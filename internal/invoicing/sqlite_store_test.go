package invoicing

import (
	"fmt"
	"sync/atomic"
	"testing"
)

var testDBCounter int64

// newTestSQLiteStore opens a fresh in-memory SQLite database per test.
// "cache=shared" keeps the single connection (see SetMaxOpenConns(1) in
// sqlite_store.go) pointed at the same in-memory database rather than
// each connection getting its own throwaway copy — but an unqualified
// "file::memory:" name is process-global in SQLite's shared-cache mode,
// so every test would silently share (and pollute) the same database.
// Giving each store its own name avoids that cross-test leakage.
func newTestSQLiteStore(t *testing.T) *SQLiteStore {
	t.Helper()
	id := atomic.AddInt64(&testDBCounter, 1)
	dsn := fmt.Sprintf("file:testdb%d?mode=memory&cache=shared", id)
	store, err := NewSQLiteStore(dsn)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	return store
}

// These mirror the MemoryStore tests in store_test.go — the same
// business rules must hold no matter which Store implementation is
// behind the API.

func TestSQLiteStore_CreateInvoice_RequiresRealClient(t *testing.T) {
	store := newTestSQLiteStore(t)

	_, err := store.CreateInvoice(Invoice{ClientID: "does-not-exist"})
	if err == nil {
		t.Fatal("expected an error creating an invoice for a nonexistent client, got nil")
	}
}

func TestSQLiteStore_CreateInvoice_AssignsSequentialNumbers(t *testing.T) {
	store := newTestSQLiteStore(t)
	client, _ := store.CreateClient(Client{Name: "Acme Co"})

	first, err := store.CreateInvoice(Invoice{ClientID: client.ID})
	if err != nil {
		t.Fatalf("CreateInvoice: %v", err)
	}
	second, err := store.CreateInvoice(Invoice{ClientID: client.ID})
	if err != nil {
		t.Fatalf("CreateInvoice: %v", err)
	}

	if first.Number != "INV-0001" {
		t.Errorf("first invoice number = %q, want INV-0001", first.Number)
	}
	if second.Number != "INV-0002" {
		t.Errorf("second invoice number = %q, want INV-0002", second.Number)
	}
	if first.ID == second.ID {
		t.Error("two invoices got the same internal ID")
	}
}

func TestSQLiteStore_CreateInvoice_DefaultsToDraftStatus(t *testing.T) {
	store := newTestSQLiteStore(t)
	client, _ := store.CreateClient(Client{Name: "Acme Co"})

	inv, err := store.CreateInvoice(Invoice{ClientID: client.ID})
	if err != nil {
		t.Fatalf("CreateInvoice: %v", err)
	}
	if inv.Status != StatusDraft {
		t.Errorf("new invoice status = %q, want %q", inv.Status, StatusDraft)
	}
}

func TestSQLiteStore_CreateInvoice_PersistsLineItemsInOrder(t *testing.T) {
	store := newTestSQLiteStore(t)
	client, _ := store.CreateClient(Client{Name: "Acme Co"})

	created, err := store.CreateInvoice(Invoice{
		ClientID: client.ID,
		LineItems: []LineItem{
			{Description: "Design", Quantity: 2, UnitPriceCents: 5000},
			{Description: "Development", Quantity: 10, UnitPriceCents: 8000},
		},
	})
	if err != nil {
		t.Fatalf("CreateInvoice: %v", err)
	}

	reloaded, ok := store.GetInvoice(created.ID)
	if !ok {
		t.Fatal("expected invoice to be retrievable after creation")
	}
	if len(reloaded.LineItems) != 2 {
		t.Fatalf("got %d line items, want 2", len(reloaded.LineItems))
	}
	if reloaded.LineItems[0].Description != "Design" || reloaded.LineItems[1].Description != "Development" {
		t.Errorf("line items out of order: %+v", reloaded.LineItems)
	}
}

func TestSQLiteStore_UpdateInvoiceStatus(t *testing.T) {
	store := newTestSQLiteStore(t)
	client, _ := store.CreateClient(Client{Name: "Acme Co"})
	inv, _ := store.CreateInvoice(Invoice{ClientID: client.ID})

	updated, err := store.UpdateInvoiceStatus(inv.ID, StatusPaid)
	if err != nil {
		t.Fatalf("UpdateInvoiceStatus: %v", err)
	}
	if updated.Status != StatusPaid {
		t.Errorf("status = %q, want %q", updated.Status, StatusPaid)
	}

	// Confirm it actually persisted, not just returned in the response.
	reloaded, ok := store.GetInvoice(inv.ID)
	if !ok || reloaded.Status != StatusPaid {
		t.Errorf("expected persisted status to be %q, got %q (ok=%v)", StatusPaid, reloaded.Status, ok)
	}
}

func TestSQLiteStore_UpdateInvoiceStatus_NotFound(t *testing.T) {
	store := newTestSQLiteStore(t)
	if _, err := store.UpdateInvoiceStatus("does-not-exist", StatusPaid); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
