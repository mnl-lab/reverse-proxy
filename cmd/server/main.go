package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"reverse_proxy/pkg/config"
	"reverse_proxy/pkg/proxy"
	"time"
)

func main() {

	// parse flags to use: go run main.go -config=config.json
	configFile := flag.String("config", "confg.json", "config file path")
	flag.Parse()

	// load configurations
	conf, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatal("failed to load config", err)
	}
	log.Printf("Loaded config succesfully: Port=%d, Strategy=%s", conf.Port, conf.Strategy)

	var strategy proxy.Strategy

	switch conf.Strategy {
	case "least-conn":
		strategy = &proxy.LeastConnections{}
	case "round-robin":
		strategy = &proxy.RoundRobin{}
	case "weighted-round-robin":
		strategy = &proxy.WeightedRoundRobin{}
	default:
		strategy = &proxy.RoundRobin{} // Default fallback
	}

	// we manually build the pool to handle the URL parsing safely
	serverPool := &proxy.ServerPool{
		Backends: make([]*proxy.Backend, 0),
		Strategy: strategy,
	}

	// add config backends to the pool
	for _, b := range conf.Backends {
		// transform the backen url into a URL object
		parsedURL, err := url.Parse(b.URL)
		if err != nil {
			log.Println("skipping, unvalid url: ", b.URL)
			continue
		}
		// make a backend object
		backend := &proxy.Backend{
			URL:   parsedURL,
			Alive: b.Alive,
			Weight: b.Weight,
			
		}
		serverPool.AddBackend(backend)

	}
	// parse the duration string into time.Duration
	freq, err := time.ParseDuration(conf.HealthCheckFreq)
	if err != nil {
		log.Println("Invalid health check frequency, defaulting to 10s")
		freq = 10 * time.Second
	}

	// launch background health check
	go serverPool.HealthCheck(freq)

	// admin dashboard wiring
	// go routine ofc
	go func() {
		// create a separate router for Admin so normal users can't access it
		adminMux := http.NewServeMux()
		adminMux.HandleFunc("/status", serverPool.GetStatus)
		adminMux.HandleFunc("/backends", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost:
				serverPool.AddBackendHandler(w, r)
			case http.MethodDelete:
				serverPool.RemoveBackendHandler(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})

		log.Println("Admin API started on port :8081")
		if err := http.ListenAndServe(":8081", adminMux); err != nil {
			log.Fatalf("Admin server failed: %v", err)
		}
	}()

	// start HTTP server (again...)
	port := fmt.Sprintf(":%d", conf.Port)
	server := http.Server{
		Addr:    port,
		Handler: http.HandlerFunc(serverPool.Proxy),
	}

	log.Println("Secure Load Balancer started on port: ", port)

	// ListenAndServeTLS takes the cert file and key file as parameters
	if err := server.ListenAndServeTLS("cert.pem", "key.pem"); err != nil {
		log.Fatalf("Secure Server failed: %v", err)
	}

}
