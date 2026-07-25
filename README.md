# Invoicing Tool — showcase project

A simple invoicing tool for a freelancer/small business: manage clients,
create invoices with line items and tax, track status (draft/sent/paid),
and export a clean invoice as a PDF via the browser's print function.

Built as a freelance portfolio piece, demonstrating money-math correctness
(cents-based arithmetic, tested), auth-protected APIs, and a practical,
dependency-free approach to "PDF export."

## Architecture decisions

- **Money is stored and calculated in cents (`int64`), never floats.**
  Floating-point arithmetic on money values is a classic source of subtle
  bugs (e.g. rounding errors accumulating across many invoices). All
  totals/subtotals/tax are covered by tests in `internal/invoicing/calc_test.go`.
- **The entire API requires authentication.** Unlike the booking system,
  this tool has no public-facing side — every user of this API is the
  freelancer managing their own business, so there's no split between
  public and admin routes.
- **PDF export uses the browser's native "Print > Save as PDF"**, not a
  server-side PDF library — there's no internet access in this dev
  environment to fetch one, and print-to-PDF is a legitimate, widely-used
  approach for exactly this kind of tool. `style.css` has a `@media print`
  section that hides everything except the invoice itself when printing.
- **Go standard library only, no external dependencies**, same reasoning
  as the booking system: in-memory storage behind a `Store` interface,
  swappable for a real database later without touching any handler code.

## Running it

```
go test ./... -v
go run ./cmd/server
```

Then open http://localhost:8081/ (default credentials: `admin` /
`changeme`, override via `ADMIN_USER` / `ADMIN_PASSWORD` env vars).

Workflow: add a client, create an invoice for them with one or more line
items, then open the invoice to view/print it or mark it sent/paid.

## Deploying (Render)

1. Push this repo to GitHub.
2. On Render: New → Web Service → connect the repo.
3. Build command: `go build -o app ./cmd/server`
4. Start command: `./app`
5. Environment variables: `ADMIN_USER`, `ADMIN_PASSWORD`
   (Render sets `PORT` automatically — the app already reads it).

Note: storage is in-memory, so data resets on every restart/redeploy —
fine for a portfolio demo.

## Screenshots

![Invoice list](invoice.png)
![Invoice detail](<invoice(2).png>)
