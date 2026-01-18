package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	// We will launch 3 servers in parallel using Goroutines
	ports := []string{"8084", "8082", "8083"}

	// Channel to keep the main function from exiting
	forever := make(chan bool)

	for _, port := range ports {
		// Start a separate server for each port
		go func(p string) {
			mux := http.NewServeMux()
			
			// Handle the root path "/"
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				// Print to the server's console so we see traffic arriving
				log.Printf("Server on port %s received request\n", p)
				
				// Reply to the browser
				fmt.Fprintf(w, "Hello from Backend Server on Port %s!", p)
			})

			log.Printf("Backend server starting on port %s...", p)
			err := http.ListenAndServe(":"+p, mux)
			if err != nil {
				log.Fatal(err)
			}
		}(port)
	}

	// Block forever so the program doesn't exit
	<-forever
}