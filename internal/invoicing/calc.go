package invoicing

import "math"

// LineTotalCents is quantity * unit price, rounded to the nearest cent.
// Quantity is a float (so "2.5 hours" works), which is exactly why the
// result needs explicit rounding — without it, a value like 2.5 * 3333
// could carry tiny floating-point fractions of a cent that have no
// business appearing on an invoice.
func (li LineItem) LineTotalCents() int64 {
	return int64(math.Round(li.Quantity * float64(li.UnitPriceCents)))
}

// SubtotalCents sums every line item's total, before tax.
func (inv Invoice) SubtotalCents() int64 {
	var sum int64
	for _, li := range inv.LineItems {
		sum += li.LineTotalCents()
	}
	return sum
}

// TaxCents is the tax amount owed on the subtotal, rounded to the
// nearest cent. A zero tax rate (the common case for many freelancers,
// depending on jurisdiction) simply yields zero.
func (inv Invoice) TaxCents() int64 {
	subtotal := inv.SubtotalCents()
	return int64(math.Round(float64(subtotal) * inv.TaxRatePercent / 100))
}

// TotalCents is what the client actually owes: subtotal plus tax.
func (inv Invoice) TotalCents() int64 {
	return inv.SubtotalCents() + inv.TaxCents()
}
