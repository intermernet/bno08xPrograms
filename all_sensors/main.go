// Package main runs a comprehensive test of all sensors on a BNO08x.
// It prints product id entries and fields, enables all sensible reports,
// then counts and prints a summary of received events every 5 seconds.
package main

import (
	"runtime"
	"time"

	"machine"

	"tinygo.org/x/drivers/bno08x"
)

func main() {
	m := new(runtime.MemStats)
	// Small delay for host to be ready
	time.Sleep(2 * time.Second)

	println("BNO08x Comprehensive Sensor Test")
	println("================================")

	// Initialize I2C
	i2c := machine.I2C0
	err := i2c.Configure(machine.I2CConfig{Frequency: 400 * machine.KHz})
	if err != nil {
		println("I2C configure error:", err.Error())
		return
	}

	// Create device and configure (default)
	sensor := bno08x.New(i2c)
	if err := sensor.Configure(bno08x.Config{}); err != nil {
		println("Sensor configure error:", err.Error())
		return
	}

	println("Sensor initialized")

	// Print product ID entries if available
	prod := sensor.ProductIDs()
	println("Product ID entries:")
	for i := 0; i < int(prod.NumEntries); i++ {
		p := prod.Entries[i]
		println(" Entry", i)
		println("  PartNumber:", p.PartNumber)
		println("  Version:", p.VersionMajor, ".", p.VersionMinor, ".", p.VersionPatch)
		println("  Build:", p.BuildNumber)
		println("  ResetCause:", p.ResetCause)
	}

	// List of sensor IDs to enable
	sensors := []bno08x.SensorID{
		bno08x.SensorRawAccelerometer,
		bno08x.SensorAccelerometer,
		bno08x.SensorLinearAcceleration,
		bno08x.SensorGravity,
		bno08x.SensorRawGyroscope,
		bno08x.SensorGyroscope,
		bno08x.SensorGyroscopeUncalibrated,
		bno08x.SensorRawMagnetometer,
		bno08x.SensorMagneticField,
		bno08x.SensorMagneticFieldUncalibrated,
		bno08x.SensorRotationVector,
		bno08x.SensorGameRotationVector,
		bno08x.SensorGeomagneticRotationVector,
		bno08x.SensorPressure,
		bno08x.SensorAmbientLight,
		bno08x.SensorHumidity,
		bno08x.SensorProximity,
		bno08x.SensorTemperature,
		bno08x.SensorTapDetector,
		bno08x.SensorStepDetector,
		bno08x.SensorStepCounter,
		bno08x.SensorSignificantMotion,
		bno08x.SensorStabilityClassifier,
		bno08x.SensorShakeDetector,
		bno08x.SensorFlipDetector,
		bno08x.SensorPickupDetector,
		bno08x.SensorStabilityDetector,
		bno08x.SensorPersonalActivityClassifier,
		bno08x.SensorSleepDetector,
		bno08x.SensorTiltDetector,
		bno08x.SensorPocketDetector,
		bno08x.SensorCircleDetector,
	}

	// Sensor name mapping (define early for use during enable)
	sensorNames := map[uint8]string{
		0x01: "Accelerometer",
		0x02: "Gyroscope",
		0x03: "Magnetic Field",
		0x04: "Linear Acceleration",
		0x05: "Rotation Vector",
		0x06: "Gravity",
		0x07: "Gyroscope Uncalibrated",
		0x08: "Game Rotation Vector",
		0x09: "Geomagnetic Rotation Vector",
		0x0A: "Pressure",
		0x0B: "Ambient Light",
		0x0C: "Humidity",
		0x0D: "Proximity",
		0x0E: "Temperature",
		0x0F: "Magnetic Field Uncalibrated",
		0x10: "Tap Detector",
		0x11: "Step Counter",
		0x12: "Significant Motion",
		0x13: "Stability Classifier",
		0x14: "Raw Accelerometer",
		0x15: "Raw Gyroscope",
		0x16: "Raw Magnetometer",
		0x18: "Step Detector",
		0x19: "Shake Detector",
		0x1A: "Flip Detector",
		0x1B: "Pickup Detector",
		0x1C: "Stability Detector",
		0x1E: "Personal Activity Classifier",
		0x1F: "Sleep Detector",
		0x20: "Tilt Detector",
		0x21: "Pocket Detector",
		0x22: "Circle Detector",
	}

	println("Enabling reports (where supported)...")
	for _, id := range sensors {
		idByte := uint8(id)
		name := sensorNames[idByte]
		if name == "" {
			name = "Unknown"
		}
		// Use 10ms default (100Hz) for most sensors; 0 means disable
		if err := sensor.EnableReport(id, 10000); err != nil {
			println(" Enable failed for 0x"+formatHex(idByte)+" ("+name+"):", err.Error())
		} else {
			println(" Enabled 0x" + formatHex(idByte) + " (" + name + ")")
		}
		// Small pause between requests
		time.Sleep(20 * time.Millisecond)
	}

	// Build list of enabled sensors for tracking
	enabledSensors := make([]uint8, len(sensors))
	for i, id := range sensors {
		enabledSensors[i] = uint8(id)
	}

	// Counters - initialize all enabled sensors to 0
	totalEvents := 0
	counts := make(map[uint8]int)
	// Track which sensors have received events
	hasEvents := make(map[uint8]bool)
	for _, id := range enabledSensors {
		counts[id] = 0
		hasEvents[id] = false
	}

	lastPrint := time.Now()

	println("Listening for events. Summary every 5s...")

	for {
		event, ok := sensor.GetSensorEvent()
		if ok {
			totalEvents++
			idByte := uint8(event.ID())
			counts[idByte]++
			hasEvents[idByte] = true
		}

		if time.Since(lastPrint) >= 5*time.Second {
			println()
			println("--- Cumulative Summary ---")
			println("Total events:", totalEvents)
			// Print counts for each enabled sensor in order
			for _, id := range enabledSensors {
				c := counts[id]
				name := sensorNames[id]
				if name == "" {
					name = "Unknown"
				}
				println(" 0x"+formatHex(id)+" ("+name+"):", c)
			}
			println("--- End Summary ---")
			runtime.ReadMemStats(m)
			println("Alloc =", m.Alloc, "TotalAlloc =", m.TotalAlloc, "Sys =", m.Sys)
			lastPrint = time.Now()
		}

		time.Sleep(5 * time.Millisecond)
	}
}

// formatHex formats a byte as a 2-character hex string
func formatHex(b uint8) string {
	const hex = "0123456789ABCDEF"
	return string([]byte{hex[b>>4], hex[b&0x0F]})
}

// printEventDetails prints human-readable details of the last sensor event
func printEventDetails(id uint8, ev *bno08x.SensorValue) {
	switch id {
	// Vector3 sensors (accelerometer, gyro, mag, etc.)
	case 0x01: // Accelerometer
		v := ev.Accelerometer()
		println("    X:", formatFloat(v.X), "Y:", formatFloat(v.Y), "Z:", formatFloat(v.Z), "m/s²")
	case 0x02: // Gyroscope
		v := ev.Gyroscope()
		println("    X:", formatFloat(v.X), "Y:", formatFloat(v.Y), "Z:", formatFloat(v.Z), "rad/s")
	case 0x03: // Magnetic Field
		v := ev.MagneticField()
		println("    X:", formatFloat(v.X), "Y:", formatFloat(v.Y), "Z:", formatFloat(v.Z), "µT")
	case 0x04: // Linear Acceleration
		v := ev.LinearAcceleration()
		println("    X:", formatFloat(v.X), "Y:", formatFloat(v.Y), "Z:", formatFloat(v.Z), "m/s²")
	case 0x06: // Gravity
		v := ev.Gravity()
		println("    X:", formatFloat(v.X), "Y:", formatFloat(v.Y), "Z:", formatFloat(v.Z), "m/s²")

	// Quaternion sensors (rotation vectors)
	case 0x05: // Rotation Vector
		q := ev.Quaternion()
		println("    i:", formatFloat(q.I), "j:", formatFloat(q.J), "k:", formatFloat(q.K), "real:", formatFloat(q.Real))
		println("    Accuracy:", formatFloat(ev.QuaternionAccuracy()), "rad")
	case 0x08: // Game Rotation Vector
		q := ev.Quaternion()
		println("    i:", formatFloat(q.I), "j:", formatFloat(q.J), "k:", formatFloat(q.K), "real:", formatFloat(q.Real))
	case 0x09: // Geomagnetic Rotation Vector
		q := ev.Quaternion()
		println("    i:", formatFloat(q.I), "j:", formatFloat(q.J), "k:", formatFloat(q.K), "real:", formatFloat(q.Real))
		println("    Accuracy:", formatFloat(ev.QuaternionAccuracy()), "rad")

	// Uncalibrated sensors
	case 0x07: // Gyroscope Uncalibrated
		v := ev.GyroscopeUncal()
		println("    X:", formatFloat(v.X), "Y:", formatFloat(v.Y), "Z:", formatFloat(v.Z), "rad/s")
		println("    BiasX:", formatFloat(v.BiasX), "BiasY:", formatFloat(v.BiasY), "BiasZ:", formatFloat(v.BiasZ))
	case 0x0F: // Magnetic Field Uncalibrated
		v := ev.MagneticFieldUncal()
		println("    X:", formatFloat(v.X), "Y:", formatFloat(v.Y), "Z:", formatFloat(v.Z), "µT")
		println("    BiasX:", formatFloat(v.BiasX), "BiasY:", formatFloat(v.BiasY), "BiasZ:", formatFloat(v.BiasZ))

	// Raw sensors
	case 0x14: // Raw Accelerometer
		v := ev.RawAccelerometer()
		println("    X:", v.X, "Y:", v.Y, "Z:", v.Z, "Timestamp:", v.Timestamp)
	case 0x15: // Raw Gyroscope
		v := ev.RawGyroscope()
		println("    X:", v.X, "Y:", v.Y, "Z:", v.Z, "Temp:", v.Temperature, "Timestamp:", v.Timestamp)
	case 0x16: // Raw Magnetometer
		v := ev.RawMagnetometer()
		println("    X:", v.X, "Y:", v.Y, "Z:", v.Z, "Timestamp:", v.Timestamp)

	// Environmental sensors
	case 0x0A: // Pressure
		println("    Pressure:", formatFloat(ev.Pressure()), "hPa")
	case 0x0B: // Ambient Light
		println("    Light:", formatFloat(ev.AmbientLight()), "lux")
	case 0x0C: // Humidity
		println("    Humidity:", formatFloat(ev.Humidity()), "%")
	case 0x0D: // Proximity
		println("    Proximity:", formatFloat(ev.Proximity()), "cm")
	case 0x0E: // Temperature
		println("    Temperature:", formatFloat(ev.Temperature()), "°C")

	// Activity detectors
	case 0x10: // Tap Detector
		tap := ev.TapDetector()
		flags := tap.Flags
		axis := ""
		if flags&0x01 != 0 {
			axis = "X"
		} else if flags&0x04 != 0 {
			axis = "Y"
		} else if flags&0x10 != 0 {
			axis = "Z"
		}
		dir := "+"
		if flags&0x02 == 0 && flags&0x01 != 0 {
			dir = "-"
		} else if flags&0x08 == 0 && flags&0x04 != 0 {
			dir = "-"
		} else if flags&0x20 == 0 && flags&0x10 != 0 {
			dir = "-"
		}
		tapType := "Single"
		if flags&0x40 != 0 {
			tapType = "Double"
		}
		println("    "+tapType+" tap on", axis+dir, "axis (flags:", flags, ")")

	case 0x11: // Step Counter
		sc := ev.StepCounter()
		println("    Steps:", sc.Count, "Latency:", sc.Latency, "ms")

	case 0x12: // Significant Motion
		println("    Motion detected")

	case 0x13: // Stability Classifier
		sc := ev.StabilityClassifier()
		stability := sc.Classification
		desc := "Unknown"
		switch stability {
		case 1:
			desc = "On Table"
		case 2:
			desc = "Stationary"
		case 3:
			desc = "Stable"
		case 4:
			desc = "Motion"
		}
		println("    Stability:", desc)

	case 0x18: // Step Detector
		sd := ev.StepDetector()
		println("    Step detected (latency:", sd.Latency, "ms)")

	case 0x19: // Shake Detector
		sd := ev.ShakeDetector()
		println("    Shake detected (value:", sd.Shake, ")")

	case 0x1A: // Flip Detector
		println("    Flip detected")

	case 0x1B: // Pickup Detector
		println("    Pickup detected")

	case 0x1C: // Stability Detector
		println("    Stability event:", ev.StabilityDetector())

	case 0x1E: // Personal Activity Classifier
		pac := ev.PersonalActivityClassifier()
		activity := pac.MostLikelyState
		desc := "Unknown"
		switch activity {
		case 1:
			desc = "In Vehicle"
		case 2:
			desc = "On Bicycle"
		case 3:
			desc = "On Foot"
		case 4:
			desc = "Still"
		case 5:
			desc = "Tilting"
		case 6:
			desc = "Walking"
		case 7:
			desc = "Running"
		case 8:
			desc = "On Stairs"
		}
		println("    Activity:", desc, "(confidence:", pac.Confidence[activity], "%)")

	case 0x1F: // Sleep Detector
		println("    Sleep state:", ev.SleepDetector())

	case 0x20: // Tilt Detector
		println("    Tilt detected")

	case 0x21: // Pocket Detector
		println("    Pocket state:", ev.PocketDetector())

	case 0x22: // Circle Detector
		println("    Circle state:", ev.CircleDetector())

	default:
		// Unknown sensor type, don't print details
	}
}

// formatFloat formats a float32 with reasonable precision
func formatFloat(f float32) string {
	// Simple formatting for embedded systems without fmt
	val := int32(f * 1000)
	whole := val / 1000
	frac := val % 1000
	if frac < 0 {
		frac = -frac
	}

	sign := ""
	if val < 0 && whole == 0 {
		sign = "-"
	}

	return sign + itoa(int(whole)) + "." + itoa3(int(frac))
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

// itoa3 converts an integer to a 3-digit string (for fractional part)
func itoa3(n int) string {
	if n >= 1000 {
		n = 999
	}
	d0 := n / 100
	d1 := (n / 10) % 10
	d2 := n % 10
	return string([]byte{byte('0' + d0), byte('0' + d1), byte('0' + d2)})
}
