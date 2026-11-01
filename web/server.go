package web

import (
	"context"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

type Server struct {
	listen     string
	authToken  string
	apiHandler http.Handler
	httpServer *http.Server
}

func New(listen, authToken string, apiHandler http.Handler) *Server {
	s := &Server{listen: listen, authToken: authToken, apiHandler: apiHandler}
	root := http.NewServeMux()
	root.Handle("/api/", apiHandler)
	root.Handle("/health", apiHandler)
	root.HandleFunc("/", s.serveUI)
	s.httpServer = &http.Server{
		Handler:           securityHeaders(root),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	return s
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) Start(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.listen)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.listen, err)
	}
	log.FromContext(ctx).Infof("Starting web server on %s", ln.Addr())
	if uiURL := webURL(ln.Addr(), s.authToken); uiURL != "" {
		log.FromContext(ctx).Infof("Web UI available at %s", uiURL)
	}
	go func() {
		if err := s.httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.FromContext(ctx).Errorf("Web server error: %v", err)
		}
	}()
	go func() {
		<-ctx.Done()
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(stopCtx); err != nil {
			log.FromContext(ctx).Errorf("Web server shutdown error: %v", err)
		}
	}()
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func webURL(addr net.Addr, token string) string {
	if addr == nil {
		return ""
	}
	host, port, err := net.SplitHostPort(addr.String())
	if err != nil {
		return ""
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsUnspecified() {
		if ip.To4() != nil {
			host = "127.0.0.1"
		} else {
			host = "::1"
		}
	}
	u := url.URL{Scheme: "http", Host: net.JoinHostPort(host, port), Path: "/"}
	if token != "" {
		query := u.Query()
		query.Set("token", token)
		u.RawQuery = query.Encode()
	}
	return u.String()
}

func (s *Server) serveUI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.Method == http.MethodGet {
		if token := strings.TrimSpace(r.URL.Query().Get("token")); token != "" {
			query := url.Values{"token": {token}}
			w.Header().Set("Cache-Control", "no-store")
			http.Redirect(w, r, "/api/v1/auth/token?"+query.Encode(), http.StatusSeeOther)
			return
		}
	}
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		http.Error(w, "Web UI not built: run make web", http.StatusServiceUnavailable)
		return
	}
	index, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		http.Error(w, "Web UI not built: run make web", http.StatusServiceUnavailable)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path != "" {
		if file, err := sub.Open(path); err == nil {
			info, statErr := file.Stat()
			_ = file.Close()
			if statErr == nil && !info.IsDir() {
				if strings.HasPrefix(path, "assets/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				http.FileServer(http.FS(sub)).ServeHTTP(w, r)
				return
			}
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(index)
}
