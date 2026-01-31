# Concurrent Load‑Balancing Reverse Proxy with AI WAF (Go)

A reverse proxy built in Go that distributes traffic across multiple backend services using concurrent, thread‑safe load‑balancing strategies. The system includes continuous health monitoring, a runtime admin API, sticky sessions, TLS termination, and an optional AI‑based Web Application Firewall. All services can be run locally using Docker.

This project is designed to be readable, extensible, and close to how a small production gateway might actually be wired together.

---

## Overview

At its core, the proxy accepts incoming HTTPS requests, selects a healthy backend using a configurable strategy, forwards the request, and returns the response to the client. Backend availability is monitored continuously in the background, and backends can be added or removed at runtime via an admin API.

In addition to the core proxy logic, the project includes:

* Sticky sessions for session affinity
* TLS support
* A Python‑based AI WAF for request inspection
* Docker and docker‑compose for local orchestration

The extra components are optional at runtime but are part of the intended system design.

---

## System Architecture

The system is composed of three main layers:

### 1. Reverse Proxy (Go)

* Request routing and forwarding
* Load‑balancing strategies
* Sticky session handling
* Health monitoring
* Admin API

### 2. AI Web Application Firewall (Python, optional)

* Receives requests before forwarding
* Uses an ML classifier to detect malicious payloads
* Blocks suspicious requests (SQLi, XSS)

### 3. Backend Services (Node.js)

* Simple demo applications
* Color‑coded responses to visualize routing behavior

![System Diagram](image.png)

---

## Core Data Models

### Backend

```go
type Backend struct {
    URL          *url.URL     `json:"url"`
    Alive        bool         `json:"alive"`
    CurrentConns int64        `json:"current_connections"`
    mux          sync.RWMutex
}
```

### ServerPool

```go
type ServerPool struct {
    Backends []*Backend `json:"backends"`
    Current  uint64     `json:"current"`
}
```

### ProxyConfig

```go
type ProxyConfig struct {
    Port            int           `json:"port"`
    Strategy        string        `json:"strategy"`
    HealthCheckFreq time.Duration `json:"health_check_frequency"`
}
```

---

## Load Balancing

Backend selection is abstracted behind a strategy interface, allowing different algorithms to be swapped without changing request‑handling logic.

Supported strategies:

* **Round‑Robin**: sequential distribution using an atomic counter
* **Least‑Connections**: selects the backend with the lowest active connection count
* **Weighted Round‑Robin**: distributes traffic based on backend capacity

All shared state is protected using `sync.RWMutex` and `sync/atomic`. Only backends marked as healthy are eligible to receive traffic. If no backend is available, the proxy returns `503 Service Unavailable`.

---

## Proxy Behavior & Context Handling

The proxy is implemented using `net/http` and `httputil.ReverseProxy`.

Key behaviors:

* Request contexts are forwarded to backends
* Backend work is canceled if the client disconnects
* Connection counters are incremented before forwarding and decremented after completion
* Backend failures detected during request handling immediately mark the backend as unhealthy

---

## Health Monitoring

A background goroutine runs periodic health checks:

* Triggered using `time.Ticker`
* Sends lightweight HTTP probes to each backend
* Updates backend state in a thread‑safe manner
* Logs state transitions when backends go up or down

This allows the proxy to react automatically to backend failures without manual intervention.

---

## Admin API

The admin API runs on port `:8081` and allows runtime inspection and modification of the server pool.

### Endpoints

#### GET /status

Returns the current state of the system:

```json
{
  "total_backends": 3,
  "active_backends": 2,
  "backends": [
    {
      "url": "http://localhost:8082",
      "alive": true,
      "current_connections": 4
    }
  ]
}
```

#### POST /backends

```bash
curl -X POST http://localhost:8081/backends \
  -H "Content-Type: application/json" \
  -d '{"url": "http://localhost:8084"}'
```

#### DELETE /backends

```bash
curl -X DELETE http://localhost:8081/backends \
  -H "Content-Type: application/json" \
  -d '{"url": "http://localhost:8083"}'
```

---

## Sticky Sessions

Sticky sessions provide session affinity so that a client is consistently routed to the same backend.

* Can be based on client IP or cookies
* Useful for stateful backend services
* Implemented on top of existing load‑balancing strategies

Sticky sessions can be enabled or disabled at startup.

---

## AI Web Application Firewall

An optional AI‑based WAF runs as a separate Python service:

* Implemented with Flask and Scikit‑Learn
* Classifies requests as safe or malicious
* Detects common attack patterns such as SQL injection and XSS

Example behavior:

* `/?search=shoes` → `200 OK`
* `/?search=' OR 1=1 --` → `403 Forbidden`

The WAF is decoupled from the proxy and can be removed without affecting core functionality.

---

## Docker & Project Initialization

Docker is an intentional part of the project and is used to orchestrate supporting services.

The following components are containerized:

* AI WAF (Python)
* Backend services (Node.js)

### Start supporting services

```bash
docker-compose up --build -d
```

This launches the WAF and backend services required by the proxy. The Go proxy itself can be run locally or containerized separately.

---

## Configuration

Example configuration file:

```json
{
  "port": 8080,
  "strategy": "weighted-round-robin",
  "health_check_frequency": "10s",
  "backends": [
    { "url": "http://localhost:8082", "weight": 3 },
    { "url": "http://localhost:8083", "weight": 1 }
  ]
}
```

---

## Running the Proxy

### Generate TLS certificates

```bash
openssl req -x509 -newkey rsa:4096 \
  -keyout key.pem -out cert.pem \
  -days 365 -nodes -subj "/CN=localhost"
```
### Start supporting services
```bash
docker-compose up --build -d
```

### Start the proxy

```bash
go mod tidy
go run cmd/server/main.go 
```

Sticky sessions can be disabled:

```bash
go run cmd/server/main.go -sticky=false
```

---

## Demo Videos

Demo recordings showing routing behavior, health checks, and sticky sessions are hosted in GitHub Releases to keep the repository lightweight:

[https://github.com/mnl-lab/reverse-proxy/releases](https://github.com/mnl-lab/reverse-proxy/releases)

---

## Tests

Tests cover:

* Load‑balancing behavior
* Backend failover and recovery
* Concurrency safety under parallel requests

Run all tests with:

```bash
go test ./...
```

---

## Notes

* Self‑signed TLS certificates will trigger browser warnings
* Docker must be installed to run the WAF and backend services
* The system is designed so optional components can be removed without affecting the core proxy



