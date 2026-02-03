// Package main provides the entry point for the Shared-Wisdom gateway server.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/thoughtracker/shared-wisdom/internal/api"
	"github.com/thoughtracker/shared-wisdom/internal/store"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("DEBUG: Gateway starting...")

	// Parse command line flags
	addr := flag.String("addr", ":8080", "Server address")
	dbPath := flag.String("db", "gateway.db", "Database path")
	hubURL := flag.String("hub", "https://hub1.wisdom.spawning.de", "Upstream hub URL")
	flag.Parse()

	log.Printf("DEBUG: addr=%s, db=%s, hub=%s", *addr, *dbPath, *hubURL)

	// Initialize store
	log.Println("DEBUG: Initializing store...")
	s, err := store.NewSQLiteStore(*dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}
	defer s.Close()
	log.Println("DEBUG: Store initialized")

	// Initialize API server
	log.Println("DEBUG: Creating API server...")
	srv := api.NewServer(s, api.Config{
		Addr:   *addr,
		HubURL: *hubURL,
	})
	log.Println("DEBUG: API server created")

	// Start server in background
	go func() {
		log.Printf("DEBUG: Starting HTTP listener on %s", *addr)
		if err := srv.Start(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	log.Printf("Gateway started on %s (hub: %s)", *addr, *hubURL)

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
