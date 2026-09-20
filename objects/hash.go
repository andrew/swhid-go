package objects

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/pjbgf/sha1cd"
)

var ErrSHA1Collision = errors.New("SHA-1 collision detected")
var ErrObjectHashMismatch = errors.New("object hash does not match content")

const (
	objectHashBytes = 20
	copyBufferSize  = 32 * 1024
	decimalBase     = 10
	hexNibbleBits   = 4
)

type copyBuffer [copyBufferSize]byte

type readerOnly struct {
	io.Reader
}

var copyBufferPool = sync.Pool{
	New: func() any {
		return new(copyBuffer)
	},
}

func computeObjectHash(objectType string, size int64, content io.Reader) (string, error) {
	if size < 0 {
		return "", errors.New("object size cannot be negative")
	}
	h := sha1cd.New().(sha1cd.CollisionResistantHash)
	var headerStorage [64]byte
	header := append(headerStorage[:0], objectType...)
	header = append(header, ' ')
	header = strconv.AppendInt(header, size, decimalBase)
	header = append(header, 0)
	if _, err := h.Write(header); err != nil {
		return "", err
	}
	var written int64
	var err error
	switch reader := content.(type) {
	case *bytes.Reader:
		written, err = reader.WriteTo(h)
	case *strings.Reader:
		written, err = reader.WriteTo(h)
	default:
		buffer := copyBufferPool.Get().(*copyBuffer)
		written, err = io.CopyBuffer(h, readerOnly{Reader: content}, buffer[:])
		copyBufferPool.Put(buffer)
	}
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
	if _, ok := decodeObjectID(value); !ok {
		return fmt.Errorf("invalid %s hash", field)
	}
	return nil
}

func decodeObjectID(value string) ([objectHashBytes]byte, bool) {
	var decoded [objectHashBytes]byte
	if len(value) != objectHashBytes*2 {
		return decoded, false
	}
	for i := range decoded {
		high, ok := hexNibble(value[i*2])
		if !ok {
			return decoded, false
		}
		low, ok := hexNibble(value[i*2+1])
		if !ok {
			return decoded, false
		}
		decoded[i] = high<<hexNibbleBits | low
	}
	return decoded, true
}

func hexNibble(value byte) (byte, bool) {
	switch {
	case value >= '0' && value <= '9':
		return value - '0', true
	case value >= 'a' && value <= 'f':
		return value - 'a' + decimalBase, true
	default:
		return 0, false
	}
}

func validateExtraHeaders(headers [][2]string) error {
	for _, header := range headers {
		if header[0] == "" || strings.ContainsAny(header[0], " \n") {
			return fmt.Errorf("invalid extra header name %q", header[0])
		}
	}
	return nil
}
