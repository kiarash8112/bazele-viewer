// Package search finds products by name.
package search

import (
	"sort"
	"strings"

	"example.com/bazelviewer/example/internal/catalog"
	"example.com/bazelviewer/example/pkg/strutil"
)

// Index searches a catalog.
type Index struct {
	catalog *catalog.Catalog
}

// New returns an Index over c.
func New(c *catalog.Catalog) *Index { return &Index{catalog: c} }

// Find returns the products whose name contains query, by name.
func (i *Index) Find(query string) []catalog.Product {
	q := strutil.Slug(query)
	var found []catalog.Product
	for _, p := range i.catalog.All() {
		if strings.Contains(p.Slug, q) {
			found = append(found, p)
		}
	}
	sort.Slice(found, func(a, b int) bool { return found[a].Name < found[b].Name })
	return found
}
