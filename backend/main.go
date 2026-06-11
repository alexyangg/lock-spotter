package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"lockspotter-backend/config"
	"lockspotter-backend/database"
	"lockspotter-backend/handlers"
)

func main() {
	log.Println("[*] Starting LockSpotter Geospatial Server Core...")

	// 1. Initialize configuration values
	cfg := config.LoadConfig()

	// 2. Initialize Thread-Safe connection pools
	storage, err := database.InitStorage(cfg)
	if err != nil {
		log.Fatalf("[!] Storage initialization block failed: %v", err)
	}
	defer storage.Close()

	// 3. Initialize Handler instances
	rackHandler := handlers.NewRackHandler(storage)

	// 4. Setup multiplexer request routes
	mux := http.NewServeMux()
	mux.HandleFunc("/api/racks/nearby", rackHandler.HandleGetNearbyRacks)

	// 5. Construct structural server parameters
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 6. Graceful Shutdown listener interceptor
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("[+] High-performance Go worker listening on Port %s...", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[!] Core transport layer failed: %v", err)
		}
	}()

	// Wait until an interrupt signal hits the terminal
	<-shutdownChan
	log.Println("[*] System intercept registered. Terminating operational threads gracefully...")
}