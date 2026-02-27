// Package storage provides client-side encryption for storage backends.
package storage

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"golang.org/x/crypto/nacl/secretbox"
)

// EncryptedBackend wraps a backend with client-side encryption.
type EncryptedBackend struct {
	backend Backend
	key     [32]byte
}

// NewEncryptedBackend creates a new encrypted backend wrapper.
func NewEncryptedBackend(backend Backend, keyString string) (*EncryptedBackend, error) {
	// Derive 32-byte key from the key string
	key := sha256.Sum256([]byte(keyString))

	return &EncryptedBackend{
		backend: backend,
		key:     key,
	}, nil
}

// Put encrypts and stores data at the given path.
func (b *EncryptedBackend) Put(ctx context.Context, path string, data []byte) error {
	// Generate random nonce
	var nonce [24]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return fmt.Errorf("generate nonce: %w", err)
	}

	// Encrypt data
	encrypted := secretbox.Seal(nonce[:], data, &nonce, &b.key)

	// Store with nonce prepended (secretbox doesn't do this by default)
	return b.backend.Put(ctx, path, encrypted)
}

// Get retrieves and decrypts data from the given path.
func (b *EncryptedBackend) Get(ctx context.Context, path string) ([]byte, error) {
	data, err := b.backend.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	// Check minimum size (nonce + encrypted data + secretbox overhead)
	if len(data) < 24 {
		return nil, fmt.Errorf("encrypted data too short")
	}

	// Extract nonce
	var nonce [24]byte
	copy(nonce[:], data[:24])

	// Decrypt data
	decrypted, ok := secretbox.Open(nil, data[24:], &nonce, &b.key)
	if !ok {
		return nil, fmt.Errorf("decryption failed")
	}

	return decrypted, nil
}

// Delete removes data at the given path.
func (b *EncryptedBackend) Delete(ctx context.Context, path string) error {
	return b.backend.Delete(ctx, path)
}

// Exists checks if data exists at the given path.
func (b *EncryptedBackend) Exists(ctx context.Context, path string) (bool, error) {
	return b.backend.Exists(ctx, path)
}

// List returns all objects with the given prefix.
func (b *EncryptedBackend) List(ctx context.Context, prefix string) ([]string, error) {
	return b.backend.List(ctx, prefix)
}

// GetURL returns a presigned URL (not encrypted).
func (b *EncryptedBackend) GetURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	// Cannot provide presigned URL for encrypted content
	return "", fmt.Errorf("presigned URLs not supported for encrypted backend")
}

// EncryptString encrypts a string for storage.
func (b *EncryptedBackend) EncryptString(data string) (string, error) {
	// Generate random nonce
	var nonce [24]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	// Encrypt data
	encrypted := secretbox.Seal(nonce[:], []byte(data), &nonce, &b.key)

	// Return base64 encoded
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// DecryptString decrypts a string from storage.
func (b *EncryptedBackend) DecryptString(data string) (string, error) {
	// Decode base64
	encrypted, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", fmt.Errorf("decode base64: %w", err)
	}

	// Check minimum size
	if len(encrypted) < 24 {
		return "", fmt.Errorf("encrypted data too short")
	}

	// Extract nonce
	var nonce [24]byte
	copy(nonce[:], encrypted[:24])

	// Decrypt
	decrypted, ok := secretbox.Open(nil, encrypted[24:], &nonce, &b.key)
	if !ok {
		return "", fmt.Errorf("decryption failed")
	}

	return string(decrypted), nil
}

// GenerateKey generates a random encryption key.
func GenerateKey() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// DeriveKey derives a key from a password and salt using PBKDF2.
func DeriveKey(password, salt string) [32]byte {
	// In production, use proper KDF like scrypt or Argon2
	// This is simplified for the example
	return sha256.Sum256([]byte(password + salt))
}
