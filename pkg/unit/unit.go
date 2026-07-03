package unit

import (
	"fmt"
	"math"
)

type Category string

const (
	CatLength      Category = "Length"
	CatWeight      Category = "Weight/Mass"
	CatTemperature Category = "Temperature"
	CatDataSize    Category = "Data Size"
	CatArea        Category = "Area"
	CatSpeed       Category = "Speed"
)

type Unit struct {
	Name        string
	Symbol      string
	FactorToBase float64 // Multiply by this to get base unit
}

var Categories = map[Category][]Unit{
	CatLength: {
		{"Millimeter", "mm", 0.001},
		{"Centimeter", "cm", 0.01},
		{"Meter", "m", 1.0},
		{"Kilometer", "km", 1000.0},
		{"Inch", "in", 0.0254},
		{"Foot", "ft", 0.3048},
		{"Yard", "yd", 0.9144},
		{"Mile", "mi", 1609.344},
	},
	CatWeight: {
		{"Milligram", "mg", 0.001},
		{"Gram", "g", 1.0},
		{"Kilogram", "kg", 1000.0},
		{"Ounce", "oz", 28.349523125},
		{"Pound", "lb", 453.59237},
		{"Stone", "st", 6350.29318},
	},
	CatTemperature: {
		{"Celsius", "°C", 1.0},
		{"Fahrenheit", "°F", 1.0},
		{"Kelvin", "K", 1.0},
	},
	CatDataSize: {
		{"Byte", "B", 1.0},
		{"Kilobyte", "KB", 1000.0},
		{"Megabyte", "MB", 1000000.0},
		{"Gigabyte", "GB", 1000000000.0},
		{"Terabyte", "TB", 1000000000000.0},
		{"Petabyte", "PB", 1000000000000000.0},
		{"Kibibyte", "KiB", 1024.0},
		{"Mebibyte", "MiB", 1048576.0},
		{"Gibibyte", "GiB", 1073741824.0},
		{"Tebibyte", "TiB", 1099511627776.0},
	},
	CatArea: {
		{"Square Millimeter", "mm²", 0.000001},
		{"Square Centimeter", "cm²", 0.0001},
		{"Square Meter", "m²", 1.0},
		{"Hectare", "ha", 10000.0},
		{"Square Kilometer", "km²", 1000000.0},
		{"Square Inch", "in²", 0.00064516},
		{"Square Foot", "ft²", 0.09290304},
		{"Square Yard", "yd²", 0.83612736},
		{"Acre", "ac", 4046.8564224},
		{"Square Mile", "mi²", 2589988.110336},
	},
	CatSpeed: {
		{"Meter per second", "m/s", 1.0},
		{"Kilometer per hour", "km/h", 1.0 / 3.6},
		{"Mile per hour", "mph", 0.44704},
		{"Knot", "kn", 0.514444},
	},
}

// Convert converts a value from one unit to another within a category.
func Convert(val float64, cat Category, fromName, toName string) (float64, error) {
	units, exists := Categories[cat]
	if !exists {
		return 0, fmt.Errorf("unknown category: %s", cat)
	}

	if cat == CatTemperature {
		return convertTemp(val, fromName, toName)
	}

	var fromUnit, toUnit *Unit
	for i := range units {
		if units[i].Name == fromName {
			fromUnit = &units[i]
		}
		if units[i].Name == toName {
			toUnit = &units[i]
		}
	}

	if fromUnit == nil || toUnit == nil {
		return 0, fmt.Errorf("invalid units for category: %s -> %s", fromName, toName)
	}

	// Convert to base unit first, then to target unit
	valInBase := val * fromUnit.FactorToBase
	result := valInBase / toUnit.FactorToBase

	return result, nil
}

func convertTemp(val float64, from, to string) (float64, error) {
	if from == to {
		return val, nil
	}

	// Convert to Celsius first
	var celsius float64
	switch from {
	case "Celsius":
		celsius = val
	case "Fahrenheit":
		celsius = (val - 32.0) * 5.0 / 9.0
	case "Kelvin":
		celsius = val - 273.15
	default:
		return 0, fmt.Errorf("unknown temperature unit: %s", from)
	}

	// Convert Celsius to target unit
	switch to {
	case "Celsius":
		return celsius, nil
	case "Fahrenheit":
		return (celsius * 9.0 / 5.0) + 32.0, nil
	case "Kelvin":
		return celsius + 273.15, nil
	default:
		return 0, fmt.Errorf("unknown temperature unit: %s", to)
	}
}

// FormatResult formats the float with appropriate decimal places.
func FormatResult(val float64) string {
	if math.Abs(val) < 0.000001 && val != 0 {
		return fmt.Sprintf("%.6e", val)
	}
	// Trim trailing zeros but keep up to 6 decimal places
	s := fmt.Sprintf("%.6f", val)
	for s[len(s)-1] == '0' {
		s = s[:len(s)-1]
	}
	if s[len(s)-1] == '.' {
		s = s[:len(s)-1]
	}
	return s
}
