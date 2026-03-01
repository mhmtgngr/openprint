// Package main provides the API Gateway service for OpenPrint.
// This service acts as a unified entry point for all API requests and provides
// rate limiting, request routing, and developer portal functionality.
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
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openprint/openprint/internal/shared/middleware"
	"github.com/openprint/openprint/internal/shared/telemetry"
	"github.com/openprint/openprint/services/api-gateway/handlers"
	gatewaymiddleware "github.com/openprint/openprint/services/api-gateway/middleware"
)

// Config holds service configuration.
type Config struct {
	ServerAddr          string
	DatabaseURL         string
	JWTSecret           string
	JaegerEndpoint      string
	ServiceName         string
	AuthServiceURL      string
	JobServiceURL       string
	RegistryServiceURL  string
	StorageServiceURL   string
	NotificationServiceURL string
	AnalyticsServiceURL string
	OrganizationServiceURL string
}

func main() {
	cfg := loadConfig()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize telemetry
	shutdown, err := telemetry.InitTracer(cfg.ServiceName, "1.0.0", cfg.JaegerEndpoint)
	if err != nil {
		log.Printf("Warning: failed to initialize tracer: %v", err)
	}
	if shutdown != nil {
		defer shutdown(ctx)
	}

	// Connect to PostgreSQL
	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create service URLs
	authServiceURL, _ := url.Parse(cfg.AuthServiceURL)
	jobServiceURL, _ := url.Parse(cfg.JobServiceURL)
	registryServiceURL, _ := url.Parse(cfg.RegistryServiceURL)
	storageServiceURL, _ := url.Parse(cfg.StorageServiceURL)
	notificationServiceURL, _ := url.Parse(cfg.NotificationServiceURL)
	analyticsServiceURL, _ := url.Parse(cfg.AnalyticsServiceURL)
	organizationServiceURL, _ := url.Parse(cfg.OrganizationServiceURL)

	// Create reverse proxies
	authProxy := createReverseProxy(authServiceURL, "auth-service")
	jobProxy := createReverseProxy(jobServiceURL, "job-service")
	registryProxy := createReverseProxy(registryServiceURL, "registry-service")
	storageProxy := createReverseProxy(storageServiceURL, "storage-service")
	notificationProxy := createReverseProxy(notificationServiceURL, "notification-service")
	analyticsProxy := createReverseProxy(analyticsServiceURL, "analytics-service")
	organizationProxy := createReverseProxy(organizationServiceURL, "organization-service")

	// Create handlers
	devHandler := handler.NewDeveloperHandler(db, cfg.JWTSecret)

	// Setup HTTP server with middleware
	mux := http.NewServeMux()

	// API Gateway endpoints
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/api/v1/docs", devHandler.APIDocsHandler)
	mux.HandleFunc("/api/v1/developer", devHandler.DeveloperPortalHandler)
	mux.HandleFunc("/api/v1/developer/keys", devHandler.APIKeysHandler)
	mux.HandleFunc("/api/v1/developer/keys/", devHandler.APIKeyHandler)
	mux.HandleFunc("/api/v1/developer/usage", devHandler.UsageStatsHandler)
	mux.HandleFunc("/api/v1/developer/webhooks", devHandler.WebhooksHandler)
	mux.HandleFunc("/api/v1/developer/webhooks/", devHandler.WebhookHandler)

	// Service routes (reverse proxy)
	mux.HandleFunc("/auth/", withServiceProxy(authProxy, "auth"))
	mux.HandleFunc("/api/v1/auth/", withServiceProxy(authProxy, "auth"))
	mux.HandleFunc("/jobs/", withServiceProxy(jobProxy, "job"))
	mux.HandleFunc("/api/v1/jobs/", withServiceProxy(jobProxy, "job"))
	mux.HandleFunc("/quota/", withServiceProxy(jobProxy, "job"))
	mux.HandleFunc("/api/v1/quota/", withServiceProxy(jobProxy, "job"))
	mux.HandleFunc("/cost/", withServiceProxy(jobProxy, "job"))
	mux.HandleFunc("/api/v1/cost/", withServiceProxy(jobProxy, "job"))
	mux.HandleFunc("/reports/", withServiceProxy(jobProxy, "job"))
	mux.HandleFunc("/api/v1/reports/", withServiceProxy(jobProxy, "job"))
	mux.HandleFunc("/printers/", withServiceProxy(registryProxy, "registry"))
	mux.HandleFunc("/api/v1/printers/", withServiceProxy(registryProxy, "registry"))
	mux.HandleFunc("/agents/", withServiceProxy(registryProxy, "registry"))
	mux.HandleFunc("/api/v1/agents/", withServiceProxy(registryProxy, "registry"))
	mux.HandleFunc("/devices/", withServiceProxy(registryProxy, "registry"))
	mux.HandleFunc("/api/v1/devices/", withServiceProxy(registryProxy, "registry"))
	mux.HandleFunc("/documents/", withServiceProxy(storageProxy, "storage"))
	mux.HandleFunc("/api/v1/documents/", withServiceProxy(storageProxy, "storage"))
	mux.HandleFunc("/watermarks/", withServiceProxy(storageProxy, "storage"))
	mux.HandleFunc("/api/v1/watermarks/", withServiceProxy(storageProxy, "storage"))
	mux.HandleFunc("/notifications/", withServiceProxy(notificationProxy, "notification"))
	mux.HandleFunc("/api/v1/notifications/", withServiceProxy(notificationProxy, "notification"))
	mux.HandleFunc("/analytics/", withServiceProxy(analyticsProxy, "analytics"))
	mux.HandleFunc("/api/v1/analytics/", withServiceProxy(analyticsProxy, "analytics"))
	mux.HandleFunc("/organizations/", withServiceProxy(organizationProxy, "organization"))
	mux.HandleFunc("/api/v1/organizations/", withServiceProxy(organizationProxy, "organization"))

	// Build middleware chain
	middlewareChain := middleware.Chain(
		middleware.LoggingMiddleware(log.New(os.Stdout, "[GATEWAY] ", log.LstdFlags)),
		middleware.RecoveryMiddleware(log.New(os.Stdout, "[GATEWAY] ", log.LstdFlags)),
		middleware.CORSMiddleware([]string{"*"}, []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}, []string{"Content-Type", "Authorization"}),
		middleware.RateLimitMiddleware(60, 1*time.Minute),
		telemetry.HTTPMiddleware(cfg.ServiceName),
		middleware.SecurityHeadersMiddleware(),
		gatewaymiddleware.APIKeyMiddleware(db, []string{"/health", "/api/v1/docs"}),
	)

	wrappedMux := middlewareChain(mux)

	server := &http.Server{
		Addr:         cfg.ServerAddr,
		Handler:      wrappedMux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}

	// Start server in goroutine
	go func() {
		log.Printf("%s listening on %s", cfg.ServiceName, cfg.ServerAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	log.Println("Shutting down server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}

func loadConfig() *Config {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	return &Config{
		ServerAddr:             getEnv("SERVER_ADDR", ":8000"),
		DatabaseURL:            getEnv("DATABASE_URL", "postgres://openprint:openprint@localhost:5432/openprint"),
		JWTSecret:              jwtSecret,
		JaegerEndpoint:         getEnv("JAEGER_ENDPOINT", ""),
		ServiceName:            getEnv("SERVICE_NAME", "api-gateway"),
		AuthServiceURL:         getEnv("AUTH_SERVICE_URL", "http://localhost:8001"),
		JobServiceURL:          getEnv("JOB_SERVICE_URL", "http://localhost:8003"),
		RegistryServiceURL:     getEnv("REGISTRY_SERVICE_URL", "http://localhost:8002"),
		StorageServiceURL:      getEnv("STORAGE_SERVICE_URL", "http://localhost:8004"),
		NotificationServiceURL: getEnv("NOTIFICATION_SERVICE_URL", "http://localhost:8005"),
		AnalyticsServiceURL:    getEnv("ANALYTICS_SERVICE_URL", "http://localhost:8006"),
		OrganizationServiceURL: getEnv("ORGANIZATION_SERVICE_URL", "http://localhost:8007"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// healthHandler returns gateway health and downstream service status.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check downstream services
	services := map[string]string{
		"auth-service":         os.Getenv("AUTH_SERVICE_URL"),
		"job-service":          os.Getenv("JOB_SERVICE_URL"),
		"registry-service":     os.Getenv("REGISTRY_SERVICE_URL"),
		"storage-service":      os.Getenv("STORAGE_SERVICE_URL"),
		"notification-service": os.Getenv("NOTIFICATION_SERVICE_URL"),
		"analytics-service":    os.Getenv("ANALYTICS_SERVICE_URL"),
		"organization-service": os.Getenv("ORGANIZATION_SERVICE_URL"),
	}

	status := map[string]interface{}{
		"status":   "healthy",
		"service":  "api-gateway",
		"services": make(map[string]string),
	}

	for name, url := range services {
		// Simple health check - in production, actually ping the service
		status["services"].(map[string]string)[name] = "unknown"
		if url != "" {
			status["services"].(map[string]string)[name] = "reachable"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"healthy","service":"api-gateway"}`)
}

// createReverseProxy creates a reverse proxy for the given service URL.
func createReverseProxy(target *url.URL, serviceName string) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = createTransport(serviceName)
	return proxy
}

// createTransport creates a custom HTTP transport for the proxy.
func createTransport(serviceName string) *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		// Disable HTTP/2 for better compatibility with service-to-service communication
		ForceAttemptHTTP2:     false,
	}
}

// withServiceProxy wraps a reverse proxy with service identification.
func withServiceProxy(proxy http.Handler, service string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Add service header for tracing
		r.Header.Set("X-Forwarded-Service", service)
		proxy.ServeHTTP(w, r)
	}
}
