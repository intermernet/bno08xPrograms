// Package main tests the SetFeature command directly
package main

import (
	"encoding/binary"
	"machine"
	"time"
)

func main() {
	time.Sleep(2 * time.Second) // Wait for sensor to power up
	println("=== BNO08x SetFeature Command Test ===")
	println()

	// Initialize I2C
	i2c := machine.I2C0
	err := i2c.Configure(machine.I2CConfig{
		Frequency: 400 * machine.KHz,
	})
	if err != nil {
		println("FAILED to configure I2C:", err.Error())
		return
	}

	address := uint16(0x4A)

	// Send soft reset
	println("Sending soft reset...")
	softReset := []byte{5, 0, 1, 0, 1}
	err = i2c.Tx(address, softReset, nil)
	if err != nil {
		println("FAILED:", err.Error())
		return
	}
	time.Sleep(500 * time.Millisecond)

	// Drain any responses
	println("Draining initial responses...")
	for i := 0; i < 5; i++ {
		header := make([]byte, 4)
		i2c.Tx(address, nil, header)
		length := binary.LittleEndian.Uint16(header[0:2])
		channel := header[2]

		if length > 0 && length < 1000 && (length&0x8000) == 0 {
			println("  Got packet, length:", length, "channel:", channel)

			// If it's channel 0, this is an advertisement - let's read and parse it
			if channel == 0 && length > 4 {
				payload := make([]byte, length-4)
				err = i2c.Tx(address, nil, payload)
				if err == nil {
					println("  Advertisement payload (first 50 bytes):")
					for j := 0; j < 50 && j < len(payload); j += 10 {
						end := j + 10
						if end > len(payload) {
							end = len(payload)
						}
						print("    ")
						for k := j; k < end; k++ {
							print(payload[k], " ")
						}
						println()
					}

					// Parse TLV (Tag-Length-Value) format
					println("  Parsing advertisement TLV tags:")
					parseAdvertisement(payload)
				}
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	println()

	// Send Initialize command (required before sensor config)
	println("Sending Initialize command...")
	initPayload := []byte{
		0xF2,                                                 // Report ID: COMMAND_REQUEST
		0x00,                                                 // Sequence number
		0x04,                                                 // Command: Initialize
		0x01,                                                 // Subcommand: System
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Padding
	}
	initFrameLen := 4 + len(initPayload)
	initFrame := make([]byte, initFrameLen)
	binary.LittleEndian.PutUint16(initFrame[0:2], uint16(initFrameLen))
	initFrame[2] = 2 // Channel: Control
	initFrame[3] = 0 // Sequence
	copy(initFrame[4:], initPayload)

	err = i2c.Tx(address, initFrame, nil)
	if err != nil {
		println("FAILED to send initialize:", err.Error())
		return
	}
	println("  SUCCESS: Initialize sent")
	time.Sleep(200 * time.Millisecond)
	println()

	// Now send a SetFeature command for Accelerometer at 100Hz (simpler sensor)
	println("Sending SetFeature command for Accelerometer (ID=0x01) at 100Hz...")

	// Build SHTP packet for control channel (2)
	// SetFeature report = 0xFD
	payload := []byte{
		0xFD,       // Report ID: SET_FEATURE
		0x01,       // Sensor ID: Accelerometer (calibrated)
		0x00,       // Flags: none
		0x00, 0x00, // Change sensitivity: 0
		0x10, 0x27, 0x00, 0x00, // Report interval: 10000 microseconds (100Hz)
		0x00, 0x00, 0x00, 0x00, // Batch interval: 0
		0x00, 0x00, 0x00, 0x00, // Sensor specific: 0
	}

	// Build SHTP frame
	frameLen := 4 + len(payload) // header + payload
	frame := make([]byte, frameLen)
	binary.LittleEndian.PutUint16(frame[0:2], uint16(frameLen))
	frame[2] = 2 // Channel: Control
	frame[3] = 0 // Sequence: 0
	copy(frame[4:], payload)

	println("  Frame length:", frameLen)
	println("  Payload:", payload)

	err = i2c.Tx(address, frame, nil)
	if err != nil {
		println("FAILED to send:", err.Error())
		return
	}
	println("  SUCCESS: Command sent")
	time.Sleep(100 * time.Millisecond) // Wait 100ms for sensor to start
	println()

	// Poll for responses
	println("Polling for sensor reports (30 attempts)...")
	for i := 0; i < 30; i++ {
		header := make([]byte, 4)
		err = i2c.Tx(address, nil, header)
		if err != nil {
			println("  Attempt", i+1, "- Read error:", err.Error())
			time.Sleep(100 * time.Millisecond)
			continue
		}

		length := binary.LittleEndian.Uint16(header[0:2])
		channel := header[2]
		seq := header[3]

		// Check for continuation bit
		if length&0x8000 != 0 {
			// No data
			time.Sleep(100 * time.Millisecond)
			continue
		}

		if length > 0 && length < 1000 {
			println("  Attempt", i+1, "- Length:", length, "Channel:", channel, "Seq:", seq)

			// Read full packet
			if length > 4 {
				remaining := make([]byte, length-4)
				err = i2c.Tx(address, nil, remaining)
				if err == nil {
					println("    Payload[0]:", remaining[0], "- might be sensor ID")
					if channel == 3 || channel == 4 || channel == 5 {
						println("    This is a sensor report channel!")
					}
				}
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	println()
	println("Test complete")
}

func parseAdvertisement(payload []byte) {
	// Advertisement uses TLV format: Tag (1 byte), Length (1 byte), Value (Length bytes)
	// Looking for channel tags (6=normal channel, 7=wake channel)
	i := 0
	for i < len(payload)-2 {
		tag := payload[i]
		length := int(payload[i+1])
		i += 2

		if i+length > len(payload) {
			break
		}

		value := payload[i : i+length]
		i += length

		if tag == 6 {
			// Normal channel
			if length > 1 {
				chanNum := value[0]
				name := string(value[1:])
				println("    Channel", chanNum, "=", name)
			}
		} else if tag == 7 {
			// Wake channel
			if length > 1 {
				chanNum := value[0]
				name := string(value[1:])
				println("    Wake Channel", chanNum, "=", name)
			}
		} else if tag == 0x80 {
			// Version string
			println("    Version:", string(value))
		}
	}
}
