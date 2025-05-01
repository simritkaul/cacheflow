package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/simritkaul/cacheflow/internal/cache"
	"github.com/simritkaul/cacheflow/internal/cluster"
)

type Server struct {
	cache *cache.Cache
	mux *http.ServeMux
	nodeManager *cluster.NodeManager
	replicationManager *cache.ReplicationManager
}

// Creates a new HTTP server for the cache
func NewServer (cache *cache.Cache, mux *http.ServeMux) (*Server) {
	return &Server{
		cache: cache,
		mux: mux,
	}
}

// Sets the Node Manager for the server
func (s *Server) SetNodeManager (nm *cluster.NodeManager) {
	s.nodeManager = nm;
}

// Sets the Replication Manager for the server
func (s *Server) SetReplicationManager (rm *cache.ReplicationManager) {
	s.replicationManager = rm;
}

// SetupHandlers sets up the HTTP handlers
func (s *Server) SetupHandlers() {
	s.mux.HandleFunc("/get", s.handleGet)
	s.mux.HandleFunc("/set", s.handleSet)
	s.mux.HandleFunc("/delete", s.handleDelete)
}

// Handle GET requests to retrieve values from cache
func (s *Server) handleGet (w http.ResponseWriter, r *http.Request) {
	if (r.Method != http.MethodGet) {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed);
		return;
	}

	key := r.URL.Query().Get("key");
	if key == "" {
		http.Error(w, "Key is required", http.StatusBadRequest);
		return;
	}

	// Check if we have a node manager and the key belongs to some other node
	if s.nodeManager != nil  {
		node := s.nodeManager.GetNodeForKey(key);
		// If it was for another node, forward the request to that node
		if node != nil && node.ID != s.nodeManager.GetLocalNode().ID {
			s.forwardRequest(w, r, node);
			return;
		}
	}

	value, found := s.cache.Get(key);

	if !found {
		http.Error(w, "Key not found", http.StatusNotFound);
		return;
	}

	w.Header().Set("Content-Type", "application/json");
	json.NewEncoder(w).Encode(map[string]interface{} {
		"key": key,
		"value": value,
	})
}

// Handle POST request to set a value in the cache
func (s *Server) handleSet (w http.ResponseWriter, r *http.Request) {
	if (r.Method != http.MethodPost) {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed);
		return;
	}

	// The DTO for the request body
	var data struct {
		Key string `json:"key"`
		Value interface{} `json:"value"`
		TTL int64	`json:"ttl"` // ttl in seconds
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest);
		return;
	}

	if data.Key == "" {
		http.Error(w, "Key is required", http.StatusBadRequest);
		return;
	}

	// Check if we have a node manager and the key belongs to some other node
	if s.nodeManager != nil  {
		node := s.nodeManager.GetNodeForKey(data.Key);
		// If it was for another node, forward the request to that node
		if node != nil && node.ID != s.nodeManager.GetLocalNode().ID {
			s.forwardRequest(w, r, node);
			return;
		}
	}

	ttl := time.Duration(data.TTL) * time.Second;

	s.cache.Set(data.Key, data.Value, ttl);

	w.WriteHeader(http.StatusCreated);
	w.Header().Set("Content-Type", "application/json");
	json.NewEncoder(w).Encode(map[string]string {
		"status": "success",
	})
}

func (s *Server) handleDelete (w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed);
		return;
	}

	key := r.URL.Query().Get("key");

	if key == "" {
		http.Error(w, "Key is required", http.StatusBadRequest);
		return;
	}

	s.cache.Delete(key);

	w.Header().Set("Content-Type", "application/json");
	json.NewEncoder(w).Encode(map[string]string {
		"status": "success",
	})
}

// Forwards a request to another node
func (s *Server) forwardRequest (w http.ResponseWriter, r *http.Request, node *cluster.Node) {
	
	// Create a new URL for forwarding
	forwardUrl := fmt.Sprintf("%s%s", node.Address, r.URL.Path);

	var req *http.Request;
	var err error;

	// Create a new request based on the original method
	switch r.Method {
	case http.MethodGet, http.MethodDelete:
		// For GET and DELETE, forward the query params
		forwardUrl = fmt.Sprintf("%s?%s", forwardUrl, r.URL.RawQuery);
		req, err = http.NewRequest(r.Method, forwardUrl, nil);
	case http.MethodPost:
		// For POST, forward the request body
		bodyBytes, err := io.ReadAll(r.Body);
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError);
			return;
		}

		req, err = http.NewRequest(r.Method, forwardUrl, bytes.NewBuffer(bodyBytes));
		req.Header.Set("Content-Type", "application/json");
	default:
		http.Error(w, "Method not supported for forwarding", http.StatusMethodNotAllowed);
		return;
	}

	if err != nil {
		http.Error(w, "Error creating the forwarded request", http.StatusInternalServerError);
		return;
	}

	// Forward the request
	client := &http.Client{}
	resp, err := client.Do(req);

	if err != nil {
		http.Error(w, fmt.Sprintf("Error forwarding the request: %v", err), http.StatusInternalServerError);
		return;
	}
	defer resp.Body.Close();

	// Copy the response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value);
		}
	}

	// Copy the status code
	w.WriteHeader(resp.StatusCode);

	// Copy the response body
	io.Copy(w, resp.Body);

	http.Error(w, fmt.Sprintf("Key belongs to node %s at %s", node.ID, node.Address), http.StatusTemporaryRedirect);
}