// Package catalog holds the products for sale.
package catalog

import (
	"strconv"
	"time"

	"example.com/bazelviewer/example/internal/cache"
	"example.com/bazelviewer/example/internal/db"
	"example.com/bazelviewer/example/pkg/clock"
	"example.com/bazelviewer/example/pkg/money"
	"example.com/bazelviewer/example/pkg/strutil"
)

// Product is something for sale.
type Product struct {
	ID    string
	Slug  string
	Name  string
	Price money.Amount
}

// Catalog stores products in the database and caches lookups.
type Catalog struct {
	db    *db.DB
	cache *cache.Cache[Product]
}

// New returns a Catalog backed by d.
func New(d *db.DB) *Catalog {
	return &Catalog{db: d, cache: cache.New[Product](time.Minute, clock.System)}
}

// Add puts a new product up for sale.
func (c *Catalog) Add(name string, price money.Amount) Product {
	r := c.db.Insert("products", map[string]string{
		"name":  name,
		"slug":  strutil.Slug(name),
		"price": strconv.FormatInt(int64(price), 10),
	})
	return fromRow(r)
}

// Get returns the product with id.
func (c *Catalog) Get(id string) (Product, bool) {
	if p, ok := c.cache.Get(id); ok {
		return p, true
	}
	r, ok := c.db.Get("products", id)
	if !ok {
		return Product{}, false
	}
	p := fromRow(r)
	c.cache.Put(id, p)
	return p, true
}

// All returns every product.
func (c *Catalog) All() []Product {
	rows := c.db.All("products")
	products := make([]Product, 0, len(rows))
	for _, r := range rows {
		products = append(products, fromRow(r))
	}
	return products
}

func fromRow(r db.Row) Product {
	cents, _ := strconv.ParseInt(r.Fields["price"], 10, 64)
	return Product{ID: r.ID, Slug: r.Fields["slug"], Name: r.Fields["name"], Price: money.Amount(cents)}
}
