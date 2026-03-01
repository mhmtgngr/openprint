// Package middleware provides agent-specific JWT authentication middleware.
package middleware

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// AgentIDKey is the context key for agent ID
	AgentIDKey contextKey = "agent_id"
	// AgentCertThumbprintKey is the context key for agent certificate thumbprint
	AgentCertThumbprintKey contextKey = "agent_cert_thumbprint"
	// AgentHostnameKey is the context key for agent hostname
	AgentHostnameKey contextKey = "agent_hostname"
)

// AgentCertificate represents an agent's X.509 certificate.
type AgentCertificate struct {
	Thumbprint string
	Subject    string
	NotBefore  time.Time
	NotAfter   time.Time
	IsRevoked  bool
}

// CertificateValidator validates agent certificates.
type CertificateValidator interface {
	Validate(thumbprint string) (*AgentCertificate, error)
	IsRevoked(thumbprint string) bool
}

// DatabaseCertificateValidator validates certificates against a database.
type DatabaseCertificateValidator struct {
	// In production, this would query a database
	// For now, we'll have a simple in-memory implementation
	revoked map[string]bool
	certs   map[string]*AgentCertificate
}

// NewDatabaseCertificateValidator creates a new certificate validator.
func NewDatabaseCertificateValidator() *DatabaseCertificateValidator {
	return &DatabaseCertificateValidator{
		revoked: make(map[string]bool),
		certs:   make(map[string]*AgentCertificate),
	}
}

// Validate validates a certificate thumbprint.
func (v *DatabaseCertificateValidator) Validate(thumbprint string) (*AgentCertificate, error) {
	cert, exists := v.certs[thumbprint]
	if !exists {
		return nil, errors.New("certificate not found")
	}

	if v.IsRevoked(thumbprint) {
		return nil, errors.New("certificate is revoked")
	}

	return cert, nil
}

// IsRevoked checks if a certificate is revoked.
func (v *DatabaseCertificateValidator) IsRevoked(thumbprint string) bool {
	return v.revoked[thumbprint]
}

// AddCertificate adds a certificate to the validator.
func (v *DatabaseCertificateValidator) AddCertificate(thumbprint string, cert *AgentCertificate) {
	v.certs[thumbprint] = cert
}

// Revoke revokes a certificate.
func (v *DatabaseCertificateValidator) Revoke(thumbprint string) {
	v.revoked[thumbprint] = true
}

// AgentJWTConfig holds agent JWT authentication configuration.
type AgentJWTConfig struct {
	SecretKey            string
	PublicKey            *rsa.PublicKey
	CertificateValidator CertificateValidator
	SkipPaths            []string
	TokenIssuer          string
	TokenAudience        string
}

// AgentAuthMiddleware creates agent JWT authentication middleware.
func AgentAuthMiddleware(cfg AgentJWTConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if path should be skipped
			for _, skipPath := range cfg.SkipPaths {
				if strings.HasPrefix(r.URL.Path, skipPath) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondAgentAuthError(w, "missing authorization header")
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				respondAgentAuthError(w, "invalid authorization header format")
				return
			}

			// Parse and validate token
			claims, err := validateAgentToken(tokenString, cfg)
			if err != nil {
				respondAgentAuthError(w, fmt.Sprintf("invalid token: %v", err))
				return
			}

			// Check certificate validity if validator is provided
			if cfg.CertificateValidator != nil && claims.CertificateThumbprint != "" {
				if cfg.CertificateValidator.IsRevoked(claims.CertificateThumbprint) {
					respondAgentAuthError(w, "agent certificate is revoked")
					return
				}
			}

			// Add agent info to context
			ctx := r.Context()
			ctx = context.WithValue(ctx, AgentIDKey, claims.AgentID)
			ctx = context.WithValue(ctx, AgentCertThumbprintKey, claims.CertificateThumbprint)
			ctx = context.WithValue(ctx, AgentHostnameKey, claims.Hostname)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AgentCertificateAuthMiddleware authenticates agents using X.509 certificates.
// This is used for mutual TLS authentication.
func AgentCertificateAuthMiddleware(validator CertificateValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get certificate from TLS connection
			if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
				respondAgentAuthError(w, "client certificate required")
				return
			}

			cert := r.TLS.PeerCertificates[0]

			// Calculate thumbprint
			thumbprint := calculateThumbprint(cert)

			// Validate certificate
			agentCert, err := validator.Validate(thumbprint)
			if err != nil {
				respondAgentAuthError(w, fmt.Sprintf("invalid certificate: %v", err))
				return
			}

			// Check if certificate is expired
			if time.Now().After(agentCert.NotAfter) {
				respondAgentAuthError(w, "certificate has expired")
				return
			}

			if time.Now().Before(agentCert.NotBefore) {
				respondAgentAuthError(w, "certificate is not yet valid")
				return
			}

			// Extract agent ID from certificate subject or serial number
			agentID := extractAgentIDFromCert(cert, agentCert)

			// Add agent info to context
			ctx := r.Context()
			ctx = context.WithValue(ctx, AgentIDKey, agentID)
			ctx = context.WithValue(ctx, AgentCertThumbprintKey, thumbprint)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CombinedAgentAuthMiddleware tries JWT first, then certificate authentication.
func CombinedAgentAuthMiddleware(jwtCfg AgentJWTConfig, certValidator CertificateValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Try JWT authentication first
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				tokenString := strings.TrimPrefix(authHeader, "Bearer ")
				claims, err := validateAgentToken(tokenString, jwtCfg)
				if err == nil {
					ctx = context.WithValue(ctx, AgentIDKey, claims.AgentID)
					ctx = context.WithValue(ctx, AgentCertThumbprintKey, claims.CertificateThumbprint)
					ctx = context.WithValue(ctx, AgentHostnameKey, claims.Hostname)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			// Try certificate authentication
			if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
				cert := r.TLS.PeerCertificates[0]
				thumbprint := calculateThumbprint(cert)

				agentCert, err := certValidator.Validate(thumbprint)
				if err == nil {
					agentID := extractAgentIDFromCert(cert, agentCert)
					ctx = context.WithValue(ctx, AgentIDKey, agentID)
					ctx = context.WithValue(ctx, AgentCertThumbprintKey, thumbprint)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			// Neither authentication method succeeded
			respondAgentAuthError(w, "authentication required")
		})
	}
}

// AgentClaims represents JWT claims for agents.
type AgentClaims struct {
	AgentID              string `json:"agent_id"`
	CertificateThumbprint string `json:"certificate_thumbprint,omitempty"`
	Hostname              string `json:"hostname,omitempty"`
	jwt.RegisteredClaims
}

// validateAgentToken validates an agent JWT token.
func validateAgentToken(tokenString string, cfg AgentJWTConfig) (*AgentClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AgentClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Explicitly verify the signing method to prevent algorithm confusion attacks.
		// Only RSA or HMAC signing methods are accepted.
		// This prevents the "none" algorithm attack.
		switch method := token.Method.(type) {
		case *jwt.SigningMethodRSA:
			// RSA signature - use public key for verification
			if cfg.PublicKey == nil {
				return nil, errors.New("RSA signing requires public key configuration")
			}
			return cfg.PublicKey, nil
		case *jwt.SigningMethodHMAC:
			// HMAC signature - use secret key
			if cfg.SecretKey == "" {
				return nil, errors.New("HMAC signing requires secret key configuration")
			}
			return []byte(cfg.SecretKey), nil
		default:
			// Reject any other signing method (including "none")
			return nil, fmt.Errorf("unexpected signing method: %v (only RSA and HMAC are supported)", method.Alg())
		}
	}, jwt.WithValidMethods([]string{"RS256", "HS256"}))

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*AgentClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	// Validate issuer and audience if configured
	if cfg.TokenIssuer != "" && claims.Issuer != cfg.TokenIssuer {
		return nil, errors.New("invalid token issuer")
	}

	if cfg.TokenAudience != "" {
		found := false
		for _, aud := range claims.Audience {
			if aud == cfg.TokenAudience {
				found = true
				break
			}
		}
		if !found {
			return nil, errors.New("invalid token audience")
		}
	}

	// Check expiration
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("token has expired")
	}

	return claims, nil
}

// calculateThumbprint calculates the SHA-256 thumbprint of a certificate.
func calculateThumbprint(cert *x509.Certificate) string {
	h := sha256.New()
	h.Write(cert.Raw)
	return fmt.Sprintf("%X", h.Sum(nil))
}

// extractAgentIDFromCert extracts the agent ID from a certificate.
func extractAgentIDFromCert(cert *x509.Certificate, agentCert *AgentCertificate) string {
	// Try to get agent ID from Subject CN
	if cert.Subject.CommonName != "" {
		return cert.Subject.CommonName
	}

	// Try Subject Alternative Name
	for _, name := range cert.DNSNames {
		if strings.HasPrefix(name, "agent-") {
			return name
		}
	}

	// Use subject from validator
	if agentCert.Subject != "" {
		return agentCert.Subject
	}

	// Fallback to serial number
	return fmt.Sprintf("%x", cert.SerialNumber)
}

// RequireAgentAuth wraps a handler that requires agent authentication.
func RequireAgentAuth(fn func(w http.ResponseWriter, r *http.Request, agentID string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agentID := GetAgentID(r)
		if agentID == "" {
			respondAgentAuthError(w, "unauthorized")
			return
		}

		fn(w, r, agentID)
	}
}

// GetAgentID extracts the agent ID from the request context.
func GetAgentID(r *http.Request) string {
	return GetStringFromContext(r.Context(), AgentIDKey)
}

// GetAgentCertThumbprint extracts the certificate thumbprint from the request context.
func GetAgentCertThumbprint(r *http.Request) string {
	return GetStringFromContext(r.Context(), AgentCertThumbprintKey)
}

// GetAgentHostname extracts the hostname from the request context.
func GetAgentHostname(r *http.Request) string {
	return GetStringFromContext(r.Context(), AgentHostnameKey)
}

// respondAgentAuthError sends an authentication error response for agent auth.
func respondAgentAuthError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    "AUTHENTICATION_ERROR",
		"message": message,
	})
}

// GenerateAgentToken generates a JWT token for an agent.
func GenerateAgentToken(agentID, hostname, thumbprint, secret string, expiry time.Duration) (string, error) {
	claims := AgentClaims{
		AgentID:              agentID,
		Hostname:             hostname,
		CertificateThumbprint: thumbprint,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			Issuer:    "openprint-server",
			Subject:   agentID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseCertificatePEM parses a PEM-encoded certificate.
func ParseCertificatePEM(pemData []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return cert, nil
}

// ParsePublicKeyPEM parses a PEM-encoded public key.
func ParsePublicKeyPEM(pemData []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not an RSA public key")
	}

	return rsaPub, nil
}

// ValidateCSR validates a Certificate Signing Request.
func ValidateCSR(csrData []byte) (*x509.CertificateRequest, error) {
	block, _ := pem.Decode(csrData)
	if block == nil {
		// Try to parse directly without PEM
	} else if block.Type == "CERTIFICATE REQUEST" || block.Type == "NEW CERTIFICATE REQUEST" {
		csrData = block.Bytes
	}

	csr, err := x509.ParseCertificateRequest(csrData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSR: %w", err)
	}

	if err := csr.CheckSignature(); err != nil {
		return nil, fmt.Errorf("invalid CSR signature: %w", err)
	}

	return csr, nil
}

// GenerateAgentCertificate generates a certificate for an agent (for development/testing).
// In production, this would use a proper CA.
func GenerateAgentCertificate(agentID, hostname string, validity time.Duration, caKey *rsa.PrivateKey, caCert *x509.Certificate) ([]byte, error) {
	// Generate serial number
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{CommonName: agentID, Organization: []string{"OpenPrint Agents"}},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(validity),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{hostname, agentID},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, caCert, &template.PublicKey, caKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	}), nil
}
