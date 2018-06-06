package vespyr

import (
	"math"
)

// TruncateFloat truncates the float to the specified number of
// decimal places.
func TruncateFloat(f float64, precision uint) float64 {
	x := math.Pow(10, float64(precision))
	return float64(int(f*x)) / x
}
