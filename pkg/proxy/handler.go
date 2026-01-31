package proxy

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"sync/atomic"
	"time"
)

// handle incoming HTTP request and forward it to a backend
func (s *ServerPool) Proxy(w http.ResponseWriter, r *http.Request) {

	// ---  AI GUARDIAN CHECK  ---
	// We send the full URL path and query (e.g., "/?search=' OR 1=1")
	fullURL := r.URL.String()

	if checkWAF(fullURL) {
		log.Printf("[BLOCKED] AI Guardian stopped malicious request: %s", fullURL)
		w.WriteHeader(http.StatusForbidden) // 403 Forbidden
		w.Write([]byte("<h1>403 Forbidden</h1><p>Malicious Request Detected by AI Guardian.</p>"))
		return // Stop processing! Don't touch the backends.
	}

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
			// If no backends are available, return 503
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
	// decrement the count upon completion
	defer atomic.AddInt64(&peer.CurrentConns, -1)

	// setting up the reverse proxy
	// using the standart library helper
	rp := httputil.NewSingleHostReverseProxy(peer.URL)

	// handle backend timeouts and connection logic
	// this ensures slow backends don't hang the proxy indefinitely
	rp.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second, // max time to connect to backend
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
	}

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
	// ServeHTTP uses the request context automatically, handling client cancellations [cite: 64, 84]
	rp.ServeHTTP(w, r)

}

// helper function to encrypt backends
// creates a safe hash from the backend URL for the cookie
func generateBackendID(url string) string {
	hash := md5.Sum([]byte(url))
	return hex.EncodeToString(hash[:])
}

// WAFResponse defines what we expect back from Python
type WAFResponse struct {
	Prediction int    `json:"prediction"`
	Status     string `json:"status"`
}

// checkWAF sends the URL to the Python AI and returns true if malicious
func checkWAF(targetURL string) bool {
	// 1. Prepare the JSON payload
	requestBody, _ := json.Marshal(map[string]string{
		"url": targetURL,
	})

	// 2. Create the HTTP request to Python (Port 5000)
	// We set a short timeout (500ms) so the WAF doesn't slow down the site
	client := http.Client{
		Timeout: 500 * time.Millisecond,
	}

	resp, err := client.Post("http://localhost:5000/predict", "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		// If Python is down, we "Fail Open" (Log error but let traffic pass)
		// In high security, you would "Fail Closed" (Block everything)
		log.Println("WAF ERROR: Could not contact AI Guardian:", err)
		return false
	}
	defer resp.Body.Close()

	// 3. Read the verdict
	body, _ := io.ReadAll(resp.Body)
	var wafResult WAFResponse
	json.Unmarshal(body, &wafResult)

	// Return true if prediction is 1 (Malicious)
	return wafResult.Prediction == 1
}
