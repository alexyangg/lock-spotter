package main

import (
	"log"
	"lockspotter-backend/config"
	"lockspotter-backend/database"
)

func main() {
	log.Println("[*] Launching LockSpotter Core Backend Engine...")

	// Load settings
	cfg := config.LoadConfig()

	// Initialize storage engine pools
	storage, err := database.InitStorage(cfg)
	if err != nil {
		log.Fatalf("[!] Core infrastructure initialization failed: %v", err)
	}
	defer storage.Close()

	log.Println("[+] Bootstrapping sequence successful. LockSpotter data engine is completely online.")
}