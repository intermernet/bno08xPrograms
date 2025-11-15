// Package main provides a diagnostic tool to test BNO08x I2C connectivity
// and help troubleshoot "operation timed out" errors.
package main

import (
	"machine"
	"time"

	"tinygo.org/x/drivers/bno08x"
)

func main() {
	time.Sleep(2 * time.Second)
	println("=== BNO08x I2C Diagnostic Tool ===")
	println()

	// Initialize I2C bus
	println("Step 1: Initializing I2C bus...")
	i2c := machine.I2C0
	err := i2c.Configure(machine.I2CConfig{
		Frequency: 400 * machine.KHz,
	})
	if err != nil {
		println("FAILED: Could not configure I2C:", err.Error())
		return
	}
	println("SUCCESS: I2C configured at 400 KHz")
	println()

	// Test I2C connectivity
	println("Step 2: Testing I2C connectivity...")
	addresses := []uint16{0x4A, 0x4B}
	foundAddress := uint16(0)

	for _, addr := range addresses {
		println("  Trying address 0x", formatHex(uint8(addr)), "...")
		buf := make([]byte, 4)
		err := i2c.Tx(addr, nil, buf)
		if err == nil {
			println("  FOUND: Device responds at 0x", formatHex(uint8(addr)))
			foundAddress = addr
			break
		} else {
			println("  No response:", err.Error())
		}
	}

	if foundAddress == 0 {
		println()
		println("ERROR: No BNO08x device found on I2C bus")
		println("Troubleshooting:")
		println("  1. Check wiring (SDA, SCL, VCC, GND)")
		println("  2. Verify 3.3V power supply")
		println("  3. Check I2C pull-up resistors (2.2K - 10K to 3.3V)")
		println("  4. Try different I2C pins if available")
		return
	}
	println()

	// Initialize sensor
	println("Step 3: Initializing BNO08x sensor...")
	sensor := bno08x.New(i2c)

	config := bno08x.Config{
		Address:      foundAddress,
		StartupDelay: 200 * time.Millisecond,
	}

	println("  Using extended startup delay (200ms)...")
	err = sensor.Configure(config)
	if err != nil {
		println("FAILED:", err.Error())
		println()
		println("Troubleshooting:")
		println("  1. The sensor may need a hardware reset")
		println("  2. Try connecting RST pin and add to config:")
		println("     config.ResetPin = machine.D2  // or your chosen pin")
		println("  3. Power cycle the sensor")
		println("  4. Increase StartupDelay to 500ms or 1s")
		return
	}
	println("SUCCESS: Sensor initialized")
	println()

	// Get product IDs
	println("Step 4: Reading product information...")
	ids := sensor.ProductIDs()
	if ids.NumEntries > 0 {
		id := ids.Entries[0]
		println("  Part Number:", id.PartNumber)
		println("  Build Number:", id.BuildNumber)
		println("  Version:", id.VersionMajor, ".", id.VersionMinor, ".", id.VersionPatch)
	} else {
		println("  No product IDs available")
	}
	println()

	// Enable a test sensor
	println("Step 5: Enabling sensors...")
	println("  Enabling Game Rotation Vector at 10Hz...")
	// Game rotation vector doesn't need magnetometer, often more reliable
	err = sensor.EnableReport(bno08x.SensorGameRotationVector, 100000) // 10 Hz
	if err != nil {
		println("FAILED:", err.Error())
		return
	}
	println("  Enabling Raw Accelerometer at 10Hz...")
	err = sensor.EnableReport(bno08x.SensorRawAccelerometer, 100000) // 10 Hz
	if err != nil {
		println("FAILED:", err.Error())
		return
	}
	println("SUCCESS: Sensors enabled")
	println()

	// Give the sensor time to start producing data
	println("Waiting for sensor to start producing data...")
	time.Sleep(2 * time.Second)

	// Read a few samples
	println("Step 6: Reading sensor data...")
	println("(Polling for 10 seconds...)")
	successCount := 0
	startTime := time.Now()
	attempts := 0
	serviceErrors := 0

	for time.Since(startTime) < 10*time.Second {
		attempts++

		// Service the sensor to poll for data
		err := sensor.Service()
		if err != nil {
			serviceErrors++
			if serviceErrors < 5 {
				println("  Service error:", err.Error())
			}
		}

		// Try to get an event
		event, ok := sensor.GetSensorEvent()
		if ok {
			if event.ID() == bno08x.SensorGameRotationVector {
				successCount++
				q := event.Quaternion()
				println("  GRV Sample", successCount, ": Q =", q.Real, q.I, q.J, q.K)
			} else if event.ID() == bno08x.SensorRawAccelerometer {
				successCount++
				a := event.RawAccelerometer()
				println("  Raw Accel Sample", successCount, ": X=", a.X, "Y=", a.Y, "Z=", a.Z)
			} else {
				println("  Received unexpected sensor type:", uint8(event.ID()))
			}
		}

		time.Sleep(10 * time.Millisecond)
	}

	println()
	println("Polling complete: Made", attempts, "attempts")
	println()

	if successCount > 0 {
		println("=== DIAGNOSTIC PASSED ===")
		println("Your BNO08x sensor is working correctly!")
		println("Received", successCount, "valid sensor readings")
	} else {
		println("=== WARNING ===")
		println("Sensor initialized but no data received")
		println("This may indicate a sensor configuration issue")
	}
}

func formatHex(b uint8) string {
	const hex = "0123456789ABCDEF"
	return string([]byte{hex[b>>4], hex[b&0x0F]})
}
