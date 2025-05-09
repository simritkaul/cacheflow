package cluster

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

// NodeStatus represents the status of a node (up or down)
type NodeStatus string;

const (
	NodeStatusUp NodeStatus = "up";
	NodeStatusDown NodeStatus = "down";
)

// Node represents a cache node in the cluster
type Node struct {
	ID	string;
	Address string;
	Status NodeStatus;
	LastSeen time.Time;
}

// NodeManager manages the nodes in the cluster
type NodeManager struct {
	nodes map[string]*Node;
	hash *ConsistentHash;
	localNode *Node;
	nodeCheckTime time.Duration;
	mu sync.RWMutex;
}

// Creates a new Node Manager
func NewNodeManager (localId, localAddr string, checkTime time.Duration) *NodeManager {
	localNode := &Node{
		ID: localId,
		Address: localAddr,
		Status: NodeStatusUp,
		LastSeen: time.Now(),
	}

	nm := &NodeManager{
		nodes: make(map[string]*Node),
		hash: NewConsistentHash(10), // 10 virtual nodes for each physical node in the cluster
		localNode: localNode,
		nodeCheckTime: checkTime,
	}

	// Add the local node
	nm.nodes[localId] = localNode;
	nm.hash.Add(localId);

	return nm;
}

// Registers a new node in the node cluster
func (nm *NodeManager) RegisterNode (id, addrs string) {
	nm.mu.Lock();
	defer nm.mu.Unlock();

	if node, exists := nm.nodes[id]; exists {
		// Update the existing node
		node.Address = addrs;
		node.Status = NodeStatusUp;
		node.LastSeen = time.Now();
		return;
	}

	// Add a new node
	newNode := &Node{
		ID: id,
		Address: addrs,
		Status: NodeStatusUp,
		LastSeen: time.Now(),
	}

	nm.nodes[id] = newNode;
	nm.hash.Add(id);

	log.Printf("Node %s registered at %s", id, addrs);
}

// Returns the node responsible for the given key
func (nm *NodeManager) GetNodeForKey (key string) *Node {
	nm.mu.RLock();
	defer nm.mu.RUnlock();

	nodeId := nm.hash.Get(key);
	return nm.nodes[nodeId];
}

// Get all nodes in the node cluster
func (nm *NodeManager) GetAllNodes () []*Node {
	nm.mu.RLock();
	defer nm.mu.RUnlock();

	nodes := make([]*Node, 0, len(nm.nodes));
	for _, node := range(nm.nodes) {
		nodes = append(nodes, node);
	}

	return nodes;
}

// Starts a background goroutine to check node health
func (nm *NodeManager) StartHealthCheck () {
	go func () {
		ticker := time.NewTicker(nm.nodeCheckTime);
		defer ticker.Stop();

		for range ticker.C {
			nm.checkNodeHealth();
		}
	}();
}

func (nm *NodeManager) checkNodeHealth () {
	nm.mu.Lock();
	defer nm.mu.Unlock();

	for id, node := range nm.nodes {
		// Skip local node
		// Local node is the one running this function, so we don’t need to check its own health.
		if id == nm.localNode.ID {
			continue;
		}

		// Check if the node is alive
		// Waiting twice the check interval ensures we don’t mistakenly remove a healthy but slow-responding node.
		if time.Since(node.LastSeen) > nm.nodeCheckTime * 2 {
			log.Printf("Node %s at %s is down", node.ID, node.Address);
			node.Status = NodeStatusDown;
		}

		// In a real system, we would attempt to contact the node 
		// and only mark it as down if we couldn't reach it
	}
}

// Sets up HTTP handlers for node management
func (nm *NodeManager) SetupHTTPHandlers (mux *http.ServeMux) {
	mux.HandleFunc("/nodes/register", nm.handleRegister);
	mux.HandleFunc("/nodes/heartbeat", nm.handleHeartbeat);
	mux.HandleFunc("/nodes/list", nm.handleListNodes);
}

// Handles node registration requests
func (nm *NodeManager) handleRegister (w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed);
		return;
	}

	var data struct {
		ID	string	`json:"id"`;
		Address	string `json:"address"`;
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest);
		return;
	}

	nm.RegisterNode(data.ID, data.Address);

	w.Header().Set("Content-Type", "application/json");
	json.NewEncoder(w).Encode(map[string]string {
		"status": "success",
	})
}

// Handles node heartbeat requests i.e. this node sent a heartbeat i.e. it is up
func (nm *NodeManager) handleHeartbeat (w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed);
		return;
	}

	var data struct {
		ID	string 	`json:"id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest);
		return;
	}

	nm.mu.Lock();

	if node, exists := nm.nodes[data.ID]; exists {
		node.LastSeen = time.Now();
		node.Status = NodeStatusUp;
	}

	nm.mu.Unlock();

	w.Header().Set("Content-Type", "application/json");
	json.NewEncoder(w).Encode(map[string]string {
		"status": "success",
	})
}

// Handles request to get list of all nodes
func (nm *NodeManager) handleListNodes (w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed);
		return;
	}

	type nodeInfo struct {
		ID	string `json:"id"`
		Address	string `json:"address"`
		Status NodeStatus `json:"status"`
	}

	response := make([]nodeInfo, len(nm.nodes));

	nodes := nm.GetAllNodes();

	for i, node := range nodes {
		response[i] = nodeInfo {
			ID: node.ID,
			Address: node.Address,
			Status: node.Status,
		}
	}

	w.Header().Set("Content-Type", "application/json");
	json.NewEncoder(w).Encode(response);
}

// Gets the local node for the node manager
func (nm *NodeManager) GetLocalNode () *Node {
	return nm.localNode;
}

// Returns a list of nodes that should store the given key
func (nm *NodeManager) GetNodesForKey (key string, count int) []string {
	nm.mu.Lock();
	defer nm.mu.Unlock();

	// Get primary node
	primaryNodeId := nm.hash.Get(key);
	nodes := []string{primaryNodeId};

	// If we only need one node, return the primary
	if count <= 1 || len(nm.nodes) <= 1 {
		return nodes;
	}

	// Get the hash ring
	hashRing := nm.hash.GetHashRing();
	numNodes := len(hashRing);

	// Find the position of the primary node in the ring
	var primaryPos int;
	for i, hash := range hashRing {
		if nm.hash.GetNodeForHash(hash) == primaryNodeId {
			primaryPos = i;
			break;
		}
	}

	// Get the remaining count - 1 nodes
	added := 1; // Already counted primary
	for i := 1; i < numNodes && added < count; i++ {
		pos := (primaryPos + i) % numNodes;
		nodeId := nm.hash.GetNodeForHash(hashRing[pos]);

		// Add each node only once
		if !containsString(nodes, nodeId) {
			nodes = append(nodes, nodeId);
			added++;
		}
	}

	return nodes;
}

// Utility function for checking if a slice contains a string
func containsString (slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true;
		}
	}

	return false;
}

// Handles a node failure
func (nm *NodeManager) HandleNodeFailure (nodeId string) {
	nm.mu.Lock();
	
	// Check if node exists and already marked as down
	node, exists := nm.nodes[nodeId];
	if !exists || node.Status == NodeStatusDown {
		nm.mu.Unlock();
		return;
	}
	
	// Mark the status as down
	node.Status = NodeStatusDown;
log.Printf("Node %s marked as down, starting recovery", nodeId);
	nm.mu.Unlock();

	// Start the recovery process
	go nm.recoverNodeData(nodeId);
}

// Handles data recovery for a failed node
func (nm *NodeManager) recoverNodeData (failedNodeId string) {
	// Get all keys that should be on the failed node
	// This is a simplified approach. In a real system, you would need a way to
	// know what keys were stored on the failed node.

	// For each key that needs to be recovered:
	// 1. Determine where it should go now
	// 2. Find a replica that has the data
	// 3. Copy the data to the new node

	log.Printf("Recovery for node %s complete", failedNodeId)
}