// Package main is the entry point for the OpenPrint Auth Service.
// This service handles user authentication, session management, and identity provider integration.
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openprint/openprint/internal/auth/jwt"
	"github.com/openprint/openprint/internal/auth/oidc"
	"github.com/openprint/openprint/internal/auth/password"
	"github.com/openprint/openprint/internal/auth/saml"
	_ "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/internal/shared/middleware"
	"github.com/openprint/openprint/internal/shared/telemetry"
	"github.com/openprint/openprint/internal/shared/telemetry/prometheus"
	"github.com/openprint/openprint/services/auth-service/handler"
	"github.com/openprint/openprint/services/auth-service/repository"
	"github.com/redis/go-redis/v9"
)

// Config holds service configuration.
type Config struct {
	ServerAddr     string
	MetricsPort    int
	DatabaseURL    string
	RedisURL       string
	JWTSecret      string
	JaegerEndpoint string
	ServiceName    string
}

func main() {
	cfg := loadConfig()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Prometheus metrics registry
	registry, err := prometheus.NewRegistry(prometheus.DefaultConfig(cfg.ServiceName))
	if err != nil {
		log.Fatalf("Failed to create Prometheus registry: %v", err)
	}
	prometheus.SetRegistry(registry)
	metrics := prometheus.NewMetrics(registry)

	// Start metrics server on dedicated port
	metricsPort := cfg.MetricsPort
	if metricsPort == 0 {
		metricsPort = prometheus.GetDefaultMetricsPort(cfg.ServiceName)
	}
	metricsServer, err := prometheus.StartMetricsServer(registry, metricsPort)
	if err != nil {
		log.Fatalf("Failed to start metrics server: %v", err)
	}
	defer func() {
		if err := metricsServer.Shutdown(ctx); err != nil {
			log.Printf("Metrics server shutdown error: %v", err)
		}
	}()

	// Wrap database with metrics collector
	// This will track connection pool stats and query metrics
	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	prometheus.WrapPgxPool(db, registry, prometheus.DBConfig{
		ServiceName: cfg.ServiceName,
		DBName:      "openprint",
		DBSystem:    prometheus.DBSystemPostgreSQL,
	})

	// Connect to Redis with metrics
	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to parse Redis URL: %v", err)
	}
	redisClient := redis.NewClient(redisOpts)

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Wrap Redis with metrics collector
	prometheus.WrapRedisClient(redisClient, registry, prometheus.RedisConfig{
		ServiceName: cfg.ServiceName,
		DBName:      "0",
	})

	// Initialize telemetry (tracing)
	shutdown, err := telemetry.InitTracer(cfg.ServiceName, "1.0.0", cfg.JaegerEndpoint)
	if err != nil {
		log.Printf("Warning: failed to initialize tracer: %v", err)
	}
	if shutdown != nil {
		defer shutdown(ctx)
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(redisClient)

	// Initialize auth components
	jwtCfg, err := jwt.DefaultConfig(cfg.JWTSecret)
	if err != nil {
		log.Fatalf("Failed to create JWT config: %v", err)
	}
	jwtManager, err := jwt.NewManager(jwtCfg)
	if err != nil {
		log.Fatalf("Failed to create JWT manager: %v", err)
	}
	passwordHasher := password.DefaultHasher()

	// Initialize OIDC providers
	oidcRegistry := oidc.NewRegistry()
	// Providers would be registered here based on configuration

	// Initialize SAML (if configured)
	var samlManager *saml.Manager
	// SAML would be initialized here based on configuration

	// Create handlers with metrics
	h := handler.New(handler.Config{
		UserRepo:       userRepo,
		SessionRepo:    sessionRepo,
		JWTManager:     jwtManager,
		PasswordHasher: passwordHasher,
		OIDCRegistry:   oidcRegistry,
		SAMLManager:    samlManager,
		Metrics:        metrics,
		ServiceName:    cfg.ServiceName,
	})

	// Setup HTTP server with middleware
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/auth/register", h.Register)
	mux.HandleFunc("/auth/login", h.Login)
	mux.HandleFunc("/auth/logout", h.Logout)
	mux.HandleFunc("/auth/refresh", h.RefreshToken)
	mux.HandleFunc("/auth/me", h.GetCurrentUser)
	mux.HandleFunc("/auth/oidc/", h.OIDCHandler)
	mux.HandleFunc("/auth/saml/metadata", h.SAMLMetadataHandler)
	mux.HandleFunc("/auth/saml/acs", h.SAMLACSHandler)

	// Apply security middleware chain
	// Rate limiting strategy:
	// - Strict rate limiting (5 per 5 minutes) for login/register to prevent brute force attacks
	// - Moderate rate limiting (10 per 5 minutes) for refresh token endpoint
	// - Permissive rate limiting (60 per 5 minutes) for general endpoints
	// Rate limiting is applied per IP address
	strictRateLimiter := middleware.RateLimitMiddleware(5, 5*time.Minute)   // For login/register
	authRateLimiter := middleware.RateLimitMiddleware(10, 5*time.Minute)    // For refresh/oidc/saml
	generalRateLimiter := middleware.RateLimitMiddleware(60, 5*time.Minute) // For other endpoints

	// Security headers middleware
	securityHeaders := middleware.SecurityHeadersMiddleware()
	// CORS (allow specific origins in production)
	corsMiddleware := middleware.CORSMiddleware(
		[]string{"*"}, // Configure appropriately for production
		[]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		[]string{"Content-Type", "Authorization"},
	)

	// Apply rate limiting to all authentication endpoints
	// Login and register have strict rate limiting to prevent credential stuffing and brute force
	protectedMux := http.NewServeMux()
	protectedMux.Handle("/auth/register", strictRateLimiter(http.HandlerFunc(h.Register)))
	protectedMux.Handle("/auth/login", strictRateLimiter(http.HandlerFunc(h.Login)))
	protectedMux.Handle("/auth/refresh", authRateLimiter(http.HandlerFunc(h.RefreshToken)))
	protectedMux.Handle("/auth/logout", authRateLimiter(http.HandlerFunc(h.Logout)))
	protectedMux.Handle("/auth/me", generalRateLimiter(http.HandlerFunc(h.GetCurrentUser)))
	protectedMux.Handle("/auth/oidc/", authRateLimiter(http.HandlerFunc(h.OIDCHandler)))
	protectedMux.Handle("/auth/saml/metadata", generalRateLimiter(http.HandlerFunc(h.SAMLMetadataHandler)))
	protectedMux.Handle("/auth/saml/acs", authRateLimiter(http.HandlerFunc(h.SAMLACSHandler)))
	protectedMux.Handle("/health", http.HandlerFunc(healthHandler))

	// Wrap the mux with middleware
	wrappedMux := corsMiddleware(securityHeaders(generalRateLimiter(protectedMux)))

	// Apply Prometheus metrics middleware
	wrappedMux = middleware.MetricsMiddleware(middleware.MetricsMiddlewareConfig{
		Registry:           registry,
		ServiceName:        cfg.ServiceName,
		SkipPaths:          []string{"/health"},
		ExcludeStaticFiles: true,
	})(wrappedMux)

	// Apply telemetry middleware (tracing)
	wrappedMux = telemetry.HTTPMiddleware(cfg.ServiceName)(wrappedMux)

	server := &http.Server{
		Addr:         cfg.ServerAddr,
		Handler:      wrappedMux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}

	// Start server in goroutine
	go func() {
		log.Printf("%s listening on %s (metrics on :%d)", cfg.ServiceName, cfg.ServerAddr, metricsPort)
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
		log.Fatal("JWT_SECRET environment variable is required and must be set")
	}
	// Validate JWT secret meets minimum security requirements (32 characters for HS256)
	if len(jwtSecret) < 32 {
		log.Fatal("JWT_SECRET must be at least 32 characters long for secure HMAC-SHA256 signing")
	}

	return &Config{
		ServerAddr:     getEnv("SERVER_ADDR", ":8001"),
		MetricsPort:    getEnvInt("METRICS_PORT", 0), // 0 = use default
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://openprint:openprint@localhost:5432/openprint"),
		RedisURL:       getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:      jwtSecret,
		JaegerEndpoint: getEnv("JAEGER_ENDPOINT", ""),
		ServiceName:    getEnv("SERVICE_NAME", "auth-service"),
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

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "HEAD" {
		http.Error(w, "method not allowed", 405)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	if r.Method == "GET" {
		fmt.Fprintf(w, `{"status":"healthy","service":"%s"}`, "auth-service")
	}
}
