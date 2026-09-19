package objects

import (
	"bytes"
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
	MessagePresent     bool
	ExtraHeaders       [][2]string // Additional headers like gpgsig
}

// ComputeRevisionHash computes the Git commit hash for a revision.
func ComputeRevisionHash(meta RevisionMetadata) (string, error) {
	serialized, err := serializeRevision(meta)
	if err != nil {
		return "", err
	}
	return computeObjectHash("commit", int64(len(serialized)), bytes.NewReader(serialized))
}

func serializeRevision(meta RevisionMetadata) ([]byte, error) {
	if err := validateObjectID("directory", meta.Directory); err != nil {
		return nil, err
	}
	for _, parent := range meta.Parents {
		if err := validateObjectID("parent", parent); err != nil {
			return nil, err
		}
	}
	if err := validateExtraHeaders(meta.ExtraHeaders); err != nil {
		return nil, err
	}

	var lines []string

	// Tree
	lines = append(lines, "tree "+meta.Directory)

	// Parents
	for _, parent := range meta.Parents {
		lines = append(lines, "parent "+parent)
	}

	// Author
	authorEscaped := escapeNewlines(meta.Author)
	lines = append(lines, fmt.Sprintf("author %s %d %s", authorEscaped, meta.AuthorTimestamp, escapeNewlines(meta.AuthorTimezone)))

	// Committer
	committerEscaped := escapeNewlines(meta.Committer)
	lines = append(lines, fmt.Sprintf("committer %s %d %s", committerEscaped, meta.CommitterTimestamp, escapeNewlines(meta.CommitterTimezone)))

	// Extra headers
	for _, header := range meta.ExtraHeaders {
		lines = append(lines, formatHeaderLine(header[0], header[1]))
	}

	result := strings.Join(lines, "\n") + "\n"

	if meta.MessagePresent || meta.Message != "" {
		result += "\n" + meta.Message
	}

	return []byte(result), nil
}

func escapeNewlines(s string) string {
	return strings.ReplaceAll(s, "\n", "\n ")
}

func formatHeaderLine(key, value string) string {
	valueEscaped := escapeNewlines(value)
	return key + " " + valueEscaped
}
