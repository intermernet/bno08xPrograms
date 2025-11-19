// An example of using the BNO08x driver
// to read rotation vector (quaternion) data from the sensor.
package main

import (
	"machine"
	"time"

	"tinygo.org/x/drivers/bno08x"
)

func main() {
	time.Sleep(2 * time.Second) // Wait for sensor to power up

	// Configure watchdog to reset if main loop stalls
	wdc := machine.WatchdogConfig{
		TimeoutMillis: 1000,
	}
	machine.Watchdog.Configure(wdc)
	machine.Watchdog.Start()

	// Initialize I2C bus
	i2c := machine.I2C0
	err := i2c.Configure(machine.I2CConfig{
		Frequency: 400 * machine.KHz,
	})
	if err != nil {
		println("Failed to configure I2C:", err.Error())
		return
	}

	// Create and configure sensor
	sensor := bno08x.New(i2c)
	err = sensor.Configure(bno08x.Config{})
	if err != nil {
		println("Failed to configure sensor:", err.Error())
		return
	}

	// Enable Game Rotation Vector reports at 100Hz (10000 microseconds = 10ms interval)
	err = sensor.EnableReport(bno08x.SensorGameRotationVector, 10000)
	if err != nil {
		println("Failed to enable game rotation vector:", err.Error())
		return
	}

	// Add a delay after enabling reports
	time.Sleep(100 * time.Millisecond)

	// Main loop - read and display quaternion data
	for {
		// Reset watchdog timer
		machine.Watchdog.Update()
		event, ok := sensor.GetSensorEvent()
		if ok && event.ID() == bno08x.SensorGameRotationVector {
			q := event.Quaternion()
			print(q.I)
			print(",")
			print(q.J)
			print(",")
			print(q.K)
			print(",")
			println(q.Real)
		}

		// 10ms delay in loop
		time.Sleep(10 * time.Millisecond)
	}
}
