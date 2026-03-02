// Package prometheus provides a dedicated HTTP server for exposing metrics.
package prometheus

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsServer manages the Prometheus metrics HTTP endpoint.
// It runs on a separate port from the main service HTTP server.
type MetricsServer struct {
	registry    *Registry
	server      *http.Server
	port        int
	addr        string
	certFile    string
	keyFile     string
	basicAuth   *BasicAuthConfig
}

// BasicAuthConfig holds basic authentication configuration for the metrics endpoint.
type BasicAuthConfig struct {
	Username string
	Password string
}

// ServerConfig holds configuration for the metrics server.
type ServerConfig struct {
	// Registry is the Prometheus registry to serve metrics from.
	Registry *Registry

	// Port is the port to listen on (default: 9090).
	Port int

	// Host is the host to bind to (default: 0.0.0.0).
	Host string

	// CertFile is the path to the TLS certificate file (optional).
	CertFile string

	// KeyFile is the path to the TLS key file (optional).
	KeyFile string

	// BasicAuth configures basic authentication for the metrics endpoint.
	BasicAuth *BasicAuthConfig
}

// DefaultServerConfig returns a default configuration for the metrics server.
func DefaultServerConfig(registry *Registry) ServerConfig {
	return ServerConfig{
		Registry: registry,
		Port:     9090,
		Host:     "0.0.0.0",
	}
}

// NewMetricsServer creates a new metrics HTTP server.
func NewMetricsServer(cfg ServerConfig) *MetricsServer {
	if cfg.Port == 0 {
		cfg.Port = 9090
	}
	if cfg.Host == "" {
		cfg.Host = "0.0.0.0"
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", metricsHandler(cfg.Registry, cfg.BasicAuth))
	mux.HandleFunc("/health", healthHandler(cfg.Registry))
	mux.HandleFunc("/", rootHandler(cfg.Registry))

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return &MetricsServer{
		registry:  cfg.Registry,
		server:    server,
		port:      cfg.Port,
		addr:      addr,
		certFile:  cfg.CertFile,
		keyFile:   cfg.KeyFile,
		basicAuth: cfg.BasicAuth,
	}
}

// Start starts the metrics server in a background goroutine.
// Returns an error if the server fails to start.
func (s *MetricsServer) Start() error {
	log.Printf("Starting metrics server on %s", s.addr)

	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.addr, err)
	}

	go func() {
		var err error
		if s.certFile != "" && s.keyFile != "" {
			log.Printf("Metrics server listening with TLS on %s", s.addr)
			err = s.server.ServeTLS(ln, s.certFile, s.keyFile)
		} else {
			log.Printf("Metrics server listening on %s", s.addr)
			err = s.server.Serve(ln)
		}
		if err != nil && err != http.ErrServerClosed {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	return nil
}

// Shutdown gracefully shuts down the metrics server.
func (s *MetricsServer) Shutdown(ctx context.Context) error {
	log.Println("Shutting down metrics server...")
	return s.server.Shutdown(ctx)
}

// Addr returns the address the server is listening on.
func (s *MetricsServer) Addr() string {
	return s.addr
}

// Port returns the port the server is listening on.
func (s *MetricsServer) Port() int {
	return s.port
}

// metricsHandler serves the Prometheus metrics endpoint.
func metricsHandler(registry *Registry, auth *BasicAuthConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only allow GET requests
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", 405)
			return
		}

		// Check basic auth if configured
		if auth != nil {
			username, password, ok := r.BasicAuth()
			if !ok || username != auth.Username || password != auth.Password {
				w.Header().Set("WWW-Authenticate", `Basic realm="Metrics"`)
				http.Error(w, "Unauthorized", 401)
				return
			}
		}

		// Set content type
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

		// Gather and write metrics
		handler := promhttp.HandlerFor(registry.Registry(), promhttp.HandlerOpts{})
		handler.ServeHTTP(w, r)
	}
}

// healthHandler serves the health check endpoint.
func healthHandler(registry *Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","service":"%s"}`, registry.ServiceName())
	}
}

// rootHandler serves the root endpoint with helpful information.
func rootHandler(registry *Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
    <title>%s Metrics</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        h1 { color: #333; }
        a { color: #0066cc; text-decoration: none; }
        a:hover { text-decoration: underline; }
        .endpoint { margin: 10px 0; padding: 10px; background: #f5f5f5; border-radius: 4px; }
    </style>
</head>
<body>
    <h1>%s Metrics</h1>
    <p>Service: %s</p>
    <p>Version: %s</p>

    <h2>Endpoints</h2>
    <div class="endpoint">
        <strong><a href="/metrics">/metrics</a></strong> - Prometheus metrics
    </div>
    <div class="endpoint">
        <strong><a href="/health">/health</a></strong> - Health check
    </div>
</body>
</html>`, registry.ServiceName(), registry.ServiceName(), registry.ServiceName(), registry.ServiceVersion())
	}
}

// StartMetricsServer is a convenience function to create and start a metrics server.
func StartMetricsServer(registry *Registry, port int) (*MetricsServer, error) {
	cfg := DefaultServerConfig(registry)
	if port > 0 {
		cfg.Port = port
	}

	server := NewMetricsServer(cfg)
	if err := server.Start(); err != nil {
		return nil, err
	}

	return server, nil
}

// MustStartMetricsServer creates and starts a metrics server, panicking on error.
func MustStartMetricsServer(registry *Registry, port int) *MetricsServer {
	server, err := StartMetricsServer(registry, port)
	if err != nil {
		panic(err)
	}
	return server
}

// GetDefaultMetricsPort returns the default metrics port for a service.
func GetDefaultMetricsPort(serviceName string) int {
	// Each service gets its own metrics port
	switch serviceName {
	case ServiceAuthService:
		return 9091
	case ServiceRegistryService:
		return 9092
	case ServiceJobService:
		return 9093
	case ServiceStorageService:
		return 9094
	case ServiceNotificationService:
		return 9095
	default:
		return 9090
	}
}

// MultiRegistryServer serves metrics from multiple registries.
// This is useful for running a centralized metrics server.
type MultiRegistryServer struct {
	registries map[string]*Registry
	server     *http.Server
	port       int
	addr       string
}

// NewMultiRegistryServer creates a new multi-registry metrics server.
func NewMultiRegistryServer(port int, host string) *MultiRegistryServer {
	if port == 0 {
		port = 9090
	}
	if host == "" {
		host = "0.0.0.0"
	}

	addr := fmt.Sprintf("%s:%d", host, port)

	return &MultiRegistryServer{
		registries: make(map[string]*Registry),
		port:       port,
		addr:       addr,
	}
}

// AddRegistry adds a registry to be served.
func (s *MultiRegistryServer) AddRegistry(name string, registry *Registry) {
	s.registries[name] = registry
}

// Start starts the multi-registry metrics server.
func (s *MultiRegistryServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", s.metricsHandler())
	mux.HandleFunc("/health", s.healthHandler())
	mux.HandleFunc("/", s.rootHandler())

	s.server = &http.Server{
		Addr:              s.addr,
		Handler:           mux,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("Starting multi-registry metrics server on %s", s.addr)

	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.addr, err)
	}

	go func() {
		if err := s.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("Multi-registry metrics server error: %v", err)
		}
	}()

	return nil
}

// Shutdown shuts down the multi-registry metrics server.
func (s *MultiRegistryServer) Shutdown(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

func (s *MultiRegistryServer) metricsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

		// Gather metrics from all registries
		for name, registry := range s.registries {
			fmt.Fprintf(w, "# Service: %s\n", name)
			handler := promhttp.HandlerFor(registry.Registry(), promhttp.HandlerOpts{})
			handler.ServeHTTP(w, r)
			fmt.Fprintln(w)
		}
	}
}

func (s *MultiRegistryServer) healthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","services":%d}`, len(s.registries))
	}
}

func (s *MultiRegistryServer) rootHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
    <title>OpenPrint Metrics</title>
</head>
<body>
    <h1>OpenPrint Metrics</h1>
    <p>Serving %d services</p>
    <h2>Endpoints</h2>
    <div><strong><a href="/metrics">/metrics</a></strong> - All metrics</div>
    <div><strong><a href="/health">/health</a></strong> - Health check</div>
</body>
</html>`, len(s.registries))
	}
}
