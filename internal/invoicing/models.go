package invoicing

import "time"

// Client is one of the freelancer's customers — who an invoice gets
// billed to. This tool has no public-facing side (unlike the booking
// system); every user of this API is the freelancer themselves, managing
// their own clients and invoices.
type Client struct {
	ID      string
	Name    string
	Email   string
	Address string
}

// InvoiceStatus is a small closed set of valid states — using a defined
// type instead of a bare string means the compiler catches a typo like
// "Paidd" at build time instead of a user hitting it at runtime.
type InvoiceStatus string

const (
	StatusDraft InvoiceStatus = "draft"
	StatusSent  InvoiceStatus = "sent"
	StatusPaid  InvoiceStatus = "paid"
)

// LineItem is one billable row on an invoice — e.g. "Website redesign,
// 10 hours, $50/hour". Prices are stored in cents (int64), never as a
// float, to avoid classic floating-point rounding errors showing up on
// something as sensitive as money (e.g. $19.99 * 3 not landing on an
// exact cent value in float64 arithmetic).
type LineItem struct {
	Description    string
	Quantity       float64
	UnitPriceCents int64
}

// Invoice is a bill sent to one client, made up of line items plus an
// optional flat tax rate.
type Invoice struct {
	ID             string
	Number         string // e.g. "INV-0001", shown to the client
	ClientID       string
	IssueDate      time.Time
	DueDate        time.Time
	LineItems      []LineItem
	TaxRatePercent float64
	Notes          string
	Status         InvoiceStatus
}
