package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"invoicing-tool/internal/invoicing"
)

func newTestServer() *Server {
	return NewServer(invoicing.NewMemoryStore(), "admin", "secret")
}

func testRoutes(s *Server) http.Handler {
	return s.Routes(fstest.MapFS{})
}

func authed(req *http.Request) *http.Request {
	req.SetBasicAuth("admin", "secret")
	return req
}

func TestClientsRequireAuth(t *testing.T) {
	srv := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/clients", nil)
	rec := httptest.NewRecorder()
	testRoutes(srv).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func createTestClient(t *testing.T, srv *Server) ClientDTO {
	t.Helper()
	body, _ := json.Marshal(CreateClientRequest{Name: "Acme Co", Email: "billing@acme.test"})
	req := authed(httptest.NewRequest(http.MethodPost, "/api/clients", bytes.NewReader(body)))
	rec := httptest.NewRecorder()
	testRoutes(srv).ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("create client: status = %d, body: %s", rec.Code, rec.Body.String())
	}
	var client ClientDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &client); err != nil {
		t.Fatalf("bad JSON: %v", err)
	}
	return client
}

func TestCreateInvoice_HappyPath(t *testing.T) {
	srv := newTestServer()
	client := createTestClient(t, srv)

	reqBody, _ := json.Marshal(CreateInvoiceRequest{
		ClientID:  client.ID,
		IssueDate: "2026-08-01",
		DueDate:   "2026-08-15",
		LineItems: []CreateInvoiceLineItemRequest{
			{Description: "Website redesign", Quantity: 10, UnitPriceCents: 5000},
		},
		TaxRatePercent: 10,
	})
	req := authed(httptest.NewRequest(http.MethodPost, "/api/invoices", bytes.NewReader(reqBody)))
	rec := httptest.NewRecorder()
	testRoutes(srv).ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var invoice InvoiceDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &invoice); err != nil {
		t.Fatalf("bad JSON: %v", err)
	}

	if invoice.Number != "INV-0001" {
		t.Errorf("invoice number = %q, want INV-0001", invoice.Number)
	}
	if invoice.SubtotalCents != 50000 {
		t.Errorf("subtotal = %d, want 50000", invoice.SubtotalCents)
	}
	if invoice.TotalCents != 55000 {
		t.Errorf("total (with 10%% tax) = %d, want 55000", invoice.TotalCents)
	}
	if invoice.Client.Name != "Acme Co" {
		t.Errorf("expected embedded client name 'Acme Co', got %q", invoice.Client.Name)
	}
	if invoice.Status != "draft" {
		t.Errorf("new invoice status = %q, want draft", invoice.Status)
	}
}

func TestCreateInvoice_RejectsUnknownClient(t *testing.T) {
	srv := newTestServer()

	body, _ := json.Marshal(CreateInvoiceRequest{
		ClientID: "does-not-exist",
		LineItems: []CreateInvoiceLineItemRequest{
			{Description: "Work", Quantity: 1, UnitPriceCents: 1000},
		},
	})
	req := authed(httptest.NewRequest(http.MethodPost, "/api/invoices", bytes.NewReader(body)))
	rec := httptest.NewRecorder()
	testRoutes(srv).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUpdateInvoiceStatus_HappyPath(t *testing.T) {
	srv := newTestServer()
	client := createTestClient(t, srv)

	createBody, _ := json.Marshal(CreateInvoiceRequest{
		ClientID: client.ID,
		LineItems: []CreateInvoiceLineItemRequest{
			{Description: "Work", Quantity: 1, UnitPriceCents: 1000},
		},
	})
	createReq := authed(httptest.NewRequest(http.MethodPost, "/api/invoices", bytes.NewReader(createBody)))
	createRec := httptest.NewRecorder()
	testRoutes(srv).ServeHTTP(createRec, createReq)

	var created InvoiceDTO
	json.Unmarshal(createRec.Body.Bytes(), &created)

	statusBody, _ := json.Marshal(UpdateStatusRequest{Status: "paid"})
	statusReq := authed(httptest.NewRequest(http.MethodPost, "/api/invoices/"+created.ID+"/status", bytes.NewReader(statusBody)))
	statusRec := httptest.NewRecorder()
	testRoutes(srv).ServeHTTP(statusRec, statusReq)

	if statusRec.Code != http.StatusOK {
		t.Fatalf("status update: status = %d, body: %s", statusRec.Code, statusRec.Body.String())
	}

	var updated InvoiceDTO
	json.Unmarshal(statusRec.Body.Bytes(), &updated)
	if updated.Status != "paid" {
		t.Errorf("status = %q, want paid", updated.Status)
	}
}

func TestUpdateInvoiceStatus_RejectsInvalidStatus(t *testing.T) {
	srv := newTestServer()
	client := createTestClient(t, srv)

	createBody, _ := json.Marshal(CreateInvoiceRequest{
		ClientID: client.ID,
		LineItems: []CreateInvoiceLineItemRequest{
			{Description: "Work", Quantity: 1, UnitPriceCents: 1000},
		},
	})
	createReq := authed(httptest.NewRequest(http.MethodPost, "/api/invoices", bytes.NewReader(createBody)))
	createRec := httptest.NewRecorder()
	testRoutes(srv).ServeHTTP(createRec, createReq)

	var created InvoiceDTO
	json.Unmarshal(createRec.Body.Bytes(), &created)

	statusBody, _ := json.Marshal(UpdateStatusRequest{Status: "not-a-real-status"})
	statusReq := authed(httptest.NewRequest(http.MethodPost, "/api/invoices/"+created.ID+"/status", bytes.NewReader(statusBody)))
	statusRec := httptest.NewRecorder()
	testRoutes(srv).ServeHTTP(statusRec, statusReq)

	if statusRec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", statusRec.Code, http.StatusBadRequest)
	}
}
