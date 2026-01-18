package objects

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
)

// ComputeContentHash computes the Git blob hash for file content.
// The hash is computed using Git's blob format: "blob <size>\0<content>"
func ComputeContentHash(data []byte) string {
	header := fmt.Sprintf("blob %d\x00", len(data))
	h := sha1.New()
	h.Write([]byte(header))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}
