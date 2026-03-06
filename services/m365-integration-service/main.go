// Package main is the entry point for the OpenPrint Microsoft 365 Integration Service.
// This service handles Microsoft 365 integration for OneDrive, SharePoint, and document handling.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openprint/openprint/internal/auth/jwt"
	"github.com/openprint/openprint/internal/shared/middleware"
	"github.com/openprint/openprint/internal/shared/telemetry"
)

// Config holds service configuration.
type Config struct {
	ServerAddr       string
	DatabaseURL      string
	JWTSecret        string
	JaegerEndpoint   string
	ServiceName      string
	M365ClientID     string
	M365ClientSecret string
	M365TenantID     string
	StoragePath      string
	JobServiceURL    string
}

// Service provides Microsoft 365 integration functionality.
type Service struct {
	db              *pgxpool.Pool
	config          *Config
	httpClient      *http.Client
	graphAPIBaseURL string
}

// M365Document represents a document from Microsoft 365.
type M365Document struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	DriveID       string                 `json:"drive_id"`
	ItemID        string                 `json:"item_id"`
	DownloadURL   string                 `json:"download_url"`
	Size          int64                  `json:"size"`
	MimeType      string                 `json:"mime_type"`
	CreatedBy     string                 `json:"created_by"`
	CreatedAt     time.Time              `json:"created_at"`
	ModifiedAt    time.Time              `json:"modified_at"`
	SharePointURL string                 `json:"sharepoint_url,omitempty"`
	OneDriveURL   string                 `json:"onedrive_url,omitempty"`
	Path          string                 `json:"path"`
	WebURL        string                 `json:"web_url"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// PrintJobSource represents the source of a print job from Microsoft 365.
type PrintJobSource struct {
	SourceID     string                 `json:"source_id"`
	SourceType   string                 `json:"source_type"` // onedrive, sharepoint, outlook
	DocumentID   string                 `json:"document_id"`
	DocumentURL  string                 `json:"document_url"`
	UserID       string                 `json:"user_id"`
	UserEmail    string                 `json:"user_email"`
	FileName     string                 `json:"file_name"`
	FileSize     int64                  `json:"file_size"`
	AccessToken  string                 `json:"access_token"`
	RefreshToken string                 `json:"refresh_token"`
	SiteID       string                 `json:"site_id,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// SharePointSite represents a SharePoint site.
type SharePointSite struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	URL          string    `json:"url"`
	Description  string    `json:"description"`
	CreatedAt    time.Time `json:"created_at"`
	LastModified time.Time `json:"last_modified"`
	WebURL       string    `json:"web_url"`
}

// OneDriveDrive represents a OneDrive drive.
type OneDriveDrive struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	DriveType string `json:"drive_type"`
	OwnerID   string `json:"owner_id"`
	OwnerName string `json:"owner_name"`
}

// M365Connection represents a user's Microsoft 365 connection.
type M365Connection struct {
	ID           string                 `json:"id"`
	UserID       string                 `json:"user_id"`
	UserEmail    string                 `json:"user_email"`
	TenantID     string                 `json:"tenant_id"`
	AccessToken  string                 `json:"access_token"`
	RefreshToken string                 `json:"refresh_token"`
	TokenExpiry  time.Time              `json:"token_expiry"`
	Scopes       []string               `json:"scopes"`
	ConnectedAt  time.Time              `json:"connected_at"`
	LastUsed     *time.Time             `json:"last_used,omitempty"`
	IsActive     bool                   `json:"is_active"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// PrintJobM365Metadata stores metadata for Microsoft 365 print jobs.
type PrintJobM365Metadata struct {
	JobID          string                 `json:"job_id"`
	SourceID       string                 `json:"source_id"`
	SourceType     string                 `json:"source_type"`
	DocumentID     string                 `json:"document_id"`
	DocumentName   string                 `json:"document_name"`
	OriginalURL    string                 `json:"original_url"`
	DownloadedAt   time.Time              `json:"downloaded_at"`
	DownloadedSize int64                  `json:"downloaded_size"`
	StoredPath     string                 `json:"stored_path"`
	HashCode       string                 `json:"hash_code"`
	Properties     map[string]interface{} `json:"properties,omitempty"`
}

// GraphAPIResponse represents a standard Microsoft Graph API response.
type GraphAPIResponse struct {
	Value []json.RawMessage `json:"value,omitempty"`
}

// GraphAPIError represents an error from the Graph API.
type GraphAPIError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// tokenResponse represents the OAuth token response from Microsoft identity platform.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
}

// tokenErrorResponse represents an OAuth error from Microsoft identity platform.
type tokenErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	ErrorCodes       []int  `json:"error_codes"`
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

	// Create storage directory
	if cfg.StoragePath != "" {
		if err := os.MkdirAll(cfg.StoragePath, 0755); err != nil {
			log.Printf("Warning: failed to create storage directory: %v", err)
		}
	}

	// Initialize service
	svc := &Service{
		db:     db,
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		graphAPIBaseURL: "https://graph.microsoft.com/v1.0",
	}

	// Create JWT manager for authentication
	jwtCfg, err := jwt.DefaultConfig(cfg.JWTSecret)
	if err != nil {
		log.Fatalf("Failed to create JWT config: %v", err)
	}
	jwtManager, err := jwt.NewManager(jwtCfg)
	if err != nil {
		log.Fatalf("Failed to create JWT manager: %v", err)
	}

	// Setup HTTP server with middleware
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", healthHandler)

	// Microsoft 365 integration endpoints
	mux.HandleFunc("/api/v1/m365/authorize", svc.authorizeHandler)
	mux.HandleFunc("/api/v1/m365/callback", svc.callbackHandler)
	mux.HandleFunc("/api/v1/m365/connections", svc.connectionsHandler)
	mux.HandleFunc("/api/v1/m365/connections/", svc.connectionHandler)
	mux.HandleFunc("/api/v1/m365/onedrive/files", svc.oneDriveFilesHandler)
	mux.HandleFunc("/api/v1/m365/onedrive/download", svc.oneDriveDownloadHandler)
	mux.HandleFunc("/api/v1/m365/sharepoint/sites", svc.sharePointSitesHandler)
	mux.HandleFunc("/api/v1/m365/sharepoint/files", svc.sharePointFilesHandler)
	mux.HandleFunc("/api/v1/m365/print/submit", svc.printSubmitHandler)
	mux.HandleFunc("/api/v1/m365/print/status/", svc.printStatusHandler)

	// Build middleware chain
	middlewareChain := middleware.Chain(
		middleware.LoggingMiddleware(log.New(os.Stdout, "[M365-INTEGRATION] ", log.LstdFlags)),
		middleware.RecoveryMiddleware(log.New(os.Stdout, "[M365-INTEGRATION] ", log.LstdFlags)),
		middleware.AuthMiddleware(middleware.JWTConfig{
			SecretKey:  cfg.JWTSecret,
			JWTManager: jwtManager,
			SkipPaths:  []string{"/health", "/api/v1/m365/callback"},
		}),
		telemetry.HTTPMiddleware(cfg.ServiceName),
		middleware.SecurityHeadersMiddleware(),
		middleware.CORSMiddleware(
			[]string{"*"},
			[]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			[]string{"Content-Type", "Authorization"},
		),
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

	storagePath := os.Getenv("M365_STORAGE_PATH")
	if storagePath == "" {
		storagePath = "/tmp/openprint/m365"
	}

	return &Config{
		ServerAddr:       getEnv("SERVER_ADDR", ":8008"),
		DatabaseURL:      getEnv("DATABASE_URL", "postgres://openprint:openprint@localhost:5432/openprint"),
		JWTSecret:        jwtSecret,
		JaegerEndpoint:   getEnv("JAEGER_ENDPOINT", ""),
		ServiceName:      getEnv("SERVICE_NAME", "m365-integration-service"),
		M365ClientID:     getEnv("M365_CLIENT_ID", ""),
		M365ClientSecret: getEnv("M365_CLIENT_SECRET", ""),
		M365TenantID:     getEnv("M365_TENANT_ID", "common"),
		StoragePath:      storagePath,
		JobServiceURL:    getEnv("JOB_SERVICE_URL", "http://localhost:8003"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// hasM365Credentials returns true if Microsoft 365 OAuth credentials are configured.
func (s *Service) hasM365Credentials() bool {
	return s.config.M365ClientID != "" && s.config.M365ClientSecret != ""
}

// requireM365Credentials checks that M365 credentials are configured and writes
// an error response if they are not. Returns true if credentials are present.
func (s *Service) requireM365Credentials(w http.ResponseWriter) bool {
	if s.hasM365Credentials() {
		return true
	}
	respondJSON(w, http.StatusServiceUnavailable, map[string]string{
		"error":   "m365_not_configured",
		"message": "Microsoft 365 integration is not configured: M365_CLIENT_ID and M365_CLIENT_SECRET environment variables are required",
	})
	return false
}

// tokenEndpointURL returns the Microsoft identity platform token endpoint for the configured tenant.
func (s *Service) tokenEndpointURL() string {
	return fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", s.config.M365TenantID)
}

// graphAPIRequest performs an authenticated request to the Microsoft Graph API.
// It returns the response body bytes or an error.
func (s *Service) graphAPIRequest(ctx context.Context, method, path, accessToken string, body io.Reader) ([]byte, error) {
	reqURL := s.graphAPIBaseURL + path

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("graph API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var graphErr GraphAPIError
		if jsonErr := json.Unmarshal(respBody, &graphErr); jsonErr == nil && graphErr.Error.Code != "" {
			return nil, fmt.Errorf("graph API error (HTTP %d): %s - %s", resp.StatusCode, graphErr.Error.Code, graphErr.Error.Message)
		}
		return nil, fmt.Errorf("graph API returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "m365-integration-service",
	})
}

// authorizeHandler initiates the OAuth authorization flow.
func (s *Service) authorizeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !s.requireM365Credentials(w) {
		return
	}

	var req struct {
		RedirectURI string `json:"redirect_uri"`
		State       string `json:"state"`
		Scopes      string `json:"scopes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.RedirectURI == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "missing_redirect_uri",
			"message": "redirect_uri is required",
		})
		return
	}

	scopes := req.Scopes
	if scopes == "" {
		scopes = "User.Read Files.Read.All Sites.Read.All offline_access"
	}

	params := url.Values{}
	params.Set("client_id", s.config.M365ClientID)
	params.Set("response_type", "code")
	params.Set("redirect_uri", req.RedirectURI)
	params.Set("scope", scopes)
	if req.State != "" {
		params.Set("state", req.State)
	}

	authURL := fmt.Sprintf(
		"https://login.microsoftonline.com/%s/oauth2/v2.0/authorize?%s",
		s.config.M365TenantID,
		params.Encode(),
	)

	respondJSON(w, http.StatusOK, map[string]string{
		"authorization_url": authURL,
	})
}

// callbackHandler handles the OAuth callback by exchanging the authorization code for tokens.
func (s *Service) callbackHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errorCode := r.URL.Query().Get("error")

	if errorCode != "" {
		errorDescription := r.URL.Query().Get("error_description")
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error":   errorCode,
			"message": errorDescription,
		})
		return
	}

	if code == "" {
		http.Error(w, "missing authorization code", http.StatusBadRequest)
		return
	}

	if !s.hasM365Credentials() {
		respondJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error":   "m365_not_configured",
			"message": "Microsoft 365 integration is not configured: M365_CLIENT_ID and M365_CLIENT_SECRET environment variables are required",
		})
		return
	}

	redirectURI := r.URL.Query().Get("redirect_uri")
	if redirectURI == "" {
		redirectURI = fmt.Sprintf("http://%s/api/v1/m365/callback", r.Host)
	}

	// Exchange authorization code for tokens via Microsoft identity platform
	formData := url.Values{}
	formData.Set("client_id", s.config.M365ClientID)
	formData.Set("client_secret", s.config.M365ClientSecret)
	formData.Set("code", code)
	formData.Set("redirect_uri", redirectURI)
	formData.Set("grant_type", "authorization_code")

	tokenReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, s.tokenEndpointURL(), strings.NewReader(formData.Encode()))
	if err != nil {
		log.Printf("Error creating token request: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "token_exchange_failed",
			"message": "failed to create token exchange request",
		})
		return
	}
	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(tokenReq)
	if err != nil {
		log.Printf("Error exchanging code for tokens: %v", err)
		respondJSON(w, http.StatusBadGateway, map[string]string{
			"error":   "token_exchange_failed",
			"message": fmt.Sprintf("failed to contact Microsoft identity platform: %v", err),
		})
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading token response: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "token_exchange_failed",
			"message": "failed to read token response",
		})
		return
	}

	if resp.StatusCode != http.StatusOK {
		var tokenErr tokenErrorResponse
		if jsonErr := json.Unmarshal(respBody, &tokenErr); jsonErr == nil && tokenErr.Error != "" {
			log.Printf("Token exchange error: %s - %s", tokenErr.Error, tokenErr.ErrorDescription)
			respondJSON(w, http.StatusBadRequest, map[string]string{
				"error":   tokenErr.Error,
				"message": tokenErr.ErrorDescription,
			})
			return
		}
		log.Printf("Token exchange failed with HTTP %d: %s", resp.StatusCode, string(respBody))
		respondJSON(w, http.StatusBadGateway, map[string]string{
			"error":   "token_exchange_failed",
			"message": fmt.Sprintf("Microsoft identity platform returned HTTP %d", resp.StatusCode),
		})
		return
	}

	var tokenResp tokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		log.Printf("Error parsing token response: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "token_parse_failed",
			"message": "failed to parse token response from Microsoft",
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"state":         state,
		"access_token":  tokenResp.AccessToken,
		"refresh_token": tokenResp.RefreshToken,
		"expires_in":    tokenResp.ExpiresIn,
		"token_type":    tokenResp.TokenType,
		"scope":         tokenResp.Scope,
	})
}

// connectionsHandler manages user connections to Microsoft 365.
func (s *Service) connectionsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		connections, err := s.listConnections(ctx)
		if err != nil {
			log.Printf("Failed to list connections: %v", err)
			respondJSON(w, http.StatusInternalServerError, map[string]string{
				"error":   "database_error",
				"message": "failed to retrieve connections",
			})
			return
		}
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"connections": connections,
		})

	case http.MethodPost:
		// Create new connection
		var req M365Connection
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		req.ID = uuid.New().String()
		req.ConnectedAt = time.Now()
		req.IsActive = true

		// Store connection in database
		if err := s.storeConnection(ctx, &req); err != nil {
			log.Printf("Failed to store connection: %v", err)
			respondJSON(w, http.StatusInternalServerError, map[string]string{
				"error":   "database_error",
				"message": "failed to store connection",
			})
			return
		}

		respondJSON(w, http.StatusCreated, req)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// connectionHandler handles individual connection operations.
func (s *Service) connectionHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	connectionID := extractIDFromPath(r.URL.Path, "/api/v1/m365/connections/")

	if connectionID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "missing_connection_id",
			"message": "connection ID is required",
		})
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Get connection
		connection, err := s.getConnection(ctx, connectionID)
		if err != nil {
			respondJSON(w, http.StatusNotFound, map[string]string{
				"error":   "not_found",
				"message": fmt.Sprintf("connection %s not found", connectionID),
			})
			return
		}
		respondJSON(w, http.StatusOK, connection)

	case http.MethodDelete:
		// Delete connection
		if err := s.deleteConnection(ctx, connectionID); err != nil {
			respondJSON(w, http.StatusInternalServerError, map[string]string{
				"error":   "delete_failed",
				"message": fmt.Sprintf("failed to delete connection: %v", err),
			})
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// oneDriveFilesHandler lists files from OneDrive via Microsoft Graph API.
func (s *Service) oneDriveFilesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !s.requireM365Credentials(w) {
		return
	}

	accessToken := extractBearerToken(r)
	if accessToken == "" {
		respondJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "missing_access_token",
			"message": "M365 access token is required; pass it as a Bearer token in the Authorization header or as the access_token query parameter",
		})
		return
	}

	// Determine the Graph API path based on query parameters
	folderPath := r.URL.Query().Get("path")
	graphPath := "/me/drive/root/children"
	if folderPath != "" {
		graphPath = fmt.Sprintf("/me/drive/root:/%s:/children", url.PathEscape(folderPath))
	}

	// Add query parameters for selecting fields
	graphPath += "?$select=id,name,size,file,folder,parentReference,createdDateTime,lastModifiedDateTime,webUrl,createdBy&$top=100"

	respBody, err := s.graphAPIRequest(r.Context(), http.MethodGet, graphPath, accessToken, nil)
	if err != nil {
		log.Printf("Failed to list OneDrive files: %v", err)
		respondJSON(w, http.StatusBadGateway, map[string]string{
			"error":   "graph_api_error",
			"message": fmt.Sprintf("failed to list OneDrive files: %v", err),
		})
		return
	}

	// Parse the Graph API response into our document format
	var graphResp struct {
		Value []json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(respBody, &graphResp); err != nil {
		log.Printf("Failed to parse Graph API response: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "parse_error",
			"message": "failed to parse Microsoft Graph API response",
		})
		return
	}

	files := make([]M365Document, 0, len(graphResp.Value))
	for _, raw := range graphResp.Value {
		doc, err := parseGraphDriveItem(raw)
		if err != nil {
			log.Printf("Skipping unparseable drive item: %v", err)
			continue
		}
		files = append(files, *doc)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"files": files,
		"total": len(files),
	})
}

// oneDriveDownloadHandler downloads a file from OneDrive via Graph API and stores it locally.
func (s *Service) oneDriveDownloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !s.requireM365Credentials(w) {
		return
	}

	var req struct {
		DocumentID  string `json:"document_id"`
		AccessToken string `json:"access_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.DocumentID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "missing_document_id",
			"message": "document_id is required",
		})
		return
	}

	if req.AccessToken == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "missing_access_token",
			"message": "access_token is required for downloading files from OneDrive",
		})
		return
	}

	// Sanitize document ID to prevent path traversal attacks
	sanitizedID := filepath.Base(req.DocumentID)
	if sanitizedID == "" || sanitizedID == "." || sanitizedID == ".." {
		http.Error(w, "invalid document_id", http.StatusBadRequest)
		return
	}

	// First, get the document metadata from Graph API to learn the filename and download URL
	doc, err := s.getDocumentFromOneDrive(r.Context(), req.AccessToken, req.DocumentID)
	if err != nil {
		log.Printf("Failed to get document metadata from OneDrive: %v", err)
		respondJSON(w, http.StatusBadGateway, map[string]string{
			"error":   "graph_api_error",
			"message": fmt.Sprintf("failed to retrieve document from OneDrive: %v", err),
		})
		return
	}

	// Download the file content
	fileData, err := downloadFile(doc.DownloadURL, req.AccessToken)
	if err != nil {
		log.Printf("Failed to download file from OneDrive: %v", err)
		respondJSON(w, http.StatusBadGateway, map[string]string{
			"error":   "download_failed",
			"message": fmt.Sprintf("failed to download file: %v", err),
		})
		return
	}

	// Store to local storage
	storageName := sanitizedID + "_" + sanitizeFilename(doc.Name)
	storedPath, written, err := s.copyToStorage(bytes.NewReader(fileData), storageName)
	if err != nil {
		log.Printf("Failed to store downloaded file: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "storage_failed",
			"message": fmt.Sprintf("failed to store downloaded file: %v", err),
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"document_id": req.DocumentID,
		"name":        doc.Name,
		"local_path":  storedPath,
		"size":        written,
		"mime_type":   doc.MimeType,
		"status":      "downloaded",
	})
}

// sharePointSitesHandler lists SharePoint sites the user has access to via Graph API.
func (s *Service) sharePointSitesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !s.requireM365Credentials(w) {
		return
	}

	accessToken := extractBearerToken(r)
	if accessToken == "" {
		respondJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "missing_access_token",
			"message": "M365 access token is required; pass it as a Bearer token in the Authorization header or as the access_token query parameter",
		})
		return
	}

	// Search for SharePoint sites the user has access to
	searchQuery := r.URL.Query().Get("search")
	graphPath := "/sites?search=" + url.QueryEscape(searchQuery) + "&$select=id,displayName,webUrl,description,createdDateTime,lastModifiedDateTime"
	if searchQuery == "" {
		// Default: list followed sites
		graphPath = "/me/followedSites?$select=id,displayName,webUrl,description,createdDateTime,lastModifiedDateTime"
	}

	respBody, err := s.graphAPIRequest(r.Context(), http.MethodGet, graphPath, accessToken, nil)
	if err != nil {
		log.Printf("Failed to list SharePoint sites: %v", err)
		respondJSON(w, http.StatusBadGateway, map[string]string{
			"error":   "graph_api_error",
			"message": fmt.Sprintf("failed to list SharePoint sites: %v", err),
		})
		return
	}

	var graphResp struct {
		Value []json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(respBody, &graphResp); err != nil {
		log.Printf("Failed to parse SharePoint sites response: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "parse_error",
			"message": "failed to parse SharePoint sites response",
		})
		return
	}

	sites := make([]SharePointSite, 0, len(graphResp.Value))
	for _, raw := range graphResp.Value {
		site, err := parseGraphSite(raw)
		if err != nil {
			log.Printf("Skipping unparseable site: %v", err)
			continue
		}
		sites = append(sites, *site)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"sites": sites,
		"total": len(sites),
	})
}

// sharePointFilesHandler lists files from a SharePoint site's default drive via Graph API.
func (s *Service) sharePointFilesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !s.requireM365Credentials(w) {
		return
	}

	accessToken := extractBearerToken(r)
	if accessToken == "" {
		respondJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "missing_access_token",
			"message": "M365 access token is required; pass it as a Bearer token in the Authorization header or as the access_token query parameter",
		})
		return
	}

	siteID := r.URL.Query().Get("site_id")
	if siteID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "missing_site_id",
			"message": "site_id query parameter is required",
		})
		return
	}

	folderPath := r.URL.Query().Get("path")
	graphPath := fmt.Sprintf("/sites/%s/drive/root/children", url.PathEscape(siteID))
	if folderPath != "" {
		graphPath = fmt.Sprintf("/sites/%s/drive/root:/%s:/children", url.PathEscape(siteID), url.PathEscape(folderPath))
	}
	graphPath += "?$select=id,name,size,file,folder,parentReference,createdDateTime,lastModifiedDateTime,webUrl,createdBy&$top=100"

	respBody, err := s.graphAPIRequest(r.Context(), http.MethodGet, graphPath, accessToken, nil)
	if err != nil {
		log.Printf("Failed to list SharePoint files for site %s: %v", siteID, err)
		respondJSON(w, http.StatusBadGateway, map[string]string{
			"error":   "graph_api_error",
			"message": fmt.Sprintf("failed to list SharePoint files: %v", err),
		})
		return
	}

	var graphResp struct {
		Value []json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(respBody, &graphResp); err != nil {
		log.Printf("Failed to parse SharePoint files response: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "parse_error",
			"message": "failed to parse SharePoint files response",
		})
		return
	}

	files := make([]M365Document, 0, len(graphResp.Value))
	for _, raw := range graphResp.Value {
		doc, err := parseGraphDriveItem(raw)
		if err != nil {
			log.Printf("Skipping unparseable drive item: %v", err)
			continue
		}
		files = append(files, *doc)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"site_id": siteID,
		"files":   files,
		"total":   len(files),
	})
}

// printSubmitHandler submits a print job from a Microsoft 365 source.
func (s *Service) printSubmitHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !s.requireM365Credentials(w) {
		return
	}

	var req PrintJobSource
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.AccessToken == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "missing_access_token",
			"message": "access_token is required to download the document from Microsoft 365",
		})
		return
	}

	if req.DocumentID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "missing_document_id",
			"message": "document_id is required",
		})
		return
	}

	if req.SourceType == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "missing_source_type",
			"message": "source_type is required (onedrive or sharepoint)",
		})
		return
	}

	// 1. Retrieve document metadata from M365
	var doc *M365Document
	var err error

	switch req.SourceType {
	case "onedrive":
		doc, err = s.getDocumentFromOneDrive(ctx, req.AccessToken, req.DocumentID)
	case "sharepoint":
		if req.SiteID == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{
				"error":   "missing_site_id",
				"message": "site_id is required for SharePoint source type",
			})
			return
		}
		doc, err = s.getDocumentFromSharePoint(ctx, req.AccessToken, req.SiteID, req.DocumentID)
	default:
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_source_type",
			"message": fmt.Sprintf("unsupported source_type: %s (must be onedrive or sharepoint)", req.SourceType),
		})
		return
	}

	if err != nil {
		log.Printf("Failed to retrieve document metadata: %v", err)
		respondJSON(w, http.StatusBadGateway, map[string]string{
			"error":   "document_retrieval_failed",
			"message": fmt.Sprintf("failed to retrieve document from %s: %v", req.SourceType, err),
		})
		return
	}

	// 2. Download the file content
	fileData, err := downloadFile(doc.DownloadURL, req.AccessToken)
	if err != nil {
		log.Printf("Failed to download document: %v", err)
		respondJSON(w, http.StatusBadGateway, map[string]string{
			"error":   "download_failed",
			"message": fmt.Sprintf("failed to download document: %v", err),
		})
		return
	}

	// 3. Store the file locally
	storageName := uuid.New().String() + "_" + sanitizeFilename(doc.Name)
	storedPath, written, err := s.copyToStorage(bytes.NewReader(fileData), storageName)
	if err != nil {
		log.Printf("Failed to store document: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "storage_failed",
			"message": fmt.Sprintf("failed to store downloaded document: %v", err),
		})
		return
	}

	// 4. Create a print job in the job-service
	jobID, err := s.createPrintJob(ctx, &req, storedPath)
	if err != nil {
		log.Printf("Failed to create print job: %v", err)
		respondJSON(w, http.StatusBadGateway, map[string]string{
			"error":   "job_creation_failed",
			"message": fmt.Sprintf("failed to create print job: %v", err),
		})
		return
	}

	metadata := &PrintJobM365Metadata{
		JobID:          jobID,
		SourceID:       req.SourceID,
		SourceType:     req.SourceType,
		DocumentID:     req.DocumentID,
		DocumentName:   doc.Name,
		OriginalURL:    doc.WebURL,
		DownloadedAt:   time.Now(),
		DownloadedSize: written,
		StoredPath:     storedPath,
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"job_id":     jobID,
		"status":     "pending",
		"created_at": time.Now().Format(time.RFC3339),
		"metadata":   metadata,
	})
}

// printStatusHandler returns the status of a Microsoft 365 print job by querying the job-service.
func (s *Service) printStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	jobID := extractIDFromPath(r.URL.Path, "/api/v1/m365/print/status/")
	if jobID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "missing_job_id",
			"message": "job ID is required in the URL path",
		})
		return
	}

	// Query job status from the job-service
	jobURL := fmt.Sprintf("%s/api/v1/jobs/%s", s.config.JobServiceURL, url.PathEscape(jobID))
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, jobURL, nil)
	if err != nil {
		log.Printf("Failed to create job status request: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "failed to create job status request",
		})
		return
	}

	// Forward the authorization header to the job-service
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to query job status from job-service: %v", err)
		respondJSON(w, http.StatusBadGateway, map[string]string{
			"error":   "job_service_unavailable",
			"message": fmt.Sprintf("failed to reach job-service: %v", err),
		})
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read job-service response: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "failed to read job status response",
		})
		return
	}

	// Forward the job-service response as-is
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)
}

// Database operations

func (s *Service) storeConnection(ctx context.Context, conn *M365Connection) error {
	query := `
		INSERT INTO m365_connections
		(id, user_id, user_email, tenant_id, access_token, refresh_token,
		 token_expiry, scopes, connected_at, is_active, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (user_id) DO UPDATE
		SET access_token = EXCLUDED.access_token,
		    refresh_token = EXCLUDED.refresh_token,
		    token_expiry = EXCLUDED.token_expiry,
		    last_used = NOW()
	`

	scopesJSON, _ := json.Marshal(conn.Scopes)
	metadataJSON, _ := json.Marshal(conn.Metadata)

	_, err := s.db.Exec(ctx, query,
		conn.ID, conn.UserID, conn.UserEmail, conn.TenantID,
		conn.AccessToken, conn.RefreshToken, conn.TokenExpiry,
		scopesJSON, conn.ConnectedAt, conn.IsActive, metadataJSON,
	)

	return err
}

func (s *Service) getConnection(ctx context.Context, connectionID string) (*M365Connection, error) {
	query := `
		SELECT id, user_id, user_email, tenant_id, access_token, refresh_token,
		       token_expiry, scopes, connected_at, last_used, is_active, metadata
		FROM m365_connections
		WHERE id = $1
	`

	var conn M365Connection
	var scopesJSON, metadataJSON []byte

	err := s.db.QueryRow(ctx, query, connectionID).Scan(
		&conn.ID, &conn.UserID, &conn.UserEmail, &conn.TenantID,
		&conn.AccessToken, &conn.RefreshToken, &conn.TokenExpiry,
		&scopesJSON, &conn.ConnectedAt, &conn.LastUsed, &conn.IsActive,
		&metadataJSON,
	)

	if err != nil {
		return nil, err
	}

	json.Unmarshal(scopesJSON, &conn.Scopes)
	json.Unmarshal(metadataJSON, &conn.Metadata)

	return &conn, nil
}

func (s *Service) listConnections(ctx context.Context) ([]M365Connection, error) {
	query := `
		SELECT id, user_id, user_email, tenant_id, access_token, refresh_token,
		       token_expiry, scopes, connected_at, last_used, is_active, metadata
		FROM m365_connections
		WHERE is_active = true
		ORDER BY connected_at DESC
	`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query connections: %w", err)
	}
	defer rows.Close()

	var connections []M365Connection
	for rows.Next() {
		var conn M365Connection
		var scopesJSON, metadataJSON []byte

		if err := rows.Scan(
			&conn.ID, &conn.UserID, &conn.UserEmail, &conn.TenantID,
			&conn.AccessToken, &conn.RefreshToken, &conn.TokenExpiry,
			&scopesJSON, &conn.ConnectedAt, &conn.LastUsed, &conn.IsActive,
			&metadataJSON,
		); err != nil {
			return nil, fmt.Errorf("failed to scan connection row: %w", err)
		}

		json.Unmarshal(scopesJSON, &conn.Scopes)
		json.Unmarshal(metadataJSON, &conn.Metadata)
		connections = append(connections, conn)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating connection rows: %w", err)
	}

	if connections == nil {
		connections = []M365Connection{}
	}

	return connections, nil
}

func (s *Service) deleteConnection(ctx context.Context, connectionID string) error {
	query := `DELETE FROM m365_connections WHERE id = $1`

	cmdTag, err := s.db.Exec(ctx, query, connectionID)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("connection not found")
	}

	return nil
}

// Helper functions

func extractIDFromPath(path, prefix string) string {
	if len(path) > len(prefix) {
		return path[len(prefix):]
	}
	return ""
}

// extractBearerToken extracts the M365 access token from the request.
// It checks for an X-M365-Access-Token header first, then the access_token query parameter.
// (The standard Authorization header is used for the service's own JWT auth.)
func extractBearerToken(r *http.Request) string {
	if token := r.Header.Get("X-M365-Access-Token"); token != "" {
		return token
	}
	if token := r.URL.Query().Get("access_token"); token != "" {
		return token
	}
	return ""
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// downloadFile downloads a file from a URL using the provided access token for authentication.
func downloadFile(fileURL, accessToken string) ([]byte, error) {
	if fileURL == "" {
		return nil, fmt.Errorf("download URL is empty")
	}

	client := &http.Client{Timeout: 120 * time.Second}

	req, err := http.NewRequest(http.MethodGet, fileURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create download request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("download failed with HTTP %d: %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read download response: %w", err)
	}

	return data, nil
}

// refreshToken refreshes an expired access token using the Microsoft identity platform token endpoint.
func (s *Service) refreshToken(refreshTokenValue string) (string, string, time.Time, error) {
	if !s.hasM365Credentials() {
		return "", "", time.Time{}, fmt.Errorf("Microsoft 365 credentials are not configured: M365_CLIENT_ID and M365_CLIENT_SECRET are required")
	}

	if refreshTokenValue == "" {
		return "", "", time.Time{}, fmt.Errorf("refresh token is empty")
	}

	formData := url.Values{}
	formData.Set("client_id", s.config.M365ClientID)
	formData.Set("client_secret", s.config.M365ClientSecret)
	formData.Set("refresh_token", refreshTokenValue)
	formData.Set("grant_type", "refresh_token")

	req, err := http.NewRequest(http.MethodPost, s.tokenEndpointURL(), strings.NewReader(formData.Encode()))
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("failed to create refresh token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("refresh token request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("failed to read refresh token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var tokenErr tokenErrorResponse
		if jsonErr := json.Unmarshal(respBody, &tokenErr); jsonErr == nil && tokenErr.Error != "" {
			return "", "", time.Time{}, fmt.Errorf("token refresh failed: %s - %s", tokenErr.Error, tokenErr.ErrorDescription)
		}
		return "", "", time.Time{}, fmt.Errorf("token refresh failed with HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var tokenResp tokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return "", "", time.Time{}, fmt.Errorf("failed to parse refresh token response: %w", err)
	}

	expiry := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	return tokenResp.AccessToken, tokenResp.RefreshToken, expiry, nil
}

// getDocumentFromOneDrive retrieves a document's metadata from OneDrive via the Microsoft Graph API.
func (s *Service) getDocumentFromOneDrive(ctx context.Context, accessToken, documentID string) (*M365Document, error) {
	if accessToken == "" {
		return nil, fmt.Errorf("access token is required")
	}
	if documentID == "" {
		return nil, fmt.Errorf("document ID is required")
	}

	graphPath := fmt.Sprintf("/me/drive/items/%s?$select=id,name,size,file,parentReference,createdDateTime,lastModifiedDateTime,webUrl,createdBy,@microsoft.graph.downloadUrl", url.PathEscape(documentID))

	respBody, err := s.graphAPIRequest(ctx, http.MethodGet, graphPath, accessToken, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get OneDrive document: %w", err)
	}

	doc, err := parseGraphDriveItem(respBody)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OneDrive document response: %w", err)
	}

	return doc, nil
}

// getDocumentFromSharePoint retrieves a document's metadata from a SharePoint site via the Microsoft Graph API.
func (s *Service) getDocumentFromSharePoint(ctx context.Context, accessToken, siteID, documentID string) (*M365Document, error) {
	if accessToken == "" {
		return nil, fmt.Errorf("access token is required")
	}
	if siteID == "" {
		return nil, fmt.Errorf("site ID is required")
	}
	if documentID == "" {
		return nil, fmt.Errorf("document ID is required")
	}

	graphPath := fmt.Sprintf("/sites/%s/drive/items/%s?$select=id,name,size,file,parentReference,createdDateTime,lastModifiedDateTime,webUrl,createdBy,@microsoft.graph.downloadUrl", url.PathEscape(siteID), url.PathEscape(documentID))

	respBody, err := s.graphAPIRequest(ctx, http.MethodGet, graphPath, accessToken, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get SharePoint document: %w", err)
	}

	doc, err := parseGraphDriveItem(respBody)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SharePoint document response: %w", err)
	}

	return doc, nil
}

// listSharePointDrives lists drives (document libraries) in a SharePoint site via the Microsoft Graph API.
func (s *Service) listSharePointDrives(ctx context.Context, accessToken, siteID string) ([]OneDriveDrive, error) {
	if accessToken == "" {
		return nil, fmt.Errorf("access token is required")
	}
	if siteID == "" {
		return nil, fmt.Errorf("site ID is required")
	}

	graphPath := fmt.Sprintf("/sites/%s/drives?$select=id,name,driveType,owner", url.PathEscape(siteID))

	respBody, err := s.graphAPIRequest(ctx, http.MethodGet, graphPath, accessToken, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list SharePoint drives: %w", err)
	}

	var graphResp struct {
		Value []json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(respBody, &graphResp); err != nil {
		return nil, fmt.Errorf("failed to parse drives response: %w", err)
	}

	drives := make([]OneDriveDrive, 0, len(graphResp.Value))
	for _, raw := range graphResp.Value {
		var item struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			DriveType string `json:"driveType"`
			Owner     struct {
				User struct {
					ID          string `json:"id"`
					DisplayName string `json:"displayName"`
				} `json:"user"`
			} `json:"owner"`
		}
		if err := json.Unmarshal(raw, &item); err != nil {
			log.Printf("Skipping unparseable drive: %v", err)
			continue
		}
		drives = append(drives, OneDriveDrive{
			ID:        item.ID,
			Name:      item.Name,
			DriveType: item.DriveType,
			OwnerID:   item.Owner.User.ID,
			OwnerName: item.Owner.User.DisplayName,
		})
	}

	return drives, nil
}

// getUserInfo retrieves user information from Microsoft Graph API.
func (s *Service) getUserInfo(ctx context.Context, accessToken string) (map[string]interface{}, error) {
	if accessToken == "" {
		return nil, fmt.Errorf("access token is required")
	}

	respBody, err := s.graphAPIRequest(ctx, http.MethodGet, "/me?$select=id,displayName,mail,userPrincipalName", accessToken, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	var userInfo map[string]interface{}
	if err := json.Unmarshal(respBody, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to parse user info response: %w", err)
	}

	return userInfo, nil
}

// createPrintJob creates a print job by calling the job-service API.
func (s *Service) createPrintJob(ctx context.Context, source *PrintJobSource, localPath string) (string, error) {
	if s.config.JobServiceURL == "" {
		return "", fmt.Errorf("job service URL is not configured: set JOB_SERVICE_URL environment variable")
	}

	jobRequest := map[string]interface{}{
		"source_type":   "m365_" + source.SourceType,
		"document_id":   source.DocumentID,
		"document_path": localPath,
		"file_name":     source.FileName,
		"file_size":     source.FileSize,
		"user_id":       source.UserID,
		"user_email":    source.UserEmail,
		"metadata":      source.Metadata,
	}

	body, err := json.Marshal(jobRequest)
	if err != nil {
		return "", fmt.Errorf("failed to marshal print job request: %w", err)
	}

	jobURL := fmt.Sprintf("%s/api/v1/jobs", s.config.JobServiceURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, jobURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create print job request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to reach job-service: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read job-service response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("job-service returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var jobResp struct {
		ID    string `json:"id"`
		JobID string `json:"job_id"`
	}
	if err := json.Unmarshal(respBody, &jobResp); err != nil {
		return "", fmt.Errorf("failed to parse job-service response: %w", err)
	}

	jobID := jobResp.ID
	if jobID == "" {
		jobID = jobResp.JobID
	}
	if jobID == "" {
		return "", fmt.Errorf("job-service returned empty job ID")
	}

	return jobID, nil
}

// copyToStorage copies a downloaded file to permanent storage.
func (s *Service) copyToStorage(reader io.Reader, filename string) (string, int64, error) {
	// Sanitize filename to prevent path traversal
	sanitizedFilename := sanitizeFilename(filename)
	sanitizedFilename = filepath.Base(sanitizedFilename)

	if sanitizedFilename == "" || sanitizedFilename == "." || sanitizedFilename == ".." {
		return "", 0, fmt.Errorf("invalid filename")
	}

	storagePath := filepath.Join(s.config.StoragePath, sanitizedFilename)

	// Validate the path is within the allowed storage directory
	absPath, err := filepath.Abs(storagePath)
	if err != nil {
		return "", 0, fmt.Errorf("invalid path: %w", err)
	}

	absStoragePath, err := filepath.Abs(s.config.StoragePath)
	if err != nil {
		return "", 0, fmt.Errorf("storage path error: %w", err)
	}

	// Ensure the resulting path is within the storage directory
	if !strings.HasPrefix(absPath, absStoragePath+string(filepath.Separator)) && absPath != absStoragePath {
		return "", 0, fmt.Errorf("path traversal detected")
	}

	file, err := os.Create(storagePath)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	written, err := io.Copy(file, reader)
	if err != nil {
		return "", 0, err
	}

	return storagePath, written, nil
}

// parseGraphDriveItem parses a Microsoft Graph drive item JSON into an M365Document.
func parseGraphDriveItem(raw json.RawMessage) (*M365Document, error) {
	var item struct {
		ID               string `json:"id"`
		Name             string `json:"name"`
		Size             int64  `json:"size"`
		WebURL           string `json:"webUrl"`
		DownloadURL      string `json:"@microsoft.graph.downloadUrl"`
		CreatedDateTime  string `json:"createdDateTime"`
		ModifiedDateTime string `json:"lastModifiedDateTime"`
		File             *struct {
			MimeType string `json:"mimeType"`
		} `json:"file"`
		ParentReference *struct {
			DriveID string `json:"driveId"`
			Path    string `json:"path"`
		} `json:"parentReference"`
		CreatedBy *struct {
			User struct {
				DisplayName string `json:"displayName"`
			} `json:"user"`
		} `json:"createdBy"`
	}

	if err := json.Unmarshal(raw, &item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal drive item: %w", err)
	}

	doc := &M365Document{
		ID:          item.ID,
		Name:        item.Name,
		Size:        item.Size,
		WebURL:      item.WebURL,
		DownloadURL: item.DownloadURL,
		ItemID:      item.ID,
	}

	if item.File != nil {
		doc.MimeType = item.File.MimeType
	}

	if item.ParentReference != nil {
		doc.DriveID = item.ParentReference.DriveID
		doc.Path = item.ParentReference.Path
	}

	if item.CreatedBy != nil {
		doc.CreatedBy = item.CreatedBy.User.DisplayName
	}

	if t, err := time.Parse(time.RFC3339, item.CreatedDateTime); err == nil {
		doc.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, item.ModifiedDateTime); err == nil {
		doc.ModifiedAt = t
	}

	return doc, nil
}

// parseGraphSite parses a Microsoft Graph site JSON into a SharePointSite.
func parseGraphSite(raw json.RawMessage) (*SharePointSite, error) {
	var item struct {
		ID               string `json:"id"`
		DisplayName      string `json:"displayName"`
		WebURL           string `json:"webUrl"`
		Description      string `json:"description"`
		CreatedDateTime  string `json:"createdDateTime"`
		ModifiedDateTime string `json:"lastModifiedDateTime"`
	}

	if err := json.Unmarshal(raw, &item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal site: %w", err)
	}

	site := &SharePointSite{
		ID:          item.ID,
		Name:        item.DisplayName,
		URL:         item.WebURL,
		Description: item.Description,
		WebURL:      item.WebURL,
	}

	if t, err := time.Parse(time.RFC3339, item.CreatedDateTime); err == nil {
		site.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, item.ModifiedDateTime); err == nil {
		site.LastModified = t
	}

	return site, nil
}

// isValidM365URL validates if a URL is from a valid Microsoft 365 source.
func isValidM365URL(rawURL string) bool {
	validDomains := []string{
		"onedrive.live.com",
		"1drv.ms",
		"sharepoint.com",
	}

	for _, domain := range validDomains {
		if strings.Contains(rawURL, domain) {
			return true
		}
	}

	return false
}

// sanitizeFilename sanitizes a filename for safe storage.
func sanitizeFilename(name string) string {
	// Remove any characters that aren't safe for filenames
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	name = strings.ReplaceAll(name, ":", "-")
	name = strings.ReplaceAll(name, "*", "")
	name = strings.ReplaceAll(name, "?", "")
	name = strings.ReplaceAll(name, "\"", "")
	name = strings.ReplaceAll(name, "<", "")
	name = strings.ReplaceAll(name, ">", "")
	name = strings.ReplaceAll(name, "|", "")

	// Limit length
	if len(name) > 255 {
		name = name[:255]
	}

	return name
}
