package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/config"
)

var server *http.Server

// Init initializes and starts the HTTP API server
func Init(ctx context.Context) error {
	cfg := config.C()
	if !cfg.API.Enable {
		return nil
	}

	// Validate that token is configured when API is enabled
	if cfg.API.Token == "" {
		return fmt.Errorf("API is enabled but token is not configured. Please set 'api.token' in your configuration file for security")
	}

	logger := log.FromContext(ctx).WithPrefix("api")

	mux := http.NewServeMux()

	// Register API routes
	registerRoutes(mux)

	// Wrap with middleware
	handler := loggingMiddleware(logger)(authMiddleware(mux))

	server = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.API.Port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Infof("Starting API server on port %d", cfg.API.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf("API server error: %v", err)
		}
	}()

	// Graceful shutdown on context cancellation
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Errorf("Failed to shutdown API server: %v", err)
		} else {
			logger.Info("API server stopped")
		}
	}()

	return nil
}

func registerRoutes(mux *http.ServeMux) {
	// Health check endpoint (no auth required)
	mux.HandleFunc("/health", handleHealth)

	// API v1 endpoints
	mux.HandleFunc("POST /api/v1/tasks", handleCreateTask)
	mux.HandleFunc("GET /api/v1/tasks/{id}", handleGetTask)
	mux.HandleFunc("GET /api/v1/tasks", handleListTasks)
	mux.HandleFunc("DELETE /api/v1/tasks/{id}", handleCancelTask)
}
