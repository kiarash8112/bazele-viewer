// Command worker emails customers a confirmation for each new order.
package main

import (
	"flag"
	"log"
	"time"

	"example.com/bazelviewer/internal/db"
	"example.com/bazelviewer/internal/metrics"
	"example.com/bazelviewer/internal/notify"
	"example.com/bazelviewer/pkg/config"
	"example.com/bazelviewer/pkg/logging"
)

func main() {
	every := flag.Duration("every", 10*time.Second, "how often to look for new orders")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	d, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	mailer := notify.New(cfg.MailFrom)
	m := metrics.NewRegistry()
	l := logging.New("worker")

	sent := map[string]bool{}
	for range time.Tick(*every) {
		for _, r := range d.All("orders") {
			if sent[r.ID] {
				continue
			}
			if err := mailer.Send(r.Fields["customer"], "Your order "+r.ID); err != nil {
				l.Printf("order %s: %v", r.ID, err)
				continue
			}
			sent[r.ID] = true
			m.Inc("worker.confirmations")
		}
	}
}
