package proxy

import (
	"crypto/md5"
	"encoding/hex"
	"log"
	"net/http"
	"net/http/httputil"
	"sync/atomic"
)

// handle incoming HTTP request and forward it to a backend
func (s *ServerPool) Proxy(w http.ResponseWriter, r *http.Request) {
	// the pool asks its current strategy for a peer
	s.mux.RLock()
	backends := s.Backends
	s.mux.RUnlock()
	// peer, err := s.Strategy.GetPeer(backends)
	var peer *Backend
	var err error

	if s.Sticky {
		// check for existing cookie
		cookie, cookieErr := r.Cookie("proxy_session")
		if cookieErr == nil {
			// the user has a cookie, we find the matching backend
			cookieID := cookie.Value
			for _, b := range backends {
				if generateBackendID(b.URL.String()) == cookieID && b.IsAlive() {
					peer = b
					log.Printf("Sticky session detected, routing directly to: %s", b.URL)
					break
				}
			}
		}
	}

	// if no peer found yet , use load balancer
	// this runs if they had no cookie, OR if their sticky backend died.
	if peer == nil {
		peer, err = s.Strategy.GetPeer(backends)

		if err != nil {
			http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
			return
		}

		// issue the cookie for their next visit
		if s.Sticky {
			backendID := generateBackendID(peer.URL.String())
			http.SetCookie(w, &http.Cookie{
				Name:     "proxy_session",
				Value:    backendID,
				Path:     "/", 
				HttpOnly: true, 
				MaxAge:   3600, 
			})
		}
	}

	// Safely increment connections
	atomic.AddInt64(&peer.CurrentConns, 1)
	defer atomic.AddInt64(&peer.CurrentConns, -1)

	// setting up the reverse proxy
	// using the standart library helper
	rp := httputil.NewSingleHostReverseProxy(peer.URL)

	// update the headers to allow the backend to know the original host
	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))

	// logging
	log.Println("forwarding request to: ", peer.URL)

	// assign a custom error handler
	rp.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, e error) {
		log.Printf("[%s] %s\n", peer.URL.Host, e.Error())

		// mark the backend as dead immediately
		s.SetBackendStatus(peer.URL, false)

		// tell the user something went wrong
		writer.WriteHeader(http.StatusBadGateway)
		writer.Write([]byte("The server is down"))
	}

	// forward request -> Wait for Response -> Copy back to user
	rp.ServeHTTP(w, r)

}

// helper function to encrypt backends
// creates a safe hash from the backend URL for the cookie
func generateBackendID(url string) string {
	hash := md5.Sum([]byte(url))
	return hex.EncodeToString(hash[:])
}
