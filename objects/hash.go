package objects

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/pjbgf/sha1cd"
)

var ErrSHA1Collision = errors.New("SHA-1 collision detected")
var ErrObjectHashMismatch = errors.New("object hash does not match content")

func computeObjectHash(objectType string, size int64, content io.Reader) (string, error) {
	if size < 0 {
		return "", errors.New("object size cannot be negative")
	}
	h := sha1cd.New().(sha1cd.CollisionResistantHash)
	if _, err := fmt.Fprintf(h, "%s %d\x00", objectType, size); err != nil {
		return "", err
	}
	written, err := io.Copy(h, content)
	if err != nil {
		return "", err
	}
	if written != size {
		return "", fmt.Errorf("content size is %d bytes, want %d", written, size)
	}
	digest, collision := h.CollisionResistantSum(nil)
	if collision {
		return "", ErrSHA1Collision
	}
	return hex.EncodeToString(digest), nil
}

// VerifyObjectHash checks an object's expected hash using collision-detecting SHA-1.
func VerifyObjectHash(objectType string, size int64, content io.Reader, expected string) error {
	if err := validateObjectID("expected", expected); err != nil {
		return err
	}
	actual, err := computeObjectHash(objectType, size, content)
	if err != nil {
		return err
	}
	if actual != expected {
		return fmt.Errorf("%w: got %s, want %s", ErrObjectHashMismatch, actual, expected)
	}
	return nil
}

func validateObjectID(field, value string) error {
	if value != strings.ToLower(value) {
		return fmt.Errorf("invalid %s hash", field)
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != 20 {
		return fmt.Errorf("invalid %s hash", field)
	}
	return nil
}

func validateExtraHeaders(headers [][2]string) error {
	for _, header := range headers {
		if header[0] == "" || strings.ContainsAny(header[0], " \n") {
			return fmt.Errorf("invalid extra header name %q", header[0])
		}
	}
	return nil
}
