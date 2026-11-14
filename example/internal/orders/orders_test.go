package orders

import (
	"testing"

	"example.com/bazelviewer/example/internal/catalog"
	"example.com/bazelviewer/example/internal/db"
	"example.com/bazelviewer/example/internal/inventory"
	"example.com/bazelviewer/example/internal/metrics"
	"example.com/bazelviewer/example/internal/payments"
)

func TestPlace(t *testing.T) {
	d, err := db.Open("memory://")
	if err != nil {
		t.Fatal(err)
	}
	m := metrics.NewRegistry()
	c := catalog.New(d)
	stock := inventory.New(m)
	mug := c.Add("Blue Mug", 1250)
	stock.Restock(mug.ID, 3)
	s := New(d, c, stock, payments.New(m), m)

	o, err := s.Place("ana@example.com", map[string]int{mug.ID: 2})
	if err != nil {
		t.Fatal(err)
	}
	if o.Total != 2500 || o.Receipt == "" {
		t.Errorf("order = %+v, want total $25.00 with a receipt", o)
	}
	if left := stock.Units(mug.ID); left != 1 {
		t.Errorf("units left = %d, want 1", left)
	}
	if _, err := s.Place("ana@example.com", map[string]int{mug.ID: 2}); err == nil {
		t.Error("ordering more than is in stock succeeded")
	}
}
