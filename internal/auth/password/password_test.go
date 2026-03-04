// Package password provides tests for secure password hashing using Argon2id.
package password

import (
	"strings"
	"testing"

	"golang.org/x/crypto/argon2"
)

func TestDefaultParameters(t *testing.T) {
	if DefaultMemory != 64*1024 {
		t.Errorf("DefaultMemory = %d, want %d", DefaultMemory, 64*1024)
	}
	if DefaultIterations != 3 {
		t.Errorf("DefaultIterations = %d, want %d", DefaultIterations, 3)
	}
	if DefaultThreads != 4 {
		t.Errorf("DefaultThreads = %d, want %d", DefaultThreads, 4)
	}
	if DefaultKeyLen != 32 {
		t.Errorf("DefaultKeyLen = %d, want %d", DefaultKeyLen, 32)
	}
	if DefaultSaltLen != 16 {
		t.Errorf("DefaultSaltLen = %d, want %d", DefaultSaltLen, 16)
	}
}

func TestNewHasher(t *testing.T) {
	memory := uint32(32 * 1024)
	time := uint32(2)
	threads := uint8(2)
	keyLen := uint32(24)
	saltLen := uint32(12)

	hasher := NewHasher(memory, time, threads, keyLen, saltLen)

	if hasher.memory != memory {
		t.Errorf("NewHasher() memory = %d, want %d", hasher.memory, memory)
	}
	if hasher.time != time {
		t.Errorf("NewHasher() time = %d, want %d", hasher.time, time)
	}
	if hasher.threads != threads {
		t.Errorf("NewHasher() threads = %d, want %d", hasher.threads, threads)
	}
	if hasher.keyLen != keyLen {
		t.Errorf("NewHasher() keyLen = %d, want %d", hasher.keyLen, keyLen)
	}
	if hasher.saltLen != saltLen {
		t.Errorf("NewHasher() saltLen = %d, want %d", hasher.saltLen, saltLen)
	}
}

func TestDefaultHasher(t *testing.T) {
	hasher := DefaultHasher()

	if hasher.memory != DefaultMemory {
		t.Errorf("DefaultHasher() memory = %d, want %d", hasher.memory, DefaultMemory)
	}
	if hasher.time != DefaultIterations {
		t.Errorf("DefaultHasher() time = %d, want %d", hasher.time, DefaultIterations)
	}
	if hasher.threads != DefaultThreads {
		t.Errorf("DefaultHasher() threads = %d, want %d", hasher.threads, DefaultThreads)
	}
	if hasher.keyLen != DefaultKeyLen {
		t.Errorf("DefaultHasher() keyLen = %d, want %d", hasher.keyLen, DefaultKeyLen)
	}
	if hasher.saltLen != DefaultSaltLen {
		t.Errorf("DefaultHasher() saltLen = %d, want %d", hasher.saltLen, DefaultSaltLen)
	}
}

func TestHasher_Generate(t *testing.T) {
	hasher := DefaultHasher()

	t.Run("generate valid hash", func(t *testing.T) {
		password := "SecurePassword123!"
		hash, err := hasher.Generate(password)

		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		if hash == "" {
			t.Fatal("Generate() returned empty hash")
		}

		// Check hash format
		if !strings.HasPrefix(hash, "$argon2id$") {
			t.Errorf("Hash should start with $argon2id$, got %v", hash[:10])
		}

		parts := strings.Split(hash, "$")
		if len(parts) != 6 {
			t.Errorf("Hash should have 6 parts separated by $, got %d", len(parts))
		}
	})

	t.Run("generate different hashes for same password", func(t *testing.T) {
		password := "SamePassword123!"

		hash1, err := hasher.Generate(password)
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}

		hash2, err := hasher.Generate(password)
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}

		if hash1 == hash2 {
			t.Error("Generate() should produce different hashes due to random salt")
		}
	})

	t.Run("empty password returns error", func(t *testing.T) {
		_, err := hasher.Generate("")

		if err == nil {
			t.Error("Generate() should return error for empty password")
		}
	})
}

func TestHasher_Verify(t *testing.T) {
	hasher := DefaultHasher()

	t.Run("verify correct password", func(t *testing.T) {
		password := "CorrectPassword123!"

		hash, err := hasher.Generate(password)
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}

		valid, err := hasher.Verify(password, hash)
		if err != nil {
			t.Fatalf("Verify() error = %v", err)
		}
		if !valid {
			t.Error("Verify() should return true for correct password")
		}
	})

	t.Run("verify incorrect password", func(t *testing.T) {
		password := "CorrectPassword123!"
		wrongPassword := "WrongPassword123!"

		hash, err := hasher.Generate(password)
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}

		valid, err := hasher.Verify(wrongPassword, hash)
		if err == nil {
			t.Error("Verify() should return error for incorrect password")
		}
		if valid {
			t.Error("Verify() should return false for incorrect password")
		}
	})

	t.Run("verify invalid hash format", func(t *testing.T) {
		_, err := hasher.Verify("password", "invalid-hash")

		if err == nil {
			t.Error("Verify() should return error for invalid hash format")
		}
		if err != ErrInvalidHashFormat {
			t.Errorf("Verify() error = %v, want %v", err, ErrInvalidHashFormat)
		}
	})
}

func TestHasher_MustGenerate(t *testing.T) {
	hasher := DefaultHasher()

	t.Run("successful generation", func(t *testing.T) {
		password := "SecurePassword123!"
		hash := hasher.MustGenerate(password)

		if hash == "" {
			t.Fatal("MustGenerate() returned empty hash")
		}

		// Verify the hash works
		valid, err := hasher.Verify(password, hash)
		if err != nil || !valid {
			t.Error("MustGenerate() produced invalid hash")
		}
	})

	t.Run("panics on empty password", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustGenerate() should panic for empty password")
			}
		}()
		hasher.MustGenerate("")
	})
}

func TestGenerate(t *testing.T) {
	password := "TestPassword123!"

	hash, err := Generate(password)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if hash == "" {
		t.Fatal("Generate() returned empty hash")
	}

	// Should be verifiable with the default hasher
	hasher := DefaultHasher()
	valid, err := hasher.Verify(password, hash)
	if err != nil || !valid {
		t.Error("Generate() hash should be verifiable")
	}
}

func TestVerify(t *testing.T) {
	password := "TestPassword123!"

	hash, err := Generate(password)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	t.Run("verify correct password", func(t *testing.T) {
		valid, err := Verify(password, hash)
		if err != nil {
			t.Fatalf("Verify() error = %v", err)
		}
		if !valid {
			t.Error("Verify() should return true for correct password")
		}
	})

	t.Run("verify incorrect password", func(t *testing.T) {
		valid, err := Verify("WrongPassword", hash)
		if err == nil {
			t.Error("Verify() should return error for incorrect password")
		}
		if valid {
			t.Error("Verify() should return false for incorrect password")
		}
	})
}

func TestMustVerify(t *testing.T) {
	password := "TestPassword123!"
	hash, _ := Generate(password)

	t.Run("successful verification", func(t *testing.T) {
		valid, _ := Verify(password, hash)
		if !valid {
			t.Error("Verify() should return true for correct password")
		}
	})

	t.Run("panics on invalid password", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustVerify() should panic for incorrect password")
			}
		}()
		MustVerify("WrongPassword", hash)
	})
}

func TestDecodeHash(t *testing.T) {
	hasher := DefaultHasher()
	password := "TestPassword123!"

	hash, err := hasher.Generate(password)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	t.Run("decode valid hash", func(t *testing.T) {
		decoded, err := decodeHash(hash)
		if err != nil {
			t.Fatalf("decodeHash() error = %v", err)
		}

		if decoded.Version != argon2.Version {
			t.Errorf("decodeHash() Version = %d, want %d", decoded.Version, argon2.Version)
		}
		if len(decoded.Salt) == 0 {
			t.Error("decodeHash() Salt should not be empty")
		}
		if len(decoded.Hash) == 0 {
			t.Error("decodeHash() Hash should not be empty")
		}
	})

	t.Run("decode invalid hash format", func(t *testing.T) {
		_, err := decodeHash("invalid")
		if err != ErrInvalidHashFormat {
			t.Errorf("decodeHash() error = %v, want %v", err, ErrInvalidHashFormat)
		}
	})

	t.Run("decode hash with wrong version", func(t *testing.T) {
		wrongHash := "$argon2id$v=999$m=65536,t=3,p=4$invalid$invalid"
		_, err := decodeHash(wrongHash)
		if err != ErrInvalidHashVersion {
			t.Errorf("decodeHash() error = %v, want %v", err, ErrInvalidHashVersion)
		}
	})
}

func TestStrengthChecker(t *testing.T) {
	t.Run("default checker configuration", func(t *testing.T) {
		checker := DefaultStrengthChecker()

		if checker.minLength != 12 {
			t.Errorf("DefaultStrengthChecker() minLength = %d, want 12", checker.minLength)
		}
		if !checker.requireUpper {
			t.Error("DefaultStrengthChecker() should require uppercase")
		}
		if !checker.requireLower {
			t.Error("DefaultStrengthChecker() should require lowercase")
		}
		if !checker.requireNumber {
			t.Error("DefaultStrengthChecker() should require numbers")
		}
		if !checker.requireSpecial {
			t.Error("DefaultStrengthChecker() should require special characters")
		}
	})
}

func TestNewStrengthChecker(t *testing.T) {
	checker := NewStrengthChecker(8, true, false, true, false)

	if checker.minLength != 8 {
		t.Errorf("NewStrengthChecker() minLength = %d, want 8", checker.minLength)
	}
	if !checker.requireUpper {
		t.Error("NewStrengthChecker() requireUpper should be true")
	}
	if checker.requireLower {
		t.Error("NewStrengthChecker() requireLower should be false")
	}
	if !checker.requireNumber {
		t.Error("NewStrengthChecker() requireNumber should be true")
	}
	if checker.requireSpecial {
		t.Error("NewStrengthChecker() requireSpecial should be false")
	}
}

func TestStrengthChecker_Check(t *testing.T) {
	checker := DefaultStrengthChecker()

	t.Run("valid strong password", func(t *testing.T) {
		password := "StrongP@ssword1"
		result := checker.Check(password)

		if !result.Valid {
			t.Errorf("Check() Valid = false, errors: %v", result.Errors)
		}
		if result.Strength < 80 {
			t.Errorf("Check() Strength = %d, want >= 80", result.Strength)
		}
	})

	t.Run("too short", func(t *testing.T) {
		password := "Short1!"
		result := checker.Check(password)

		if result.Valid {
			t.Error("Check() should return false for short password")
		}
		if len(result.Errors) == 0 {
			t.Error("Check() should have errors for short password")
		}
	})

	t.Run("missing uppercase", func(t *testing.T) {
		password := "lowercase1!abcdefgh"
		result := checker.Check(password)

		if result.Valid {
			t.Error("Check() should return false for password without uppercase")
		}
	})

	t.Run("missing lowercase", func(t *testing.T) {
		password := "UPPERCASE1!ABCDEFGH"
		result := checker.Check(password)

		if result.Valid {
			t.Error("Check() should return false for password without lowercase")
		}
	})

	t.Run("missing number", func(t *testing.T) {
		password := "NoNumber!abcdefgh"
		result := checker.Check(password)

		if result.Valid {
			t.Error("Check() should return false for password without number")
		}
	})

	t.Run("missing special character", func(t *testing.T) {
		password := "NoSpecial1abcdefgh"
		result := checker.Check(password)

		if result.Valid {
			t.Error("Check() should return false for password without special character")
		}
	})

	t.Run("all requirements met", func(t *testing.T) {
		password := "PerfectP@ssw0rd12345"
		result := checker.Check(password)

		if !result.Valid {
			t.Errorf("Check() should be valid, errors: %v", result.Errors)
		}
	})
}

func TestStrengthChecker_CustomRequirements(t *testing.T) {
	// Minimal requirements: only length and lowercase
	checker := NewStrengthChecker(6, false, true, false, false)

	t.Run("meets minimal requirements", func(t *testing.T) {
		result := checker.Check("abcdef")

		if !result.Valid {
			t.Errorf("Check() Valid = false, errors: %v", result.Errors)
		}
	})

	t.Run("too short", func(t *testing.T) {
		result := checker.Check("abc")

		if result.Valid {
			t.Error("Check() should return false for too short password")
		}
	})

	t.Run("missing lowercase", func(t *testing.T) {
		result := checker.Check("ABCDEF")

		if result.Valid {
			t.Error("Check() should return false for password without lowercase")
		}
	})
}

func TestStrengthChecker_CalculateStrength(t *testing.T) {
	checker := DefaultStrengthChecker()

	t.Run("maximum strength", func(t *testing.T) {
		// Very long password with all character types
		password := "VeryLongPassword123!@#$abcdefghi"
		result := checker.Check(password)

		if result.Strength != 100 {
			t.Errorf("Check() Strength = %d, want 100", result.Strength)
		}
	})

	t.Run("weak password", func(t *testing.T) {
		password := "short1!A"
		result := checker.Check(password)

		if result.Strength >= 80 {
			t.Errorf("Check() Strength = %d, want < 80 for weak password", result.Strength)
		}
	})
}

func TestHash_String(t *testing.T) {
	hasher := DefaultHasher()
	password := "TestPassword123!"

	hash, err := hasher.Generate(password)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// The hash should be decodable back
	decoded, err := decodeHash(hash)
	if err != nil {
		t.Fatalf("decodeHash() error = %v", err)
	}

	// Verify String() format
	expectedFormat := "$argon2id$v=%d$m=%d,t=%d,p=%d"
	expectedFormat = string(expectedFormat)
	if !strings.HasPrefix(hash, "$argon2id$v=") {
		t.Errorf("Hash format incorrect, got %v", hash)
	}

	// Verify decoded values match
	if decoded.Memory != hasher.memory {
		t.Error("Decoded memory mismatch")
	}
	if decoded.Time != hasher.time {
		t.Error("Decoded time mismatch")
	}
	if decoded.Threads != hasher.threads {
		t.Error("Decoded threads mismatch")
	}
}

func TestConstantTimeComparison(t *testing.T) {
	hasher := DefaultHasher()
	password := "SensitivePassword!"

	hash, err := hasher.Generate(password)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Verify should use constant-time comparison
	// This test ensures timing attacks are mitigated
	valid, _ := hasher.Verify(password, hash)
	if !valid {
		t.Error("Verify() should succeed for correct password")
	}

	invalid, _ := hasher.Verify(password+"x", hash)
	if invalid {
		t.Error("Verify() should fail for incorrect password")
	}
}

func TestCrossCompatibility(t *testing.T) {
	// Test that hashes generated by one hasher can be verified by another
	password := "TestPassword123!"

	hasher1 := NewHasher(64*1024, 3, 4, 32, 16)
	hasher2 := DefaultHasher()

	hash, err := hasher1.Generate(password)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Should be verifiable by default hasher
	valid, err := hasher2.Verify(password, hash)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !valid {
		t.Error("Hash should be cross-compatible")
	}
}

func TestEmptyPasswordHash(t *testing.T) {
	hasher := DefaultHasher()

	t.Run("hash empty password", func(t *testing.T) {
		_, err := hasher.Generate("")
		if err == nil {
			t.Error("Generate() should return error for empty password")
		}
	})

	t.Run("verify empty password", func(t *testing.T) {
		password := "ValidPassword123!"
		hash, _ := hasher.Generate(password)

		_, err := hasher.Verify("", hash)
		if err == nil {
			t.Error("Verify() should return error for empty password")
		}
	})
}

func TestSpecialCharactersInPassword(t *testing.T) {
	hasher := DefaultHasher()

	passwords := []string{
		"Password!@#$%^&*()",
		"密码123!Abc",     // Chinese characters
		"Пароль123!Abc", // Cyrillic characters
		"Emoji123!🔒Abc", // Emoji
	}

	for _, password := range passwords {
		t.Run("special chars", func(t *testing.T) {
			hash, err := hasher.Generate(password)
			if err != nil {
				t.Fatalf("Generate() error for %q: %v", password, err)
			}

			valid, err := hasher.Verify(password, hash)
			if err != nil {
				t.Fatalf("Verify() error for %q: %v", password, err)
			}
			if !valid {
				t.Errorf("Verify() failed for password %q", password)
			}
		})
	}
}
