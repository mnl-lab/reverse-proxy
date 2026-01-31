package proxy_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	proxy "reverse_proxy/pkg/proxy"
)

func mustBackend(t *testing.T, raw string) *proxy.Backend {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse backend url: %v", err)
	}
	return &proxy.Backend{URL: u, Alive: true, Weight: 1}
}

func TestRoundRobinCyclesAcrossAliveBackends(t *testing.T) {
	rr := &proxy.RoundRobin{}
	backends := []*proxy.Backend{
		mustBackend(t, "http://localhost:8081"),
		mustBackend(t, "http://localhost:8082"),
		mustBackend(t, "http://localhost:8083"),
	}

	seen := map[string]struct{}{}
	for i := 0; i < len(backends); i++ {
		peer, err := rr.GetPeer(backends)
		if err != nil {
			t.Fatalf("round robin returned error: %v", err)
		}
		seen[peer.URL.String()] = struct{}{}
	}

	if len(seen) != len(backends) {
		t.Fatalf("expected %d unique peers, got %d", len(backends), len(seen))
	}
}

func TestRoundRobinSkipsDeadBackends(t *testing.T) {
	rr := &proxy.RoundRobin{}
	backends := []*proxy.Backend{
		mustBackend(t, "http://localhost:8081"),
		mustBackend(t, "http://localhost:8082"),
	}
	backends[0].Alive = false

	for i := 0; i < 4; i++ {
		peer, err := rr.GetPeer(backends)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if peer.URL.String() == backends[0].URL.String() {
			t.Fatalf("round robin returned dead backend: %s", peer.URL)
		}
	}
}

func TestLeastConnectionsPrefersLeastBusy(t *testing.T) {
	lc := &proxy.LeastConnections{}
	backends := []*proxy.Backend{
		mustBackend(t, "http://localhost:9001"),
		mustBackend(t, "http://localhost:9002"),
	}
	atomic.StoreInt64(&backends[0].CurrentConns, 5)
	atomic.StoreInt64(&backends[1].CurrentConns, 1)

	peer, err := lc.GetPeer(backends)
	if err != nil {
		t.Fatalf("least connections returned error: %v", err)
	}
	if peer.URL.String() != backends[1].URL.String() {
		t.Fatalf("expected %s, got %s", backends[1].URL, peer.URL)
	}
}

func TestLeastConnectionsErrorsWhenNoAliveBackend(t *testing.T) {
	lc := &proxy.LeastConnections{}
	backends := []*proxy.Backend{mustBackend(t, "http://localhost:9101")}
	backends[0].Alive = false

	if _, err := lc.GetPeer(backends); err == nil {
		t.Fatal("expected error when no alive backend")
	}
}

func TestWeightedRoundRobinHonorsWeights(t *testing.T) {
	wrr := &proxy.WeightedRoundRobin{}
	heavy := mustBackend(t, "http://localhost:9201")
	heavy.Weight = 2
	light := mustBackend(t, "http://localhost:9202")
	light.Weight = 1

	counts := map[string]int{}
	for i := 0; i < 6; i++ {
		peer, err := wrr.GetPeer([]*proxy.Backend{heavy, light})
		if err != nil {
			t.Fatalf("weighted round robin error: %v", err)
		}
		counts[peer.URL.String()]++
	}

	if counts[heavy.URL.String()] != 4 {
		t.Fatalf("expected heavy backend 4 selections, got %d", counts[heavy.URL.String()])
	}
	if counts[light.URL.String()] != 2 {
		t.Fatalf("expected light backend 2 selections, got %d", counts[light.URL.String()])
	}
}

func TestServerPoolAddSetRemoveBackend(t *testing.T) {
	pool := &proxy.ServerPool{}
	backend := mustBackend(t, "http://localhost:9301")

	pool.AddBackend(backend)
	if len(pool.Backends) != 1 {
		t.Fatalf("expected 1 backend, got %d", len(pool.Backends))
	}

	pool.SetBackendStatus(backend.URL, false)
	if backend.IsAlive() {
		t.Fatal("expected backend to be marked dead")
	}

	pool.RemoveBackend(backend.URL)
	if len(pool.Backends) != 0 {
		t.Fatalf("expected backend removal, remaining: %d", len(pool.Backends))
	}
}

func TestGetStatusReturnsPoolState(t *testing.T) {
	pool := &proxy.ServerPool{}
	pool.AddBackend(mustBackend(t, "http://localhost:9401"))

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rr := httptest.NewRecorder()
	pool.GetStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var payload []proxy.Backend
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode status payload: %v", err)
	}
	if len(payload) != 1 {
		t.Fatalf("expected 1 backend in payload, got %d", len(payload))
	}
}

func TestAddBackendHandlerAddsBackend(t *testing.T) {
	pool := &proxy.ServerPool{}
	body := bytes.NewBufferString(`{"url":"http://localhost:9501"}`)
	req := httptest.NewRequest(http.MethodPost, "/backends", body)
	rr := httptest.NewRecorder()

	pool.AddBackendHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if len(pool.Backends) != 1 {
		t.Fatalf("expected backend to be added, got %d entries", len(pool.Backends))
	}
}

func TestRemoveBackendHandlerRemovesBackend(t *testing.T) {
	pool := &proxy.ServerPool{}
	backend := mustBackend(t, "http://localhost:9601")
	pool.AddBackend(backend)

	body := bytes.NewBufferString(`{"url":"http://localhost:9601"}`)
	req := httptest.NewRequest(http.MethodDelete, "/backends", body)
	rr := httptest.NewRecorder()

	pool.RemoveBackendHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if len(pool.Backends) != 0 {
		t.Fatalf("expected backend removal, remaining: %d", len(pool.Backends))
	}
}
