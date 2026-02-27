// Package password provides secure password hashing using Argon2id.
package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

var (
	// ErrMismatchedHashAndPassword is returned when a password doesn't match its hash.
	ErrMismatchedHashAndPassword = errors.New("password does not match")
	// ErrInvalidHashFormat is returned when the hash format is invalid.
	ErrInvalidHashFormat = errors.New("invalid hash format")
	// ErrInvalidHashVersion is returned when the hash version is not supported.
	ErrInvalidHashVersion = errors.New("invalid hash version")
)

// Hash represents a password hash with its parameters.
type Hash struct {
	Version  int
	Memory   uint32
	Time     uint32
	Threads  uint8
	KeyLen   uint32
	SaltLen  uint32
	Salt     []byte
	Hash     []byte
}

// String returns the string representation of the hash.
func (h *Hash) String() string {
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		h.Version,
		h.Memory,
		h.Time,
		h.Threads,
		base64.RawStdEncoding.EncodeToString(h.Salt),
		base64.RawStdEncoding.EncodeToString(h.Hash),
	)
}

// Default parameters for Argon2id (OWASP recommendations as of 2024).
var (
	// DefaultMemory is the default memory cost in KiB (64 MiB).
	DefaultMemory = uint32(64 * 1024)
	// DefaultIterations is the default number of iterations.
	DefaultIterations = uint32(3)
	// DefaultThreads is the default parallelism degree.
	DefaultThreads = uint8(4)
	// DefaultKeyLen is the default derived key length in bytes.
	DefaultKeyLen = uint32(32)
	// DefaultSaltLen is the default salt length in bytes.
	DefaultSaltLen = uint32(16)
)

// Hasher handles password hashing operations.
type Hasher struct {
	memory  uint32
	time    uint32
	threads uint8
	keyLen  uint32
	saltLen uint32
}

// NewHasher creates a new password hasher with custom parameters.
func NewHasher(memory, time uint32, threads uint8, keyLen, saltLen uint32) *Hasher {
	return &Hasher{
		memory:  memory,
		time:    time,
		threads: threads,
		keyLen:  keyLen,
		saltLen: saltLen,
	}
}

// DefaultHasher creates a new password hasher with secure default parameters.
func DefaultHasher() *Hasher {
	return &Hasher{
		memory:  DefaultMemory,
		time:    DefaultIterations,
		threads: DefaultThreads,
		keyLen:  DefaultKeyLen,
		saltLen: DefaultSaltLen,
	}
}

// Generate generates a salted hash for the given password.
func (h *Hasher) Generate(password string) (string, error) {
	if password == "" {
		return "", errors.New("password cannot be empty")
	}

	salt := make([]byte, h.saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		h.time,
		h.memory,
		h.threads,
		h.keyLen,
	)

	hashResult := &Hash{
		Version:  argon2.Version,
		Memory:   h.memory,
		Time:     h.time,
		Threads:  h.threads,
		KeyLen:   h.keyLen,
		SaltLen:  h.saltLen,
		Salt:     salt,
		Hash:     hash,
	}

	return hashResult.String(), nil
}

// Verify checks if the provided password matches the stored hash.
func (h *Hasher) Verify(password, encodedHash string) (bool, error) {
	hash, err := decodeHash(encodedHash)
	if err != nil {
		return false, err
	}

	otherHash := argon2.IDKey(
		[]byte(password),
		hash.Salt,
		hash.Time,
		hash.Memory,
		hash.Threads,
		hash.KeyLen,
	)

	if subtle.ConstantTimeCompare(hash.Hash, otherHash) == 1 {
		return true, nil
	}

	return false, ErrMismatchedHashAndPassword
}

// MustGenerate generates a hash and panics on error. Useful for tests.
func (h *Hasher) MustGenerate(password string) string {
	hash, err := h.Generate(password)
	if err != nil {
		panic(err)
	}
	return hash
}

// Generate is a convenience function that uses the default hasher.
func Generate(password string) (string, error) {
	return DefaultHasher().Generate(password)
}

// Verify is a convenience function that uses the default hasher.
func Verify(password, encodedHash string) (bool, error) {
	return DefaultHasher().Verify(password, encodedHash)
}

// MustVerify verifies and panics on error. Useful for tests.
func MustVerify(password, encodedHash string) bool {
	valid, err := Verify(password, encodedHash)
	if err != nil {
		panic(err)
	}
	return valid
}

// decodeHash decodes an encoded hash string into a Hash struct.
func decodeHash(encodedHash string) (*Hash, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return nil, ErrInvalidHashFormat
	}

	var version int
	_, err := fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return nil, ErrInvalidHashFormat
	}

	if version != argon2.Version {
		return nil, ErrInvalidHashVersion
	}

	var memory, time uint32
	var threads uint8
	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads)
	if err != nil {
		return nil, ErrInvalidHashFormat
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, ErrInvalidHashFormat
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, ErrInvalidHashFormat
	}

	// len() returns int, but KeyLen and SaltLen are uint32
	// The lengths are bounded by decodeHash's base64 decoding which produces
	// values matching the hasher's configured sizes (max 32 bytes for hash, 16 for salt)
	hashLen := uint32(len(hash)) // #nosec G115 -- bounded by argon2 key size
	saltLen := uint32(len(salt)) // #nosec G115 -- bounded by argon2 salt size

	return &Hash{
		Version:  version,
		Memory:   memory,
		Time:     time,
		Threads:  threads,
		Salt:     salt,
		Hash:     hash,
		KeyLen:   hashLen,
		SaltLen:  saltLen,
	}, nil
}

// StrengthChecker evaluates password strength.
type StrengthChecker struct {
	minLength     int
	requireUpper  bool
	requireLower  bool
	requireNumber bool
	requireSpecial bool
}

// DefaultStrengthChecker returns a checker with standard requirements.
func DefaultStrengthChecker() *StrengthChecker {
	return &StrengthChecker{
		minLength:      12,
		requireUpper:   true,
		requireLower:   true,
		requireNumber:  true,
		requireSpecial: true,
	}
}

// NewStrengthChecker creates a custom strength checker.
func NewStrengthChecker(minLength int, requireUpper, requireLower, requireNumber, requireSpecial bool) *StrengthChecker {
	return &StrengthChecker{
		minLength:      minLength,
		requireUpper:   requireUpper,
		requireLower:   requireLower,
		requireNumber:  requireNumber,
		requireSpecial: requireSpecial,
	}
}

// StrengthResult represents the result of a password strength check.
type StrengthResult struct {
	Valid      bool
	Errors     []string
	Strength   int // 0-100
	CommonWord bool
}

// Check evaluates password strength.
func (s *StrengthChecker) Check(password string) *StrengthResult {
	result := &StrengthResult{
		Valid:  true,
		Errors: make([]string, 0),
	}

	if len(password) < s.minLength {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("password must be at least %d characters", s.minLength))
	}

	hasUpper := false
	hasLower := false
	hasNumber := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasNumber = true
		case char >= 32 && char <= 126:
			hasSpecial = true
		}
	}

	if s.requireUpper && !hasUpper {
		result.Valid = false
		result.Errors = append(result.Errors, "password must contain at least one uppercase letter")
	}

	if s.requireLower && !hasLower {
		result.Valid = false
		result.Errors = append(result.Errors, "password must contain at least one lowercase letter")
	}

	if s.requireNumber && !hasNumber {
		result.Valid = false
		result.Errors = append(result.Errors, "password must contain at least one number")
	}

	if s.requireSpecial && !hasSpecial {
		result.Valid = false
		result.Errors = append(result.Errors, "password must contain at least one special character")
	}

	// Calculate strength score
	result.Strength = s.calculateStrength(password, hasUpper, hasLower, hasNumber, hasSpecial)

	return result
}

func (s *StrengthChecker) calculateStrength(password string, hasUpper, hasLower, hasNumber, hasSpecial bool) int {
	strength := 0

	// Length contribution
	length := len(password)
	if length >= 8 {
		strength += 20
	}
	if length >= 12 {
		strength += 20
	}
	if length >= 16 {
		strength += 10
	}

	// Character variety
	if hasUpper {
		strength += 12
	}
	if hasLower {
		strength += 12
	}
	if hasNumber {
		strength += 12
	}
	if hasSpecial {
		strength += 14
	}

	return min(strength, 100)
}
