package cluster

import (
	"crypto/md5"
	"encoding/binary"
	"sort"
	"sync"
)

type ConsistentHash struct {
	replicas int	// Number of virtual nodes per actual node
	hashRing []uint32	// Sorted list of hash values (hash ring where nodes and keys are hashed)
	nodeMap map[uint32]string	// Maps hash values to actual node addresses
	nodes map[string]bool	// Keeps track of actual nodes
	mu sync.RWMutex	// Read Write lock for concurrency safety
}

// Creates a new Consistent Hash Ring
func NewConsistentHash (replicas int) (*ConsistentHash) {
	return &ConsistentHash{
		replicas: replicas,
		hashRing: make([]uint32, 0),
		nodeMap: make(map[uint32]string),
		nodes: make(map[string]bool),		
	}
}

// Adds a new node to the hash ring
func (ch *ConsistentHash) Add (node string) {
	ch.mu.Lock();
	defer ch.mu.Unlock();

	// If already exists
	if _, exists := ch.nodes[node]; exists {
		return;
	}

	ch.nodes[node] = true;

	// Add virtual nodes (replicas) for the newly added node
	for i := 0; i < ch.replicas; i++ {
		hash := ch.hashKey(node + string(rune(i)));
		ch.hashRing = append(ch.hashRing, hash);
		ch.nodeMap[hash] = node;
	}

	sort.Slice(ch.hashRing, func(i, j int) bool {
		return ch.hashRing[i] < ch.hashRing[j];
	})
}

// Removes a node from the hash ring
func (ch *ConsistentHash) Remove (node string) {
	ch.mu.Lock();
	defer ch.mu.Unlock();

	// If does not exist in the hash ring
	if _, exists := ch.nodes[node]; !exists {
		return;
	}

	delete(ch.nodes, node);

	// Remove all replicas
	newHashRing := make([]uint32, 0);
	for _, hash := range ch.hashRing {
		if ch.nodeMap[hash] != node {
			newHashRing = append(newHashRing, hash);
		} else {
			delete(ch.nodeMap, hash);
		}
	}

	ch.hashRing = newHashRing;
}

// Get returns the node responsible for the given key
// (The clockwise walk thing)
func (ch *ConsistentHash) Get (key string) string {
	ch.mu.RLock();
	defer ch.mu.RUnlock();

	if len(ch.hashRing) == 0 {
		return "";
	}

	hash := ch.hashKey(key);

	// Binary search
	idx := sort.Search(len(ch.hashRing), func (i int) bool {
		return ch.hashRing[i] >= hash
	})

	// Wrap around to the first node (since it is a ring)
	if idx == len(ch.hashRing) {
		idx = 0;
	}

	return ch.nodeMap[ch.hashRing[idx]];
}

// GetNodes returns all the nodes in the hash ring
func (ch *ConsistentHash) GetNodes () ([]string) {
	ch.mu.RLock();
	defer ch.mu.RUnlock();

	nodes := make([]string, 0, len(ch.nodes));

	for node := range ch.nodes {
		nodes = append(nodes, node);
	}

	return nodes;
}

// Creates a hash for the given key
func (ch* ConsistentHash) hashKey (key string) (uint32) {
	hasher := md5.New();
	hasher.Write([]byte(key));
	hash := hasher.Sum(nil);
	return binary.LittleEndian.Uint32(hash[:4]);
}

// Returns the hash ring
func (ch *ConsistentHash) GetHashRing() []uint32 {
	return ch.hashRing;
}

// Returns the node for the given hash
func (ch *ConsistentHash) GetNodeForHash (hash uint32) string {
	ch.mu.RLock();
	defer ch.mu.RUnlock();

	// Binary search
	idx := sort.Search(len(ch.hashRing), func (i int) bool {
		return ch.hashRing[i] >= hash
	})

	// Wrap around to the first node (since it is a ring)
	if idx == len(ch.hashRing) {
		idx = 0;
	}

	return ch.nodeMap[ch.hashRing[idx]];
}