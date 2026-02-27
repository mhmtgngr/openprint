// Package saml provides SAML 2.0 SSO integration for OpenPrint authentication.
// This enables enterprise single sign-on with identity providers like Okta, Azure AD, and ADFS.
package saml

import (
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

	return nil
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

	// Parse the SAML response
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
