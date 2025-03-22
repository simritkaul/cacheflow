package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/simritkaul/cacheflow/internal/cache"
)

type Server struct {
	cache *cache.Cache
	addr string
}

// Creates a new HTTP server for the cache
func NewServer (cache *cache.Cache, addr string) (*Server) {
	return &Server{
		cache: cache,
		addr: addr,
	}
}

// Starts the HTTP server
func (s *Server) Start () error {
	mux := http.NewServeMux();

	mux.HandleFunc("/get", s.handleGet);
	mux.HandleFunc("/set", s.handleSet);
	mux.HandleFunc("/delete", s.handleDelete);

	return http.ListenAndServe(s.addr, mux);
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