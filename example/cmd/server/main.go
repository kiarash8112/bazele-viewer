// Command server runs the shop's HTTP API.
package main

import (
	"log"
	"net/http"

	"example.com/bazelviewer/example/internal/auth"
	"example.com/bazelviewer/example/internal/catalog"
	"example.com/bazelviewer/example/internal/db"
	"example.com/bazelviewer/example/internal/httpapi"
	"example.com/bazelviewer/example/internal/inventory"
	"example.com/bazelviewer/example/internal/metrics"
	"example.com/bazelviewer/example/internal/orders"
	"example.com/bazelviewer/example/internal/payments"
	"example.com/bazelviewer/example/internal/search"
	"example.com/bazelviewer/example/pkg/config"
	"example.com/bazelviewer/example/pkg/money"
	"example.com/bazelviewer/example/pkg/store"
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

	m := metrics.NewRegistry()
	c := catalog.New(d)
	stock := inventory.New(m)
	for _, p := range []struct {
		name  string
		price money.Amount
	}{{"Blue Mug", 1250}, {"Tea Pot", 3400}, {"Linen Apron", 2200}} {
		stock.Restock(c.Add(p.name, p.price).ID, 10)
	}

	api := httpapi.New(httpapi.Deps{
		Auth:    auth.New(d),
		Orders:  orders.New(d, c, stock, payments.New(m), m),
		Search:  search.New(c),
		Metrics: m,
		Visits:  store.New(),
	})
	log.Printf("listening on %s", cfg.Addr)
	log.Fatal(http.ListenAndServe(cfg.Addr, api))
}
