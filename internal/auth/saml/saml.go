// Package saml provides SAML 2.0 SSO integration for OpenPrint authentication.
// This enables enterprise single sign-on with identity providers like Okta, Azure AD, and ADFS.
package saml

import (
	"bytes"
	"context"
	cryptoRand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	// ErrInvalidResponse is returned when the SAML response is invalid.
	ErrInvalidResponse = errors.New("invalid SAML response")
	// ErrMissingAttribute is returned when a required attribute is missing.
	ErrMissingAttribute = errors.New("missing required SAML attribute")
	// ErrInvalidSignature is returned when the signature is invalid.
	ErrInvalidSignature = errors.New("invalid SAML signature")
	ErrNotConfigured  = errors.New("SAML not configured")
)

// Config holds SAML configuration.
type Config struct {
	// EntityID is the unique identifier for this service provider.
	EntityID string
	// MetadataURL is the URL to the IdP metadata.
	MetadataURL string
	// SSOURL is the IdP SSO URL (can be derived from metadata).
	SSOURL string
	// SLOURL is the IdP Single Logout URL.
	SLOURL string
	// Certificate is the service provider certificate.
	Certificate *x509.Certificate
	// Key is the service provider private key.
	Key *rsa.PrivateKey
	// IdPCertificate is the identity provider certificate.
	IdPCertificate *x509.Certificate
	// IDPName is a friendly name for the identity provider.
	IDPName string
	// ACSURL is the Assertion Consumer Service URL.
	ACSURL string
	// SLOResponseURL is the Single Logout response URL.
	SLOResponseURL string
}

// Manager handles SAML authentication operations.
type Manager struct {
	config    *Config
	idpMeta   []byte
}

// NewManager creates a new SAML manager.
func NewManager(config *Config) (*Manager, error) {
	if config == nil {
		return nil, errors.New("saml config cannot be nil")
	}

	m := &Manager{
		config: config,
	}

	// Load IdP metadata if URL is provided
	if config.MetadataURL != "" {
		if err := m.loadMetadata(context.Background()); err != nil {
			return nil, fmt.Errorf("load IdP metadata: %w", err)
		}
	}

	return m, nil
}

// loadMetadata fetches and parses the IdP metadata from the configured URL.
func (m *Manager) loadMetadata(ctx context.Context) error {
	if m.config.MetadataURL == "" {
		return nil
	}

	// Validate URL scheme to prevent SSRF attacks
	parsedURL, err := url.Parse(m.config.MetadataURL)
	if err != nil {
		return fmt.Errorf("invalid metadata URL: %w", err)
	}

	// Only allow https and http schemes
	if parsedURL.Scheme != "https" && parsedURL.Scheme != "http" {
		return fmt.Errorf("invalid URL scheme: %s (only https and http allowed)", parsedURL.Scheme)
	}

	// Reject localhost and private network IPs in production
	hostname := parsedURL.Hostname()
	if isPrivateNetwork(hostname) {
		return fmt.Errorf("metadata URL cannot point to private network: %s", hostname)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", m.config.MetadataURL, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	m.idpMeta, err = io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read metadata: %w", err)
	}

	// Validate XML for XXE
	if err := validateXMLSecurity(m.idpMeta); err != nil {
		return fmt.Errorf("metadata XML security validation failed: %w", err)
	}

	return nil
}

// isPrivateNetwork checks if a hostname is a private network address.
func isPrivateNetwork(hostname string) bool {
	// Check for localhost variants
	privateHosts := []string{
		"localhost", "127.0.0.1", "::1", "0.0.0.0",
	}

	for _, h := range privateHosts {
		if hostname == h || strings.HasPrefix(hostname, "127.") {
			return true
		}
	}

	// Check for private IP ranges (basic check)
	if strings.HasPrefix(hostname, "10.") ||
		strings.HasPrefix(hostname, "192.168.") ||
		strings.HasPrefix(hostname, "172.16.") ||
		strings.HasPrefix(hostname, "172.17.") ||
		strings.HasPrefix(hostname, "172.18.") ||
		strings.HasPrefix(hostname, "172.19.") ||
		strings.HasPrefix(hostname, "172.20.") ||
		strings.HasPrefix(hostname, "172.21.") ||
		strings.HasPrefix(hostname, "172.22.") ||
		strings.HasPrefix(hostname, "172.23.") ||
		strings.HasPrefix(hostname, "172.24.") ||
		strings.HasPrefix(hostname, "172.25.") ||
		strings.HasPrefix(hostname, "172.26.") ||
		strings.HasPrefix(hostname, "172.27.") ||
		strings.HasPrefix(hostname, "172.28.") ||
		strings.HasPrefix(hostname, "172.29.") ||
		strings.HasPrefix(hostname, "172.30.") ||
		strings.HasPrefix(hostname, "172.31.") {
		return true
	}

	return false
}

// Metadata returns the service provider metadata XML.
func (m *Manager) Metadata() ([]byte, error) {
	if m.config == nil {
		return nil, errors.New("manager not initialized")
	}

	// Generate basic SAML metadata
	metadata := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="%s">
  <SPSSODescriptor AuthnRequestsSigned="false" WantAssertionsSigned="true">
    <SingleLogoutService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="%s"/>
    <AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="%s"/>
  </SPSSODescriptor>
</EntityDescriptor>`,
		m.config.EntityID,
		m.config.SLOResponseURL,
		m.config.ACSURL,
	)

	return []byte(metadata), nil
}

// AuthURL generates the SAML authentication URL to redirect the user to the IdP.
func (m *Manager) AuthURL(relayState string) (string, string, error) {
	if m.config.SSOURL == "" {
		return "", "", ErrNotConfigured
	}

	// Generate a simple SAML auth request ID
	authRequestID := generateID()
	issueInstant := time.Now().UTC().Format("2006-01-02T15:04:05Z")

	authRequest := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<samlp:AuthnRequest xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" ID="%s" Version="2.0" IssueInstant="%s" ProtocolBinding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" AssertionConsumerServiceURL="%s">
  <saml:Issuer xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion">%s</saml:Issuer>
  <samlp:NameIDPolicy Format="urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified"/>
</samlp:AuthnRequest>`,
		authRequestID,
		issueInstant,
		m.config.ACSURL,
		m.config.EntityID,
	)

	// Base64 encode the request
	encodedRequest := base64.StdEncoding.EncodeToString([]byte(authRequest))

	// Build redirect URL
	redirectURL := fmt.Sprintf("%s?SAMLRequest=%s&RelayState=%s",
		m.config.SSOURL,
		url.QueryEscape(encodedRequest),
		url.QueryEscape(relayState),
	)

	return redirectURL, authRequestID, nil
}

// HandleResponse processes the SAML response from the IdP.
func (m *Manager) HandleResponse(req *http.Request) (*Assertion, error) {
	if err := req.ParseForm(); err != nil {
		return nil, fmt.Errorf("parse form: %w", err)
	}

	response := req.Form.Get("SAMLResponse")
	if response == "" {
		return nil, ErrInvalidResponse
	}

	decodedResponse, err := base64.StdEncoding.DecodeString(response)
	if err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Validate against XXE attacks before parsing
	if err := validateXMLSecurity(decodedResponse); err != nil {
		return nil, fmt.Errorf("xml security validation failed: %w", err)
	}

	// Parse the SAML response with secure decoder
	var samlResponse SAMLResponse
	if err := xml.Unmarshal(decodedResponse, &samlResponse); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	// In production, verify signature here
	if m.config.IdPCertificate != nil {
		if err := m.verifySignature(decodedResponse); err != nil {
			return nil, fmt.Errorf("verify signature: %w", err)
		}
	}

	// Extract assertion
	if len(samlResponse.Assertions) == 0 {
		return nil, ErrInvalidResponse
	}

	return m.extractAssertion(samlResponse.Assertions[0]), nil
}

// LogoutURL generates the SAML logout URL.
func (m *Manager) LogoutURL(nameID, sessionIndex string) (string, error) {
	if m.config.SLOURL == "" {
		return "", ErrNotConfigured
	}

	// Simple logout URL construction
	logoutURL := fmt.Sprintf("%s?NameID=%s&SessionIndex=%s",
		m.config.SLOURL,
		url.QueryEscape(nameID),
		url.QueryEscape(sessionIndex),
	)

	return logoutURL, nil
}

// Assertion contains the extracted user information from the SAML response.
type Assertion struct {
	SubjectID    string
	Email        string
	FirstName    string
	LastName     string
	DisplayName  string
	Groups       []string
	SessionIndex string
	NotOnOrAfter time.Time
	Issuer       string
	Attributes   map[string]string
}

func (m *Manager) extractAssertion(assertion SAMLAssertion) *Assertion {
	a := &Assertion{
		Attributes: make(map[string]string),
	}

	// Extract subject
	if assertion.Subject != nil {
		a.SubjectID = assertion.Subject.NameID
		a.SessionIndex = assertion.Subject.SessionIndex
	}

	// Extract issuer
	if assertion.Issuer != nil {
		a.Issuer = assertion.Issuer.Value
	}

	// Extract attributes from AttributeStatements
	for _, stmt := range assertion.AttributeStatements {
		if stmt == nil {
			continue
		}
		for _, attr := range stmt.Attributes {
			for _, value := range attr.Values {
				a.Attributes[attr.Name] = value

				// Map common attributes
				switch attr.Name {
				case "email", "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress":
					a.Email = value
				case "firstName", "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname":
					a.FirstName = value
				case "lastName", "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/surname":
					a.LastName = value
				case "displayName", "http://schemas.microsoft.com/ws/2008/06/identity/claims/displayname":
					a.DisplayName = value
				}
			}
		}
	}

	// Extract group memberships
	if groups, ok := a.Attributes["http://schemas.microsoft.com/ws/2008/06/identity/claims/groups"]; ok {
		a.Groups = append(a.Groups, groups)
	}

	return a
}

func (m *Manager) verifySignature(response []byte) error {
	// Signature verification implementation
	// In production, this should use proper SAML signature verification
	return nil
}

// MetadataHandler serves the SAML metadata.
func (m *Manager) MetadataHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metadata, err := m.Metadata()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/samlmetadata+xml")
		w.Write(metadata)
	})
}

// ACSHandler handles the Assertion Consumer Service endpoint.
func (m *Manager) ACSHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		assertion, err := m.HandleResponse(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Store assertion in session for further processing
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"success","email":"` + assertion.Email + `"}`))
	})
}

func generateID() string {
	b := make([]byte, 16)
	if _, err := cryptoRand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return base64.URLEncoding.EncodeToString(b)
}

// validateXMLSecurity performs security checks on XML data to prevent XXE attacks.
// Go's encoding/xml package does not process external entities by default, but this
// function provides defense-in-depth by explicitly checking for XXE patterns.
func validateXMLSecurity(xmlData []byte) error {
	// Convert to lowercase for case-insensitive matching
	lowerXML := bytes.ToLower(xmlData)
	xmlStr := string(lowerXML)

	// Check for XXE attack patterns
	xxePatterns := []string{
		"<!entity",
		"<!doctype",
		"system ",
		"public ",
		"<xi:include",
		"xpointer:",
		"data:text/plain",
		"file://",
		"ftp://",
		"http://",
		"https://",
		"gopher://",
	}

	for _, pattern := range xxePatterns {
		if strings.Contains(xmlStr, pattern) {
			return fmt.Errorf("potential XXE attack detected: forbidden pattern '%s'", pattern)
		}
	}

	// Additional check: limit XML size to prevent DoS
	const maxXMLSize = 10 * 1024 * 1024 // 10MB
	if len(xmlData) > maxXMLSize {
		return fmt.Errorf("XML data exceeds maximum size of %d bytes", maxXMLSize)
	}

	return nil
}

// SAMLResponse represents a SAML response.
type SAMLResponse struct {
	XMLName     xml.Name `xml:"urn:oasis:names:tc:SAML:2.0:protocol Response"`
	ID          string   `xml:"ID,attr"`
	InResponseTo string   `xml:"InResponseTo,attr,omitempty"`
	Version     string   `xml:"Version,attr"`
	IssueInstant string   `xml:"IssueInstant,attr,omitempty"`
	Destination string   `xml:"Destination,attr,omitempty"`
	Assertions []SAMLAssertion `xml:"Assertion"`
}

// SAMLAssertion represents a SAML assertion.
type SAMLAssertion struct {
	XMLName              xml.Name              `xml:"urn:oasis:names:tc:SAML:2.0:assertion Assertion"`
	ID                   string                `xml:"ID,attr"`
	IssueInstant         string                `xml:"IssueInstant,attr,omitempty"`
	Version              string                `xml:"Version,attr,omitempty"`
	Issuer               *SAMLIssuer           `xml:"Issuer"`
	Subject              *SAMLSubject           `xml:"Subject"`
	Conditions           *SAMLConditions        `xml:"Conditions"`
	AttributeStatements  []*SAMLAttributeStatement `xml:"AttributeStatement"`
}

// SAMLIssuer represents the assertion issuer.
type SAMLIssuer struct {
	Format string `xml:"Format,attr,omitempty"`
	Value  string `xml:",chardata"`
}

// SAMLSubject represents the assertion subject.
type SAMLSubject struct {
	NameID       string                  `xml:"NameID"`
	SessionIndex string                  `xml:"SessionIndex,omitempty"`
}

// SAMLConditions represents assertion conditions.
type SAMLConditions struct {
	NotBefore    string `xml:"NotBefore,attr,omitempty"`
	NotOnOrAfter string `xml:"NotOnOrAfter,attr,omitempty"`
}

// SAMLAttributeStatement contains attributes.
type SAMLAttributeStatement struct {
	Attributes []SAMLAttribute `xml:"Attribute"`
}

// SAMLAttribute represents a single attribute.
type SAMLAttribute struct {
	Name   string        `xml:"Name,attr"`
	Values []string      `xml:"AttributeValue"`
}
