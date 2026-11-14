// Package db is the shop's in-memory database of tables of rows.
package db

import (
	"fmt"
	"sync"
	"time"

	"example.com/bazelviewer/pkg/clock"
	"example.com/bazelviewer/pkg/ids"
	"example.com/bazelviewer/pkg/logging"
)

// Row is one stored record.
type Row struct {
	ID      string
	Fields  map[string]string
	Created time.Time
}

// DB holds tables of rows.
type DB struct {
	mu     sync.Mutex
	tables map[string][]Row
	clock  clock.Clock
	log    *logging.Logger
}

// Open opens the database at url. Only "memory://" is supported.
func Open(url string) (*DB, error) {
	if url != "memory://" {
		return nil, fmt.Errorf("unsupported database %q", url)
	}
	return &DB{tables: map[string][]Row{}, clock: clock.System, log: logging.New("db")}, nil
}

// Migrate creates the tables that do not exist yet.
func (d *DB) Migrate(tables ...string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, t := range tables {
		if _, ok := d.tables[t]; !ok {
			d.tables[t] = nil
			d.log.Printf("created table %s", t)
		}
	}
}

// Insert adds a row to table and returns it with a new ID.
func (d *DB) Insert(table string, fields map[string]string) Row {
	d.mu.Lock()
	defer d.mu.Unlock()
	r := Row{ID: ids.New(table), Fields: fields, Created: d.clock.Now()}
	d.tables[table] = append(d.tables[table], r)
	return r
}

// Get returns the row of table with the given ID.
func (d *DB) Get(table, id string) (Row, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, r := range d.tables[table] {
		if r.ID == id {
			return r, true
		}
	}
	return Row{}, false
}

// All returns every row of table, oldest first.
func (d *DB) All(table string) []Row {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]Row(nil), d.tables[table]...)
}
