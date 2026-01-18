package proxy

import (
	"errors"
	"sync/atomic"

	"net/url"
)

type LoadBalancer interface {
	GetNextValidPeer() (*Backend, error)
	AddBackend(backend *Backend)
	SetBackendStatus(uri *url.URL, alive bool)
}

func (s *ServerPool) GetNextValidPeer() (*Backend, error) {
	// if no backends are available
	if len(s.Backends) == 0 {
		return nil, errors.New("no backends available")
	}
	// loop through the backends to find a valid one

	// automatically add 1 to s.Current and returns the NEW value
	// used to determine where to start looking exactly when multiple users are accessing the function
	// even if 1,000 requests come in instantly, Request A gets next=1, Request B gets next=2, etc
	next := atomic.AddUint64(&s.Current, 1)
	for i := range s.Backends {
		//this is the core of the Round-Robin strategy, to create a loop in selection no matter the current index
		idx := int((next + uint64(i)) % uint64(len(s.Backends)))
		if s.Backends[idx].IsAlive() {
			// return the first alive valid next peer

			return s.Backends[idx], nil
		}
	}
	// nothing is found by here
	return nil, errors.New("no valid peer found")

}

func (s *ServerPool) AddBackend(backend *Backend) {
	// lock the slice -> add -> unlock
	s.mux.Lock()
	s.Backends = append(s.Backends, backend)
	s.mux.Unlock()
}

func (s *ServerPool) SetBackendStatus(uri *url.URL, alive bool) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	for _, b := range s.Backends {
		if b.URL.String() == uri.String() { // b.URL and uri are pointers, we compare values
			b.mux.Lock()
			b.Alive = alive
			b.mux.Unlock()
			break // match found, no need to keep iterating
		}
	}
}
