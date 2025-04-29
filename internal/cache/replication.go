package cache

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Handles cache data replication
type ReplicationManager struct {
	cache *Cache
	replicaCount int
	nodeManager interface{ GetNodesForKey(key string, count int) []string}
	localNode string
}

// Creates a new replication manager
func NewReplicationManager (cache *Cache, replicaCount int, nodeManager interface{ GetNodesForKey(key string, count int) []string}, localNode string) *ReplicationManager {
	return &ReplicationManager{
		cache: cache,
		replicaCount: replicaCount,
		nodeManager: nodeManager,
		localNode: localNode,
	}
}

// Replicates a set operation to replica nodes
func (rm *ReplicationManager) ReplicateSet (key string, value interface{}, ttl time.Duration) {
	// Get replica nodes
	nodes := rm.nodeManager.GetNodesForKey(key, rm.replicaCount+1);	// +1 for the primary node

	// Replicate to the other nodes
	for _, node := range nodes {
		// Skip the local node
		if node == rm.localNode {
			continue;
		}

		// Replicate asynchronously
		go func (node string) {
			// Create replication request
			url := fmt.Sprintf("%s/replicate/set", node);
			data := map[string]interface{}{
				"key": key,
				"value": value,
				"ttl": int64(ttl.Seconds()),
			}

			jsonData, err := json.Marshal(data);
			if err != nil {
				log.Printf("Error marshaling replication data: %v", err);
				return;
			}

			// Send replication request
			resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData));
			if err != nil {
				log.Printf("Error replicating to node %s: %v", node, err);
				return;
			}
			defer resp.Body.Close();

			if (resp.StatusCode != http.StatusOK) {
				log.Printf("Replication to node %s failed with status %d", node, resp.StatusCode)
			}
		}(node);
	}
}

// Replicates delete operation to the replica nodes
func (rm *ReplicationManager) ReplicateDelete (key string) {
	// Get replica nodes
	nodes := rm.nodeManager.GetNodesForKey(key, rm.replicaCount+1);

	// Replicate to other nodes
	for _, node := range nodes {
		// Skip local node
		if node == rm.localNode {
			continue;
		}

		// Replicate asynchronously
		go func (node string) {
			// Create replication request
			url := fmt.Sprintf("%s/replication/delete?key=%s", node, key);

			req, err := http.NewRequest(http.MethodDelete, url, nil);
			if err != nil {
				log.Printf("Error creating delete request: %v", err);
				return;
			}

			// Send replication request
			client := &http.Client{};
			resp, err := client.Do(req);
			if err != nil {
				log.Printf("Error replicating delete to node %s: %v", node, err);
				return;
			}

			defer resp.Body.Close();

			if resp.StatusCode != http.StatusOK {
				log.Printf("Delete replication to node %s failed with status %d", node, resp.StatusCode);
			}
		}(node);
	}
}

// Handles the incoming set replication requests
func (rm *ReplicationManager) HandleReplicateSet (w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed);
		return;
	}

	var data struct {
		Key string `json:"key"`;
		Value interface{} `json:"value"`;
		TTL int64 `json:"ttl"`;
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest);
		return;
	}

	ttl := time.Duration(data.TTL) * time.Second;
	rm.cache.Set(data.Key, data.Value, ttl);

	w.Header().Set("Content-Type", "application/json");
	json.NewEncoder(w).Encode(map[string]string {"status": "success"});
}

// Handles the incoming delete replication requests
func (rm *ReplicationManager) HandleReplicateDelete (w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed);
		return;
	}

	key := r.URL.Query().Get("key");

	if key == "" {
		http.Error(w, "Key is required", http.StatusBadRequest);
		return;
	}

	rm.cache.Delete(key);

	w.Header().Set("Content-Type", "application/json");
	json.NewEncoder(w).Encode(map[string]string{"status": "succcess"});
}

// Sets up the HTTP handlers for replication
func (rm *ReplicationManager) SetupHTTPHandlers (mux *http.ServeMux) {
	mux.HandleFunc("/replicate/set", rm.HandleReplicateSet);
	mux.HandleFunc("/replicate/delete", rm.HandleReplicateDelete);
}
