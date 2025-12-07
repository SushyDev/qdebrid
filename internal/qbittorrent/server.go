package qbittorrent

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Server wraps the HTTP server with routing
type Server struct {
	handler    *Handler
	httpServer *http.Server
	logger     *zap.Logger
}

// NewServer creates a new HTTP server
func NewServer(handler *Handler, host string, port int, logger *zap.Logger) *Server {
	mux := http.NewServeMux()

	// Health check endpoint (no auth required)
	mux.HandleFunc("/health", handler.Health)

	// Register routes
	const apiPrefix = "/api/v2"

	// Auth endpoints
	mux.HandleFunc(apiPrefix+"/auth/login", handler.Login)

	// App endpoints
	mux.HandleFunc(apiPrefix+"/app/webapiVersion", handler.Version)
	mux.HandleFunc(apiPrefix+"/app/preferences", handler.Preferences)

	// Torrent endpoints
	mux.HandleFunc(apiPrefix+"/torrents/add", handler.Add)
	mux.HandleFunc(apiPrefix+"/torrents/info", handler.Info)
	mux.HandleFunc(apiPrefix+"/torrents/properties", handler.Properties)
	mux.HandleFunc(apiPrefix+"/torrents/files", handler.Files)
	mux.HandleFunc(apiPrefix+"/torrents/delete", handler.Delete)
	mux.HandleFunc(apiPrefix+"/torrents/categories", handler.Categories)

	// Apply middleware
	var finalHandler http.Handler = mux
	finalHandler = loggingMiddleware(logger)(finalHandler)
	finalHandler = recoveryMiddleware(logger)(finalHandler)
	finalHandler = timeoutMiddleware(5 * time.Minute)(finalHandler)

	addr := fmt.Sprintf("%s:%d", host, port)

	httpServer := &http.Server{
		Addr:           addr,
		Handler:        finalHandler,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   5 * time.Minute, // Long timeout for large file operations
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	return &Server{
		handler:    handler,
		httpServer: httpServer,
		logger:     logger,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.logger.Info("starting server", zap.String("addr", s.httpServer.Addr))

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down server")

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown error: %w", err)
	}

	s.logger.Info("server shutdown complete")
	return nil
}

// loggingMiddleware logs HTTP requests
func loggingMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)

			logger.Info("http request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("remote_addr", r.RemoteAddr),
				zap.Int("status", wrapped.statusCode),
				zap.Duration("duration", duration),
			)
		})
	}
}

// recoveryMiddleware recovers from panics
func recoveryMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("panic recovered",
						zap.Any("error", err),
						zap.String("path", r.URL.Path),
						zap.Stack("stack"),
					)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// timeoutMiddleware adds request timeout
func timeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
