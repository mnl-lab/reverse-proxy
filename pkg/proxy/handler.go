package proxy

import (
	"log"
	"net/http"
	"net/http/httputil"
)

// handle incoming HTTP request and forward it to a backend
func (s *ServerPool) Proxy(w http.ResponseWriter, r *http.Request) {
	// get a valid backend
	peer, err := s.GetNextValidPeer()

	// no valid backend is available
	if err != nil {
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}
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
