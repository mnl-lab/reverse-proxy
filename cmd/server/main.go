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

	// we manually build the pool to handle the URL parsing safely
	serverPool := &proxy.ServerPool{
		Backends: make([]*proxy.Backend, 0),
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

	// start HTTP server (again...)
	port := fmt.Sprintf(":%d", conf.Port)
	server := http.Server{
		Addr:    port,
		Handler: http.HandlerFunc(serverPool.Proxy),
	}

	log.Println("Load Balancer started on port: ", port)

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}

}
