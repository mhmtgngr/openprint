// Package handler provides agent registration handlers.
package handler

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	stderrors "errors"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/openprint/openprint/internal/agent"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/internal/shared/middleware"
	registryrepository "github.com/openprint/openprint/services/registry-service/repository"
)

// AgentRepository defines the interface for agent repository operations.
type AgentRepository interface {
	Create(ctx context.Context, agent *registryrepository.Agent) error
	FindByID(ctx context.Context, id string) (*registryrepository.Agent, error)
	FindByHostname(ctx context.Context, hostname string) (*registryrepository.Agent, error)
	Update(ctx context.Context, agent *registryrepository.Agent) error
}

// AgentCertificateRepository defines the interface for agent certificate operations.
type AgentCertificateRepository interface {
	Store(ctx context.Context, cert *AgentCertificateRecord) error
	FindByThumbprint(ctx context.Context, thumbprint string) (*AgentCertificateRecord, error)
	FindByAgentID(ctx context.Context, agentID string) ([]*AgentCertificateRecord, error)
	Revoke(ctx context.Context, certID string, reason string) error
	IsRevoked(ctx context.Context, thumbprint string) (bool, error)
}

// EnrollmentTokenRepository defines the interface for enrollment token operations.
type EnrollmentTokenRepository interface {
	Validate(ctx context.Context, token, organizationID string) (bool, error)
	IncrementUseCount(ctx context.Context, tokenID string) error
	FindByToken(ctx context.Context, token string) (interface{}, error)
}

// AgentCertificateRecord represents an agent certificate in the database.
type AgentCertificateRecord struct {
	CertificateID    string
	AgentID          string
	SerialNumber     string
	Thumbprint       string
	Subject          string
	Issuer           string
	NotValidBefore   time.Time
	NotValidAfter    time.Time
	IsRevoked        bool
	RevokedAt        *time.Time
	RevocationReason string
	CreatedAt        time.Time
}

// AgentRegistrationConfig holds agent registration dependencies.
type AgentRegistrationConfig struct {
	AgentRepo           AgentRepository
	CertRepo            AgentCertificateRepository
	EnrollmentTokenRepo EnrollmentTokenRepository // For enrollment token validation
	CAPrivateKey        *rsa.PrivateKey
	CACertificate       *x509.Certificate
	TokenValidity       time.Duration
	CertValidity        time.Duration
	JWTSecret           string // JWT secret for token validation (loaded from config)
}

// AgentRegistrationHandler handles agent registration and certificate issuance.
type AgentRegistrationHandler struct {
	agentRepo           AgentRepository
	certRepo            AgentCertificateRepository
	enrollmentTokenRepo EnrollmentTokenRepository
	caPrivateKey        *rsa.PrivateKey
	caCertificate       *x509.Certificate
	tokenValidity       time.Duration
	certValidity        time.Duration
	jwtSecret           string // JWT secret from configuration
}

// NewAgentRegistrationHandler creates a new agent registration handler.
func NewAgentRegistrationHandler(cfg AgentRegistrationConfig) *AgentRegistrationHandler {
	// Validate JWT secret is provided
	if cfg.JWTSecret == "" {
		panic("JWTSecret is required in AgentRegistrationConfig")
	}
	return &AgentRegistrationHandler{
		agentRepo:           cfg.AgentRepo,
		certRepo:            cfg.CertRepo,
		enrollmentTokenRepo: cfg.EnrollmentTokenRepo,
		caPrivateKey:        cfg.CAPrivateKey,
		caCertificate:       cfg.CACertificate,
		tokenValidity:       cfg.TokenValidity,
		certValidity:        cfg.CertValidity,
		jwtSecret:           cfg.JWTSecret,
	}
}

// RegisterAgent handles new agent registration requests.
// POST /agents/register
func (h *AgentRegistrationHandler) RegisterAgent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req agent.AgentRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondAgentError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate required fields
	if req.Name == "" || req.Hostname == "" || req.OS == "" {
		respondAgentError(w, apperrors.New("name, hostname, and os are required", http.StatusBadRequest))
		return
	}

	// Validate enrollment token if provided
	if req.EnrollmentToken != "" {
		valid, err := h.validateEnrollmentToken(ctx, req.EnrollmentToken, req.OrganizationID)
		if err != nil {
			respondAgentError(w, apperrors.Wrap(err, "failed to validate enrollment token", http.StatusInternalServerError))
			return
		}
		if !valid {
			respondAgentError(w, apperrors.New("invalid enrollment token", http.StatusUnauthorized))
			return
		}
	}

	// Check if agent already exists by hostname
	existingAgent, _ := h.agentRepo.FindByHostname(ctx, req.Hostname)

	var agentID string
	var isNewAgent bool

	if existingAgent != nil {
		// Re-register existing agent
		agentID = existingAgent.ID
		isNewAgent = false

		// Update agent info
		existingAgent.Name = req.Name
		existingAgent.Version = req.Version
		existingAgent.OS = req.OS
		existingAgent.Architecture = req.Architecture
		existingAgent.Status = "online"
		existingAgent.UpdatedAt = time.Now()

		if err := h.agentRepo.Update(ctx, existingAgent); err != nil {
			respondAgentError(w, apperrors.Wrap(err, "failed to update agent", http.StatusInternalServerError))
			return
		}
	} else {
		// Create new agent
		agentID = uuid.New().String()
		isNewAgent = true

		newAgent := &registryrepository.Agent{
			ID:           agentID,
			Name:         req.Name,
			Version:      req.Version,
			OS:           req.OS,
			Architecture: req.Architecture,
			Hostname:     req.Hostname,
			Status:       "online",
			UpdatedAt:    time.Now(),
		}

		// Set OrganizationID if provided
		newAgent.OrganizationID = ""

		if err := h.agentRepo.Create(ctx, newAgent); err != nil {
			respondAgentError(w, apperrors.Wrap(err, "failed to register agent", http.StatusInternalServerError))
			return
		}
	}

	// Handle certificate issuance
	var certificate []byte
	var certificateChain [][]byte
	var err error

	if len(req.CSR) > 0 {
		// Process CSR and issue certificate
		certificate, certificateChain, err = h.issueCertificateFromCSR(ctx, agentID, req.CSR, req.Hostname)
		if err != nil {
			respondAgentError(w, apperrors.Wrap(err, "failed to issue certificate", http.StatusInternalServerError))
			return
		}
	} else if isNewAgent {
		// Generate a self-signed certificate for new agents without CSR
		certificate, certificateChain, err = h.generateSelfSignedCertificate(agentID, req.Hostname)
		if err != nil {
			respondAgentError(w, apperrors.Wrap(err, "failed to generate certificate", http.StatusInternalServerError))
			return
		}
	}

	// Get public key for JWT signing (in production, use proper key management)
	serverPublicKey := h.getServerPublicKey()

	// Build response
	response := agent.AgentRegistrationResponse{
		AgentID:          agentID,
		Certificate:      certificate,
		CertificateChain: certificateChain,
		ServerPublicKey:  serverPublicKey,
		Config: map[string]interface{}{
			"heartbeat_interval_seconds": 30,
			"job_poll_interval_seconds":  10,
			"max_retry_count":            3,
			"supported_document_types":   []string{"pdf", "txt", "doc", "docx", "xls", "xlsx"},
		},
		HeartbeatInterval: 30,
	}

	respondAgentJSON(w, http.StatusCreated, response)
}

// ValidateToken validates an agent's JWT token.
// POST /agents/validate
func (h *AgentRegistrationHandler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondAgentError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	if req.Token == "" {
		respondAgentError(w, apperrors.New("token is required", http.StatusBadRequest))
		return
	}

	// Validate token
	secret := h.getSecret()
	claims, err := validateAgentTokenString(req.Token, secret)
	if err != nil {
		respondAgentError(w, apperrors.Wrap(err, "invalid token", http.StatusUnauthorized))
		return
	}

	// Check if agent exists
	agentInfo, err := h.agentRepo.FindByID(ctx, claims.AgentID)
	if err != nil {
		respondAgentError(w, apperrors.New("agent not found", http.StatusNotFound))
		return
	}

	// Check if certificate is revoked
	if claims.CertificateThumbprint != "" {
		revoked, _ := h.certRepo.IsRevoked(ctx, claims.CertificateThumbprint)
		if revoked {
			respondAgentError(w, apperrors.New("certificate is revoked", http.StatusUnauthorized))
			return
		}
	}

	respondAgentJSON(w, http.StatusOK, map[string]interface{}{
		"valid":      true,
		"agent_id":   agentInfo.ID,
		"hostname":   agentInfo.Hostname,
		"status":     agentInfo.Status,
		"expires_at": claims.ExpiresAt,
	})
}

// RenewCertificate handles certificate renewal requests.
// POST /agents/{agent_id}/certificates/renew
func (h *AgentRegistrationHandler) RenewCertificate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract agent ID from path
	agentID := extractIDFromPath(r.URL.Path, "agents", "certificates", "renew")
	if agentID == "" {
		respondAgentError(w, apperrors.New("agent ID is required", http.StatusBadRequest))
		return
	}

	var req struct {
		CSR []byte `json:"csr"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondAgentError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Verify agent exists
	agent, err := h.agentRepo.FindByID(ctx, agentID)
	if err != nil {
		respondAgentError(w, apperrors.ErrNotFound)
		return
	}

	var certificate []byte
	var certificateChain [][]byte

	if len(req.CSR) > 0 {
		certificate, certificateChain, err = h.issueCertificateFromCSR(ctx, agentID, req.CSR, agent.Hostname)
	} else {
		certificate, certificateChain, err = h.generateSelfSignedCertificate(agentID, agent.Hostname)
	}

	if err != nil {
		respondAgentError(w, apperrors.Wrap(err, "failed to issue certificate", http.StatusInternalServerError))
		return
	}

	respondAgentJSON(w, http.StatusOK, map[string]interface{}{
		"certificate":       certificate,
		"certificate_chain": certificateChain,
	})
}

// RevokeCertificate revokes an agent's certificate.
// POST /agents/{agent_id}/certificates/revoke
func (h *AgentRegistrationHandler) RevokeCertificate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agentID := extractIDFromPath(r.URL.Path, "agents", "certificates", "revoke")
	if agentID == "" {
		respondAgentError(w, apperrors.New("agent ID is required", http.StatusBadRequest))
		return
	}

	var req struct {
		Thumbprint string `json:"thumbprint"`
		Reason     string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondAgentError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	if req.Thumbprint == "" {
		respondError(w, apperrors.New("thumbprint is required", http.StatusBadRequest))
		return
	}

	// Find certificate by thumbprint
	cert, err := h.certRepo.FindByThumbprint(ctx, req.Thumbprint)
	if err != nil {
		respondAgentError(w, apperrors.ErrNotFound)
		return
	}

	// Verify certificate belongs to agent
	if cert.AgentID != agentID {
		respondError(w, apperrors.New("certificate does not belong to this agent", http.StatusForbidden))
		return
	}

	if err := h.certRepo.Revoke(ctx, cert.CertificateID, req.Reason); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to revoke certificate", http.StatusInternalServerError))
		return
	}

	respondAgentJSON(w, http.StatusOK, map[string]interface{}{
		"thumbprint": req.Thumbprint,
		"revoked_at": time.Now().Format(time.RFC3339),
	})
}

// GetCertificates returns all certificates for an agent.
// GET /agents/{agent_id}/certificates
func (h *AgentRegistrationHandler) GetCertificates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agentID := extractIDFromPath(r.URL.Path, "agents", "certificates", "")
	if agentID == "" {
		respondAgentError(w, apperrors.New("agent ID is required", http.StatusBadRequest))
		return
	}

	certificates, err := h.certRepo.FindByAgentID(ctx, agentID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get certificates", http.StatusInternalServerError))
		return
	}

	response := make([]map[string]interface{}, len(certificates))
	for i, cert := range certificates {
		response[i] = map[string]interface{}{
			"certificate_id":   cert.CertificateID,
			"serial_number":    cert.SerialNumber,
			"thumbprint":       cert.Thumbprint,
			"subject":          cert.Subject,
			"issuer":           cert.Issuer,
			"not_valid_before": cert.NotValidBefore.Format(time.RFC3339),
			"not_valid_after":  cert.NotValidAfter.Format(time.RFC3339),
			"is_revoked":       cert.IsRevoked,
			"revoked_at":       cert.RevokedAt,
		}
	}

	respondAgentJSON(w, http.StatusOK, map[string]interface{}{
		"certificates": response,
		"count":        len(response),
	})
}

// Helper methods

func (h *AgentRegistrationHandler) validateEnrollmentToken(ctx context.Context, token, orgID string) (bool, error) {
	// If no enrollment token repository is configured, deny all tokens
	// (secure by default)
	if h.enrollmentTokenRepo == nil {
		return false, nil
	}

	// Validate token against database
	valid, err := h.enrollmentTokenRepo.Validate(ctx, token, orgID)
	if err != nil {
		return false, fmt.Errorf("validate enrollment token: %w", err)
	}

	return valid, nil
}

func (h *AgentRegistrationHandler) issueCertificateFromCSR(ctx context.Context, agentID string, csrData []byte, hostname string) ([]byte, [][]byte, error) {
	// Parse CSR
	csr, err := middleware.ValidateCSR(csrData)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid CSR: %w", err)
	}

	// Create certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               csr.Subject,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(h.certValidity),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{hostname},
	}

	// Sign certificate with CA
	certDER, err := x509.CreateCertificate(rand.Reader, template, h.caCertificate, csr.PublicKey, h.caPrivateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Encode certificate to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// Parse issued certificate to get thumbprint
	parsedCert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse issued certificate: %w", err)
	}

	thumbprint := calculateThumbprint(parsedCert)

	// Store certificate record
	certRecord := &AgentCertificateRecord{
		CertificateID:  uuid.New().String(),
		AgentID:        agentID,
		SerialNumber:   serialNumber.String(),
		Thumbprint:     thumbprint,
		Subject:        parsedCert.Subject.String(),
		Issuer:         parsedCert.Issuer.String(),
		NotValidBefore: parsedCert.NotBefore,
		NotValidAfter:  parsedCert.NotAfter,
		IsRevoked:      false,
		CreatedAt:      time.Now(),
	}

	if err := h.certRepo.Store(ctx, certRecord); err != nil {
		// Log but don't fail - certificate is still valid
		fmt.Printf("Warning: failed to store certificate record: %v", err)
	}

	// Build certificate chain
	caPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: h.caCertificate.Raw,
	})

	certificateChain := [][]byte{certPEM, caPEM}

	return certPEM, certificateChain, nil
}

func (h *AgentRegistrationHandler) generateSelfSignedCertificate(agentID, hostname string) ([]byte, [][]byte, error) {
	// Generate private key for agent
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   agentID,
			Organization: []string{"OpenPrint Agents"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(h.certValidity),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{hostname, agentID},
	}

	// Sign with CA (or self-sign if no CA)
	var parent *x509.Certificate
	var signingKey *rsa.PrivateKey

	if h.caCertificate != nil && h.caPrivateKey != nil {
		parent = h.caCertificate
		signingKey = h.caPrivateKey
	} else {
		parent = template
		signingKey = privateKey
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, parent, &privateKey.PublicKey, signingKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Encode certificate to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// Parse issued certificate
	parsedCert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse issued certificate: %w", err)
	}

	thumbprint := calculateThumbprint(parsedCert)

	// Store certificate record in production via certRepo.Store
	// For now, we just log it
	fmt.Printf("Generated certificate for agent %s: thumbprint=%s\n", agentID, thumbprint)

	return certPEM, [][]byte{certPEM}, nil
}

func (h *AgentRegistrationHandler) getServerPublicKey() []byte {
	if h.caCertificate != nil {
		return pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: h.caCertificate.Raw,
		})
	}

	// Return a placeholder
	return []byte("-----BEGIN PUBLIC KEY-----\nPLACEHOLDER\n-----END PUBLIC KEY-----")
}

func (h *AgentRegistrationHandler) getSecret() string {
	// Return the configured JWT secret
	return h.jwtSecret
}

func calculateThumbprint(cert *x509.Certificate) string {
	h := sha256.New()
	h.Write(cert.Raw)
	return hex.EncodeToString(h.Sum(nil))
}

// Helper functions

// validateAgentTokenString validates an agent JWT token string and returns the claims.
// SECURITY: Explicitly restricts to HS256 algorithm to prevent algorithm confusion attacks.
// The 'none' algorithm and other unexpected signing methods are rejected.
func validateAgentTokenString(tokenString, secret string) (*middleware.AgentClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &middleware.AgentClaims{}, func(token *jwt.Token) (interface{}, error) {
		// SECURITY: Validate signing method - MUST be HMAC
		// This check prevents algorithm confusion attacks where an attacker
		// could attempt to use the 'none' algorithm or a different algorithm.
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v (only HMAC is allowed)", token.Header["alg"])
		}
		return []byte(secret), nil
		// SECURITY: Explicit algorithm whitelist - only HS256 is accepted
		// This prevents the 'none' algorithm attack and other algorithm confusion attacks
	}, jwt.WithValidMethods([]string{"HS256"}))

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*middleware.AgentClaims)
	if !ok || !token.Valid {
		return nil, stderrors.New("invalid token claims")
	}

	// Check expiration
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return nil, stderrors.New("token has expired")
	}

	return claims, nil
}

func extractIDFromPath(path, resource, subresource, action string) string {
	parts := splitPath(path)
	for i, part := range parts {
		if part == resource && i+1 < len(parts) {
			if subresource == "" || (i+2 < len(parts) && parts[i+2] == subresource) {
				if action == "" || (i+3 < len(parts) && parts[i+3] == action) {
					return parts[i+1]
				}
			}
		}
	}
	return ""
}

func splitPath(path string) []string {
	path = trimPath(path)
	if path == "" {
		return []string{}
	}
	parts := make([]string, 0)
	current := ""
	for _, ch := range path {
		if ch == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func trimPath(path string) string {
	for len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}
	for len(path) > 0 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}
	return path
}

func respondAgentJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondAgentError(w http.ResponseWriter, err error) {
	var appErr *apperrors.AppError
	if stderrors.As(err, &appErr) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(appErr.StatusCode)
		json.NewEncoder(w).Encode(apperrors.ToJSON(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(map[string]string{
		"code":    "INTERNAL_ERROR",
		"message": "An internal error occurred",
	})
}
