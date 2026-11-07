// Command server runs a tiny greeting HTTP server.
package main

import (
	"flag"
	"log"
	"net/http"

	"example.com/bazelviewer/internal/handler"
	"example.com/bazelviewer/pkg/store"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	flag.Parse()
	log.Printf("listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, handler.New(store.New())))
}
