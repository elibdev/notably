package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/elibdev/notably/pkg/server"
)

func main() {
	// Parse command-line flags
	var addr string
	flag.StringVar(&addr, "addr", ":8080", "HTTP listen address")
	flag.Parse()

	// Initialize server configuration
	config := server.DefaultConfig()
	
	// Override address from flag
	if addr != "" {
		config.Addr = addr
	}

	// Validate required environment variables
	if config.TableName == "" {
		log.Fatal("DYNAMODB_TABLE_NAME environment variable is required")
	}

	// Create server instance
	srv, err := server.NewServer(config)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Set up signal handling for graceful shutdown
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on %s", config.Addr)
		if err := srv.Run(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for termination signal
	<-stopChan
	log.Println("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Graceful shutdown
	if err := srv.Stop(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	log.Println("Server gracefully stopped")
}