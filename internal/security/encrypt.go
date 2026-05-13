package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"sync"
)

type Encryptor struct {
	key []byte
	mu  sync.RWMutex
}

func NewEncryptor(key []byte) (*Encryptor, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes for AES-256, got %d", len(key))
	}
	return &Encryptor{key: key}, nil
}

func (e *Encryptor) Encrypt(plaintext []byte) ([]byte, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("new cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("new GCM: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("nonce: %w", err)
	}

	return aesGCM.Seal(nonce, nonce, plaintext, nil), nil
}

func (e *Encryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("new cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("new GCM: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	plaintext, err := aesGCM.Open(nil, ciphertext[:nonceSize], ciphertext[nonceSize:], nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	return plaintext, nil
}