package unit

import (
	"math"
	"testing"
)

func TestUnitConversions(t *testing.T) {
	// 1. Length conversion test: 5 meters to centimeters
	res, err := Convert(5.0, CatLength, "Meter", "Centimeter")
	if err != nil {
		t.Fatalf("length conversion failed: %v", err)
	}
	if math.Abs(res-500.0) > 1e-9 {
		t.Errorf("expected 500.0 cm, got %f", res)
	}

	// 2. Data Size conversion test: 1 Megabyte to Kilobyte
	res, err = Convert(1.0, CatDataSize, "Megabyte", "Kilobyte")
	if err != nil {
		t.Fatalf("data conversion failed: %v", err)
	}
	if math.Abs(res-1000.0) > 1e-9 {
		t.Errorf("expected 1000.0 KB, got %f", res)
	}

	// 3. Temperature conversion test: 100°C to Fahrenheit (expected 212°F)
	res, err = Convert(100.0, CatTemperature, "Celsius", "Fahrenheit")
	if err != nil {
		t.Fatalf("temp conversion failed: %v", err)
	}
	if math.Abs(res-212.0) > 1e-9 {
		t.Errorf("expected 212.0°F, got %f", res)
	}

	// 4. Temperature conversion test: 0 Kelvin to Celsius (expected -273.15°C)
	res, err = Convert(0.0, CatTemperature, "Kelvin", "Celsius")
	if err != nil {
		t.Fatalf("temp conversion failed: %v", err)
	}
	if math.Abs(res - -273.15) > 1e-9 {
		t.Errorf("expected -273.15°C, got %f", res)
	}
}
