// Package routes provides route registration for rate limit APIs.
package routes

import (
	"net/http"

	"github.com/openprint/openprint/services/auth-service/handlers"
)

// RegisterRateLimitRoutes registers rate limit management routes.
func RegisterRateLimitRoutes(mux *http.ServeMux, rlHandler *handlers.RateLimitHandler) {
	// Policy management routes
	mux.HandleFunc("/api/v1/ratelimit/policies", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			rlHandler.CreatePolicy(w, r)
		case http.MethodGet:
			rlHandler.ListPolicies(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/ratelimit/policies/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			rlHandler.GetPolicy(w, r)
		case http.MethodPut, http.MethodPatch:
			rlHandler.UpdatePolicy(w, r)
		case http.MethodDelete:
			rlHandler.DeletePolicy(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Rate limit check routes
	mux.HandleFunc("/api/v1/ratelimit/check", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			rlHandler.CheckRateLimit(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/ratelimit/reset", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			rlHandler.ResetRateLimit(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/ratelimit/usage", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			rlHandler.GetUsage(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Violation log routes
	violationsHandler := handlers.NewViolationsHandler(rlHandler)
	mux.HandleFunc("/api/v1/ratelimit/violations", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			violationsHandler.ListViolations(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/ratelimit/violations/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			violationsHandler.GetViolation(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/ratelimit/violations/stats", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			violationsHandler.GetViolationStats(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/ratelimit/violations/cleanup", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			violationsHandler.ClearOldViolations(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Trusted client routes
	trustedClientsHandler := handlers.NewTrustedClientsHandler(rlHandler)
	mux.HandleFunc("/api/v1/ratelimit/trusted-clients", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			trustedClientsHandler.CreateTrustedClient(w, r)
		case http.MethodGet:
			trustedClientsHandler.ListTrustedClients(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/ratelimit/trusted-clients/", func(w http.ResponseWriter, r *http.Request) {
		// Check if it's the regenerate endpoint
		if len(r.URL.Path) > 30 && r.URL.Path[len(r.URL.Path)-10:] == "/regenerate" {
			if r.Method == http.MethodPost {
				trustedClientsHandler.RegenerateAPIKey(w, r)
				return
			}
		}

		switch r.Method {
		case http.MethodGet:
			trustedClientsHandler.GetTrustedClient(w, r)
		case http.MethodPut, http.MethodPatch:
			trustedClientsHandler.UpdateTrustedClient(w, r)
		case http.MethodDelete:
			trustedClientsHandler.DeleteTrustedClient(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Circuit breaker routes
	cbHandler := handlers.NewCircuitBreakerHandler(rlHandler)
	mux.HandleFunc("/api/v1/ratelimit/circuit-breakers", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			cbHandler.ListCircuitStates(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/ratelimit/circuit-breakers/state", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			cbHandler.GetCircuitState(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/ratelimit/circuit-breakers/reset", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			cbHandler.ResetCircuit(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/ratelimit/circuit-breakers/open", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			cbHandler.ForceOpenCircuit(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/ratelimit/circuit-breakers/close", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			cbHandler.ForceCloseCircuit(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/ratelimit/circuit-breakers/stats", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			cbHandler.GetCircuitStats(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}
