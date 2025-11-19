// Package main demonstrates using the BNO08x sensor to control a NeoPixel LED
// based on orientation. Roll, Pitch, and Yaw control Red, Green, and Blue values.
package main

import (
	"image/color"
	"machine"
	"math"
	"time"

	"tinygo.org/x/drivers/bno08x"
	"tinygo.org/x/drivers/ws2812"
)

const ledPin = machine.WS2812

func main() {
	time.Sleep(2 * time.Second) // Wait for sensor to power up

	println("BNO08x NeoPixel Control")
	println("======================")

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

	// Enable Game Rotation Vector reports at 50Hz (20000 microseconds)
	err = sensor.EnableReport(bno08x.SensorGameRotationVector, 20000)
	if err != nil {
		println("Failed to enable game rotation vector:", err.Error())
		return
	}

	// Initialize NeoPixel
	ledPin.Configure(machine.PinConfig{Mode: machine.PinOutput})
	neo := ws2812.New(ledPin)
	led := make([]color.RGBA, 1) // Single RGB LED

	println("Starting LED control...")
	println("Roll -> Red, Pitch -> Green, Yaw -> Blue")

	// Main loop - read quaternions, convert to Euler angles, and control LED
	for {
		event, ok := sensor.GetSensorEvent()
		if ok && event.ID() == bno08x.SensorGameRotationVector {
			q := event.Quaternion()

			// Convert quaternion to Euler angles (radians)
			roll, pitch, yaw := quaternionToEuler(q)

			// Convert angles to RGB values (0-255)
			// Map -90° to +90° range to 0-255
			red := angleToRGB(roll)
			green := angleToRGB(pitch)
			blue := angleToRGB(yaw)

			// Update LED color
			led[0].R = red
			led[0].G = green
			led[0].B = blue
			neo.WriteColors(led)

			// Log values to serial console
			println("Roll:", formatFloat(roll*180.0/math.Pi), "° -> R:", red,
				"| Pitch:", formatFloat(pitch*180.0/math.Pi), "° -> G:", green,
				"| Yaw:", formatFloat(yaw*180.0/math.Pi), "° -> B:", blue)
		}
		time.Sleep(20 * time.Millisecond)
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

// angleToRGB converts an angle in radians to an RGB value (0-255)
// Maps -90° to +90° to the full 0-255 range, clamping values outside this range
func angleToRGB(angle float32) uint8 {
	// Convert radians to degrees
	degrees := angle * 180.0 / math.Pi

	// Clamp to -90° to +90° range
	if degrees < -90.0 {
		degrees = -90.0
	}
	if degrees > 90.0 {
		degrees = 90.0
	}

	// Map -90° to +90° to 0-255
	// Add 90 to shift range to 0-180, then scale to 0-255
	normalized := (degrees + 90.0) / 180.0
	value := normalized * 255.0

	return uint8(value)
}

// formatFloat formats a float32 with reasonable precision
func formatFloat(f float32) string {
	// Simple formatting for embedded systems without fmt
	val := int32(f * 100)
	whole := val / 100
	frac := val % 100
	if frac < 0 {
		frac = -frac
	}

	sign := ""
	if val < 0 && whole == 0 {
		sign = "-"
	}

	return sign + itoa(int(whole)) + "." + itoa2(int(frac))
}

// itoa converts an integer to string
func itoa(n int) string {
	if n == 0 {
		return "0"
	}

	negative := n < 0
	if negative {
		n = -n
	}

	// Use fixed-size buffer to avoid allocations
	var buf [10]byte
	i := len(buf) - 1
	for n > 0 {
		buf[i] = byte('0' + n%10)
		n /= 10
		i--
	}

	if negative {
		return "-" + string(buf[i+1:])
	}
	return string(buf[i+1:])
}

// itoa2 converts an integer to a 2-digit string (for fractional part)
func itoa2(n int) string {
	if n >= 100 {
		n = 99
	}
	d0 := n / 10
	d1 := n % 10
	return string([]byte{byte('0' + d0), byte('0' + d1)})
}
