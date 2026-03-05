package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/charmbracelet/log"
	"github.com/krau/SaveAny-Bot/config"
)

// Server API 服务器
type Server struct {
	httpServer *http.Server
	factory    *TaskFactory
}

// NewServer 创建新的 API 服务器
func NewServer(ctx context.Context) *Server {
	cfg := config.C().API

	factory := NewTaskFactory(ctx)
	handlers := NewHandlers(factory)

	// 设置路由
	mux := http.NewServeMux()

	// 健康检查
	mux.HandleFunc("/health", handlers.HealthCheckHandler)

	// API v1 路由
	mux.HandleFunc("/api/v1/tasks", handlers.CreateTaskHandler)
	mux.HandleFunc("/api/v1/tasks/", func(w http.ResponseWriter, r *http.Request) {
		// 根据方法和路径分发
		switch r.Method {
		case http.MethodGet:
			if r.URL.Path == "/api/v1/tasks" {
				handlers.ListTasksHandler(w, r)
			} else {
				handlers.GetTaskHandler(w, r)
			}
		case http.MethodDelete:
			handlers.CancelTaskHandler(w, r)
		default:
			MethodNotAllowedHandler(w, r)
		}
	})
	mux.HandleFunc("/api/v1/storages", handlers.ListStoragesHandler)
	mux.HandleFunc("/api/v1/task-types", handlers.GetTaskTypesHandler)

	// 404 处理
	mux.HandleFunc("/", NotFoundHandler)

	// 应用中间件
	var handler http.Handler = mux

	// 添加认证中间件
	token := cfg.Token
	if token == "" {
		log.FromContext(ctx).Warn("API server is enabled but no token is set, this is insecure!")
	}
	if token != "" {
		handler = AuthMiddleware()(handler)
	}

	// 添加日志中间件
	handler = loggingMiddleware(handler)

	// 添加恢复中间件
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

// Start 启动服务器
func (s *Server) Start(ctx context.Context) error {
	logger := log.FromContext(ctx).With("module", "api")

	logger.Infof("Starting API server on %s", s.httpServer.Addr)

	// 在 goroutine 中启动服务器
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf("API server error: %v", err)
		}
	}()

	// 监听 context 取消
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

// loggingMiddleware 日志中间件
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 包装 ResponseWriter 以获取状态码
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		log.Infof("%s %s %d %s", r.Method, r.URL.Path, wrapped.statusCode, time.Since(start))
	})
}

// recoveryMiddleware 恢复中间件
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

// responseWriter 包装 http.ResponseWriter 以捕获状态码
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Start 初始化并启动 API 服务器
func Start(ctx context.Context) error {
	cfg := config.C().API

	if !cfg.Enable {
		return nil
	}

	if cfg.Token == "" {
		log.FromContext(ctx).Warn("API server is enabled but no token is set, this is insecure!")
	}

	server := NewServer(ctx)
	return server.Start(ctx)
}
