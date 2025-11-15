// Package main - Debug channel mapping and data flow
package main

import (
	"encoding/binary"
	"machine"
	"time"
)

func main() {
	time.Sleep(2 * time.Second)
	println("=== BNO08x Channel Debug ===")
	println()

	i2c := machine.I2C0
	err := i2c.Configure(machine.I2CConfig{Frequency: 400 * machine.KHz})
	if err != nil {
		println("FAILED:", err.Error())
		return
	}

	addr := uint16(0x4A)
	seq := [6]uint8{0, 0, 0, 0, 0, 0}

	// Soft reset
	println("1. Soft reset")
	softReset := []byte{5, 0, 1, 0, 1}
	i2c.Tx(addr, softReset, nil)
	time.Sleep(300 * time.Millisecond)
	println("   Done")
	println()

	// Read advertisement and parse channel assignments
	println("2. Reading advertisement")
	header := make([]byte, 4)
	i2c.Tx(addr, nil, header)
	advertLen := binary.LittleEndian.Uint16(header[0:2]) & 0x7FFF
	println("   Length:", advertLen, "Channel:", header[2])

	if advertLen > 4 && advertLen < 500 {
		advert := make([]byte, advertLen)
		i2c.Tx(addr, nil, advert)

		// Parse advertisement tags to find channel assignments
		// Advertisement format: [header(4)] [reportID(1)] [tags...]
		println("   Parsing channel assignments:")
		cursor := 4 // Skip 4-byte header (length is re-read in full packet)
		reportID := advert[cursor]
		println("   Report ID:", reportID)
		cursor++ // Skip report ID

		channels := make(map[string]uint8)
		currentChan := uint8(0)

		for cursor < int(advertLen) {
			if cursor+1 >= int(advertLen) {
				break
			}
			tag := advert[cursor]
			length := advert[cursor+1]
			cursor += 2

			if cursor+int(length) > int(advertLen) {
				break
			}

			// TAG_NORMAL_CHANNEL = 4
			if tag == 4 && length == 1 {
				currentChan = advert[cursor]
				println("     Normal channel:", currentChan)
			}
			// TAG_CHANNEL_NAME = 8
			if tag == 8 && length > 0 && currentChan > 0 {
				name := string(advert[cursor : cursor+int(length)])
				channels[name] = currentChan
				println("     Channel", currentChan, "=", name)
			}

			cursor += int(length)
		}
		println()

		// Show what we found
		println("   Channel map:")
		for name, ch := range channels {
			println("    ", name, "->", ch)
		}
	}
	println()

	// Send initialize command (channel 2 = control)
	println("3. Initialize command")
	initCmd := []byte{0x02} // COMMAND_INITIALIZE
	sendOnChannel(i2c, addr, &seq, 2, initCmd)
	time.Sleep(100 * time.Millisecond)
	println("   Sent")
	println()

	// Enable Game Rotation Vector
	println("4. Enable Game Rotation Vector")
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
	time.Sleep(100 * time.Millisecond)
	println("   Sent")
	println()

	// Poll and show ALL data on ALL channels
	println("5. Polling all channels (100 attempts, 10ms between each)")
	channelCounts := make(map[uint8]int)

	for i := 0; i < 100; i++ {
		header := make([]byte, 4)
		err = i2c.Tx(addr, nil, header)
		if err != nil {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		length := binary.LittleEndian.Uint16(header[0:2])

		// Check continuation bit
		if length&0x8000 != 0 {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		if length == 0 {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		length &= 0x7FFF

		if length > 4 && length < 500 {
			// Re-read full packet
			packet := make([]byte, length)
			err = i2c.Tx(addr, nil, packet)
			if err != nil {
				time.Sleep(10 * time.Millisecond)
				continue
			}

			channel := packet[2]
			channelCounts[channel]++

			println("   Packet on channel", channel, "length:", length, "seq:", packet[3])
			if length > 4 {
				print("     Payload bytes:")
				for j := 4; j < int(length) && j < 12; j++ {
					print(" ", packet[j])
				}
				println()
			}
		}

		time.Sleep(10 * time.Millisecond)
	}

	println()
	println("Summary - packets per channel:")
	for ch := uint8(0); ch < 6; ch++ {
		if count, ok := channelCounts[ch]; ok {
			println("  Channel", ch, ":", count, "packets")
		}
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
