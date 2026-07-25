package invoicing

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite driver, no cgo required
)

// SQLiteStore is a Store implementation backed by a real SQLite database.
// It satisfies the exact same Store interface as MemoryStore, so the API
// layer and every handler are completely unaware which one is active.
//
// Unlike the booking system (one flat table), invoicing data is
// genuinely relational: an invoice has many line items. That's modeled
// here as two tables joined on invoice_id, rather than flattening line
// items into a JSON blob column — the point of using SQL at all.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens (or creates) a SQLite database at path and creates
// the schema if it doesn't exist yet.
func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// SQLite only really supports one writer at a time; capping the pool
	// at 1 connection avoids "database is locked" errors under
	// concurrent requests, trading a little throughput for correctness.
	db.SetMaxOpenConns(1)

	store := &SQLiteStore{db: db}
	if err := store.migrate(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *SQLiteStore) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS clients (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL,
			address TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS invoices (
			id TEXT PRIMARY KEY,
			number TEXT NOT NULL,
			client_id TEXT NOT NULL REFERENCES clients(id),
			issue_date TEXT NOT NULL,
			due_date TEXT NOT NULL,
			tax_rate_percent REAL NOT NULL,
			notes TEXT NOT NULL,
			status TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS invoice_line_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			invoice_id TEXT NOT NULL REFERENCES invoices(id),
			position INTEGER NOT NULL,
			description TEXT NOT NULL,
			quantity REAL NOT NULL,
			unit_price_cents INTEGER NOT NULL
		);
		CREATE TABLE IF NOT EXISTS counters (
			name TEXT PRIMARY KEY,
			value INTEGER NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}
	return nil
}

func (s *SQLiteStore) nextID(tx *sql.Tx, counter string) (int, error) {
	if _, err := tx.Exec(
		`INSERT INTO counters (name, value) VALUES (?, 1)
		 ON CONFLICT(name) DO UPDATE SET value = value + 1`,
		counter,
	); err != nil {
		return 0, err
	}
	var next int
	if err := tx.QueryRow(`SELECT value FROM counters WHERE name = ?`, counter).Scan(&next); err != nil {
		return 0, err
	}
	return next, nil
}

func (s *SQLiteStore) ListClients() []Client {
	rows, err := s.db.Query(`SELECT id, name, email, address FROM clients`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var out []Client
	for rows.Next() {
		var c Client
		if err := rows.Scan(&c.ID, &c.Name, &c.Email, &c.Address); err == nil {
			out = append(out, c)
		}
	}
	return out
}

func (s *SQLiteStore) CreateClient(c Client) (Client, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return Client{}, err
	}
	defer tx.Rollback()

	next, err := s.nextID(tx, "client")
	if err != nil {
		return Client{}, err
	}
	c.ID = fmt.Sprintf("clt-%d", next)

	if _, err := tx.Exec(
		`INSERT INTO clients (id, name, email, address) VALUES (?, ?, ?, ?)`,
		c.ID, c.Name, c.Email, c.Address,
	); err != nil {
		return Client{}, err
	}
	if err := tx.Commit(); err != nil {
		return Client{}, err
	}
	return c, nil
}

func (s *SQLiteStore) GetClient(id string) (Client, bool) {
	var c Client
	err := s.db.QueryRow(`SELECT id, name, email, address FROM clients WHERE id = ?`, id).
		Scan(&c.ID, &c.Name, &c.Email, &c.Address)
	if err != nil {
		return Client{}, false
	}
	return c, true
}

// lineItemsFor loads an invoice's line items in the order they were
// entered — the position column preserves that order since SQL rows have
// no inherent ordering guarantee otherwise.
func (s *SQLiteStore) lineItemsFor(invoiceID string) []LineItem {
	rows, err := s.db.Query(
		`SELECT description, quantity, unit_price_cents FROM invoice_line_items
		 WHERE invoice_id = ? ORDER BY position`,
		invoiceID,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var out []LineItem
	for rows.Next() {
		var li LineItem
		if err := rows.Scan(&li.Description, &li.Quantity, &li.UnitPriceCents); err == nil {
			out = append(out, li)
		}
	}
	return out
}

func (s *SQLiteStore) scanInvoice(row *sql.Row) (Invoice, error) {
	var inv Invoice
	var issueDate, dueDate string
	if err := row.Scan(&inv.ID, &inv.Number, &inv.ClientID, &issueDate, &dueDate,
		&inv.TaxRatePercent, &inv.Notes, &inv.Status); err != nil {
		return Invoice{}, err
	}
	inv.IssueDate, _ = time.Parse(time.RFC3339, issueDate)
	inv.DueDate, _ = time.Parse(time.RFC3339, dueDate)
	inv.LineItems = s.lineItemsFor(inv.ID)
	return inv, nil
}

func (s *SQLiteStore) ListInvoices() []Invoice {
	rows, err := s.db.Query(`SELECT id FROM invoices`)
	if err != nil {
		return nil
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()

	var out []Invoice
	for _, id := range ids {
		if inv, ok := s.GetInvoice(id); ok {
			out = append(out, inv)
		}
	}
	return out
}

// CreateInvoice mirrors MemoryStore's rule that an invoice must belong to
// a client that actually exists, and assigns both an internal ID and a
// sequential, client-facing invoice number.
func (s *SQLiteStore) CreateInvoice(inv Invoice) (Invoice, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return Invoice{}, err
	}
	defer tx.Rollback()

	var exists int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM clients WHERE id = ?`, inv.ClientID).Scan(&exists); err != nil {
		return Invoice{}, err
	}
	if exists == 0 {
		return Invoice{}, fmt.Errorf("client %q does not exist", inv.ClientID)
	}

	idNum, err := s.nextID(tx, "invoice")
	if err != nil {
		return Invoice{}, err
	}
	numNum, err := s.nextID(tx, "invoice_number")
	if err != nil {
		return Invoice{}, err
	}
	inv.ID = fmt.Sprintf("inv-%d", idNum)
	inv.Number = fmt.Sprintf("INV-%04d", numNum)
	if inv.Status == "" {
		inv.Status = StatusDraft
	}

	if _, err := tx.Exec(
		`INSERT INTO invoices (id, number, client_id, issue_date, due_date, tax_rate_percent, notes, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		inv.ID, inv.Number, inv.ClientID, inv.IssueDate.Format(time.RFC3339), inv.DueDate.Format(time.RFC3339),
		inv.TaxRatePercent, inv.Notes, inv.Status,
	); err != nil {
		return Invoice{}, err
	}

	for i, li := range inv.LineItems {
		if _, err := tx.Exec(
			`INSERT INTO invoice_line_items (invoice_id, position, description, quantity, unit_price_cents)
			 VALUES (?, ?, ?, ?, ?)`,
			inv.ID, i, li.Description, li.Quantity, li.UnitPriceCents,
		); err != nil {
			return Invoice{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return Invoice{}, err
	}
	return inv, nil
}

func (s *SQLiteStore) GetInvoice(id string) (Invoice, bool) {
	row := s.db.QueryRow(
		`SELECT id, number, client_id, issue_date, due_date, tax_rate_percent, notes, status
		 FROM invoices WHERE id = ?`, id,
	)
	inv, err := s.scanInvoice(row)
	if err != nil {
		return Invoice{}, false
	}
	return inv, true
}

func (s *SQLiteStore) UpdateInvoiceStatus(id string, status InvoiceStatus) (Invoice, error) {
	result, err := s.db.Exec(`UPDATE invoices SET status = ? WHERE id = ?`, status, id)
	if err != nil {
		return Invoice{}, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return Invoice{}, err
	}
	if rows == 0 {
		return Invoice{}, ErrNotFound
	}
	inv, _ := s.GetInvoice(id)
	return inv, nil
}
