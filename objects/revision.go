package objects

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"
)

// RevisionMetadata contains the metadata for a revision (commit).
type RevisionMetadata struct {
	Directory          string   // 40-char hex hash of the tree
	Parents            []string // 40-char hex hashes of parent commits
	Author             string   // "Name <email>" format
	AuthorTimestamp    int64    // Unix timestamp
	AuthorTimezone     string   // "+0000" format
	Committer          string   // "Name <email>" format
	CommitterTimestamp int64    // Unix timestamp
	CommitterTimezone  string   // "+0000" format
	Message            string
	ExtraHeaders       [][2]string // Additional headers like gpgsig
}

// ComputeRevisionHash computes the Git commit hash for a revision.
func ComputeRevisionHash(meta RevisionMetadata) string {
	serialized := serializeRevision(meta)
	header := fmt.Sprintf("commit %d\x00", len(serialized))

	h := sha1.New()
	h.Write([]byte(header))
	h.Write(serialized)
	return hex.EncodeToString(h.Sum(nil))
}

func serializeRevision(meta RevisionMetadata) []byte {
	var lines []string

	// Tree
	lines = append(lines, "tree "+meta.Directory)

	// Parents
	for _, parent := range meta.Parents {
		lines = append(lines, "parent "+parent)
	}

	// Author
	authorTz := meta.AuthorTimezone
	if authorTz == "" {
		authorTz = "+0000"
	}
	authorEscaped := escapeNewlines(meta.Author)
	lines = append(lines, fmt.Sprintf("author %s %d %s", authorEscaped, meta.AuthorTimestamp, authorTz))

	// Committer
	committerTz := meta.CommitterTimezone
	if committerTz == "" {
		committerTz = "+0000"
	}
	committerEscaped := escapeNewlines(meta.Committer)
	lines = append(lines, fmt.Sprintf("committer %s %d %s", committerEscaped, meta.CommitterTimestamp, committerTz))

	// Extra headers
	for _, header := range meta.ExtraHeaders {
		lines = append(lines, formatHeaderLine(header[0], header[1]))
	}

	result := strings.Join(lines, "\n") + "\n"

	// Message (after blank line)
	if meta.Message != "" {
		result += "\n" + meta.Message
	}

	return []byte(result)
}

func escapeNewlines(s string) string {
	return strings.ReplaceAll(s, "\n", "\n ")
}

func formatHeaderLine(key, value string) string {
	valueEscaped := escapeNewlines(value)
	return key + " " + valueEscaped
}
