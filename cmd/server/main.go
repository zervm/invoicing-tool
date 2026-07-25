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
	store := invoicing.NewMemoryStore()

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
