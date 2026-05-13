package security

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
)

type DeviceBoundKey struct {
	DeviceID    string
	keyMaterial []byte
	boundSalt   []byte
}

func NewDeviceBoundKey(deviceID string) (*DeviceBoundKey, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("salt: %w", err)
	}

	material := make([]byte, 32)
	if _, err := rand.Read(material); err != nil {
		return nil, fmt.Errorf("material: %w", err)
	}

	deviceHash := sha256.Sum256([]byte(deviceID))
	for i := range material {
		material[i] ^= deviceHash[i%len(deviceHash)]
	}

	return &DeviceBoundKey{
		DeviceID:    deviceID,
		keyMaterial: material,
		boundSalt:   salt,
	}, nil
}

func (dbk *DeviceBoundKey) DeriveKey() []byte {
	deviceHash := sha256.Sum256([]byte(dbk.DeviceID))
	derived := make([]byte, 32)
	copy(derived, dbk.keyMaterial)
	for i := range derived {
		derived[i] ^= deviceHash[i%len(deviceHash)]
	}
	final := sha256.Sum256(derived)
	return final[:]
}