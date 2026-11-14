// Command migrate creates the shop's database tables.
package main

import (
	"log"

	"example.com/bazelviewer/example/internal/db"
	"example.com/bazelviewer/example/pkg/config"
	"example.com/bazelviewer/example/pkg/logging"
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
