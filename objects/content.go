package objects

import (
	"bytes"
	"io"
)

// ComputeContentHash computes the Git blob hash for file content.
// The hash is computed using Git's blob format: "blob <size>\0<content>"
func ComputeContentHash(data []byte) (string, error) {
	return ComputeContentHashReader(bytes.NewReader(data), int64(len(data)))
}

// ComputeContentHashReader computes a content hash without loading the content into memory.
func ComputeContentHashReader(content io.Reader, size int64) (string, error) {
	if size < 0 {
		return "", io.ErrUnexpectedEOF
	}
	return computeObjectHash("blob", size, content)
}
