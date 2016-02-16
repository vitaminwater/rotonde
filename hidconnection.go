package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/GeertJohan/go.hid"
	"github.com/HackerLoop/rotonde/shared"
	log "github.com/Sirupsen/logrus"
)

const ROTONDE_VENDOR_ID = 0x0042
const FEATURE_STOP = 0x0
const FEATURE_START = 0x1
const FEATURE_DEFINITION = 0xff
const MaxHIDFrameSize = 64
const HeaderLength = 4

func StartHID(d *Dispatcher) {
	var isOpen, openned, closed = func() (func(*hid.DeviceInfo) bool, func(*hid.DeviceInfo), func(*hid.DeviceInfo)) {
		var mutex = new(sync.Mutex)
		var openPorts = map[string]bool{}

		var deviceId = func(device *hid.DeviceInfo) string {
			return fmt.Sprintf("%x:%x:%s", device.VendorId, device.ProductId, device.SerialNumber)
		}

		return func(device *hid.DeviceInfo) bool {
				mutex.Lock()
				defer mutex.Unlock()
				isOpen, ok := openPorts[deviceId(device)]
				return ok && isOpen
			}, func(device *hid.DeviceInfo) {
				mutex.Lock()
				defer mutex.Unlock()
				openPorts[deviceId(device)] = true
			}, func(device *hid.DeviceInfo) {
				mutex.Lock()
				defer mutex.Unlock()
				openPorts[deviceId(device)] = false
			}
	}()

	go func() {
		for {
			devices, err := hid.Enumerate(ROTONDE_VENDOR_ID, 0x00)
			if err != nil {
				time.Sleep(1 * time.Second)
				continue
			}

			for _, device := range devices {
				if isOpen(device) {
					continue
				}
				cc, err := hid.Open(device.VendorId, device.ProductId, device.SerialNumber)
				if err != nil {
					log.Warning(err)
					log.Warningf("Failing device is: 0x%04x:0x%04x serial: %s", device.VendorId, device.ProductId, device.SerialNumber)
					time.Sleep(1 * time.Second)
					continue
				}
				log.Infof("HID device successfully openned 0x%04x:0x%04x serial: %s", device.VendorId, device.ProductId, device.SerialNumber)

				openned(device)
				go func() {
					defer closed(device)
					if err := startHIDConnection(device, cc, d); err != nil {
						log.Warning(err)
						time.Sleep(time.Second * 3)
					}
				}()
			}
			time.Sleep(1 * time.Second)
		}
	}()

	log.Infof("HID Listening for vendorId: 0x%04x", ROTONDE_VENDOR_ID)
}

func startHIDConnection(device *hid.DeviceInfo, cc *hid.Device, d *Dispatcher) error {
	defer cc.Close()

	c := NewConnection()
	d.AddConnection(c)
	defer c.Close()

	if _, err := cc.SendFeatureReport([]byte{0x00, FEATURE_DEFINITION}); err != nil {
		log.Warning(err)
	}

	errChan := make(chan error)
	var connErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		fixedLengthWriteBuffer := make([]byte, MaxHIDFrameSize+1)
		fixedLengthWriteBuffer[0] = 0x0
		for {
			select {
			case dispatcherPacket := <-c.InChan:
				if _, ok := dispatcherPacket.(rotonde.Definition); ok == true {
					log.Info("USB skipping Definition messages")
					continue
				}
				jsonPacket, err := rotonde.ToJSON(dispatcherPacket)
				if err != nil {
					log.Warning(err)
					continue
				}

				first := true
				currentOffset := 0
				length := len(jsonPacket)
				for currentOffset < length {
					headerLength := 0
					if first {
						headerLength = HeaderLength
					}
					toWriteLength := length - currentOffset
					// packet on the HID link can't be > MaxHIDFrameSize, split it if it's the case.
					if toWriteLength > MaxHIDFrameSize-headerLength {
						toWriteLength = MaxHIDFrameSize - headerLength
					}

					if first {
						fixedLengthWriteBuffer[1] = 0x3c
						fixedLengthWriteBuffer[2] = 0x40
						fixedLengthWriteBuffer[3] = byte(length)
						fixedLengthWriteBuffer[4] = byte(length >> 8)
						first = false
					}
					copy(fixedLengthWriteBuffer[headerLength+1:], jsonPacket[currentOffset:currentOffset+toWriteLength])

					n, err := cc.Write(fixedLengthWriteBuffer)
					if err != nil {
						log.Warning(err)
						break
					}
					if n > headerLength {
						currentOffset += n - headerLength - 1
					}
				}

			case connErr = <-errChan:
				return
			}
		}
	}()

	wg.Add(1)
	go frameReader(&wg, cc, c, errChan)

	log.Info("Treating messages")
	wg.Wait()
	log.Infof("HID Connection 0x%04x:0x%04x closed", device.VendorId, device.ProductId)
	return connErr
}

func frameReader(wg *sync.WaitGroup, cc *hid.Device, c *Connection, errChan chan error) {
	defer wg.Done()
	var buffer bytes.Buffer
	var version uint8
	var length uint16
	var crc uint8
	packet := make([]byte, MaxHIDFrameSize)

	readNBytes := func(n int) error {
		if buffer.Len() >= n {
			return nil
		}
		for buffer.Len() < n {
			n, err := cc.Read(packet)
			if err != nil {
				return err
			}
			if n == 0 {
				return fmt.Errorf("Empty message usually means disconnection")
			}

			buffer.Write(packet[0:n])
		}
		return nil
	}

	readUpToFrame := func() error {
		for {
			if _, err := buffer.ReadBytes(0x3c); err != nil {
				if err = readNBytes(64); err != nil {
					return err
				}
				continue
			}
			break
		}
		return nil
	}

	for {
		if err := readUpToFrame(); err != nil {
			errChan <- err
			return
		}

		if err := readNBytes(HeaderLength); err != nil {
			errChan <- err
			return
		}
		if err := binary.Read(&buffer, binary.LittleEndian, &version); err != nil {
			errChan <- err
			return
		}
		if err := binary.Read(&buffer, binary.LittleEndian, &length); err != nil {
			errChan <- err
			return
		}

		if err := readNBytes(int(length)); err != nil {
			errChan <- err
			return
		}
		body := make([]byte, length)
		buffer.Read(body)

		if err := readNBytes(1); err != nil {
			errChan <- err
			return
		}
		if err := binary.Read(&buffer, binary.LittleEndian, &crc); err != nil {
			errChan <- err
			return
		}

		dispatcherPacket, err := rotonde.FromJSON(bytes.NewReader(body))
		if err != nil {
			errChan <- fmt.Errorf("Failed to decode packet")
			return
		}
		c.OutChan <- dispatcherPacket
	}
}
