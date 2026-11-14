// Package httpapi serves the shop over HTTP.
package httpapi

import (
	"encoding/json"
	"net/http"

	"example.com/bazelviewer/internal/auth"
	"example.com/bazelviewer/internal/handler"
	"example.com/bazelviewer/internal/metrics"
	"example.com/bazelviewer/internal/orders"
	"example.com/bazelviewer/internal/search"
	"example.com/bazelviewer/pkg/logging"
	"example.com/bazelviewer/pkg/store"
)

// Deps are the services the API calls.
type Deps struct {
	Auth    *auth.Service
	Orders  *orders.Service
	Search  *search.Index
	Metrics *metrics.Registry
	Visits  *store.Counter
}

// New returns the shop's HTTP routes.
func New(d Deps) http.Handler {
	log := logging.New("http")
	mux := http.NewServeMux()
	mux.Handle("GET /hello", handler.New(d.Visits))
	mux.HandleFunc("POST /signup", func(w http.ResponseWriter, r *http.Request) {
		token, err := d.Auth.SignUp(r.FormValue("email"))
		reply(w, map[string]string{"token": token}, err)
	})
	mux.HandleFunc("GET /products", func(w http.ResponseWriter, r *http.Request) {
		reply(w, d.Search.Find(r.URL.Query().Get("q")), nil)
	})
	mux.HandleFunc("POST /orders", func(w http.ResponseWriter, r *http.Request) {
		customer, err := d.Auth.Customer(r.Header.Get("Authorization"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		var items map[string]int
		if err := json.NewDecoder(r.Body).Decode(&items); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		order, err := d.Orders.Place(customer, items)
		reply(w, order, err)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		d.Metrics.Inc("http.requests")
		log.Printf("%s %s", r.Method, r.URL.Path)
		mux.ServeHTTP(w, r)
	})
}

func reply(w http.ResponseWriter, v any, err error) {
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
