package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/config"
)

type Server struct {
	httpServer *http.Server
	factory    *TaskFactory
}

func NewServer(ctx context.Context) *Server {
	cfg := config.C().API

	factory := NewTaskFactory(ctx)
	handlers := NewHandlers(factory)

	mux := http.NewServeMux()

	mux.HandleFunc("/health", handlers.HealthCheckHandler)

	mux.HandleFunc("/api/v1/tasks", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.ListTasksHandler(w, r)
		case http.MethodPost:
			handlers.CreateTaskHandler(w, r)
		default:
			MethodNotAllowedHandler(w, r)
		}
	})
	mux.HandleFunc("/api/v1/tasks/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.GetTaskHandler(w, r)
		case http.MethodDelete:
			handlers.CancelTaskHandler(w, r)
		default:
			MethodNotAllowedHandler(w, r)
		}
	})
	mux.HandleFunc("/api/v1/storages", handlers.ListStoragesHandler)
	mux.HandleFunc("/api/v1/media-metadata", handlers.GetMediaMetadataHandler)
	mux.HandleFunc("/api/v1/task-types", handlers.GetTaskTypesHandler)

	mux.HandleFunc("/", NotFoundHandler)

	var handler http.Handler = mux

	token := cfg.Token
	if token != "" {
		handler = AuthMiddleware()(handler)
	}

	handler = loggingMiddleware(handler)

	handler = recoveryMiddleware(handler)

	return &Server{
		httpServer: &http.Server{
			Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
			Handler:      handler,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
		factory: factory,
	}
}

func (s *Server) Start(ctx context.Context) error {
	logger := log.FromContext(ctx).With("module", "api")

	logger.Infof("Starting API server on %s", s.httpServer.Addr)

	// Bind synchronously so listen failures are returned to the caller.
	ln, err := net.Listen("tcp", s.httpServer.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.httpServer.Addr, err)
	}

	go func() {
		if err := s.httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
			logger.Errorf("API server error: %v", err)
		}
	}()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Errorf("API server shutdown error: %v", err)
		}
	}()

	return nil
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		log.Infof("%s %s %d %s", r.Method, r.URL.Path, wrapped.statusCode, time.Since(start))
	})
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Errorf("Panic recovered: %v", err)
				WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Start initializes and starts the API server. It refuses to start without a
// token, since an open download proxy is a security risk.
func Start(ctx context.Context) error {
	cfg := config.C().API

	if !cfg.Enable {
		return nil
	}

	if cfg.Token == "" {
		return fmt.Errorf("API server is enabled but no token is set; refusing to start insecurely")
	}

	server := NewServer(ctx)
	if err := server.Start(ctx); err != nil {
		return err
	}
	StartCleanupLoop(ctx)
	return nil
}
