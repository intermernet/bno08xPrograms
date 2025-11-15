// Package main demonstrates reading multiple sensor types simultaneously
// including accelerometer, gyroscope, and magnetometer data.
package main

import (
	"machine"
	"time"

	"tinygo.org/x/drivers/bno08x"
)

func main() {
	// Initialize I2C bus
	i2c := machine.I2C0
	err := i2c.Configure(machine.I2CConfig{
		Frequency: 400 * machine.KHz,
	})
	if err != nil {
		println("Failed to configure I2C:", err.Error())
		return
	}

	println("Initializing BNO08x sensor...")

	// Create and configure sensor
	sensor := bno08x.New(i2c)
	err = sensor.Configure(bno08x.Config{})
	if err != nil {
		println("Failed to configure sensor:", err.Error())
		return
	}

	println("Sensor initialized successfully")

	// Enable multiple sensor reports at 100Hz (10000 microseconds)
	sensors := []bno08x.SensorID{
		bno08x.SensorAccelerometer,
		bno08x.SensorGyroscope,
		bno08x.SensorMagneticField,
	}

	for _, id := range sensors {
		err = sensor.EnableReport(id, 10000)
		if err != nil {
			println("Failed to enable sensor:", id, err.Error())
			return
		}
	}

	println("Reading sensor data...")

	// Track last print time for each sensor
	lastPrint := make(map[bno08x.SensorID]time.Time)
	printInterval := 500 * time.Millisecond

	// Main loop - read and display sensor data
	for {
		event, ok := sensor.GetSensorEvent()
		if !ok {
			time.Sleep(time.Millisecond)
			continue
		}

		// Rate limit printing for each sensor type
		now := time.Now()
		if now.Sub(lastPrint[event.ID]) < printInterval {
			continue
		}
		lastPrint[event.ID] = now

		// Display data based on sensor type
		switch event.ID() {
		case bno08x.SensorAccelerometer:
			a := event.Accelerometer()
			println("Accel (m/s²):", a.X, a.Y, a.Z)

		case bno08x.SensorGyroscope:
			g := event.Gyroscope()
			println("Gyro (rad/s):", g.X, g.Y, g.Z)

		case bno08x.SensorMagneticField:
			m := event.MagneticField()
			println("Mag (µT):   ", m.X, m.Y, m.Z)
		}
	}
}
