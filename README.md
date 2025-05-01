# CacheFlow: Distributed Caching System

[![Go Report Card](https://goreportcard.com/badge/github.com/simritkaul/cacheflow)](https://goreportcard.com/report/github.com/simritkaul/cacheflow)
[![GoDoc](https://godoc.org/github.com/simritkaul/cacheflow?status.svg)](https://godoc.org/github.com/simritkaul/cacheflow)

CacheFlow is a high-performance, distributed in-memory caching system written in Go. It provides a reliable and scalable solution for distributed caching with support for clustering, replication, and persistence.

## Features

- **Distributed Architecture**: Scale horizontally across multiple nodes
- **Multiple Eviction Policies**: LRU (Least Recently Used) and LFU (Least Frequently Used)
- **Automatic Cluster Management**: Node discovery and health monitoring
- **Data Replication**: Configurable replication factor for high availability
- **Persistence**: Optional disk-based persistence for data durability
- **RESTful API**: Simple HTTP API for cache operations
- **Metrics & Monitoring**: Built-in performance metrics
- **TTL Support**: Automatic expiration of cache entries
- **Configurable Settings**: Flexible configuration options

## Installation

### Prerequisites

- Go 1.18 or higher
- Git

### Building from Source

```bash
# Clone the repository
git clone https://github.com/simritkaul/cacheflow.git
cd cacheflow

# Build the binary
go build -o cacheflow cmd/server/main.go
```

## Quick Start

### Running a Single Node

```bash
./cacheflow --port 8080 --eviction lru --max-items 1000
```

### Creating a Cluster

Start the first node:

```bash
./cacheflow --port 8080 --node-id node1 --data-dir ./data1
```

Start additional nodes and connect them to the first one:

```bash
./cacheflow --port 8081 --node-id node2 --seed http://localhost:8080 --data-dir ./data2
./cacheflow --port 8082 --node-id node3 --seed http://localhost:8080 --data-dir ./data3
```

## Command-line Options

| Option          | Description                           | Default       |
| --------------- | ------------------------------------- | ------------- |
| `--port`        | HTTP server port                      | 8080          |
| `--eviction`    | Eviction policy (lru or lfu)          | lru           |
| `--max-items`   | Maximum number of items in cache      | 1000          |
| `--node-id`     | Node ID (generated if empty)          | [random UUID] |
| `--seed`        | Seed node address to join the cluster | ""            |
| `--data-dir`    | Directory for cache persistence       | "./data"      |
| `--replicas`    | Number of replicas for each key       | 2             |
| `--persistence` | Enable persistence                    | true          |

## API Reference

### Cache Operations

#### Set a Value

```
PUT /cache/{key}
Content-Type: application/json

{
  "value": "string value",
  "ttl": 3600  // optional, in seconds
}
```

#### Get a Value

```
GET /cache/{key}
```

#### Delete a Value

```
DELETE /cache/{key}
```

### Cluster Management

#### List Nodes

```
GET /nodes/list
```

#### Register a Node

```
POST /nodes/register
Content-Type: application/json

{
  "id": "node-id",
  "address": "http://host:port"
}
```

### Monitoring

#### Get Metrics

```
GET /metrics
```

## Example Client

The repository includes an example client that demonstrates how to interact with the cache:

```bash
# Run the client demo
go run examples/client/main.go

# Set a key
go run examples/client/main.go -action set -key mykey -value myvalue -ttl 60

# Get a key
go run examples/client/main.go -action get -key mykey

# Run benchmark
go run examples/client/main.go -action bench -bench-ops 5000
```

## Architecture

CacheFlow is designed with a modular architecture:

1. **Cache Module**: Core caching functionality with eviction policies
2. **Cluster Module**: Node management and service discovery
3. **API Module**: HTTP handlers for cache operations and cluster management
4. **Monitoring Module**: Performance metrics and monitoring endpoints
5. **Persistence Module**: Optional disk persistence for data durability
6. **Replication Module**: Data replication across multiple nodes

![Architecture Diagram](https://via.placeholder.com/800x400.png?text=CacheFlow+Architecture)

## Performance

CacheFlow is designed for high performance and scalability:

- Efficient concurrent access patterns using Go's sync primitives
- Optimized data structures for key lookups and eviction
- Low-overhead clustering protocol
- Configurable replication for balancing availability and performance

## Use Cases

- **API Rate Limiting**: Cache request counts and implement rate limiting
- **Database Query Caching**: Reduce database load by caching query results
- **Session Storage**: Store and retrieve user session data
- **Content Caching**: Cache rendered content for faster page loads
- **Distributed Counters**: Implement distributed atomic counters
- **Temporary Data Storage**: Store transient data with TTL

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Acknowledgments

- Inspired by projects like Redis, Memcached, and etcd
- Built with Go's powerful concurrency primitives
