// Package main demonstrates converting quaternion data to Euler angles
// (roll, pitch, yaw) for easier visualization of sensor orientation.
package main

import (
	"machine"
	"math"
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

	// Enable rotation vector reports at 50Hz (20000 microseconds)
	err = sensor.EnableReport(bno08x.SensorRotationVector, 20000)
	if err != nil {
		println("Failed to enable rotation vector:", err.Error())
		return
	}

	println("Reading orientation data...")
	println("Format: Roll Pitch Yaw (degrees)")

	// Main loop - read quaternions and convert to Euler angles
	for {
		event, ok := sensor.GetSensorEvent()
		if ok && event.ID() == bno08x.SensorRotationVector {
			q := event.Quaternion()

			// Convert quaternion to Euler angles
			roll, pitch, yaw := quaternionToEuler(q)

			// Convert radians to degrees
			rollDeg := roll * 180.0 / math.Pi
			pitchDeg := pitch * 180.0 / math.Pi
			yawDeg := yaw * 180.0 / math.Pi

			println(rollDeg, pitchDeg, yawDeg)
		}

		time.Sleep(100 * time.Millisecond)
	}
}

// quaternionToEuler converts a quaternion to Euler angles (roll, pitch, yaw).
// Roll is rotation around X axis, Pitch around Y axis, Yaw around Z axis.
// All angles are returned in radians.
func quaternionToEuler(q bno08x.Quaternion) (roll, pitch, yaw float32) {
	// Roll (x-axis rotation)
	sinr_cosp := 2.0 * (q.Real*q.I + q.J*q.K)
	cosr_cosp := 1.0 - 2.0*(q.I*q.I+q.J*q.J)
	roll = float32(math.Atan2(float64(sinr_cosp), float64(cosr_cosp)))

	// Pitch (y-axis rotation)
	sinp := 2.0 * (q.Real*q.J - q.K*q.I)
	if math.Abs(float64(sinp)) >= 1 {
		pitch = float32(math.Copysign(math.Pi/2, float64(sinp)))
	} else {
		pitch = float32(math.Asin(float64(sinp)))
	}

	// Yaw (z-axis rotation)
	siny_cosp := 2.0 * (q.Real*q.K + q.I*q.J)
	cosy_cosp := 1.0 - 2.0*(q.J*q.J+q.K*q.K)
	yaw = float32(math.Atan2(float64(siny_cosp), float64(cosy_cosp)))

	return roll, pitch, yaw
}
