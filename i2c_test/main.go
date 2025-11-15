// Package main provides a minimal I2C test to verify basic communication
package main

import (
	"encoding/binary"
	"machine"
	"time"
)

func main() {
	time.Sleep(2 * time.Second) // Wait for sensor to power up
	println("=== BNO08x Minimal I2C Test ===")
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
	println("I2C configured at 400 KHz")

	address := uint16(0x4A)
	println("Testing address 0x4A")
	println()

	// Test 1: Send soft reset
	println("Test 1: Sending soft reset packet...")
	softReset := []byte{5, 0, 1, 0, 1}
	err = i2c.Tx(address, softReset, nil)
	if err != nil {
		println("  FAILED:", err.Error())
		return
	}
	println("  SUCCESS: Soft reset sent")
	time.Sleep(500 * time.Millisecond)
	println()

	// Test 2: Try to read SHTP header
	println("Test 2: Reading SHTP headers (10 attempts)...")
	for i := 0; i < 10; i++ {
		header := make([]byte, 4)
		err = i2c.Tx(address, nil, header)
		if err != nil {
			println("  Attempt", i+1, "- Read error:", err.Error())
		} else {
			length := binary.LittleEndian.Uint16(header[0:2])
			channel := header[2]
			seq := header[3]
			println("  Attempt", i+1, "- Length:", length, "Channel:", channel, "Seq:", seq)

			if length > 0 && length < 1000 {
				// Try to read the full packet
				packet := make([]byte, length)
				copy(packet[:4], header)
				if length > 4 {
					remaining := make([]byte, length-4)
					err = i2c.Tx(address, nil, remaining)
					if err == nil {
						copy(packet[4:], remaining)
						println("    Full packet:", packet[:min(int(length), 20)])
					}
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	println()
	println("Test complete")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
