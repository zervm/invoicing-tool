package invoicing

import "testing"

func TestLineTotalCents(t *testing.T) {
	tests := []struct {
		name     string
		item     LineItem
		wantCents int64
	}{
		{
			name:      "whole number quantity",
			item:      LineItem{Quantity: 3, UnitPriceCents: 2000},
			wantCents: 6000,
		},
		{
			name:      "fractional hours",
			item:      LineItem{Quantity: 2.5, UnitPriceCents: 5000},
			wantCents: 12500,
		},
		{
			name:      "quantity that could produce a rounding artifact",
			item:      LineItem{Quantity: 0.1, UnitPriceCents: 999},
			wantCents: 100, // 0.1 * 999 = 99.9, rounds to 100
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.item.LineTotalCents()
			if got != tt.wantCents {
				t.Fatalf("LineTotalCents() = %d, want %d", got, tt.wantCents)
			}
		})
	}
}

func TestInvoiceTotals(t *testing.T) {
	tests := []struct {
		name         string
		invoice      Invoice
		wantSubtotal int64
		wantTax      int64
		wantTotal    int64
	}{
		{
			name: "no tax",
			invoice: Invoice{
				LineItems: []LineItem{
					{Quantity: 10, UnitPriceCents: 5000}, // $500.00
					{Quantity: 1, UnitPriceCents: 2500},  // $25.00
				},
				TaxRatePercent: 0,
			},
			wantSubtotal: 52500,
			wantTax:      0,
			wantTotal:    52500,
		},
		{
			name: "with tax",
			invoice: Invoice{
				LineItems: []LineItem{
					{Quantity: 10, UnitPriceCents: 5000}, // $500.00 subtotal
				},
				TaxRatePercent: 20, // 20% tax
			},
			wantSubtotal: 50000,
			wantTax:      10000, // $100.00
			wantTotal:    60000, // $600.00
		},
		{
			name: "empty invoice",
			invoice: Invoice{
				LineItems:      nil,
				TaxRatePercent: 15,
			},
			wantSubtotal: 0,
			wantTax:      0,
			wantTotal:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.invoice.SubtotalCents(); got != tt.wantSubtotal {
				t.Errorf("SubtotalCents() = %d, want %d", got, tt.wantSubtotal)
			}
			if got := tt.invoice.TaxCents(); got != tt.wantTax {
				t.Errorf("TaxCents() = %d, want %d", got, tt.wantTax)
			}
			if got := tt.invoice.TotalCents(); got != tt.wantTotal {
				t.Errorf("TotalCents() = %d, want %d", got, tt.wantTotal)
			}
		})
	}
}
