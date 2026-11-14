// Command migrate creates the shop's database tables.
package main

import (
	"log"

	"example.com/bazelviewer/internal/db"
	"example.com/bazelviewer/pkg/config"
	"example.com/bazelviewer/pkg/logging"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	d, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	d.Migrate("users", "products", "orders")
	logging.New("migrate").Printf("database %s is up to date", cfg.DatabaseURL)
}
