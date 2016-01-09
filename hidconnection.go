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

func StartHID(d *Dispatcher) {
	var isOpen, openned, closed = func() (func(*hid.DeviceInfo) bool, func(*hid.DeviceInfo), func(*hid.DeviceInfo)) {
		var mutex = new(sync.Mutex)
		var openPorts = map[string]bool{}
		return func(device *hid.DeviceInfo) bool {
				mutex.Lock()
				defer mutex.Unlock()
				var port = fmt.Sprintf("%x:%x", device.VendorId, device.ProductId)
				isOpen, ok := openPorts[port]
				return ok && isOpen
			}, func(device *hid.DeviceInfo) {
				mutex.Lock()
				defer mutex.Unlock()
				var port = fmt.Sprintf("%x:%x", device.VendorId, device.ProductId)
				openPorts[port] = true
			}, func(device *hid.DeviceInfo) {
				mutex.Lock()
				defer mutex.Unlock()
				var port = fmt.Sprintf("%x:%x", device.VendorId, device.ProductId)
				openPorts[port] = false
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

	errChan := make(chan error)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		fixedLengthWriteBuffer := make([]byte, MaxHIDFrameSize)
		for {
			select {
			case dispatcherPacket := <-c.InChan:
				jsonPacket, err := rotonde.ToJSON(dispatcherPacket)
				if err != nil {
					log.Warning(err)
					continue
				}

				currentOffset := 0
				for currentOffset < len(jsonPacket) {
					toWriteLength := len(jsonPacket) - currentOffset
					// packet on the HID link can't be > MaxHIDFrameSize, split it if it's the case.
					if toWriteLength > MaxHIDFrameSize-2 {
						toWriteLength = MaxHIDFrameSize - 2
					}

					// USB HID link requires a reportID and packet length as first bytes
					copy(fixedLengthWriteBuffer, jsonPacket[currentOffset:currentOffset+toWriteLength])

					n, err := cc.Write(fixedLengthWriteBuffer)
					if err != nil {
						log.Warning(err)
						errChan <- err
						return
					}
					if n > 2 {
						currentOffset += n - 2
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
	var length uint8
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
				continue
			}

			buffer.Write(packet[0:n])
		}
		return nil
	}

	readUpToFrame := func() {
		for {
			if _, err := buffer.ReadBytes(0x3c); err != nil {
				readNBytes(64)
				continue
			}
			break
		}
	}

	for {
		readUpToFrame()

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
