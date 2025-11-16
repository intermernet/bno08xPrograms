// Package main debugs tap detector issues by showing all sensor events
package main

import (
	"machine"
	"time"

	"tinygo.org/x/drivers/bno08x"
)

func main() {
	time.Sleep(2 * time.Second)

	println("BNO08x Tap Detector Debug")
	println("=========================")
	println()

	// Initialize I2C
	i2c := machine.I2C0
	err := i2c.Configure(machine.I2CConfig{
		Frequency: 400 * machine.KHz,
	})
	if err != nil {
		println("I2C error:", err.Error())
		return
	}

	// Create sensor
	sensor := bno08x.New(i2c)
	err = sensor.Configure(bno08x.Config{})
	if err != nil {
		println("Configure error:", err.Error())
		return
	}

	println("Sensor initialized")
	println()

	// Try enabling tap detector with different intervals
	println("Enabling tap detector...")
	err = sensor.EnableReport(bno08x.SensorTapDetector, 0)
	if err != nil {
		println("EnableReport error:", err.Error())
		return
	}

	// Also enable accelerometer as a control to see if ANY events come through
	println("Enabling accelerometer as control...")
	err = sensor.EnableReport(bno08x.SensorAccelerometer, 100000) // 10Hz
	if err != nil {
		println("Accelerometer error:", err.Error())
	}

	time.Sleep(100 * time.Millisecond)

	println()
	println("Waiting for sensor events...")
	println("(Tap detector ID: 0x10, Accelerometer ID: 0x01)")
	println()

	eventCount := 0
	tapCount := 0
	accelCount := 0
	otherSensors := make(map[uint8]int)

	lastPrint := time.Now()

	// Main loop
	for {
		event, ok := sensor.GetSensorEvent()
		if ok {
			eventCount++

			switch event.ID() {
			case bno08x.SensorTapDetector:
				tapCount++
				tap := event.TapDetector()
				println("[TAP EVENT!] Flags:", tap.Flags, "Count:", tapCount)

			case bno08x.SensorAccelerometer:
				accelCount++
				// Don't print every accel event, just count them

			default:
				otherSensors[uint8(event.ID())]++
			}

			// Print summary every 2 seconds
			if time.Since(lastPrint) > 2*time.Second {
				println()
				println("--- Event Summary ---")
				println("Total events:", eventCount)
				println("Tap events:", tapCount)
				println("Accel events:", accelCount)
				println("Other sensor IDs:")
				for id, count := range otherSensors {
					println("  Sensor", id, ":", count, "events")
				}
				println()
				lastPrint = time.Now()
			}
		}

		time.Sleep(10 * time.Millisecond)
	}
}
