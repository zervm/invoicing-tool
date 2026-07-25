package invoicing

import (
	"errors"
	"testing"
)

func TestCreateInvoice_RequiresRealClient(t *testing.T) {
	store := NewMemoryStore()

	_, err := store.CreateInvoice(Invoice{ClientID: "does-not-exist"})
	if err == nil {
		t.Fatal("expected an error creating an invoice for a nonexistent client, got nil")
	}
}

func TestCreateInvoice_AssignsSequentialNumbers(t *testing.T) {
	store := NewMemoryStore()
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

func TestCreateInvoice_DefaultsToDraftStatus(t *testing.T) {
	store := NewMemoryStore()
	client, _ := store.CreateClient(Client{Name: "Acme Co"})

	inv, err := store.CreateInvoice(Invoice{ClientID: client.ID})
	if err != nil {
		t.Fatalf("CreateInvoice: %v", err)
	}
	if inv.Status != StatusDraft {
		t.Errorf("new invoice status = %q, want %q", inv.Status, StatusDraft)
	}
}

func TestUpdateInvoiceStatus(t *testing.T) {
	store := NewMemoryStore()
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

func TestUpdateInvoiceStatus_NotFound(t *testing.T) {
	store := NewMemoryStore()
	_, err := store.UpdateInvoiceStatus("does-not-exist", StatusPaid)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
