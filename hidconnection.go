package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/GeertJohan/go.hid"
	"github.com/HackerLoop/rotonde/shared"
	log "github.com/Sirupsen/logrus"
)

const ROTONDE_VENDOR_ID = 0x03EB
const MaxHIDFrameSize = 64
const HeaderLength = 4

func StartHID(d *Dispatcher) {
	var isOpen, openned, closed = func() (func(*hid.DeviceInfo) bool, func(*hid.DeviceInfo), func(*hid.DeviceInfo)) {
		var mutex = new(sync.Mutex)
		var openPorts = map[string]bool{}

		var deviceId = func(device *hid.DeviceInfo) string {
			return fmt.Sprintf("%x:%x", device.VendorId, device.ProductId)
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
				cc, err := hid.Open(device.VendorId, device.ProductId, "")
				if err != nil {
					log.Warning(err)
					time.Sleep(1 * time.Second)
					continue
				}
				log.Infof("HID device successfully openned 0x%04x:0x%04x", device.VendorId, device.ProductId)

				go func() {
					openned(device)
					startHIDConnection(device, cc, d)
					closed(device)
				}()
			}
			time.Sleep(1 * time.Second)
		}
	}()

	log.Infof("HID Listening for vendorId: 0x%04x", ROTONDE_VENDOR_ID)
}

func startHIDConnection(device *hid.DeviceInfo, cc *hid.Device, d *Dispatcher) {
	defer cc.Close()

	c := NewConnection()
	d.AddConnection(c)
	defer c.Close()

	if _, err := cc.SendFeatureReport([]byte{0x0, 0x0, 0x0, 0x0, 0x0}); err != nil {
		log.Warning(err)
	}

	errChan := make(chan error)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		fixedLengthWriteBuffer := make([]byte, MaxHIDFrameSize)
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

				currentOffset := 0
				for currentOffset < len(jsonPacket) {
					toWriteLength := len(jsonPacket) - currentOffset
					// packet on the HID link can't be > MaxHIDFrameSize, split it if it's the case.
					if toWriteLength > MaxHIDFrameSize-HeaderLength {
						toWriteLength = MaxHIDFrameSize - HeaderLength
					}

					fixedLengthWriteBuffer[0] = 0x3c
					fixedLengthWriteBuffer[1] = 0x42
					fixedLengthWriteBuffer[2] = byte(toWriteLength)
					fixedLengthWriteBuffer[3] = byte(toWriteLength >> 8)
					copy(fixedLengthWriteBuffer[HeaderLength:], jsonPacket[currentOffset:currentOffset+toWriteLength])

					n, err := cc.Write(fixedLengthWriteBuffer)
					if err != nil {
						log.Warning(err)
						errChan <- err
						return
					}
					if n > HeaderLength {
						currentOffset += n - HeaderLength
					}
				}

			case <-errChan:
				return
			}
		}
	}()

	frameChan := make(chan io.Reader, 10)
	wg.Add(1)
	go frameReader(&wg, cc, frameChan, errChan)

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			reader, ok := <-frameChan
			if ok == false {
				log.Warning("framChan channel closed")
				return
			}
			dispatcherPacket, err := rotonde.FromJSON(reader)
			if err != nil {
				log.Warning("Failed to decode packet")
				return
			}
			c.OutChan <- dispatcherPacket
		}
	}()
	log.Info("Treating messages")
	wg.Wait()
	log.Infof("HID Connection 0x%04x:0x%04x closed", device.VendorId, device.ProductId)
}

func frameReader(wg *sync.WaitGroup, cc *hid.Device, c chan io.Reader, errChan chan error) {
	defer wg.Done()
	defer close(c)
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

		if err := readNBytes(2); err != nil {
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

		c <- bytes.NewReader(body)
	}
}
