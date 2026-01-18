package proxy	

import (
	"encoding/json"
	"net/http"
	"net/url"
)

// payload defines the expected JSON structure for Add/Remove requests
type payload struct {
	URL string `json:"url"`
}

// GetStatus handles GET /status
// returns the entire ServerPool status as JSON
func (s *ServerPool) GetStatus(w http.ResponseWriter, r *http.Request) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.Backends)
}

// AddBackendHandler handles POST /backends
func (s *ServerPool) AddBackendHandler(w http.ResponseWriter, r *http.Request) {
	var p payload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	u, err := url.Parse(p.URL)
	if err != nil {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	// Create new backend and add to pool
	b := &Backend{URL: u, Alive: true}
	s.AddBackend(b)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Backend added"))
}

// RemoveBackendHandler handles DELETE /backends
func (s *ServerPool) RemoveBackendHandler(w http.ResponseWriter, r *http.Request) {
	var p payload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	u, err := url.Parse(p.URL)
	if err != nil {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	s.RemoveBackend(u)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Backend removed"))
}