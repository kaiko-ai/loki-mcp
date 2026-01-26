package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/server"

	"github.com/grafana/loki-mcp/internal/handlers"
)

// Default configuration values
const (
	DefaultPort            = "8080"
	DefaultShutdownTimeout = 5 * time.Second
)

// version is set via -ldflags at build time
var version = "dev"

func main() {
	// Create a new MCP server
	s := server.NewMCPServer(
		"Loki MCP Server",
		version,
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
	)

	// Add Loki query tool
	lokiQueryTool := handlers.NewLokiQueryTool()
	s.AddTool(lokiQueryTool, handlers.HandleLokiQuery)

	// Add Loki label names tool
	lokiLabelNamesTool := handlers.NewLokiLabelNamesTool()
	s.AddTool(lokiLabelNamesTool, handlers.HandleLokiLabelNames)

	// Add Loki label values tool
	lokiLabelValuesTool := handlers.NewLokiLabelValuesTool()
	s.AddTool(lokiLabelValuesTool, handlers.HandleLokiLabelValues)

	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = DefaultPort
	}

	// Create SSE server for legacy SSE connections
	sseServer := server.NewSSEServer(s,
		server.WithSSEEndpoint("/sse"),
		server.WithMessageEndpoint("/mcp"),
	)

	// Create Streamable HTTP server
	streamableServer := server.NewStreamableHTTPServer(s)

	// Create a multiplexer to handle both protocols on the same port
	mux := http.NewServeMux()

	// Register SSE endpoints (legacy support)
	mux.Handle("/sse", sseServer) // SSE event stream
	mux.Handle("/mcp", sseServer) // SSE message endpoint

	// Register Streamable HTTP endpoint
	mux.Handle("/stream", streamableServer) // Streamable HTTP endpoint

	// Create HTTP server with explicit configuration
	addr := fmt.Sprintf(":%s", port)
	httpServer := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a channel to handle shutdown signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// WaitGroup to track goroutines
	var wg sync.WaitGroup

	// Start unified HTTP server
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("Starting unified MCP server on http://localhost%s", addr)
		log.Printf("SSE Endpoint (legacy): http://localhost%s/sse", addr)
		log.Printf("SSE Message Endpoint: http://localhost%s/mcp", addr)
		log.Printf("Streamable HTTP Endpoint: http://localhost%s/stream", addr)

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// For backward compatibility, also serve via stdio
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("Starting stdio server")
		if err := server.ServeStdio(s); err != nil {
			log.Printf("Stdio server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-stop
	log.Println("Shutting down servers...")

	// Cancel context to signal shutdown
	cancel()

	// Gracefully shutdown HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, DefaultShutdownTimeout)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// Wait for goroutines to finish (with timeout via context)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All servers stopped gracefully")
	case <-shutdownCtx.Done():
		log.Println("Shutdown timeout exceeded, forcing exit")
	}
}
