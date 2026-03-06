package main

import (
	"net/http"

	"github.com/openprint/openprint/internal/auth/jwt"
	"github.com/openprint/openprint/services/gateway/middleware"
)

// AuthLevel defines the authentication/authorization requirement for a route.
type AuthLevel int

const (
	// Public routes require no authentication.
	Public AuthLevel = iota
	// Authenticated routes require a valid JWT token.
	Authenticated
	// AdminOnly routes require admin, owner, org_admin, or platform_admin role.
	AdminOnly
	// PlatformAdminOnly routes require the platform_admin role.
	PlatformAdminOnly
)

// RouteDefinition describes a single API route declaratively.
type RouteDefinition struct {
	Pattern   string
	Service   string // logical service name, resolved to URL at registration time
	AuthLevel AuthLevel
}

// serviceURLs resolves logical service names to URLs from config.
func serviceURLs(cfg *Config) map[string]string {
	return map[string]string{
		"auth":         cfg.AuthServiceURL,
		"registry":     cfg.RegistryServiceURL,
		"job":          cfg.JobServiceURL,
		"storage":      cfg.StorageServiceURL,
		"notification": cfg.NotificationServiceURL,
	}
}

// apiV1Routes returns the declarative route definitions for /api/v1/*.
// Adding a new route is as simple as appending to this slice.
func apiV1Routes() []RouteDefinition {
	return []RouteDefinition{
		// --- Public auth endpoints ---
		{"/api/v1/auth/login", "auth", Public},
		{"/api/v1/auth/register", "auth", Public},
		{"/api/v1/auth/logout", "auth", Public},
		{"/api/v1/auth/refresh", "auth", Public},
		{"/api/v1/auth/me", "auth", Public},
		{"/api/v1/auth/sso/", "auth", Public},
		{"/api/v1/agents/register", "auth", Public},

		// --- Authenticated user endpoints ---
		{"/api/v1/agents/", "registry", Authenticated},
		{"/api/v1/printers/", "registry", Authenticated},
		{"/api/v1/printers", "registry", Authenticated},
		{"/api/v1/jobs/", "job", Authenticated},
		{"/api/v1/jobs", "job", Authenticated},
		{"/api/v1/documents/", "storage", Authenticated},
		{"/api/v1/documents", "storage", Authenticated},
		{"/api/v1/follow-me/", "job", Authenticated},
		{"/api/v1/releases/", "job", Authenticated},
		{"/api/v1/notifications/", "notification", Authenticated},
		{"/api/v1/users/", "auth", Authenticated},
		{"/api/v1/quotas/me", "job", Authenticated},

		// --- Admin-only endpoints ---
		{"/api/v1/analytics/", "job", AdminOnly},
		{"/api/v1/organizations/", "auth", AdminOnly},
		{"/api/v1/organizations", "auth", AdminOnly},
		{"/api/v1/quotas/organization", "job", AdminOnly},
		{"/api/v1/quotas/users/", "job", AdminOnly},
		{"/api/v1/quotas/periods", "job", AdminOnly},
		{"/api/v1/quotas/", "job", AdminOnly},
		{"/api/v1/quotas", "job", AdminOnly},
		{"/api/v1/policies/", "job", AdminOnly},
		{"/api/v1/policies", "job", AdminOnly},
		{"/api/v1/audit-logs", "auth", AdminOnly},
		{"/api/v1/email-to-print/", "job", AdminOnly},
		{"/api/v1/guest/", "auth", AdminOnly},
		{"/api/v1/webhooks/", "notification", AdminOnly},
		{"/api/v1/webhooks", "notification", AdminOnly},
		{"/api/v1/supplies/", "registry", AdminOnly},
		{"/api/v1/drivers/", "registry", AdminOnly},
		{"/api/v1/drivers", "registry", AdminOnly},
		{"/api/v1/groups/", "auth", AdminOnly},
		{"/api/v1/groups", "auth", AdminOnly},
	}
}

// registerServiceRoutes registers all service routes with reverse proxies.
func registerServiceRoutes(mux *http.ServeMux, cfg *Config, jwtManager *jwt.Manager, auditLogger *middleware.AuditLogger) {
	urls := serviceURLs(cfg)

	jwtAuthCfg := middleware.JWTAuthConfig{
		SecretKey:  cfg.JWTSecret,
		JWTManager: jwtManager,
	}

	adminRoles := []string{"admin", "owner", "org_admin", "platform_admin"}

	// Build middleware factories for each auth level
	wrappers := map[AuthLevel]func(http.Handler) http.Handler{
		Public: func(h http.Handler) http.Handler { return h },
		Authenticated: func(h http.Handler) http.Handler {
			return middleware.JWTAuthMiddleware(jwtAuthCfg)(h)
		},
		AdminOnly: func(h http.Handler) http.Handler {
			return middleware.Chain(
				middleware.JWTAuthMiddleware(jwtAuthCfg),
				middleware.RequireRole(adminRoles...),
			)(h)
		},
		PlatformAdminOnly: func(h http.Handler) http.Handler {
			return middleware.Chain(
				middleware.JWTAuthMiddleware(jwtAuthCfg),
				middleware.RequireRole("platform_admin"),
			)(h)
		},
	}

	// Register declarative API v1 routes
	for _, route := range apiV1Routes() {
		serviceURL, ok := urls[route.Service]
		if !ok {
			panic("unknown service: " + route.Service)
		}
		wrap := wrappers[route.AuthLevel]
		handler := wrap(forwardTo(serviceURL))

		if route.AuthLevel == Public {
			// Public routes use HandleFunc (unwrap to HandlerFunc)
			mux.HandleFunc(route.Pattern, forwardTo(serviceURL))
		} else {
			mux.Handle(route.Pattern, handler)
		}
	}

	// Legacy prefix-based routes for backward compatibility
	registerLegacyRoutes(mux, cfg, jwtManager)

	// Platform Admin wildcard
	platformAdminHandler := wrappers[PlatformAdminOnly](forwardTo(cfg.RegistryServiceURL))
	mux.Handle("/api/v1/admin/", http.StripPrefix("/api/v1/admin", platformAdminHandler))
}

// registerLegacyRoutes sets up the /auth/, /registry/, /jobs/, /storage/, /notifications/ prefix routes.
func registerLegacyRoutes(mux *http.ServeMux, cfg *Config, jwtManager *jwt.Manager) {
	jwtAuthCfgWithSkip := middleware.JWTAuthConfig{
		SecretKey:  cfg.JWTSecret,
		JWTManager: jwtManager,
		SkipPaths:  []string{"/health"},
	}

	type legacyRoute struct {
		prefix  string
		target  string
		public  bool
	}

	legacy := []legacyRoute{
		{"/auth/", cfg.AuthServiceURL, true},
		{"/registry/", cfg.RegistryServiceURL, false},
		{"/jobs/", cfg.JobServiceURL, false},
		{"/storage/", cfg.StorageServiceURL, false},
		{"/notifications/", cfg.NotificationServiceURL, false},
	}

	for _, lr := range legacy {
		handler := http.Handler(forwardTo(lr.target))
		if !lr.public {
			handler = middleware.JWTAuthMiddleware(jwtAuthCfgWithSkip)(handler)
		}
		mux.Handle(lr.prefix, http.StripPrefix(lr.prefix[:len(lr.prefix)-1], handler))
	}
}
