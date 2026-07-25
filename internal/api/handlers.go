package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"invoicing-tool/internal/invoicing"
)

func (s *Server) handleListClients(w http.ResponseWriter, r *http.Request) {
	clients := s.store.ListClients()
	dtos := make([]ClientDTO, len(clients))
	for i, c := range clients {
		dtos[i] = toClientDTO(c)
	}
	writeJSON(w, http.StatusOK, dtos)
}

func (s *Server) handleCreateClient(w http.ResponseWriter, r *http.Request) {
	var req CreateClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	created, err := s.store.CreateClient(invoicing.Client{
		Name:    req.Name,
		Email:   req.Email,
		Address: req.Address,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create client")
		return
	}

	writeJSON(w, http.StatusCreated, toClientDTO(created))
}

// invoiceToDTO looks up the invoice's client so the DTO can embed full
// client details (needed by the print view) rather than just an ID.
func (s *Server) invoiceToDTO(inv invoicing.Invoice) InvoiceDTO {
	client, _ := s.store.GetClient(inv.ClientID)
	return toInvoiceDTO(inv, client)
}

func (s *Server) handleListInvoices(w http.ResponseWriter, r *http.Request) {
	invoices := s.store.ListInvoices()
	dtos := make([]InvoiceDTO, len(invoices))
	for i, inv := range invoices {
		dtos[i] = s.invoiceToDTO(inv)
	}
	writeJSON(w, http.StatusOK, dtos)
}

func (s *Server) handleCreateInvoice(w http.ResponseWriter, r *http.Request) {
	var req CreateInvoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if strings.TrimSpace(req.ClientID) == "" {
		writeError(w, http.StatusBadRequest, "client_id is required")
		return
	}
	if len(req.LineItems) == 0 {
		writeError(w, http.StatusBadRequest, "at least one line item is required")
		return
	}

	issueDate := time.Now()
	if req.IssueDate != "" {
		parsed, err := parseDate(req.IssueDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, "issue_date must be YYYY-MM-DD")
			return
		}
		issueDate = parsed
	}

	var dueDate time.Time
	if req.DueDate != "" {
		parsed, err := parseDate(req.DueDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, "due_date must be YYYY-MM-DD")
			return
		}
		dueDate = parsed
	} else {
		// Sensible default: due 14 days after issue, a common freelance
		// payment term (net-14) — better than leaving it as a zero date.
		dueDate = issueDate.AddDate(0, 0, 14)
	}

	items := make([]invoicing.LineItem, len(req.LineItems))
	for i, li := range req.LineItems {
		if strings.TrimSpace(li.Description) == "" {
			writeError(w, http.StatusBadRequest, "every line item needs a description")
			return
		}
		items[i] = invoicing.LineItem{
			Description:    li.Description,
			Quantity:       li.Quantity,
			UnitPriceCents: li.UnitPriceCents,
		}
	}

	created, err := s.store.CreateInvoice(invoicing.Invoice{
		ClientID:       req.ClientID,
		IssueDate:      issueDate,
		DueDate:        dueDate,
		LineItems:      items,
		TaxRatePercent: req.TaxRatePercent,
		Notes:          req.Notes,
	})
	if err != nil {
		// The only way CreateInvoice fails today is an unknown client_id.
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, s.invoiceToDTO(created))
}

func (s *Server) handleGetInvoice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	inv, ok := s.store.GetInvoice(id)
	if !ok {
		writeError(w, http.StatusNotFound, "invoice not found")
		return
	}
	writeJSON(w, http.StatusOK, s.invoiceToDTO(inv))
}

var validStatuses = map[string]invoicing.InvoiceStatus{
	"draft": invoicing.StatusDraft,
	"sent":  invoicing.StatusSent,
	"paid":  invoicing.StatusPaid,
}

func (s *Server) handleUpdateInvoiceStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	status, ok := validStatuses[req.Status]
	if !ok {
		writeError(w, http.StatusBadRequest, "status must be one of: draft, sent, paid")
		return
	}

	updated, err := s.store.UpdateInvoiceStatus(id, status)
	if err != nil {
		if errors.Is(err, invoicing.ErrNotFound) {
			writeError(w, http.StatusNotFound, "invoice not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not update status")
		return
	}

	writeJSON(w, http.StatusOK, s.invoiceToDTO(updated))
}
