package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// serviceHealth holds the result of a single service health check.
type serviceHealth struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	URL    string `json:"url,omitempty"`
	Error  string `json:"error,omitempty"`
}

// aggregatedHealthHandler probes all downstream services concurrently.
func aggregatedHealthHandler(cfg *Config) http.HandlerFunc {
	services := []struct {
		name string
		url  string
	}{
		{"auth-service", cfg.AuthServiceURL},
		{"registry-service", cfg.RegistryServiceURL},
		{"job-service", cfg.JobServiceURL},
		{"storage-service", cfg.StorageServiceURL},
		{"notification-service", cfg.NotificationServiceURL},
	}

	client := &http.Client{Timeout: 3 * time.Second}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		results := make([]serviceHealth, len(services))
		var wg sync.WaitGroup

		for i, svc := range services {
			wg.Add(1)
			go func(idx int, name, baseURL string) {
				defer wg.Done()
				healthURL := baseURL + "/health"
				resp, err := client.Get(healthURL)
				if err != nil {
					results[idx] = serviceHealth{Name: name, Status: "unhealthy", Error: err.Error()}
					return
				}
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					results[idx] = serviceHealth{Name: name, Status: "healthy"}
				} else {
					results[idx] = serviceHealth{Name: name, Status: "unhealthy", Error: fmt.Sprintf("status %d", resp.StatusCode)}
				}
			}(i, svc.name, svc.url)
		}

		wg.Wait()

		overallStatus := "healthy"
		for _, r := range results {
			if r.Status != "healthy" {
				overallStatus = "degraded"
				break
			}
		}

		response := map[string]interface{}{
			"status":    overallStatus,
			"service":   "gateway",
			"timestamp": time.Now().Format(time.RFC3339),
			"services":  results,
		}

		w.Header().Set("Content-Type", "application/json")
		if overallStatus != "healthy" {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		json.NewEncoder(w).Encode(response)
	}
}
