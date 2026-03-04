// Package saml provides tests for SAML 2.0 SSO integration.
package saml

import (
	"bytes"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestNewManager(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := &Config{
			EntityID: "https://sp.example.com",
			SSOURL:   "https://idp.example.com/sso",
			ACSURL:   "https://sp.example.com/acs",
			SLOURL:   "https://idp.example.com/slo",
		}

		mgr, err := NewManager(cfg)
		if err != nil {
			t.Fatalf("NewManager() error = %v", err)
		}
		if mgr == nil {
			t.Fatal("NewManager() returned nil")
		}
		if mgr.config != cfg {
			t.Error("NewManager() config not set correctly")
		}
	})

	t.Run("nil config", func(t *testing.T) {
		_, err := NewManager(nil)
		if err == nil {
			t.Error("NewManager(nil) should return error")
		}
	})
}

func TestManager_Metadata(t *testing.T) {
	cfg := &Config{
		EntityID:       "https://sp.example.com",
		SSOURL:         "https://idp.example.com/sso",
		ACSURL:         "https://sp.example.com/acs",
		SLOResponseURL: "https://sp.example.com/slo",
	}

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	metadata, err := mgr.Metadata()
	if err != nil {
		t.Fatalf("Metadata() error = %v", err)
	}

	if len(metadata) == 0 {
		t.Fatal("Metadata() returned empty bytes")
	}

	metadataStr := string(metadata)
	if !strings.Contains(metadataStr, cfg.EntityID) {
		t.Errorf("Metadata() should contain EntityID, got %v", metadataStr)
	}
	if !strings.Contains(metadataStr, cfg.ACSURL) {
		t.Errorf("Metadata() should contain ACSURL, got %v", metadataStr)
	}
	if !strings.Contains(metadataStr, cfg.SLOResponseURL) {
		t.Errorf("Metadata() should contain SLOResponseURL, got %v", metadataStr)
	}
}

func TestManager_AuthURL(t *testing.T) {
	cfg := &Config{
		EntityID: "https://sp.example.com",
		SSOURL:   "https://idp.example.com/sso",
		ACSURL:   "https://sp.example.com/acs",
	}

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	t.Run("generate auth URL with relay state", func(t *testing.T) {
		relayState := "test-relay-state"
		authURL, authRequestID, err := mgr.AuthURL(relayState)

		if err != nil {
			t.Fatalf("AuthURL() error = %v", err)
		}
		if authURL == "" {
			t.Fatal("AuthURL() returned empty URL")
		}
		if authRequestID == "" {
			t.Fatal("AuthURL() returned empty request ID")
		}

		// Check URL contains SSOURL
		if !strings.Contains(authURL, cfg.SSOURL) {
			t.Errorf("AuthURL() should contain SSOURL, got %v", authURL)
		}
		// Check for relay state
		if !strings.Contains(authURL, url.QueryEscape(relayState)) {
			t.Errorf("AuthURL() should contain relay state, got %v", authURL)
		}
	})

	t.Run("generate auth URL without relay state", func(t *testing.T) {
		authURL, authRequestID, err := mgr.AuthURL("")

		if err != nil {
			t.Fatalf("AuthURL() error = %v", err)
		}
		if authURL == "" {
			t.Fatal("AuthURL() returned empty URL")
		}
		if authRequestID == "" {
			t.Fatal("AuthURL() returned empty request ID")
		}
	})

	t.Run("unconfigured SSO URL", func(t *testing.T) {
		cfg := &Config{
			EntityID: "https://sp.example.com",
			ACSURL:   "https://sp.example.com/acs",
		}
		mgr, _ := NewManager(cfg)

		_, _, err := mgr.AuthURL("")
		if err != ErrNotConfigured {
			t.Errorf("AuthURL() error = %v, want %v", err, ErrNotConfigured)
		}
	})
}

func TestManager_LogoutURL(t *testing.T) {
	cfg := &Config{
		EntityID: "https://sp.example.com",
		SLOURL:   "https://idp.example.com/slo",
		ACSURL:   "https://sp.example.com/acs",
	}

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	t.Run("generate logout URL", func(t *testing.T) {
		nameID := "user-123"
		sessionIndex := "session-456"

		logoutURL, err := mgr.LogoutURL(nameID, sessionIndex)
		if err != nil {
			t.Fatalf("LogoutURL() error = %v", err)
		}
		if logoutURL == "" {
			t.Fatal("LogoutURL() returned empty URL")
		}

		// Check URL contains SLOURL
		if !strings.Contains(logoutURL, cfg.SLOURL) {
			t.Errorf("LogoutURL() should contain SLOURL, got %v", logoutURL)
		}
		// Check for nameID and sessionIndex
		if !strings.Contains(logoutURL, url.QueryEscape(nameID)) {
			t.Errorf("LogoutURL() should contain nameID, got %v", logoutURL)
		}
	})

	t.Run("unconfigured SLO URL", func(t *testing.T) {
		cfg := &Config{
			EntityID: "https://sp.example.com",
			ACSURL:   "https://sp.example.com/acs",
		}
		mgr, _ := NewManager(cfg)

		_, err := mgr.LogoutURL("user-123", "session-456")
		if err != ErrNotConfigured {
			t.Errorf("LogoutURL() error = %v, want %v", err, ErrNotConfigured)
		}
	})
}

func TestManager_HandleResponse(t *testing.T) {
	cfg := &Config{
		EntityID: "https://sp.example.com",
		SSOURL:   "https://idp.example.com/sso",
		ACSURL:   "https://sp.example.com/acs",
	}

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	t.Run("missing SAML response", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/acs", nil)
		req.PostForm = url.Values{}

		_, err := mgr.HandleResponse(req)
		if err != ErrInvalidResponse {
			t.Errorf("HandleResponse() error = %v, want %v", err, ErrInvalidResponse)
		}
	})

	t.Run("malformed SAML response", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/acs", strings.NewReader("invalid=saml"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		_, err := mgr.HandleResponse(req)
		if err == nil {
			t.Error("HandleResponse() should return error for malformed response")
		}
	})
}

func TestManager_MetadataHandler(t *testing.T) {
	cfg := &Config{
		EntityID: "https://sp.example.com",
		ACSURL:   "https://sp.example.com/acs",
	}

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	handler := mgr.MetadataHandler()
	if handler == nil {
		t.Fatal("MetadataHandler() returned nil")
	}

	req := httptest.NewRequest("GET", "/metadata", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("MetadataHandler() status = %d, want %d", w.Code, http.StatusOK)
	}
	if w.Header().Get("Content-Type") != "application/samlmetadata+xml" {
		t.Errorf("MetadataHandler() Content-Type = %v, want application/samlmetadata+xml", w.Header().Get("Content-Type"))
	}
}

func TestManager_ACSHandler(t *testing.T) {
	cfg := &Config{
		EntityID: "https://sp.example.com",
		ACSURL:   "https://sp.example.com/acs",
	}

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	handler := mgr.ACSHandler()
	if handler == nil {
		t.Fatal("ACSHandler() returned nil")
	}

	t.Run("invalid SAML response", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/acs", strings.NewReader("SAMLResponse=invalid"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("ACSHandler() status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("wrong method", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/acs", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("ACSHandler() status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
		}
	})
}

func TestExtractAssertion(t *testing.T) {
	cfg := &Config{
		EntityID: "https://sp.example.com",
		ACSURL:   "https://sp.example.com/acs",
	}

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	// Test with minimal assertion
	assertion := &SAMLAssertion{
		ID: "assertion-123",
		Issuer: &SAMLIssuer{
			Value: "https://idp.example.com",
		},
		Subject: &SAMLSubject{
			NameID: "user@example.com",
		},
		AttributeStatements: []*SAMLAttributeStatement{
			{
				Attributes: []SAMLAttribute{
					{
						Name:   "email",
						Values: []string{"user@example.com"},
					},
					{
						Name:   "firstName",
						Values: []string{"John"},
					},
				},
			},
		},
	}

	result := mgr.extractAssertion(*assertion)

	if result.SubjectID != "user@example.com" {
		t.Errorf("SubjectID = %v, want user@example.com", result.SubjectID)
	}
	if result.Email != "user@example.com" {
		t.Errorf("Email = %v, want user@example.com", result.Email)
	}
	if result.FirstName != "John" {
		t.Errorf("FirstName = %v, want John", result.FirstName)
	}
	if result.Issuer != "https://idp.example.com" {
		t.Errorf("Issuer = %v, want https://idp.example.com", result.Issuer)
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	if id1 == "" {
		t.Fatal("generateID() returned empty string")
	}
	if id2 == "" {
		t.Fatal("generateID() returned empty string")
	}
	if id1 == id2 {
		t.Error("generateID() should produce unique IDs")
	}
}

func TestAssertion(t *testing.T) {
	assertion := &Assertion{
		SubjectID:    "user@example.com",
		Email:        "user@example.com",
		FirstName:    "John",
		LastName:     "Doe",
		DisplayName:  "John Doe",
		Groups:       []string{"admins", "users"},
		SessionIndex: "session-123",
		Issuer:       "https://idp.example.com",
		Attributes:   make(map[string]string),
	}

	if assertion.SubjectID != "user@example.com" {
		t.Error("Assertion SubjectID not set correctly")
	}
	if len(assertion.Groups) != 2 {
		t.Errorf("Assertion Groups length = %d, want 2", len(assertion.Groups))
	}
}

func TestSAMLResponse_Struct(t *testing.T) {
	// Test that SAMLResponse struct has expected fields
	response := &SAMLResponse{
		ID:           "response-123",
		InResponseTo: "request-456",
		Version:      "2.0",
		IssueInstant: "2024-01-01T00:00:00Z",
		Destination:  "https://sp.example.com/acs",
		Assertions: []SAMLAssertion{
			{
				ID:      "assertion-123",
				Version: "2.0",
				Issuer:  &SAMLIssuer{Value: "https://idp.example.com"},
				Subject: &SAMLSubject{NameID: "user@example.com"},
			},
		},
	}

	if response.ID != "response-123" {
		t.Error("SAMLResponse ID not set correctly")
	}
	if response.Version != "2.0" {
		t.Error("SAMLResponse Version not set correctly")
	}
	if len(response.Assertions) != 1 {
		t.Error("SAMLResponse should have 1 assertion")
	}
}

func TestSAMLIssuer(t *testing.T) {
	issuer := &SAMLIssuer{
		Format: "urn:oasis:names:tc:SAML:2.0:nameid-format:entity",
		Value:  "https://idp.example.com",
	}

	if issuer.Value != "https://idp.example.com" {
		t.Error("SAMLIssuer Value not set correctly")
	}
}

func TestSAMLSubject(t *testing.T) {
	subject := &SAMLSubject{
		NameID:       "user@example.com",
		SessionIndex: "session-123",
	}

	if subject.NameID != "user@example.com" {
		t.Error("SAMLSubject NameID not set correctly")
	}
}

func TestSAMLAttributeStatement(t *testing.T) {
	attr := SAMLAttribute{
		Name:   "email",
		Values: []string{"user@example.com"},
	}

	statement := &SAMLAttributeStatement{
		Attributes: []SAMLAttribute{attr},
	}

	if len(statement.Attributes) != 1 {
		t.Error("SAMLAttributeStatement should have 1 attribute")
	}
}

func TestSAMLConditions(t *testing.T) {
	conditions := &SAMLConditions{
		NotBefore:    "2024-01-01T00:00:00Z",
		NotOnOrAfter: "2024-12-31T23:59:59Z",
	}

	if conditions.NotBefore != "2024-01-01T00:00:00Z" {
		t.Error("SAMLConditions NotBefore not set correctly")
	}
}

func TestSecureXMLDecoder_ValidXML(t *testing.T) {
	validXML := `<?xml version="1.0" encoding="UTF-8"?>
<Response xmlns="urn:oasis:names:tc:SAML:2.0:protocol" ID="test" Version="2.0">
	<Issuer>https://idp.example.com</Issuer>
</Response>`

	decoder := newSecureXMLDecoder(strings.NewReader(validXML))
	var response SAMLResponse
	err := decoder.Decode(&response)

	if err != nil {
		t.Errorf("Decode() valid XML should succeed, got error: %v", err)
	}
	if response.ID != "test" {
		t.Errorf("Decode() ID = %v, want test", response.ID)
	}
}

func TestSecureXMLDecoder_SizeLimit(t *testing.T) {
	// Create XML that exceeds the 10MB limit
	// We'll create a smaller test by using a mock reader with a small limit
	smallLimitDecoder := func(r io.Reader, limit int64) *secureXMLDecoder {
		return &secureXMLDecoder{
			Decoder:  xml.NewDecoder(&countingReader{r: r, remaining: limit}),
			maxBytes: limit,
		}
	}

	// Create XML that's larger than our test limit
	largeXML := `<?xml version="1.0" encoding="UTF-8"?>
<Response xmlns="urn:oasis:names:tc:SAML:2.0:protocol" ID="test" Version="2.0">
	<Issuer>https://idp.example.com</Issuer>`
	for i := 0; i < 1000; i++ {
		largeXML += `<Extension>Very long extension data to exceed the size limit</Extension>`
	}
	largeXML += `</Response>`

	// Use a small limit to trigger the error
	testLimit := int64(100) // 100 bytes
	decoder := smallLimitDecoder(strings.NewReader(largeXML), testLimit)
	var response SAMLResponse
	err := decoder.Decode(&response)

	if err == nil {
		t.Error("Decode() should return error for XML exceeding size limit")
	}
}

func TestCountingReader_RespectsLimit(t *testing.T) {
	data := make([]byte, 1000) // 1000 bytes of data
	for i := range data {
		data[i] = 'x'
	}

	tests := []struct {
		name           string
		limit          int64
		expectRead     int
		expectFirstEOF bool // Whether first Read returns EOF
	}{
		{"read within limit", 500, 500, false},
		{"read at limit", 1000, 1000, false},   // First read exhausts data, returns all bytes
		{"read beyond limit", 100, 100, false}, // First read hits limit, returns 100 bytes
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &countingReader{
				r:         bytes.NewReader(data),
				remaining: tt.limit,
			}

			buf := make([]byte, 2000)
			n, err := reader.Read(buf)

			if n != tt.expectRead {
				t.Errorf("Read() returned %d bytes, want %d", n, tt.expectRead)
			}

			if tt.expectFirstEOF && err != io.EOF {
				t.Errorf("Read() should return EOF, got %v", err)
			}
			if !tt.expectFirstEOF && err != nil {
				t.Errorf("Read() should not return EOF on first read, got %v", err)
			}

			// Second read should return EOF when limit is exhausted
			n2, err2 := reader.Read(buf)
			if tt.limit <= 1000 {
				if err2 != io.EOF {
					t.Errorf("Second Read() should return EOF when limit exhausted, got %v (n=%d)", err2, n2)
				}
			}
		})
	}
}

func TestValidateXMLSecurity(t *testing.T) {
	tests := []struct {
		name      string
		xmlData   string
		wantError bool
	}{
		{
			name:      "valid SAML response",
			xmlData:   `<?xml version="1.0"?><Response xmlns="urn:oasis:names:tc:SAML:2.0:protocol"><Issuer>test</Issuer></Response>`,
			wantError: false,
		},
		{
			name:      "XXE attack with entity",
			xmlData:   `<?xml version="1.0"?><!DOCTYPE foo [<!ENTITY xxe SYSTEM "file:///etc/passwd">]><foo>&xxe;</foo>`,
			wantError: true,
		},
		{
			name:      "XXE attack with DOCTYPE",
			xmlData:   `<?xml version="1.0"?><!DOCTYPE foo PUBLIC "-//foo//bar" "http://example.com/foo.dtd"><foo></foo>`,
			wantError: true,
		},
		{
			name:      "XXE attack with system",
			xmlData:   `<?xml version="1.0"?><foo><system>file:///etc/passwd</system></foo>`,
			wantError: true,
		},
		{
			name:      "XXE attack with xi:include",
			xmlData:   `<?xml version="1.0"?><foo xmlns:xi="http://www.w3.org/2001/XInclude"><xi:include href="file:///etc/passwd"/></foo>`,
			wantError: true,
		},
		{
			name:      "XML exceeds maximum size",
			xmlData:   string(make([]byte, 11*1024*1024)), // 11MB
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateXMLSecurity([]byte(tt.xmlData))
			if (err != nil) != tt.wantError {
				t.Errorf("validateXMLSecurity() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
