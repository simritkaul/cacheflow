package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/simritkaul/cacheflow/internal/api"
	"github.com/simritkaul/cacheflow/internal/cache"
	"github.com/simritkaul/cacheflow/internal/cluster"
)

func main () {
	// Parse command line flags
	port := flag.Int("port", 8080, "Port to run the server on");
	evictionType := flag.String("eviction", "lru", "Eviction policy for the cache (lfu or lru)");
	maxItems := flag.Int("max-items", 1000, "Maximum capacity of items in cache");
	nodeId := flag.String("node-id", "", "Node ID (Generated if empty)");
	seedNode := flag.String("seed", "", "Seed node address to join the cluster");
	dataDir := flag.String("data-dir", "./data", "Directory for cache persistence");
	replicaCount := flag.Int("replicas", 2, "Number of replicas for each key");
	persistenceEnabled := flag.Bool("persistence", true, "Enable persistence");
	flag.Parse();

	// Generate a new node id if not provided
	if *nodeId == "" {
		*nodeId = uuid.New().String();
	}

	// Create a data directory if it doesn't exist
	if *persistenceEnabled {
		if err := os.MkdirAll(*dataDir, 0755); err != nil {
			log.Fatalf("Failed to create data directory: %v", err);
		}
	}

	// Create a new cache
	c := cache.NewCache(*evictionType, *maxItems);

	// Create node address
	addr := fmt.Sprintf(":%d", *port);
	nodeAddr := fmt.Sprintf("http://localhost%s", addr);

	// Create node manager
	nm := cluster.NewNodeManager(*nodeId, nodeAddr, 5 * time.Second);

	// Create new HTTP server for the node manager
	mux := http.NewServeMux();

	// Create a new HTTP server and setup handlers
	server := api.NewServer(c, mux);
	server.SetupHandlers();

	// Set up node management handlers
	nm.SetupHTTPHandlers(mux);

	// Create replication manager
	rm := cache.NewReplicationManager(c, *replicaCount, nm, *nodeId);
	rm.SetupHTTPHandlers(mux);
	server.SetReplicationManager(rm);

	// Create persistence manager if enabled
	var persistenceManager *cache.PersistenceManager;
	if *persistenceEnabled {
		persistencePath := filepath.Join(*dataDir, fmt.Sprintf("cache-%s.dat", *nodeId));
		persistenceManager = cache.NewPersistenceManager(c, persistencePath, 30 * time.Second);
		persistenceManager.Start();
	}

	// Start health check
	nm.StartHealthCheck();

	// Connect to seed node if provided
	if *seedNode != "" {
		go func () {
			// Wait a little to ensure our server has started
			time.Sleep(1 * time.Second);

			// Register with seed node
			if err := registerWithSeedNode(*seedNode, *nodeId, nodeAddr); err != nil {
				log.Printf("Failed to register with seed node: %v", err);
			}
		}();

	}

	// Start the server in a goroutine
	go func() {
		log.Printf("Starting server on port %d with node ID %s...", *port, *nodeId)
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Set up a graceful shutdown
	// Channel to get Signal from OS (like pressing CTRL + C)
	quit := make(chan os.Signal, 1);
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM);
	<-quit	// Waits here till it receives any signal from the quit channel

	log.Println("Shutting down the server ...");
	// Potential cleanup logic
	log.Println("Server gracefully stopped");
}

func registerWithSeedNode (seedNode, nodeId, nodeAddr string) error {
	url := fmt.Sprintf("%s/nodes/register", seedNode);
	data := map[string]string {
		"id": nodeId,
		"address": nodeAddr,
	}

	jsonData, err := json.Marshal(data);
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err);
	}

	// Send the POST request
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData));
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err);
	}
	defer resp.Body.Close();

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to register with seed node, status: %s", resp.Status);
	}

	// Implement post request to register with seed node
	log.Printf("Registering with seed node %s", seedNode);

	return nil;
}