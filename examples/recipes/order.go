// examples/recipes/order.go
package recipes

import (
	"fmt"
	"strings"

	"go-pocketbook-base/internal/toolkit/widget"
)

// OrderForm shows numeric fields with − / + buttons, a digit mask and a total
// that is recalculated whenever a field changes.
type OrderForm struct {
	Root      widget.Widget
	Quantity  *widget.Input
	UnitPrice *widget.Input
	Coupon    *widget.Input

	total *widget.Label
}

// NewOrderForm builds the form with 1 item of R$ 19,90.
func NewOrderForm() *OrderForm {
	f := &OrderForm{total: widget.NewLabel("")}

	f.Quantity = widget.NewNumberInput(1, 0)
	f.Quantity.Min, f.Quantity.Max = 1, 20

	f.UnitPrice = widget.NewNumberInput(19.9, 2)
	f.UnitPrice.Prefix, f.UnitPrice.DecimalSeparator, f.UnitPrice.Step = "R$ ", ",", 0.5

	f.Coupon = widget.NewMaskedInput("000-000", "") // 6 digits: 123-456
	f.Coupon.NoSteppers, f.Coupon.Placeholder = true, "Coupon 000-000"

	for _, in := range []*widget.Input{f.Quantity, f.UnitPrice, f.Coupon} {
		in.OnChange = func(e *widget.Event) { f.recalculate() }
	}
	f.recalculate()

	f.total.Style.Bold = true
	f.Root = widget.NewCard("Order", "Change a field to update the total",
		f.Quantity, f.UnitPrice, f.Coupon, f.total)
	return f
}

// Total is quantity × price, with 10% off when the coupon is complete.
func (f *OrderForm) Total() float64 {
	total := f.Quantity.Number() * f.UnitPrice.Number()
	if len(f.Coupon.Value) == widget.MaskSlots(f.Coupon.Mask) {
		total *= 0.9
	}
	return total
}

func (f *OrderForm) recalculate() {
	f.total.Text = "Total: R$ " + strings.Replace(fmt.Sprintf("%.2f", f.Total()), ".", ",", 1)
}
