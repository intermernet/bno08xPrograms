// Package main demonstrates using the BNO08x sensor to control MIDI CC values
// based on rotation (roll, pitch, yaw). This can be used for musical expression
// or control of MIDI-enabled software/hardware.
package main

import (
	"machine"
	"machine/usb/adc/midi"
	"math"
	"time"

	"tinygo.org/x/drivers/bno08x"
)

const (
	// MIDI CC numbers for each axis
	ccRoll  = 65
	ccPitch = 66
	ccYaw   = 67

	// MIDI cable (0-15)
	midiCable = 0
	// MIDI channel (0-15)
	midiChannel = 1

	// Threshold for detecting value changes (avoid sending redundant messages)
	changeThreshold = 1
)

var (
	lastRoll  uint8 = 255 // Invalid initial value to force first send
	lastPitch uint8 = 255
	lastYaw   uint8 = 255
)

func main() {
	time.Sleep(2 * time.Second) // Wait for sensor to power up

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
	err = sensor.EnableReport(bno08x.SensorGameRotationVector, 20000)
	if err != nil {
		println("Failed to enable rotation vector:", err.Error())
		return
	}

	println("Starting MIDI control...")
	println("Roll -> CC65, Pitch -> CC66, Yaw -> CC67")

	// Main loop - read quaternions, convert to Euler angles, and send MIDI CC
	for {
		event, ok := sensor.GetSensorEvent()
		if ok && event.ID() == bno08x.SensorGameRotationVector {
			q := event.Quaternion()

			// Convert quaternion to Euler angles (radians)
			roll, pitch, yaw := quaternionToEuler(q)

			// Convert angles to MIDI CC values (0-127)
			// Map -180° to +180° range to 0-127
			rollCC := angleToMIDI(roll)
			pitchCC := angleToMIDI(pitch)
			yawCC := angleToMIDI(yaw)

			// Send MIDI CC messages only if values changed significantly
			if abs(int16(rollCC)-int16(lastRoll)) >= changeThreshold {
				midi.Port().ControlChange(midiCable, midiChannel, ccRoll, rollCC)
				lastRoll = rollCC
			}

			if abs(int16(pitchCC)-int16(lastPitch)) >= changeThreshold {
				midi.Port().ControlChange(midiCable, midiChannel, ccPitch, pitchCC)
				lastPitch = pitchCC
			}

			if abs(int16(yawCC)-int16(lastYaw)) >= changeThreshold {
				midi.Port().ControlChange(midiCable, midiChannel, ccYaw, yawCC)
				lastYaw = yawCC
			}
			//println(rollCC, pitchCC, yawCC)
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

// angleToMIDI converts an angle in radians to a MIDI CC value (0-127)
// Maps -90° to +90° to the full 0-127 range, clamping values outside this range
func angleToMIDI(angle float32) uint8 {
	// Convert radians to degrees
	degrees := angle * 180.0 / math.Pi

	// Clamp to -90° to +90° range
	if degrees < -90.0 {
		degrees = -90.0
	}
	if degrees > 90.0 {
		degrees = 90.0
	}

	// Map -90° to +90° to 0-127
	// Add 90 to shift range to 0-180, then scale to 0-127
	normalized := (degrees + 90.0) / 180.0
	value := normalized * 127.0

	return uint8(value)
}

// abs returns the absolute value of an int16
func abs(x int16) int16 {
	if x < 0 {
		return -x
	}
	return x
}
