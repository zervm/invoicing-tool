package invoicing

import (
	"errors"
	"fmt"
	"sync"
)

var ErrNotFound = errors.New("not found")

// Store is everything the API layer needs from persistence. As with the
// booking system, this is an interface specifically so a real database
// implementation can replace MemoryStore later without any changes to
// the HTTP handlers built on top of it.
type Store interface {
	ListClients() []Client
	CreateClient(c Client) (Client, error)
	GetClient(id string) (Client, bool)

	ListInvoices() []Invoice
	CreateInvoice(inv Invoice) (Invoice, error)
	GetInvoice(id string) (Invoice, bool)
	UpdateInvoiceStatus(id string, status InvoiceStatus) (Invoice, error)
}

// MemoryStore keeps everything in memory, guarded by a mutex since
// multiple HTTP requests can arrive concurrently. Nothing here survives
// a restart — the trade-off for zero external dependencies in an
// environment with no network access to fetch a database driver.
type MemoryStore struct {
	mu             sync.Mutex
	clients        map[string]Client
	invoices       map[string]Invoice
	nextClientID   int
	nextInvoiceID  int
	nextInvoiceNum int
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		clients:  make(map[string]Client),
		invoices: make(map[string]Invoice),
	}
}

func (s *MemoryStore) ListClients() []Client {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Client, 0, len(s.clients))
	for _, c := range s.clients {
		out = append(out, c)
	}
	return out
}

func (s *MemoryStore) CreateClient(c Client) (Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextClientID++
	c.ID = fmt.Sprintf("clt-%d", s.nextClientID)
	s.clients[c.ID] = c
	return c, nil
}

func (s *MemoryStore) GetClient(id string) (Client, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.clients[id]
	return c, ok
}

func (s *MemoryStore) ListInvoices() []Invoice {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Invoice, 0, len(s.invoices))
	for _, inv := range s.invoices {
		out = append(out, inv)
	}
	return out
}

// CreateInvoice assigns both an internal ID (used in URLs/API calls) and
// a human-facing invoice number (used in the actual PDF/print view) —
// deliberately two different things. The ID can be an opaque string; the
// number is what a client expects to see and reference ("per invoice
// INV-0004"), so it needs to be sequential and presentable.
func (s *MemoryStore) CreateInvoice(inv Invoice) (Invoice, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.clients[inv.ClientID]; !ok {
		return Invoice{}, fmt.Errorf("client %q does not exist", inv.ClientID)
	}

	s.nextInvoiceID++
	s.nextInvoiceNum++
	inv.ID = fmt.Sprintf("inv-%d", s.nextInvoiceID)
	inv.Number = fmt.Sprintf("INV-%04d", s.nextInvoiceNum)
	if inv.Status == "" {
		inv.Status = StatusDraft
	}
	s.invoices[inv.ID] = inv
	return inv, nil
}

func (s *MemoryStore) GetInvoice(id string) (Invoice, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	inv, ok := s.invoices[id]
	return inv, ok
}

func (s *MemoryStore) UpdateInvoiceStatus(id string, status InvoiceStatus) (Invoice, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inv, ok := s.invoices[id]
	if !ok {
		return Invoice{}, ErrNotFound
	}
	inv.Status = status
	s.invoices[id] = inv
	return inv, nil
}
