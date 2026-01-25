package proxy

import (
	"net/url"
	"sync"
	"time"
)

type Backend struct {
	URL          *url.URL `json:"url"`
	Alive        bool     `json:"alive"`
	CurrentConns int64    `json:"current_connexctions"`
	Weight       int      `json:"weight"`
	mux          sync.RWMutex
}

type ServerPool struct {
	Backends []*Backend `json:"backends"`
	Current  uint64     `json:"current"`
	mux      sync.RWMutex
	Strategy Strategy
}

type ProxyConfig struct {
	Port            int           `json:"port"`
	Strategy        string        `json:"strategy"`
	HealthCheckFreq time.Duration `json:"health_check_frequency"`
}

// helper function to make things concurrency friendly later
func (b *Backend) IsAlive() bool {
	b.mux.RLock() // Lock for Reading (allows multiple readers, blocks writers)
	alive := b.Alive
	b.mux.RUnlock()
	return alive
}
