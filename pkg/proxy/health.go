package proxy

import (
	"log"
	// "net"
	"net/http"
	"net/url"
	"time"
)

// isAlive tries to connect to the backend URL via http GET to avoid race consitions.
// it returns true if the connection succeeds, false otherwise.
// Replace the net.DialTimeout logic in health.go
func isAlive(u *url.URL) bool {
    timeout := 2 * time.Second
	// make request
    client := http.Client{
        Timeout: timeout,
    }
    resp, err := client.Get(u.String())
    if err != nil {
        log.Println("Site unreachable:", err)
        return false
    }
    defer resp.Body.Close()
    return resp.StatusCode == http.StatusOK
}

// loops forever and performs checks every  specified interval
// REMEMBER: it starts as a go routine in main!!!!!!!!!!
func (s *ServerPool) HealthCheck(interval time.Duration) {
	// define the clock ticker
	t := time.NewTicker(interval)
	// the loop
	for {
		// wait for tick (stop until there's a tick)
		<-t.C
		// check every server
		// lock s for redingbecause it's being updated
		s.mux.RLock()
		for _, b := range s.Backends {
			// check if alive (ping)
			status := isAlive(b.URL)
			// change our knowledge of the status
			// lock backend
			b.mux.Lock()
			if b.Alive != status {
				b.Alive = status
				log.Println("URL status changed, ", b.URL, "is now: ", b.Alive)
			}
			// unlock backend
			b.mux.Unlock()

		}
		// round finished, unlock s
		s.mux.RUnlock()
	}
}
