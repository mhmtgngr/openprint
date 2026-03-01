// Package main is the entry point for the OpenPrint Microsoft 365 Integration Service.
// This service handles Microsoft 365 integration for OneDrive, SharePoint, and document handling.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
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
}

// Service provides Microsoft 365 integration functionality.
type Service struct {
	db              *pgxpool.Pool
	config          *Config
	graphAPIBaseURL string
}

// M365Document represents a document from Microsoft 365.
type M365Document struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	DriveID         string                 `json:"drive_id"`
	ItemID          string                 `json:"item_id"`
	DownloadURL     string                 `json:"download_url"`
	Size            int64                  `json:"size"`
	MimeType        string                 `json:"mime_type"`
	CreatedBy       string                 `json:"created_by"`
	CreatedAt       time.Time              `json:"created_at"`
	ModifiedAt      time.Time              `json:"modified_at"`
	SharePointURL   string                 `json:"sharepoint_url,omitempty"`
	OneDriveURL     string                 `json:"onedrive_url,omitempty"`
	Path            string                 `json:"path"`
	WebURL          string                 `json:"web_url"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// PrintJobSource represents the source of a print job from Microsoft 365.
type PrintJobSource struct {
	SourceID    string                 `json:"source_id"`
	SourceType  string                 `json:"source_type"` // onedrive, sharepoint, outlook
	DocumentID  string                 `json:"document_id"`
	DocumentURL string                 `json:"document_url"`
	UserID      string                 `json:"user_id"`
	UserEmail   string                 `json:"user_email"`
	FileName    string                 `json:"file_name"`
	FileSize    int64                  `json:"file_size"`
	AccessToken string                 `json:"access_token"`
	RefreshToken string                 `json:"refresh_token"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
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
	JobID         string                 `json:"job_id"`
	SourceID      string                 `json:"source_id"`
	SourceType    string                 `json:"source_type"`
	DocumentID    string                 `json:"document_id"`
	DocumentName  string                 `json:"document_name"`
	OriginalURL   string                 `json:"original_url"`
	DownloadedAt  time.Time              `json:"downloaded_at"`
	DownloadedSize int64                 `json:"downloaded_size"`
	StoredPath    string                 `json:"stored_path"`
	HashCode      string                 `json:"hash_code"`
	Properties    map[string]interface{} `json:"properties,omitempty"`
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
		db:              db,
		config:          cfg,
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

	var req struct {
		RedirectURI string `json:"redirect_uri"`
		State       string `json:"state"`
		Scopes      string `json:"scopes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	scopes := req.Scopes
	if scopes == "" {
		scopes = "User.Read Files.Read.All Sites.Read.All offline_access"
	}

	// Build authorization URL
	authURL := fmt.Sprintf(
		"https://login.microsoftonline.com/%s/oauth2/v2.0/authorize",
		s.config.M365TenantID,
	)
	authURL += fmt.Sprintf("?client_id=%s", s.config.M365ClientID)
	authURL += "&response_type=code"
	authURL += "&redirect_uri=" + req.RedirectURI
	authURL += "&scope=" + scopes
	authURL += "&state=" + req.State

	respondJSON(w, http.StatusOK, map[string]string{
		"authorization_url": authURL,
	})
}

// callbackHandler handles the OAuth callback.
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

	// Exchange code for tokens (in production, this would make a real API call)
	// For E2E testing, we'll return mock tokens

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"state":         state,
		"access_token":  "mock_access_token_" + uuid.New().String(),
		"refresh_token": "mock_refresh_token_" + uuid.New().String(),
		"expires_in":    3600,
	})
}

// connectionsHandler manages user connections to Microsoft 365.
func (s *Service) connectionsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		// List connections
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"connections": []M365Connection{},
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

	switch r.Method {
	case http.MethodGet:
		// Get connection
		connection, err := s.getConnection(ctx, connectionID)
		if err != nil {
			http.Error(w, "connection not found", http.StatusNotFound)
			return
		}
		respondJSON(w, http.StatusOK, connection)

	case http.MethodDelete:
		// Delete connection
		if err := s.deleteConnection(ctx, connectionID); err != nil {
			http.Error(w, "failed to delete connection", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// oneDriveFilesHandler lists files from OneDrive.
func (s *Service) oneDriveFilesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// In production, this would call the Microsoft Graph API
	// For E2E testing, return mock data

	files := []M365Document{
		{
			ID:         "file-1",
			Name:       "Document1.pdf",
			DriveID:    "drive-1",
			ItemID:     "item-1",
			Size:       1024000,
			MimeType:   "application/pdf",
			CreatedAt:  time.Now().Add(-24 * time.Hour),
			ModifiedAt: time.Now().Add(-1 * time.Hour),
			Path:       "/Documents/Document1.pdf",
			WebURL:     "https://onedrive.live.com/?id=file-1",
		},
		{
			ID:         "file-2",
			Name:       "Presentation.pptx",
			DriveID:    "drive-1",
			ItemID:     "item-2",
			Size:       5120000,
			MimeType:   "application/vnd.openxmlformats-officedocument.presentationml.presentation",
			CreatedAt:  time.Now().Add(-48 * time.Hour),
			ModifiedAt: time.Now().Add(-2 * time.Hour),
			Path:       "/Documents/Presentation.pptx",
			WebURL:     "https://onedrive.live.com/?id=file-2",
		},
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"files": files,
		"total": len(files),
	})
}

// oneDriveDownloadHandler downloads a file from OneDrive.
func (s *Service) oneDriveDownloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
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

	// In production, this would download from Microsoft Graph API
	// For E2E testing, return a mock response

	localPath := filepath.Join(s.config.StoragePath, req.DocumentID+".pdf")

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"document_id": req.DocumentID,
		"local_path":  localPath,
		"size":        1024000,
		"status":      "downloaded",
	})
}

// sharePointSitesHandler lists SharePoint sites.
func (s *Service) sharePointSitesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sites := []SharePointSite{
		{
			ID:          "site-1",
			Name:        "Marketing",
			URL:         "https://contoso.sharepoint.com/sites/marketing",
			Description: "Marketing team site",
			CreatedAt:   time.Now().Add(-365 * 24 * time.Hour),
			LastModified: time.Now().Add(-1 * 24 * time.Hour),
			WebURL:      "https://contoso.sharepoint.com/sites/marketing",
		},
		{
			ID:          "site-2",
			Name:        "Documents",
			URL:         "https://contoso.sharepoint.com/sites/documents",
			Description: "Company documents",
			CreatedAt:   time.Now().Add(-730 * 24 * time.Hour),
			LastModified: time.Now().Add(-2 * 24 * time.Hour),
			WebURL:      "https://contoso.sharepoint.com/sites/documents",
		},
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"sites": sites,
		"total": len(sites),
	})
}

// sharePointFilesHandler lists files from a SharePoint site.
func (s *Service) sharePointFilesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	siteID := r.URL.Query().Get("site_id")

	files := []M365Document{
		{
			ID:           "sp-file-1",
			Name:         "Brochure.pdf",
			DriveID:      "drive-sp-1",
			ItemID:       "item-sp-1",
			Size:         2048000,
			MimeType:     "application/pdf",
			SharePointURL: "https://contoso.sharepoint.com/sites/marketing/Shared%20Documents/Brochure.pdf",
			CreatedAt:    time.Now().Add(-7 * 24 * time.Hour),
			ModifiedAt:   time.Now().Add(-1 * 24 * time.Hour),
			Path:         "/Shared Documents/Brochure.pdf",
			WebURL:       "https://contoso.sharepoint.com/:b:/g/sites/marketing/Shared%20Documents/Brochure.pdf",
		},
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"site_id": siteID,
		"files":   files,
		"total":   len(files),
	})
}

// printSubmitHandler submits a print job from a Microsoft 365 source.
func (s *Service) printSubmitHandler(w http.ResponseWriter, r *http.Request) {
	_ = r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PrintJobSource
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Generate job ID
	jobID := uuid.New().String()

	// In production, this would:
	// 1. Download the document from Microsoft 365
	// 2. Store it locally or in S3
	// 3. Create a print job in the job-service
	// 4. Return the job ID

	metadata := &PrintJobM365Metadata{
		JobID:        jobID,
		SourceID:     req.SourceID,
		SourceType:   req.SourceType,
		DocumentID:   req.DocumentID,
		DocumentName: req.FileName,
		OriginalURL:  req.DocumentURL,
		DownloadedAt: time.Now(),
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"job_id":     jobID,
		"status":     "pending",
		"created_at": time.Now().Format(time.RFC3339),
		"metadata":   metadata,
	})
}

// printStatusHandler returns the status of a Microsoft 365 print job.
func (s *Service) printStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	jobID := extractIDFromPath(r.URL.Path, "/api/v1/m365/print/status/")

	// In production, query the actual job status

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"job_id":      jobID,
		"status":      "completed",
		"created_at":  time.Now().Add(-5 * time.Minute).Format(time.RFC3339),
		"updated_at":  time.Now().Add(-1 * time.Minute).Format(time.RFC3339),
		"pages":       10,
		"color":       true,
		"duplex":      true,
	})
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

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// downloadFile downloads a file from a URL.
func downloadFile(url, accessToken string) ([]byte, error) {
	// In production, this would make an HTTP request to download the file
	// with proper OAuth authentication
	return nil, nil
}

// refreshToken refreshes an expired access token.
func (s *Service) refreshToken(refreshToken string) (string, string, time.Time, error) {
	// In production, this would call the Microsoft token endpoint
	// to refresh the access token
	return "", "", time.Time{}, nil
}

// getDocumentFromOneDrive retrieves a document from OneDrive.
func (s *Service) getDocumentFromOneDrive(ctx context.Context, accessToken, documentID string) (*M365Document, error) {
	// In production, this would call the Microsoft Graph API
	return nil, nil
}

// getDocumentFromSharePoint retrieves a document from SharePoint.
func (s *Service) getDocumentFromSharePoint(ctx context.Context, accessToken, siteID, documentID string) (*M365Document, error) {
	// In production, this would call the Microsoft Graph API
	return nil, nil
}

// listSharePointDrives lists drives (document libraries) in a SharePoint site.
func (s *Service) listSharePointDrives(ctx context.Context, accessToken, siteID string) ([]OneDriveDrive, error) {
	// In production, this would call the Microsoft Graph API
	return nil, nil
}

// getUserInfo retrieves user information from Microsoft Graph.
func (s *Service) getUserInfo(ctx context.Context, accessToken string) (map[string]interface{}, error) {
	// In production, this would call the Microsoft Graph API
	return nil, nil
}

// createPrintJob creates a print job in the job service.
func (s *Service) createPrintJob(ctx context.Context, source *PrintJobSource, localPath string) (string, error) {
	// In production, this would call the job-service API
	return uuid.New().String(), nil
}

// copyToStorage copies a downloaded file to permanent storage.
func (s *Service) copyToStorage(reader io.Reader, filename string) (string, int64, error) {
	storagePath := filepath.Join(s.config.StoragePath, filename)
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

// isValidM365URL validates if a URL is from a valid Microsoft 365 source.
func isValidM365URL(url string) bool {
	validDomains := []string{
		"onedrive.live.com",
		"1drv.ms",
		"sharepoint.com",
	}

	for _, domain := range validDomains {
		if strings.Contains(url, domain) {
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
