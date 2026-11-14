// Command shopctl adds products and lists the catalog from the command line.
//
//	shopctl add "Blue Mug" 1250 10
//	shopctl list
package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"example.com/bazelviewer/example/internal/catalog"
	"example.com/bazelviewer/example/internal/db"
	"example.com/bazelviewer/example/internal/inventory"
	"example.com/bazelviewer/example/internal/metrics"
	"example.com/bazelviewer/example/pkg/config"
	"example.com/bazelviewer/example/pkg/money"
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
	c := catalog.New(d)
	stock := inventory.New(metrics.NewRegistry())

	args := os.Args[1:]
	switch {
	case len(args) == 4 && args[0] == "add":
		cents, err1 := strconv.ParseInt(args[2], 10, 64)
		units, err2 := strconv.Atoi(args[3])
		if err1 != nil || err2 != nil {
			log.Fatal("usage: shopctl add NAME CENTS UNITS")
		}
		p := c.Add(args[1], money.Amount(cents))
		stock.Restock(p.ID, units)
		fmt.Printf("added %s (%s) at %s\n", p.Name, p.ID, p.Price)
	case len(args) == 1 && args[0] == "list":
		for _, p := range c.All() {
			fmt.Printf("%-24s %-28s %8s %4d in stock\n", p.ID, p.Name, p.Price, stock.Units(p.ID))
		}
	default:
		log.Fatal("usage: shopctl add NAME CENTS UNITS | shopctl list")
	}
}
