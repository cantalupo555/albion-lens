package components

import (
	"fmt"
	"math"
)

// FormatNumber formats a number based on the full flag.
// If full is true, returns the full number (e.g., 4984).
// If full is false, returns abbreviated form (e.g., 4.9k) with truncation (floor).
func FormatNumber(amount int64, full bool) string {
	if full {
		return fmt.Sprintf("%d", amount)
	}
	// Abbreviated format with truncation (floor) instead of rounding
	if amount >= 1000000 {
		val := math.Floor(float64(amount)/100000.0) / 10.0
		return fmt.Sprintf("%.1fM", val)
	} else if amount >= 1000 {
		val := math.Floor(float64(amount)/100.0) / 10.0
		return fmt.Sprintf("%.1fk", val)
	}
	return fmt.Sprintf("%d", amount)
}
