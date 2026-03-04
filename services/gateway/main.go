// Package main is the entry point for the OpenPrint API Gateway.
// This service acts as the API gateway routing to all microservices (ports 8001-8005).
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/openprint/openprint/internal/auth/jwt"
	"github.com/openprint/openprint/services/gateway/middleware"
)

const (
	// ServicePort is the port the gateway listens on.
	ServicePort = 8000

	// Backend service ports.
	AuthServicePort         = 8001
	RegistryServicePort     = 8002
	JobServicePort          = 8003
	StorageServicePort      = 8004
	NotificationServicePort = 8005
)

// Config holds gateway configuration.
type Config struct {
	ServerAddr             string
	JWTSecret              string
	RequestsPerMinute      int
	ServiceHost            string
	AuthServiceURL         string
	RegistryServiceURL     string
	JobServiceURL          string
	StorageServiceURL      string
	NotificationServiceURL string
}

// ServiceRoute defines a route to a backend service.
type ServiceRoute struct {
	Pattern     string
	TargetURL   string
	RequireAuth bool
}

func main() {
	cfg := loadConfig()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize JWT manager for auth middleware
	jwtCfg, err := jwt.DefaultConfig(cfg.JWTSecret)
	if err != nil {
		log.Fatalf("Failed to create JWT config: %v", err)
	}
	jwtManager, err := jwt.NewManager(jwtCfg)
	if err != nil {
		log.Fatalf("Failed to create JWT manager: %v", err)
	}

	// Create audit logger
	auditLogger := middleware.NewAuditLogger(log.New(os.Stdout, "[GATEWAY] ", log.LstdFlags))

	// Create HTTP handler with all services routed
	mux := http.NewServeMux()

	// Register service routes
	registerServiceRoutes(mux, cfg, jwtManager, auditLogger)

	// Health check endpoint (no auth required)
	mux.HandleFunc("/health", healthHandler)

	// Apply middleware chain
	// 1. Rate limiting (IP-based, 100 req/min default)
	rateLimiter := middleware.RateLimitMiddleware(&middleware.RateLimiterConfig{
		RequestsPerMinute: cfg.RequestsPerMinute,
		CleanupInterval:   5 * time.Minute,
	})

	// 2. Audit logging (all requests)
	audit := middleware.AuditMiddleware(auditLogger)

	// 3. Security headers
	security := securityHeadersMiddleware()

	// 4. CORS
	cors := corsMiddleware()

	// Chain middleware
	handler := middleware.Chain(
		rateLimiter,
		audit,
		security,
		cors,
	)(mux)

	// Create server
	server := &http.Server{
		Addr:         cfg.ServerAddr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}

	// Start server in goroutine
	go func() {
		log.Printf("API Gateway listening on %s", cfg.ServerAddr)
		log.Printf("Routing to services:")
		log.Printf("  - auth-service    -> %s", cfg.AuthServiceURL)
		log.Printf("  - registry-service -> %s", cfg.RegistryServiceURL)
		log.Printf("  - job-service      -> %s", cfg.JobServiceURL)
		log.Printf("  - storage-service  -> %s", cfg.StorageServiceURL)
		log.Printf("  - notification-service -> %s", cfg.NotificationServiceURL)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	log.Println("Shutting down gateway...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Gateway stopped")
}

func loadConfig() *Config {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}
	if len(jwtSecret) < 32 {
		log.Fatal("JWT_SECRET must be at least 32 characters long")
	}

	serviceHost := getEnv("SERVICE_HOST", "localhost")
	requestsPerMinute := getEnvInt("REQUESTS_PER_MINUTE", 100)

	return &Config{
		ServerAddr:             fmt.Sprintf(":%d", ServicePort),
		JWTSecret:              jwtSecret,
		RequestsPerMinute:      requestsPerMinute,
		ServiceHost:            serviceHost,
		AuthServiceURL:         getEnv("AUTH_SERVICE_URL", fmt.Sprintf("http://%s:%d", serviceHost, AuthServicePort)),
		RegistryServiceURL:     getEnv("REGISTRY_SERVICE_URL", fmt.Sprintf("http://%s:%d", serviceHost, RegistryServicePort)),
		JobServiceURL:          getEnv("JOB_SERVICE_URL", fmt.Sprintf("http://%s:%d", serviceHost, JobServicePort)),
		StorageServiceURL:      getEnv("STORAGE_SERVICE_URL", fmt.Sprintf("http://%s:%d", serviceHost, StorageServicePort)),
		NotificationServiceURL: getEnv("NOTIFICATION_SERVICE_URL", fmt.Sprintf("http://%s:%d", serviceHost, NotificationServicePort)),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intVal int
		if _, err := fmt.Sscanf(value, "%d", &intVal); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// registerServiceRoutes registers all service routes with reverse proxies.
func registerServiceRoutes(mux *http.ServeMux, cfg *Config, jwtManager *jwt.Manager, auditLogger *middleware.AuditLogger) {
	// Auth service routes - handle authentication separately
	authMux := http.NewServeMux()
	authMux.HandleFunc("/health", forwardTo(cfg.AuthServiceURL))
	authMux.HandleFunc("/", forwardTo(cfg.AuthServiceURL))
	mux.Handle("/auth/", http.StripPrefix("/auth", authMux))

	// Registry service routes - requires auth
	registryAuth := middleware.JWTAuthMiddleware(middleware.JWTAuthConfig{
		SecretKey:  cfg.JWTSecret,
		JWTManager: jwtManager,
		SkipPaths:  []string{"/health"},
	})
	registryProxy := registryAuth(forwardTo(cfg.RegistryServiceURL))
	mux.Handle("/registry/", http.StripPrefix("/registry", registryProxy))

	// Job service routes - requires auth
	jobAuth := middleware.JWTAuthMiddleware(middleware.JWTAuthConfig{
		SecretKey:  cfg.JWTSecret,
		JWTManager: jwtManager,
		SkipPaths:  []string{"/health"},
	})
	jobProxy := jobAuth(forwardTo(cfg.JobServiceURL))
	mux.Handle("/jobs/", http.StripPrefix("/jobs", jobProxy))

	// Storage service routes - requires auth
	storageAuth := middleware.JWTAuthMiddleware(middleware.JWTAuthConfig{
		SecretKey:  cfg.JWTSecret,
		JWTManager: jwtManager,
		SkipPaths:  []string{"/health"},
	})
	storageProxy := storageAuth(forwardTo(cfg.StorageServiceURL))
	mux.Handle("/storage/", http.StripPrefix("/storage", storageProxy))

	// Notification service routes - requires auth for WebSocket upgrades
	notificationAuth := middleware.JWTAuthMiddleware(middleware.JWTAuthConfig{
		SecretKey:  cfg.JWTSecret,
		JWTManager: jwtManager,
		SkipPaths:  []string{"/health"},
	})
	notificationProxy := notificationAuth(forwardTo(cfg.NotificationServiceURL))
	mux.Handle("/notifications/", http.StripPrefix("/notifications", notificationProxy))

	// API v1 routes - map to appropriate services
	// Auth endpoints (public)
	mux.HandleFunc("/api/v1/auth/login", forwardTo(cfg.AuthServiceURL))
	mux.HandleFunc("/api/v1/auth/register", forwardTo(cfg.AuthServiceURL))
	mux.HandleFunc("/api/v1/auth/logout", forwardTo(cfg.AuthServiceURL))
	mux.HandleFunc("/api/v1/auth/refresh", forwardTo(cfg.AuthServiceURL))
	mux.HandleFunc("/api/v1/auth/me", forwardTo(cfg.AuthServiceURL))

	// Agent endpoints (public for registration, auth for others)
	mux.HandleFunc("/api/v1/agents/register", forwardTo(cfg.AuthServiceURL))

	// Auth-protected agent endpoints
	protectedAgentHandler := middleware.JWTAuthMiddleware(middleware.JWTAuthConfig{
		SecretKey:  cfg.JWTSecret,
		JWTManager: jwtManager,
	})(forwardTo(cfg.RegistryServiceURL))
	mux.Handle("/api/v1/agents/", protectedAgentHandler)

	// Admin endpoints - require admin role
	adminAuth := middleware.Chain(
		middleware.JWTAuthMiddleware(middleware.JWTAuthConfig{
			SecretKey:  cfg.JWTSecret,
			JWTManager: jwtManager,
		}),
		middleware.RequireRole("admin", "org_admin"),
	)
	adminHandler := adminAuth(forwardTo(cfg.RegistryServiceURL))
	mux.Handle("/api/v1/admin/", http.StripPrefix("/api/v1/admin", adminHandler))
}

// forwardTo creates a reverse proxy handler for the given target URL.
func forwardTo(targetURL string) http.HandlerFunc {
	target, err := url.Parse(targetURL)
	if err != nil {
		log.Fatalf("Failed to parse target URL %s: %v", targetURL, err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// Customize the director to preserve original host and headers
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		// Preserve the original Host header
		req.Host = req.URL.Host
		// Add X-Forwarded headers
		req.Header.Set("X-Forwarded-Host", req.Host)
		req.Header.Set("X-Forwarded-Proto", scheme(req))
		// Add forwarded for header with client IP
		if req.RemoteAddr != "" {
			ip, _, err := net.SplitHostPort(req.RemoteAddr)
			if err != nil {
				ip = req.RemoteAddr
			}
			req.Header.Set("X-Forwarded-For", ip)
		}
	}

	// Customize error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Proxy error: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, `{"code":"SERVICE_UNAVAILABLE","message":"Backend service unavailable"}`)
	}

	// Customize transport to handle connection pooling
	proxy.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	}
}

// scheme returns the scheme of the request (http or https).
func scheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if scheme := r.Header.Get("X-Forwarded-Proto"); scheme != "" {
		return scheme
	}
	return "http"
}

// healthHandler returns the health status of the gateway.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"healthy","service":"gateway","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
}

// securityHeadersMiddleware adds security headers to responses.
func securityHeadersMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Prevent MIME type sniffing
			w.Header().Set("X-Content-Type-Options", "nosniff")
			// Prevent clickjacking
			w.Header().Set("X-Frame-Options", "DENY")
			// Enable XSS filter (legacy browsers)
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			// HSTS for HTTPS enforcement
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			// Control referrer information
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			next.ServeHTTP(w, r)
		})
	}
}

// corsMiddleware handles CORS headers.
func corsMiddleware() func(http.Handler) http.Handler {
	allowedOrigins := strings.Split(getEnv("CORS_ALLOWED_ORIGINS", "*"), ",")
	allowedMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}
	allowedHeaders := []string{"Content-Type", "Authorization", "X-Request-ID"}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			for _, allowedOrigin := range allowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
					break
				}
			}

			w.Header().Set("Access-Control-Allow-Methods", strings.Join(allowedMethods, ", "))
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(allowedHeaders, ", "))
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400")

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// NewHandler creates a new gateway handler with the given config.
// This is useful for testing and embedding.
func NewHandler(cfg *Config, jwtManager *jwt.Manager, auditLogger *middleware.AuditLogger) http.Handler {
	mux := http.NewServeMux()
	registerServiceRoutes(mux, cfg, jwtManager, auditLogger)
	mux.HandleFunc("/health", healthHandler)
	return mux
}
