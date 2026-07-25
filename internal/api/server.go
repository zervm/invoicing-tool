package api

import (
	"crypto/subtle"
	"encoding/json"
	"io/fs"
	"net/http"

	"invoicing-tool/internal/invoicing"
)

type Server struct {
	store         invoicing.Store
	adminUser     string
	adminPassword string
}

func NewServer(store invoicing.Store, adminUser, adminPassword string) *Server {
	return &Server{store: store, adminUser: adminUser, adminPassword: adminPassword}
}

// Routes builds the full route set. Unlike the booking system, every
// /api/... route here requires auth — this whole tool is the
// freelancer's private workspace, there's no public customer-facing
// side. Only the static frontend files (which include the login page
// itself) are served without auth, since the login form has to be
// reachable before there's anything to authenticate with.
func (s *Server) Routes(static fs.FS) http.Handler {
	mux := http.NewServeMux()

	api := http.NewServeMux()
	api.HandleFunc("GET /api/clients", s.handleListClients)
	api.HandleFunc("POST /api/clients", s.handleCreateClient)
	api.HandleFunc("GET /api/invoices", s.handleListInvoices)
	api.HandleFunc("POST /api/invoices", s.handleCreateInvoice)
	api.HandleFunc("GET /api/invoices/{id}", s.handleGetInvoice)
	api.HandleFunc("POST /api/invoices/{id}/status", s.handleUpdateInvoiceStatus)

	mux.Handle("/api/", s.requireAdmin(api))
	mux.Handle("/", http.FileServer(http.FS(static)))

	return mux
}

// requireAdmin uses constant-time comparison for the credential check —
// a plain "==" string comparison returns as soon as it finds a mismatched
// byte, which leaks (via response timing) how many leading characters
// were correct. crypto/subtle.ConstantTimeCompare always takes the same
// time regardless of where a mismatch occurs.
func (s *Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()

		userOK := subtle.ConstantTimeCompare([]byte(user), []byte(s.adminUser)) == 1
		passOK := subtle.ConstantTimeCompare([]byte(pass), []byte(s.adminPassword)) == 1

		if !ok || !userOK || !passOK {
			w.Header().Set("WWW-Authenticate", `Basic realm="invoicing"`)
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}
