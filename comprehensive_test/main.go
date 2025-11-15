// Package main - Comprehensive test following Adafruit library exactly
package main

import (
	"encoding/binary"
	"machine"
	"time"
)

func main() {
	time.Sleep(2 * time.Second) // Wait for sensor to power up
	println("=== Comprehensive BNO08x Test (Following Adafruit Exactly) ===")
	println()

	i2c := machine.I2C0
	err := i2c.Configure(machine.I2CConfig{Frequency: 400 * machine.KHz})
	if err != nil {
		println("FAILED:", err.Error())
		return
	}

	addr := uint16(0x4A)
	seq := [6]uint8{0, 0, 0, 0, 0, 0} // Sequence counters for channels 0-5

	// Step 1: Soft reset (from i2chal_open)
	println("Step 1: Soft reset")
	softReset := []byte{5, 0, 1, 0, 1}
	for attempt := 0; attempt < 5; attempt++ {
		err = i2c.Tx(addr, softReset, nil)
		if err == nil {
			break
		}
		time.Sleep(30 * time.Millisecond)
	}
	time.Sleep(300 * time.Millisecond)
	println("  Done")
	println()

	// Step 2: Drain/read advertisement
	println("Step 2: Reading advertisement")
	for i := 0; i < 10; i++ {
		header := make([]byte, 4)
		i2c.Tx(addr, nil, header)
		length := binary.LittleEndian.Uint16(header[0:2])
		if length > 0 && length < 1000 && (length&0x8000) == 0 {
			println("  Got advertisement, length:", length, "channel:", header[2])
			if length > 4 {
				payload := make([]byte, length-4)
				i2c.Tx(addr, nil, payload)
			}
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	println()

	// Step 3: Initialize command (from _init -> sh2_open)
	println("Step 3: Sending Initialize command")
	initCmd := []byte{0xF2, 0, 0x04, 0x01, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	sendOnChannel(i2c, addr, &seq, 2, initCmd)
	time.Sleep(100 * time.Millisecond)

	// Drain responses
	for i := 0; i < 5; i++ {
		header := make([]byte, 4)
		i2c.Tx(addr, nil, header)
		time.Sleep(20 * time.Millisecond)
	}
	println("  Done")
	println()

	// Step 4: Request Product IDs (from _init -> sh2_getProdIds)
	println("Step 4: Requesting Product IDs")
	prodIDReq := []byte{0xF9, 0x00}
	sendOnChannel(i2c, addr, &seq, 2, prodIDReq)
	time.Sleep(100 * time.Millisecond)

	// Read product ID response
	for i := 0; i < 10; i++ {
		header := make([]byte, 4)
		i2c.Tx(addr, nil, header)
		length := binary.LittleEndian.Uint16(header[0:2])
		if length > 0 && length < 1000 && (length&0x8000) == 0 {
			println("  Got response, length:", length, "channel:", header[2])
			if length > 4 {
				payload := make([]byte, length-4)
				i2c.Tx(addr, nil, payload)
				if len(payload) > 0 {
					println("  Response ID:", payload[0])
				}
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	println()

	// Step 5: Enable Game Rotation Vector (from enableReport)
	println("Step 5: Enabling Game Rotation Vector at 10ms (100Hz)")
	setFeature := []byte{
		0xFD,       // SET_FEATURE
		0x08,       // Game Rotation Vector
		0x00,       // Flags
		0x00, 0x00, // Change sensitivity
		0x10, 0x27, 0x00, 0x00, // 10000 microseconds
		0x00, 0x00, 0x00, 0x00, // Batch interval
		0x00, 0x00, 0x00, 0x00, // Sensor specific
	}
	sendOnChannel(i2c, addr, &seq, 2, setFeature)
	println("  Command sent")
	println()

	// Delay after enabling report (Arduino does this in setup)
	time.Sleep(100 * time.Millisecond)

	// Step 6: Poll for sensor data (from getSensorEvent -> sh2_service)
	// Following Arduino's exact approach: read header, then re-read full packet
	println("Step 6: Polling for sensor data (100 attempts, 10ms between each)")
	reportCount := 0
	for i := 0; i < 100; i++ {
		// First, read header to get packet length
		header := make([]byte, 4)
		err = i2c.Tx(addr, nil, header)
		if err != nil {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		length := binary.LittleEndian.Uint16(header[0:2])

		// Skip if no data (continuation bit set)
		if length&0x8000 != 0 {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		if length == 0 {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		length &= ^uint16(0x8000) // Mask off continuation bit

		if length > 0 && length < 500 {
			// Now re-read the FULL packet including header (Arduino's approach)
			fullPacket := make([]byte, length)
			err = i2c.Tx(addr, nil, fullPacket)
			if err != nil {
				time.Sleep(10 * time.Millisecond)
				continue
			}

			channel := fullPacket[2]

			// Check if it's a sensor report channel (3, 4, or 5)
			if channel == 3 || channel == 4 || channel == 5 {
				reportCount++
				println("  Report", reportCount, "- Length:", length, "Channel:", channel)
				if length > 4 {
					println("    Sensor ID:", fullPacket[4], "Seq:", fullPacket[5], "Status:", fullPacket[6])
				}
			} else if channel == 2 {
				// Control channel response
				if length > 4 {
					println("  Control response, ID:", fullPacket[4])
				}
			}
		}

		time.Sleep(10 * time.Millisecond)
	}

	println()
	if reportCount > 0 {
		println("SUCCESS! Received", reportCount, "sensor reports")
	} else {
		println("WARNING: No sensor reports received")
		println("The sensor responds to commands but doesn't send data.")
		println("This may indicate:")
		println("  - Sensor firmware issue")
		println("  - Missing INT pin connection")
		println("  - Sensor needs additional undocumented initialization")
	}
}

func sendOnChannel(i2c *machine.I2C, addr uint16, seq *[6]uint8, channel uint8, payload []byte) {
	frameLen := 4 + len(payload)
	frame := make([]byte, frameLen)
	binary.LittleEndian.PutUint16(frame[0:2], uint16(frameLen))
	frame[2] = channel
	frame[3] = seq[channel]
	seq[channel]++
	copy(frame[4:], payload)
	i2c.Tx(addr, frame, nil)
}
