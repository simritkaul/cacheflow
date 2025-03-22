package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/simritkaul/cacheflow/internal/api"
	"github.com/simritkaul/cacheflow/internal/cache"
)

func main () {
	// Parse command line flags
	port := flag.Int("port", 8080, "Port to run the server on");
	evictionType := flag.String("eviction", "lru", "Eviction policy for the cache (lfu or lru)");
	maxItems := flag.Int("max-items", 1000, "Maximum capacity of items in cache");
	flag.Parse();

	// Create a new cache
	c := cache.NewCache(*evictionType, *maxItems);

	// Create and start a new HTTP server
	server := api.NewServer(c, fmt.Sprintf(":%d", *port));

	// Start the server in a goroutine
	go func () {
		log.Printf("Starting HTTP server on Port %d", *port);
		if err := server.Start(); err != nil {
			log.Fatalf("Server failed to start: %v", err);
		}
	}();

	// Set up a graceful shutdown
	quit := make(chan os.Signal, 1);
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM);
	<-quit

	log.Println("Shutting down the server ...");
	// Potential cleanup logic
	log.Println("Server gracefully stopped");
}