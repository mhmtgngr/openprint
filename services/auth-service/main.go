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

	"github.com/redis/go-redis/v9"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openprint/openprint/internal/auth/jwt"
	"github.com/openprint/openprint/internal/auth/oidc"
	"github.com/openprint/openprint/internal/auth/saml"
	"github.com/openprint/openprint/internal/auth/password"
	_ "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/internal/shared/middleware"
	"github.com/openprint/openprint/internal/shared/telemetry"
	"github.com/openprint/openprint/services/auth-service/handler"
	"github.com/openprint/openprint/services/auth-service/repository"
)

// Config holds service configuration.
type Config struct {
	ServerAddr       string
	DatabaseURL      string
	RedisURL         string
	JWTSecret        string
	JaegerEndpoint   string
	ServiceName      string
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

	// Connect to Redis
	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to parse Redis URL: %v", err)
	}
	redisClient := redis.NewClient(redisOpts)

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

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

	// Create handlers
	h := handler.New(handler.Config{
		UserRepo:       userRepo,
		SessionRepo:    sessionRepo,
		JWTManager:     jwtManager,
		PasswordHasher: passwordHasher,
		OIDCRegistry:   oidcRegistry,
		SAMLManager:    samlManager,
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
	// 1. Rate limiting for auth endpoints (10 requests per minute for sensitive endpoints)
	authRateLimiter := middleware.RateLimitMiddleware(10, 5*time.Minute)
	// 2. General rate limiting (60 requests per minute)
	generalRateLimiter := middleware.RateLimitMiddleware(60, 5*time.Minute)
	// 3. Security headers
	securityHeaders := middleware.SecurityHeadersMiddleware()
	// 4. CORS (allow specific origins in production)
	corsMiddleware := middleware.CORSMiddleware(
		[]string{"*"}, // Configure appropriately for production
		[]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		[]string{"Content-Type", "Authorization"},
	)

	// Apply stricter rate limiting to sensitive auth endpoints
	protectedMux := http.NewServeMux()
	protectedMux.Handle("/auth/register", authRateLimiter(http.HandlerFunc(h.Register)))
	protectedMux.Handle("/auth/login", authRateLimiter(http.HandlerFunc(h.Login)))
	protectedMux.Handle("/auth/refresh", authRateLimiter(http.HandlerFunc(h.RefreshToken)))
	protectedMux.Handle("/auth/logout", http.HandlerFunc(h.Logout))
	protectedMux.Handle("/auth/me", http.HandlerFunc(h.GetCurrentUser))
	protectedMux.Handle("/auth/oidc/", authRateLimiter(http.HandlerFunc(h.OIDCHandler)))
	protectedMux.Handle("/auth/saml/metadata", http.HandlerFunc(h.SAMLMetadataHandler))
	protectedMux.Handle("/auth/saml/acs", authRateLimiter(http.HandlerFunc(h.SAMLACSHandler)))
	protectedMux.Handle("/health", http.HandlerFunc(healthHandler))

	// Wrap the mux with middleware
	wrappedMux := corsMiddleware(securityHeaders(generalRateLimiter(protectedMux)))

	// Apply telemetry middleware
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
		log.Fatal("JWT_SECRET environment variable is required and must be set")
	}
	// Validate JWT secret meets minimum security requirements (32 characters for HS256)
	if len(jwtSecret) < 32 {
		log.Fatal("JWT_SECRET must be at least 32 characters long for secure HMAC-SHA256 signing")
	}

	return &Config{
		ServerAddr:     getEnv("SERVER_ADDR", ":8001"),
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

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"healthy","service":"%s"}`, "auth-service")
}
