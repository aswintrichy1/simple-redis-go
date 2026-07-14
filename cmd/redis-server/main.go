// Command redis-server starts the in-memory RESP server.
package main

import (
	"flag"
	"log"

	"simple-redis-go/internal/server"
	"simple-redis-go/internal/store"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:6379", "address to listen on (host:port)")
	flag.Parse()

	srv := server.New(store.New())
	log.Printf("simple-redis-go listening on %s", *addr)
	if err := srv.Listen(*addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
