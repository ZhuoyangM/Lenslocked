package rand

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// Generate a random byte slice with given size
func Bytes(numBytes int) ([]byte, error) {
	b := make([]byte, numBytes)
	numRead, err := rand.Read(b)
	if err != nil {
		return nil, fmt.Errorf("Bytes: %w", err)
	}
	if numRead < numBytes {
		return nil, fmt.Errorf("Bytes: didn't read enough random bytes")
	}
	return b, nil
}

// Generate a random string with given size
func String(numBytes int) (string, error) {
	b, err := Bytes(numBytes)
	if err != nil {
		return "", fmt.Errorf("String: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
