package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"

	"invoicing-tool/internal/api"
	"invoicing-tool/internal/invoicing"
)

//go:embed web
var webFiles embed.FS

func main() {
	// DB_PATH controls persistence: set it (e.g. to a file path) to use
	// real SQLite storage that survives restarts. Leave it unset and the
	// app falls back to in-memory storage.
	var store invoicing.Store
	if dbPath := os.Getenv("DB_PATH"); dbPath != "" {
		sqliteStore, err := invoicing.NewSQLiteStore(dbPath)
		if err != nil {
			log.Fatalf("opening SQLite store at %q: %v", dbPath, err)
		}
		store = sqliteStore
		log.Printf("using SQLite storage at %s", dbPath)
	} else {
		store = invoicing.NewMemoryStore()
		log.Print("using in-memory storage (set DB_PATH to persist data)")
	}

	adminUser := getEnv("ADMIN_USER", "admin")
	adminPassword := getEnv("ADMIN_PASSWORD", "changeme")

	staticFiles, err := fs.Sub(webFiles, "web")
	if err != nil {
		log.Fatal(err)
	}

	server := api.NewServer(store, adminUser, adminPassword)

	// PORT is read because hosting platforms like Render assign it
	// dynamically at deploy time.
	addr := ":" + getEnv("PORT", "8081")
	log.Printf("invoicing-tool listening on %s (admin user: %s)", addr, adminUser)
	if err := http.ListenAndServe(addr, server.Routes(staticFiles)); err != nil {
		log.Fatal(err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
