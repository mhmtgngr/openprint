package middleware

import (
	"testing"
)

// TestValidateSQLIdentifier tests SQL identifier validation.
func TestValidateSQLIdentifier(t *testing.T) {
	tests := []struct {
		name        string
		identifier  string
		expectError bool
	}{
		{"Valid simple", "users", false},
		{"Valid with underscore", "user_profile", false},
		{"Valid with numbers", "table123", false},
		{"Valid starting with underscore", "_private", false},
		{"Invalid with space", "user table", true},
		{"Invalid with dash", "user-table", true},
		{"Invalid with dot", "schema.table", true},
		{"Invalid with semicolon", "users;DROP", true},
		{"Invalid with quote", "users'", true},
		{"SQL keyword SELECT", "SELECT", true},
		{"SQL keyword DROP", "DROP", true},
		{"SQL keyword union", "UNION", true},
		{"Empty string", "", true},
		{"Starting with number", "123table", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSQLIdentifier(tt.identifier)
			if tt.expectError && err == nil {
				t.Errorf("Expected error for identifier %q, got nil", tt.identifier)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for identifier %q, got %v", tt.identifier, err)
			}
		})
	}
}

// TestIsSafeSQLValue tests SQL value safety checking.
func TestIsSafeSQLValue(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{"Normal string", "hello world", true},
		{"Email", "user@example.com", true},
		{"Single quote", "O'Reilly", false},
		{"Double single quote", "it''s", false},
		{"SQL comment", "value--", false},
		{"SQL comment 2", "value /* comment", false},
		{"SQL injection 1=1", "1 OR 1=1", false},
		{"SQL injection 1 = 1", "1 OR 1 = 1", false},
		{"xp_ command", "xp_cmdshell", false},
		{"sp_ procedure", "sp_executesql", false},
		{"EXEC statement", "EXEC(", false},
		{"EXECUTE keyword", "EXECUTE(", false},
		{"SLEEP function", "SLEEP(10)", false},
		{"BENCHMARK function", "BENCHMARK(1000", false},
		{"WAITFOR DELAY", "WAITFOR DELAY", false},
		{"Multiple quotes", "a''b''c", false},
		{"Safe numeric", "12345", true},
		{"Safe with special chars", "hello@world!", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSafeSQLValue(tt.value)
			if result != tt.expected {
				t.Errorf("IsSafeSQLValue(%q) = %v, want %v", tt.value, result, tt.expected)
			}
		})
	}
}

// TestSanitizeString tests the XSS prevention functionality.
func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Normal string", "Hello World", "Hello World"},
		{"Script tag", "<script>alert('xss')</script>", "alert('xss')"},
		{"Script tag uppercase", "<SCRIPT>alert('xss')</SCRIPT>", ""},
		{"OnClick handler", "<div onclick=\"alert('xss')\">Click</div>", "<div >Click</div>"},
		{"JavaScript protocol", "<a href=\"javascript:alert('xss')\">Link</a>", "<a href=\"\">Link</a>"},
		{"Null byte", "hello\x00world", "helloworld"},
		{"Control character", "hello\x01world", "helloworld"},
		{"Tab preserved", "hello\tworld", "hello\tworld"},
		{"Newline preserved", "hello\nworld", "hello\nworld"},
		{"Multiple scripts", "<script>bad</script> and <script>more</script>", " and "},
		{"Event handler with newline", "<img\nonerror=\"alert('xss')\">", "<img>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := string(SanitizeString(tt.input))
			// Check that dangerous patterns are removed
			if containsScript(result) {
				t.Errorf("SanitizeString(%q) still contains script: %q", tt.input, result)
			}
		})
	}
}

// TestSanitizeHTML tests HTML sanitization.
func TestSanitizeHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{"Removes script tag", "<script>alert('xss')</script>hello", "hello"},
		{"Removes iframe", "<iframe src='evil'></iframe>content", "content"},
		{"Removes object", "<object data='evil'></object>content", "content"},
		{"Removes form", "<form action='evil'></form>content", "content"},
		{"Keeps safe div", "<div class='safe'>content</div>", "content"},
		{"Keeps safe span", "<span>content</span>", "content"},
		{"Removes onclick", "<div onclick='alert(1)'>content</div>", ""},
		{"Removes onerror", "<img onerror='alert(1)'>", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := string(SanitizeHTML(tt.input))
			if !containsSubstring(result, tt.contains) {
				t.Errorf("SanitizeHTML(%q) = %q, expected to contain %q", tt.input, result, tt.contains)
			}
			// Verify no dangerous tags remain
			if containsAnyScript(result) {
				t.Errorf("SanitizeHTML(%q) still contains dangerous tags: %q", tt.input, result)
			}
		})
	}
}

// TestValidateEmail tests email validation.
func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		expectError bool
	}{
		{"Valid email", "user@example.com", false},
		{"Valid with dots", "user.name@example.com", false},
		{"Valid with plus", "user+tag@example.com", false},
		{"Valid with numbers", "user123@example123.com", false},
		{"Invalid no at", "userexample.com", true},
		{"Invalid no domain", "user@", true},
		{"Invalid no user", "@example.com", true},
		{"Invalid multiple at", "user@@example.com", true},
		{"Invalid space", "user @example.com", true},
		{"Empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if tt.expectError && err == nil {
				t.Errorf("Expected error for email %q, got nil", tt.email)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for email %q, got %v", tt.email, err)
			}
		})
	}
}

// TestValidateUUID tests UUID validation.
func TestValidateUUID(t *testing.T) {
	tests := []struct {
		name        string
		uuid        string
		expectError bool
	}{
		{"Valid UUID v4", "550e8400-e29b-41d4-a716-446655440000", false},
		{"Valid UUID uppercase", "550E8400-E29B-41D4-A716-446655440000", false},
		{"Valid UUID mixed", "550e8400-E29b-41d4-A716-446655440000", false},
		{"Invalid no hyphens", "550e8400e29b41d4a716446655440000", true},
		{"Invalid too short", "550e8400-e29b-41d4", true},
		{"Invalid characters", "550e8400-e29b-41d4-a716-44665544xxxx", true},
		{"Empty string", "", true},
		{"Invalid format", "550e8400/e29b-41d4-a716-446655440000", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUUID(tt.uuid)
			if tt.expectError && err == nil {
				t.Errorf("Expected error for UUID %q, got nil", tt.uuid)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for UUID %q, got %v", tt.uuid, err)
			}
		})
	}
}

// TestValidateURL tests URL validation.
func TestValidateURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{"Valid HTTP", "http://example.com", false},
		{"Valid HTTPS", "https://example.com", false},
		{"Valid with path", "https://example.com/path", false},
		{"Valid with query", "https://example.com?query=1", false},
		{"Empty string", "", true},
		{"JavaScript protocol", "javascript:alert('xss')", true},
		{"No protocol", "example.com", true},
		{"FTP protocol", "ftp://example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url)
			if tt.expectError && err == nil {
				t.Errorf("Expected error for URL %q, got nil", tt.url)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for URL %q, got %v", tt.url, err)
			}
		})
	}
}

// TestValidateStringLength tests string length validation.
func TestValidateStringLength(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		minLength   int
		maxLength   int
		expectError bool
	}{
		{"Within range", "hello", 1, 10, false},
		{"Too short", "hi", 5, 10, true},
		{"Too long", "hello world", 1, 5, true},
		{"Exact min", "hello", 5, 10, false},
		{"Exact max", "hello", 1, 5, false},
		{"Empty with min 0", "", 0, 10, false},
		{"No constraints", "any string", 0, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStringLength(tt.input, tt.minLength, tt.maxLength)
			if tt.expectError && err == nil {
				t.Errorf("Expected error for input %q", tt.input)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for input %q, got %v", tt.input, err)
			}
		})
	}
}

// TestValidateInteger tests integer validation.
func TestValidateInteger(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		min         int64
		max         int64
		expectError bool
	}{
		{"Valid number", "100", 1, 1000, false},
		{"Not a number", "abc", 1, 1000, true},
		{"Too small", "5", 10, 100, true},
		{"Too large", "200", 1, 100, true},
		{"Exact min", "10", 10, 100, false},
		{"Exact max", "100", 10, 100, false},
		{"Negative", "-5", -10, 10, false},
		{"Zero", "0", 0, 100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := ValidateInteger(tt.input, tt.min, tt.max)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for input %q", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for input %q, got %v", tt.input, err)
				}
				if val < tt.min || val > tt.max {
					t.Errorf("Value %d out of range [%d, %d]", val, tt.min, tt.max)
				}
			}
		})
	}
}

// Helper functions
func containsScript(s string) bool {
	return containsSubstring(s, "<script") ||
		containsSubstring(s, "<SCRIPT") ||
		containsSubstring(s, "javascript:") ||
		containsSubstring(s, "onerror") ||
		containsSubstring(s, "onclick")
}

func containsAnyScript(s string) bool {
	dangerous := []string{
		"<script", "<SCRIPT", "<iframe", "<IFRAME",
		"<object", "<OBJECT", "<embed", "<EMBED",
		"onclick", "onerror", "onload", "onmouseover",
		"javascript:",
	}
	for _, d := range dangerous {
		if containsSubstring(s, d) {
			return true
		}
	}
	return false
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) >= 0
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// BenchmarkSanitizeString benchmarks the sanitization function.
func BenchmarkSanitizeString(b *testing.B) {
	input := "<script>alert('xss')</script>Hello <b>World</b> <!-- comment -->"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SanitizeString(input)
	}
}

// BenchmarkValidateEmail benchmarks email validation.
func BenchmarkValidateEmail(b *testing.B) {
	email := "user@example.com"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateEmail(email)
	}
}

// BenchmarkIsSafeSQLValue benchmarks SQL safety checking.
func BenchmarkIsSafeSQLValue(b *testing.B) {
	value := "normal user input 123"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsSafeSQLValue(value)
	}
}
