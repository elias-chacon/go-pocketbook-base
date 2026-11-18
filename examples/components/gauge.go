package components

import (
	"go-pocketbook-base/internal/toolkit/widget"
)

// GaugeExample is a semicircular meter.
func GaugeExample() widget.Widget {
	speed := widget.NewGauge(0, 240, 135, " km/h")
	speed.Label = "Speed"
	return speed
}
