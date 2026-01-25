package proxy

import (
	"errors"
	"sync/atomic"

	"net/url"
)

type LoadBalancer interface {
	AddBackend(backend *Backend)
	SetBackendStatus(uri *url.URL, alive bool)
}

type Strategy interface {
	GetPeer(backends []*Backend) (*Backend, error)
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

func (s *ServerPool) RemoveBackend(backendURL *url.URL){
	s.mux.Lock()
	defer s.mux.Unlock()

	// iterate to finc the backend

	for i,b := range s.Backends {
		if b.URL.String() == backendURL.String() {
			// the trick ;)
			s.Backends = append(s.Backends[:i], s.Backends[i+1:]...)
			return
		}
	}
}



// Round-Robin Strategy
type RoundRobin struct {
	Current uint64
}

// Least-Connections Strategy
type LeastConnections struct {}

// weighted round robin
type WeightedRoundRobin struct {
	Current uint64
}
// TODO: add weighted least conn

// getPeer for round roibn
func (rr *RoundRobin) GetPeer(backends []*Backend) (*Backend, error) {
    // if no backends are available
	if len(backends) == 0 {
		return nil, errors.New("no backends available")
	}
	// loop through the backends to find a valid one

	// automatically add 1 to s.Current and returns the NEW value
	// used to determine where to start looking exactly when multiple users are accessing the function
	// even if 1,000 requests come in instantly, Request A gets next=1, Request B gets next=2, etc
	next := atomic.AddUint64(&rr.Current, 1)
	for i := range backends {
		//this is the core of the Round-Robin strategy, to create a loop in selection no matter the current index
		idx := int((next + uint64(i)) % uint64(len(backends)))
		if backends[idx].IsAlive() {
			// return the first alive valid next peer

			return backends[idx], nil
		}
	}
	// nothing is found by here
	return nil, errors.New("no valid peer found")
}

// get peer for least conn
func (lc *LeastConnections) GetPeer(backends []*Backend) (*Backend, error) {
    
	var bestPeer *Backend
	var minConns int64 = -1
	// if no backends are available
	if len(backends) == 0 {
		return nil, errors.New("no backends available")
	}

	for _ , b := range backends {
		// skip the backend if it's not alive
		if ! b.IsAlive() {
			continue
		}
		// use atomic load for concurrency safety
		conns := atomic.LoadInt64(&b.CurrentConns)
		// if we're just starting or a better value is found
		if minConns == -1 || conns < minConns{
			minConns = conns
			bestPeer = b
		}

	}
	if bestPeer == nil {
		return nil, errors.New("no available bachend is found")
	}
	return bestPeer, nil
}

func (wrr *WeightedRoundRobin) GetPeer(backends []*Backend) (*Backend, error){
	// if no backends are available
	if len(backends) == 0 {
		return nil, errors.New("no backends available")
	}
	// STEP 1: Calculate the Total Weight of all ALIVE backends
	var totalWeight int
	for _, b := range backends {
		if b.IsAlive() {
			totalWeight += b.Weight
		}
	}

	if totalWeight == 0 {
		return nil, errors.New("no alive backends available")
	}
	
	// STEP 2:  
	// Get the next number and mod it by totalWeight. 
	// similar to regular round robin
	next := atomic.AddUint64(&wrr.Current, 1)
	target := int(next % uint64(totalWeight))

	// STEP 3: Find which server owns this "target" number
	for _, b := range backends {
		if !b.IsAlive() {
			continue
		}

		// subtract this backend's weight from the target.
		// when target drops below 0, we have found THE ONE.
		target -= b.Weight
		if target < 0 {
			return b, nil
		}
	}
	
	return nil, errors.New("no valid peer found")
}