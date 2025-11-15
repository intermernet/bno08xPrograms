// Package main - Hybrid test: Use driver Configure(), then raw I2C reads
package main

import (
	"encoding/binary"
	"machine"
	"time"

	"tinygo.org/x/drivers/bno08x"
)

func main() {
	time.Sleep(2 * time.Second)
	println("=== Hybrid Test: Driver init + Raw I2C reads ===")
	println()

	// Initialize I2C bus
	i2c := machine.I2C0
	err := i2c.Configure(machine.I2CConfig{
		Frequency: 400 * machine.KHz,
	})
	if err != nil {
		println("Failed to configure I2C:", err.Error())
		return
	}

	println("Step 1: Use driver to initialize sensor")
	sensor := bno08x.New(i2c)
	err = sensor.Configure(bno08x.Config{})
	if err != nil {
		println("Failed to configure sensor:", err.Error())
		return
	}
	println("  Sensor initialized via driver")
	println()

	println("Step 2: Use driver to enable Game Rotation Vector")
	err = sensor.EnableReport(bno08x.SensorGameRotationVector, 10000)
	if err != nil {
		println("Failed to enable report:", err.Error())
		return
	}
	println("  Report enabled via driver")
	time.Sleep(100 * time.Millisecond)
	println()

	println("Step 3: Raw I2C polling for data (like channel_debug)")
	addr := uint16(0x4A)
	reportCount := 0
	channelCounts := make(map[uint8]int)

	for i := 0; i < 100; i++ {
		// Read header
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

			// Check if it's a sensor report channel (3, 4, or 5)
			if channel == 3 || channel == 4 || channel == 5 {
				reportCount++
				println("  Sensor report", reportCount, "on channel", channel, "length:", length)
				if length > 4 {
					print("    First bytes:")
					for j := 4; j < int(length) && j < 12; j++ {
						print(" ", packet[j])
					}
					println()
				}
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

	if reportCount > 0 {
		println()
		println("SUCCESS! Received", reportCount, "sensor reports via raw I2C")
		println("This means the sensor IS configured correctly by the driver.")
		println("The issue is in how the driver reads the data.")
	} else {
		println()
		println("FAILURE: No sensor reports received even with raw I2C")
		println("This means the driver's configuration didn't work.")
	}
}
