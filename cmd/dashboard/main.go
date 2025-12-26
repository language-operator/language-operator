/*
Copyright 2025 Langop Team.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"github.com/language-operator/language-operator/pkg/telemetry"
	"github.com/language-operator/language-operator/pkg/telemetry/adapters"
	"github.com/language-operator/language-operator/pkg/telemetry/handlers"
)

func main() {
	// Get port from environment or default to 3000
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	// Initialize telemetry adapter
	adapter, err := initializeTelemetryAdapter()
	if err != nil {
		log.Printf("Failed to initialize telemetry adapter: %v", err)
		log.Printf("Using NoOp adapter (telemetry features will be unavailable)")
		adapter = telemetry.NewNoOpAdapter()
	}

	// Create dashboard handler
	dashboardHandler := handlers.NewDashboardHandler(adapter)

	// Setup router
	router := mux.NewRouter()

	// Register telemetry API routes
	dashboardHandler.RegisterRoutes(router)

	// Add health check endpoint
	router.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if adapter.Available() {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "healthy", "telemetry": "available"}`))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "healthy", "telemetry": "unavailable"}`))
		}
	}).Methods("GET")

	// Serve Next.js static files
	staticPath := os.Getenv("STATIC_PATH")
	if staticPath == "" {
		// Default path for development
		staticPath = "./components/dashboard/.next/static/"
	}

	// Check if we have a built Next.js app to serve
	if _, err := os.Stat(staticPath); err == nil {
		log.Printf("Serving Next.js static files from: %s", staticPath)
		router.PathPrefix("/_next/static/").Handler(http.StripPrefix("/_next/static/", http.FileServer(http.Dir(staticPath))))
	} else {
		log.Printf("Next.js static files not found at %s - API-only mode", staticPath)
	}

	// CORS middleware for development
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:3001"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      c.Handler(router),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting dashboard server on port %s", port)
		log.Printf("Health check: http://localhost:%s/api/health", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// initializeTelemetryAdapter creates and initializes the appropriate telemetry adapter
// based on environment configuration.
func initializeTelemetryAdapter() (telemetry.TelemetryAdapter, error) {
	adapterType := os.Getenv("TELEMETRY_ADAPTER_TYPE")

	switch adapterType {
	case "clickhouse":
		log.Println("Initializing ClickHouse telemetry adapter...")
		adapter, err := adapters.NewClickhouseAdapter()
		if err != nil {
			return nil, fmt.Errorf("failed to create ClickHouse adapter: %w", err)
		}

		// Test availability
		if !adapter.Available() {
			return nil, fmt.Errorf("ClickHouse adapter is not available")
		}

		log.Println("ClickHouse adapter initialized successfully")
		return adapter, nil

	case "signoz":
		log.Println("Initializing SigNoz telemetry adapter...")
		endpoint := os.Getenv("TELEMETRY_ADAPTER_ENDPOINT")
		apiKey := os.Getenv("TELEMETRY_ADAPTER_API_KEY")

		if endpoint == "" {
			return nil, fmt.Errorf("SigNoz adapter requires TELEMETRY_ADAPTER_ENDPOINT")
		}
		if apiKey == "" {
			return nil, fmt.Errorf("SigNoz adapter requires TELEMETRY_ADAPTER_API_KEY")
		}

		adapter, err := adapters.NewSignozAdapter(endpoint, apiKey, 30*time.Second)
		if err != nil {
			return nil, fmt.Errorf("failed to create SigNoz adapter: %w", err)
		}

		// Test availability
		if !adapter.Available() {
			return nil, fmt.Errorf("SigNoz adapter is not available")
		}

		log.Println("SigNoz adapter initialized successfully")
		return adapter, nil

	case "noop", "":
		log.Println("Using NoOp telemetry adapter (telemetry disabled)")
		return telemetry.NewNoOpAdapter(), nil

	default:
		return nil, fmt.Errorf("unsupported telemetry adapter type: %s", adapterType)
	}
}
