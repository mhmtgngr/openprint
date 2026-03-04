// Package handler provides HTTP handlers for the developer portal.
package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
)

// DeveloperHandler handles developer portal HTTP endpoints.
type DeveloperHandler struct {
	db        *pgxpool.Pool
	jwtSecret string
}

// NewDeveloperHandler creates a new developer handler instance.
func NewDeveloperHandler(db *pgxpool.Pool, jwtSecret string) *DeveloperHandler {
	return &DeveloperHandler{
		db:        db,
		jwtSecret: jwtSecret,
	}
}

// APIKey represents an API key for developer access.
type APIKey struct {
	ID             string
	OrganizationID string
	Name           string
	Key            string // SHA256 hashed
	KeyPrefix      string // First 8 characters for identification
	Scopes         []string
	IsActive       bool
	ExpiresAt      *time.Time
	RateLimit      int // requests per minute
	CreatedBy      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	LastUsedAt     *time.Time
}

// Webhook represents a webhook configuration.
type Webhook struct {
	ID             string
	OrganizationID string
	Name           string
	URL            string
	Secret         string
	Events         []string
	IsActive       bool
	Headers        map[string]string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// UsageStats represents API usage statistics.
type UsageStats struct {
	APIKeyID     string
	Date         string
	RequestCount int
	ErrorCount   int
	AvgLatency   int // milliseconds
}

// CreateAPIKeyRequest represents a request to create an API key.
type CreateAPIKeyRequest struct {
	OrganizationID string   `json:"organization_id"`
	Name           string   `json:"name"`
	Scopes         []string `json:"scopes"`
	ExpiresInDays  int      `json:"expires_in_days,omitempty"`
	RateLimit      int      `json:"rate_limit,omitempty"`
}

// UpdateAPIKeyRequest represents a request to update an API key.
type UpdateAPIKeyRequest struct {
	Name      string   `json:"name,omitempty"`
	Scopes    []string `json:"scopes,omitempty"`
	IsActive  *bool    `json:"is_active,omitempty"`
	RateLimit int      `json:"rate_limit,omitempty"`
}

// CreateWebhookRequest represents a request to create a webhook.
type CreateWebhookRequest struct {
	OrganizationID string            `json:"organization_id"`
	Name           string            `json:"name"`
	URL            string            `json:"url"`
	Events         []string          `json:"events"`
	Secret         string            `json:"secret,omitempty"`
	Headers        map[string]string `json:"headers,omitempty"`
}

// APIDocsHandler handles serving API documentation.
func (h *DeveloperHandler) APIDocsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Serve OpenAPI documentation
	docs := map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":       "OpenPrint API",
			"version":     "v1",
			"description": "Cloud-based print management platform API",
		},
		"servers": []map[string]interface{}{
			{"url": "/api/v1", "description": "API Gateway"},
		},
		"paths": map[string]interface{}{
			"/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "User login",
					"tags":    []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"email":    map[string]interface{}{"type": "string"},
										"password": map[string]interface{}{"type": "string"},
									},
									"required": []string{"email", "password"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successful login",
						},
					},
				},
			},
			"/jobs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "List print jobs",
					"tags":    []string{"Jobs"},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "List of jobs",
						},
					},
				},
				"post": map[string]interface{}{
					"summary": "Create a new print job",
					"tags":    []string{"Jobs"},
					"responses": map[string]interface{}{
						"201": map[string]interface{}{
							"description": "Job created",
						},
					},
				},
			},
			"/quota": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Check print quota",
					"tags":    []string{"Quota"},
				},
			},
			"/printers": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "List printers",
					"tags":    []string{"Printers"},
				},
			},
			"/reports": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Get usage reports",
					"tags":    []string{"Reports"},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "JWT",
				},
				"apiKeyAuth": map[string]interface{}{
					"type": "apiKey",
					"in":   "header",
					"name": "X-API-Key",
				},
			},
		},
		"security": []map[string]interface{}{
			{"bearerAuth": []string{}},
			{"apiKeyAuth": []string{}},
		},
	}

	respondJSON(w, http.StatusOK, docs)
}

// DeveloperPortalHandler handles the main developer portal endpoint.
func (h *DeveloperHandler) DeveloperPortalHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		h.getDeveloperInfo(w, r, ctx)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *DeveloperHandler) getDeveloperInfo(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	// Get organization ID from context (set by auth middleware)
	orgID := ctx.Value("org_id")
	if orgID == nil {
		respondError(w, apperrors.New("organization not found", http.StatusNotFound))
		return
	}

	// Get API keys for this organization
	keys, err := h.listAPIKeys(ctx, fmt.Sprintf("%v", orgID))
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get api keys", http.StatusInternalServerError))
		return
	}

	// Get webhooks for this organization
	webhooks, err := h.listWebhooks(ctx, fmt.Sprintf("%v", orgID))
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get webhooks", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"organization_id": orgID,
		"api_keys":        keys,
		"webhooks":        webhooks,
		"docs_url":        "/api/v1/docs",
	})
}

// APIKeysHandler handles API key list and creation.
func (h *DeveloperHandler) APIKeysHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		h.listAPIKeysEndpoint(w, r, ctx)
	case http.MethodPost:
		h.createAPIKey(w, r, ctx)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *DeveloperHandler) listAPIKeysEndpoint(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	orgID := r.URL.Query().Get("organization_id")
	if orgID == "" {
		// Try to get from context
		if oid := ctx.Value("org_id"); oid != nil {
			orgID = fmt.Sprintf("%v", oid)
		} else {
			respondError(w, apperrors.New("organization_id is required", http.StatusBadRequest))
			return
		}
	}

	keys, err := h.listAPIKeys(ctx, orgID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to list api keys", http.StatusInternalServerError))
		return
	}

	response := make([]map[string]interface{}, len(keys))
	for i, k := range keys {
		response[i] = map[string]interface{}{
			"id":         k.ID,
			"name":       k.Name,
			"key_prefix": k.KeyPrefix,
			"scopes":     k.Scopes,
			"is_active":  k.IsActive,
			"rate_limit": k.RateLimit,
			"created_at": k.CreatedAt.Format(time.RFC3339),
			"last_used_at": func() string {
				if k.LastUsedAt != nil {
					return k.LastUsedAt.Format(time.RFC3339)
				}
				return ""
			}(),
		}
		if k.ExpiresAt != nil {
			response[i]["expires_at"] = k.ExpiresAt.Format(time.RFC3339)
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"api_keys": response,
		"count":    len(response),
	})
}

func (h *DeveloperHandler) createAPIKey(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	var req CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate request
	if req.Name == "" {
		respondError(w, apperrors.New("name is required", http.StatusBadRequest))
		return
	}
	if req.OrganizationID == "" {
		// Try to get from context
		if oid := ctx.Value("org_id"); oid != nil {
			req.OrganizationID = fmt.Sprintf("%v", oid)
		} else {
			respondError(w, apperrors.New("organization_id is required", http.StatusBadRequest))
			return
		}
	}
	if len(req.Scopes) == 0 {
		req.Scopes = []string{"read", "write"}
	}
	if req.RateLimit == 0 {
		req.RateLimit = 60 // 60 requests per minute
	}

	// Generate API key
	rawKey := generateAPIKey()
	keyHash := sha256.Sum256([]byte(rawKey))
	keyPrefix := rawKey[:8]

	var expiresAt *time.Time
	if req.ExpiresInDays > 0 {
		t := time.Now().AddDate(0, 0, req.ExpiresInDays)
		expiresAt = &t
	}

	// Get created by from context
	createdBy := "system"
	if uid := ctx.Value("user_id"); uid != nil {
		createdBy = fmt.Sprintf("%v", uid)
	}

	key := &APIKey{
		ID:             uuid.New().String(),
		OrganizationID: req.OrganizationID,
		Name:           req.Name,
		Key:            hex.EncodeToString(keyHash[:]),
		KeyPrefix:      keyPrefix,
		Scopes:         req.Scopes,
		IsActive:       true,
		ExpiresAt:      expiresAt,
		RateLimit:      req.RateLimit,
		CreatedBy:      createdBy,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := h.storeAPIKey(ctx, key); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to create api key", http.StatusInternalServerError))
		return
	}

	// Only return the full key once
	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"id":         key.ID,
		"name":       key.Name,
		"api_key":    rawKey,
		"key_prefix": key.KeyPrefix,
		"scopes":     key.Scopes,
		"rate_limit": key.RateLimit,
		"expires_at": func() string {
			if key.ExpiresAt != nil {
				return key.ExpiresAt.Format(time.RFC3339)
			}
			return ""
		}(),
		"created_at": key.CreatedAt.Format(time.RFC3339),
	})
}

// APIKeyHandler handles individual API key operations.
func (h *DeveloperHandler) APIKeyHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract key ID from path
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	keyID := parts[len(parts)-1]

	switch r.Method {
	case http.MethodGet:
		h.getAPIKey(w, r, ctx, keyID)
	case http.MethodPut, http.MethodPatch:
		h.updateAPIKey(w, r, ctx, keyID)
	case http.MethodDelete:
		h.deleteAPIKey(w, r, ctx, keyID)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *DeveloperHandler) getAPIKey(w http.ResponseWriter, r *http.Request, ctx context.Context, keyID string) {
	key, err := h.getAPIKeyByID(ctx, keyID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get api key", http.StatusInternalServerError))
		return
	}
	if key == nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":         key.ID,
		"name":       key.Name,
		"key_prefix": key.KeyPrefix,
		"scopes":     key.Scopes,
		"is_active":  key.IsActive,
		"rate_limit": key.RateLimit,
		"created_at": key.CreatedAt.Format(time.RFC3339),
		"last_used_at": func() string {
			if key.LastUsedAt != nil {
				return key.LastUsedAt.Format(time.RFC3339)
			}
			return ""
		}(),
	})
}

func (h *DeveloperHandler) updateAPIKey(w http.ResponseWriter, r *http.Request, ctx context.Context, keyID string) {
	key, err := h.getAPIKeyByID(ctx, keyID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get api key", http.StatusInternalServerError))
		return
	}
	if key == nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	var req UpdateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Update fields
	if req.Name != "" {
		key.Name = req.Name
	}
	if req.Scopes != nil {
		key.Scopes = req.Scopes
	}
	if req.IsActive != nil {
		key.IsActive = *req.IsActive
	}
	if req.RateLimit > 0 {
		key.RateLimit = req.RateLimit
	}
	key.UpdatedAt = time.Now()

	if err := h.updateAPIKeyStore(ctx, key); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to update api key", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":         key.ID,
		"name":       key.Name,
		"key_prefix": key.KeyPrefix,
		"scopes":     key.Scopes,
		"is_active":  key.IsActive,
		"rate_limit": key.RateLimit,
		"updated_at": key.UpdatedAt.Format(time.RFC3339),
	})
}

func (h *DeveloperHandler) deleteAPIKey(w http.ResponseWriter, r *http.Request, ctx context.Context, keyID string) {
	if err := h.deleteAPIKeyStore(ctx, keyID); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to delete api key", http.StatusInternalServerError))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UsageStatsHandler handles usage statistics for API keys.
func (h *DeveloperHandler) UsageStatsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgID := r.URL.Query().Get("organization_id")
	apiKeyID := r.URL.Query().Get("api_key_id")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	if orgID == "" {
		if oid := ctx.Value("org_id"); oid != nil {
			orgID = fmt.Sprintf("%v", oid)
		}
	}

	// Default to last 30 days
	if startDate == "" {
		startDate = time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	}
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}

	stats, err := h.getUsageStats(ctx, orgID, apiKeyID, startDate, endDate)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get usage stats", http.StatusInternalServerError))
		return
	}

	response := make([]map[string]interface{}, len(stats))
	totalRequests := 0
	for i, s := range stats {
		totalRequests += s.RequestCount
		response[i] = map[string]interface{}{
			"api_key_id":     s.APIKeyID,
			"date":           s.Date,
			"request_count":  s.RequestCount,
			"error_count":    s.ErrorCount,
			"avg_latency_ms": s.AvgLatency,
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"organization_id": orgID,
		"start_date":      startDate,
		"end_date":        endDate,
		"stats":           response,
		"total_requests":  totalRequests,
	})
}

// WebhooksHandler handles webhook list and creation.
func (h *DeveloperHandler) WebhooksHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		h.listWebhooksEndpoint(w, r, ctx)
	case http.MethodPost:
		h.createWebhook(w, r, ctx)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *DeveloperHandler) listWebhooksEndpoint(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	orgID := r.URL.Query().Get("organization_id")
	if orgID == "" {
		if oid := ctx.Value("org_id"); oid != nil {
			orgID = fmt.Sprintf("%v", oid)
		}
	}

	webhooks, err := h.listWebhooks(ctx, orgID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to list webhooks", http.StatusInternalServerError))
		return
	}

	response := make([]map[string]interface{}, len(webhooks))
	for i, wh := range webhooks {
		response[i] = map[string]interface{}{
			"id":         wh.ID,
			"name":       wh.Name,
			"url":        wh.URL,
			"events":     wh.Events,
			"is_active":  wh.IsActive,
			"created_at": wh.CreatedAt.Format(time.RFC3339),
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"webhooks": response,
		"count":    len(response),
	})
}

func (h *DeveloperHandler) createWebhook(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	var req CreateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	if req.Name == "" {
		respondError(w, apperrors.New("name is required", http.StatusBadRequest))
		return
	}
	if req.URL == "" {
		respondError(w, apperrors.New("url is required", http.StatusBadRequest))
		return
	}
	if len(req.Events) == 0 {
		req.Events = []string{"job.completed", "job.failed"}
	}
	if req.OrganizationID == "" {
		if oid := ctx.Value("org_id"); oid != nil {
			req.OrganizationID = fmt.Sprintf("%v", oid)
		}
	}

	// Generate secret if not provided
	if req.Secret == "" {
		req.Secret = generateWebhookSecret()
	}

	webhook := &Webhook{
		ID:             uuid.New().String(),
		OrganizationID: req.OrganizationID,
		Name:           req.Name,
		URL:            req.URL,
		Secret:         req.Secret,
		Events:         req.Events,
		IsActive:       true,
		Headers:        req.Headers,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := h.storeWebhook(ctx, webhook); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to create webhook", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"id":         webhook.ID,
		"name":       webhook.Name,
		"url":        webhook.URL,
		"events":     webhook.Events,
		"secret":     webhook.Secret,
		"is_active":  webhook.IsActive,
		"created_at": webhook.CreatedAt.Format(time.RFC3339),
	})
}

// WebhookHandler handles individual webhook operations.
func (h *DeveloperHandler) WebhookHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract webhook ID from path
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	webhookID := parts[len(parts)-1]

	switch r.Method {
	case http.MethodDelete:
		h.deleteWebhook(w, r, ctx, webhookID)
	case http.MethodPost:
		// Test webhook
		h.testWebhook(w, r, ctx, webhookID)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *DeveloperHandler) deleteWebhook(w http.ResponseWriter, r *http.Request, ctx context.Context, webhookID string) {
	if err := h.deleteWebhookStore(ctx, webhookID); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to delete webhook", http.StatusInternalServerError))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *DeveloperHandler) testWebhook(w http.ResponseWriter, r *http.Request, ctx context.Context, webhookID string) {
	webhook, err := h.getWebhookByID(ctx, webhookID)
	if err != nil || webhook == nil {
		respondError(w, apperrors.Wrap(err, "webhook not found", http.StatusNotFound))
		return
	}

	// Send test webhook
	testPayload := map[string]interface{}{
		"test":      true,
		"timestamp": time.Now().Format(time.RFC3339),
		"event":     "test",
	}

	// In production, actually send the webhook
	_ = h.sendWebhookRequest(ctx, webhook, testPayload)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Test webhook sent",
		"url":     webhook.URL,
	})
}

// Repository methods

func (h *DeveloperHandler) storeAPIKey(ctx context.Context, key *APIKey) error {
	h.initTable(ctx, "api_keys")

	query := `
		INSERT INTO api_keys (
			id, organization_id, name, key_hash, key_prefix, scopes,
			is_active, expires_at, rate_limit, created_by, created_at, updated_at
		) VALUES (
			$1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
	`

	_, err := h.db.Exec(ctx, query,
		key.ID, key.OrganizationID, key.Name, key.Key, key.KeyPrefix,
		key.Scopes, key.IsActive, key.ExpiresAt, key.RateLimit,
		key.CreatedBy, key.CreatedAt, key.UpdatedAt,
	)

	return err
}

func (h *DeveloperHandler) getAPIKeyByID(ctx context.Context, keyID string) (*APIKey, error) {
	query := `
		SELECT id, organization_id, name, key_hash, key_prefix, scopes,
		       is_active, expires_at, rate_limit, created_by, created_at, updated_at, last_used_at
		FROM api_keys
		WHERE id = $1::uuid
	`

	var key APIKey
	err := h.db.QueryRow(ctx, query, keyID).Scan(
		&key.ID, &key.OrganizationID, &key.Name, &key.Key, &key.KeyPrefix, &key.Scopes,
		&key.IsActive, &key.ExpiresAt, &key.RateLimit, &key.CreatedBy,
		&key.CreatedAt, &key.UpdatedAt, &key.LastUsedAt,
	)

	if err != nil {
		return nil, err
	}

	return &key, nil
}

func (h *DeveloperHandler) listAPIKeys(ctx context.Context, orgID string) ([]*APIKey, error) {
	query := `
		SELECT id, organization_id, name, key_hash, key_prefix, scopes,
		       is_active, expires_at, rate_limit, created_by, created_at, updated_at, last_used_at
		FROM api_keys
		WHERE organization_id = $1::uuid
		ORDER BY created_at DESC
	`

	rows, err := h.db.Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*APIKey
	for rows.Next() {
		var key APIKey
		if err := rows.Scan(
			&key.ID, &key.OrganizationID, &key.Name, &key.Key, &key.KeyPrefix, &key.Scopes,
			&key.IsActive, &key.ExpiresAt, &key.RateLimit, &key.CreatedBy,
			&key.CreatedAt, &key.UpdatedAt, &key.LastUsedAt,
		); err != nil {
			return nil, err
		}
		keys = append(keys, &key)
	}

	return keys, nil
}

func (h *DeveloperHandler) updateAPIKeyStore(ctx context.Context, key *APIKey) error {
	query := `
		UPDATE api_keys
		SET name = $2, scopes = $3, is_active = $4, rate_limit = $5, updated_at = $6
		WHERE id = $1::uuid
	`

	_, err := h.db.Exec(ctx, query, key.ID, key.Name, key.Scopes, key.IsActive, key.RateLimit, key.UpdatedAt)
	return err
}

func (h *DeveloperHandler) deleteAPIKeyStore(ctx context.Context, keyID string) error {
	query := `DELETE FROM api_keys WHERE id = $1::uuid`
	_, err := h.db.Exec(ctx, query, keyID)
	return err
}

func (h *DeveloperHandler) getUsageStats(ctx context.Context, orgID, apiKeyID, startDate, endDate string) ([]*UsageStats, error) {
	query := `
		SELECT api_key_id, DATE(created_at)::text as date,
		       COUNT(*) as request_count,
		       SUM(CASE WHEN status >= 400 THEN 1 ELSE 0 END)::integer as error_count,
		      	AVG(latency_ms)::integer as avg_latency
		FROM api_usage_logs
		WHERE organization_id = $1::uuid
		  AND ($2::uuid = ''::uuid OR api_key_id = $2::uuid)
		  AND DATE(created_at) >= $3::date
		  AND DATE(created_at) <= $4::date
		GROUP BY api_key_id, DATE(created_at)
		ORDER BY date DESC
	`

	rows, err := h.db.Query(ctx, query, nullIfEmpty(orgID), nullIfEmpty(apiKeyID), startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*UsageStats
	for rows.Next() {
		var stat UsageStats
		if err := rows.Scan(&stat.APIKeyID, &stat.Date, &stat.RequestCount, &stat.ErrorCount, &stat.AvgLatency); err != nil {
			return nil, err
		}
		stats = append(stats, &stat)
	}

	return stats, nil
}

func (h *DeveloperHandler) storeWebhook(ctx context.Context, webhook *Webhook) error {
	h.initTable(ctx, "webhooks")

	query := `
		INSERT INTO webhooks (
			id, organization_id, name, url, secret, events, is_active, headers, created_at, updated_at
		) VALUES (
			$1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9, $10
		)
	`

	headersJSON, _ := json.Marshal(webhook.Headers)

	_, err := h.db.Exec(ctx, query,
		webhook.ID, webhook.OrganizationID, webhook.Name, webhook.URL,
		webhook.Secret, webhook.Events, webhook.IsActive, headersJSON,
		webhook.CreatedAt, webhook.UpdatedAt,
	)

	return err
}

func (h *DeveloperHandler) getWebhookByID(ctx context.Context, webhookID string) (*Webhook, error) {
	query := `
		SELECT id, organization_id, name, url, secret, events, is_active, headers, created_at, updated_at
		FROM webhooks
		WHERE id = $1::uuid
	`

	var webhook Webhook
	var headersJSON []byte

	err := h.db.QueryRow(ctx, query, webhookID).Scan(
		&webhook.ID, &webhook.OrganizationID, &webhook.Name, &webhook.URL,
		&webhook.Secret, &webhook.Events, &webhook.IsActive, &headersJSON,
		&webhook.CreatedAt, &webhook.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	json.Unmarshal(headersJSON, &webhook.Headers)

	return &webhook, nil
}

func (h *DeveloperHandler) listWebhooks(ctx context.Context, orgID string) ([]*Webhook, error) {
	query := `
		SELECT id, organization_id, name, url, secret, events, is_active, headers, created_at, updated_at
		FROM webhooks
		WHERE organization_id = $1::uuid
		ORDER BY created_at DESC
	`

	rows, err := h.db.Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []*Webhook
	for rows.Next() {
		var webhook Webhook
		var headersJSON []byte

		if err := rows.Scan(
			&webhook.ID, &webhook.OrganizationID, &webhook.Name, &webhook.URL,
			&webhook.Secret, &webhook.Events, &webhook.IsActive, &headersJSON,
			&webhook.CreatedAt, &webhook.UpdatedAt,
		); err != nil {
			return nil, err
		}

		json.Unmarshal(headersJSON, &webhook.Headers)
		webhooks = append(webhooks, &webhook)
	}

	return webhooks, nil
}

func (h *DeveloperHandler) deleteWebhookStore(ctx context.Context, webhookID string) error {
	query := `DELETE FROM webhooks WHERE id = $1::uuid`
	_, err := h.db.Exec(ctx, query, webhookID)
	return err
}

func (h *DeveloperHandler) sendWebhookRequest(ctx context.Context, webhook *Webhook, payload map[string]interface{}) error {
	// In production, this would make an actual HTTP request
	// For now, just log the webhook delivery
	return nil
}

// Helper functions

func (h *DeveloperHandler) initTable(ctx context.Context, tableName string) {
	var createSQL string
	switch tableName {
	case "api_keys":
		createSQL = `
			CREATE TABLE IF NOT EXISTS api_keys (
				id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
				organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
				name VARCHAR(255) NOT NULL,
				key_hash VARCHAR(64) NOT NULL UNIQUE,
				key_prefix VARCHAR(8) NOT NULL,
				scopes TEXT[] NOT NULL DEFAULT ARRAY['read', 'write'],
				is_active BOOLEAN DEFAULT true,
				expires_at TIMESTAMPTZ,
				rate_limit INTEGER DEFAULT 60,
				created_by VARCHAR(255),
				created_at TIMESTAMPTZ DEFAULT NOW(),
				updated_at TIMESTAMPTZ DEFAULT NOW(),
				last_used_at TIMESTAMPTZ
			);
			CREATE INDEX IF NOT EXISTS idx_api_keys_org ON api_keys(organization_id);
			CREATE INDEX IF NOT EXISTS idx_api_keys_prefix ON api_keys(key_prefix);
			CREATE TRIGGER update_api_keys_updated_at BEFORE UPDATE ON api_keys
			    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
		`
	case "webhooks":
		createSQL = `
			CREATE TABLE IF NOT EXISTS webhooks (
				id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
				organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
				name VARCHAR(255) NOT NULL,
				url VARCHAR(2048) NOT NULL,
				secret VARCHAR(255) NOT NULL DEFAULT uuid_generate_v4()::text,
				events TEXT[] NOT NULL,
				is_active BOOLEAN DEFAULT true,
				headers JSONB,
				created_at TIMESTAMPTZ DEFAULT NOW(),
				updated_at TIMESTAMPTZ DEFAULT NOW()
			);
			CREATE INDEX IF NOT EXISTS idx_webhooks_org ON webhooks(organization_id);
			CREATE TRIGGER update_webhooks_updated_at BEFORE UPDATE ON webhooks
			    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
		`
	}

	if createSQL != "" {
		h.db.Exec(ctx, createSQL)
	}
}

func generateAPIKey() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return "opk_" + hex.EncodeToString(bytes)
}

func generateWebhookSecret() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(map[string]string{
		"code":    "INTERNAL_ERROR",
		"message": "An internal error occurred",
	})
}
