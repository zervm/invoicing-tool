package api

import (
	"time"

	"invoicing-tool/internal/invoicing"
)

// As with the booking system, these DTOs keep the JSON wire format
// decoupled from the internal domain models — internal fields can be
// renamed or restructured later without breaking whatever's consuming
// this API.

type ClientDTO struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Address string `json:"address"`
}

func toClientDTO(c invoicing.Client) ClientDTO {
	return ClientDTO{ID: c.ID, Name: c.Name, Email: c.Email, Address: c.Address}
}

type CreateClientRequest struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Address string `json:"address"`
}

type LineItemDTO struct {
	Description    string  `json:"description"`
	Quantity       float64 `json:"quantity"`
	UnitPriceCents int64   `json:"unit_price_cents"`
	LineTotalCents int64   `json:"line_total_cents"`
}

// InvoiceDTO includes the computed totals (subtotal/tax/total) directly,
// so the frontend never has to reimplement money math in JavaScript —
// there is exactly one place these numbers are calculated, in Go, and
// it's covered by tests.
type InvoiceDTO struct {
	ID             string        `json:"id"`
	Number         string        `json:"number"`
	Client         ClientDTO     `json:"client"`
	IssueDate      string        `json:"issue_date"`
	DueDate        string        `json:"due_date"`
	LineItems      []LineItemDTO `json:"line_items"`
	TaxRatePercent float64       `json:"tax_rate_percent"`
	Notes          string        `json:"notes"`
	Status         string        `json:"status"`
	SubtotalCents  int64         `json:"subtotal_cents"`
	TaxCents       int64         `json:"tax_cents"`
	TotalCents     int64         `json:"total_cents"`
}

func toInvoiceDTO(inv invoicing.Invoice, client invoicing.Client) InvoiceDTO {
	items := make([]LineItemDTO, len(inv.LineItems))
	for i, li := range inv.LineItems {
		items[i] = LineItemDTO{
			Description:    li.Description,
			Quantity:       li.Quantity,
			UnitPriceCents: li.UnitPriceCents,
			LineTotalCents: li.LineTotalCents(),
		}
	}

	return InvoiceDTO{
		ID:             inv.ID,
		Number:         inv.Number,
		Client:         toClientDTO(client),
		IssueDate:      inv.IssueDate.Format("2006-01-02"),
		DueDate:        inv.DueDate.Format("2006-01-02"),
		LineItems:      items,
		TaxRatePercent: inv.TaxRatePercent,
		Notes:          inv.Notes,
		Status:         string(inv.Status),
		SubtotalCents:  inv.SubtotalCents(),
		TaxCents:       inv.TaxCents(),
		TotalCents:     inv.TotalCents(),
	}
}

type CreateInvoiceLineItemRequest struct {
	Description    string  `json:"description"`
	Quantity       float64 `json:"quantity"`
	UnitPriceCents int64   `json:"unit_price_cents"`
}

type CreateInvoiceRequest struct {
	ClientID       string                         `json:"client_id"`
	IssueDate      string                         `json:"issue_date"` // YYYY-MM-DD
	DueDate        string                         `json:"due_date"`   // YYYY-MM-DD
	LineItems      []CreateInvoiceLineItemRequest `json:"line_items"`
	TaxRatePercent float64                        `json:"tax_rate_percent"`
	Notes          string                         `json:"notes"`
}

type UpdateStatusRequest struct {
	Status string `json:"status"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// parseDate is a tiny shared helper — every date in this API travels as
// a plain YYYY-MM-DD string, not a full timestamp, since invoices deal
// in calendar dates, not moments in time.
func parseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}
