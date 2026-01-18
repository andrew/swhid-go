package objects

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"
)

// TargetType represents the SWHID object type.
type TargetType string

const (
	TargetTypeContent   TargetType = "cnt"
	TargetTypeDirectory TargetType = "dir"
	TargetTypeRevision  TargetType = "rev"
	TargetTypeRelease   TargetType = "rel"
	TargetTypeSnapshot  TargetType = "snp"
)

// ReleaseTarget represents the target of a release.
type ReleaseTarget struct {
	Hash string     // 40-char hex hash
	Type TargetType // cnt, dir, rev, rel, snp
}

// GitType returns the Git object type name.
func (t ReleaseTarget) GitType() string {
	switch t.Type {
	case TargetTypeContent:
		return "blob"
	case TargetTypeDirectory:
		return "tree"
	case TargetTypeRevision:
		return "commit"
	case TargetTypeRelease:
		return "tag"
	case TargetTypeSnapshot:
		return "snapshot"
	default:
		return "commit"
	}
}

// ReleaseMetadata contains the metadata for a release (tag).
type ReleaseMetadata struct {
	Name            string
	Target          ReleaseTarget
	Author          string // "Name <email>" format, optional
	AuthorTimestamp int64  // Unix timestamp, required if Author is set
	AuthorTimezone  string // "+0000" format
	Message         string
	ExtraHeaders    [][2]string // Additional headers like gpgsig
}

// ComputeReleaseHash computes the Git tag hash for a release.
func ComputeReleaseHash(meta ReleaseMetadata) string {
	serialized := serializeRelease(meta)
	header := fmt.Sprintf("tag %d\x00", len(serialized))

	h := sha1.New()
	h.Write([]byte(header))
	h.Write(serialized)
	return hex.EncodeToString(h.Sum(nil))
}

func serializeRelease(meta ReleaseMetadata) []byte {
	var lines []string

	// Object (target hash)
	lines = append(lines, "object "+meta.Target.Hash)

	// Type
	lines = append(lines, "type "+meta.Target.GitType())

	// Tag name
	nameEscaped := escapeNewlines(meta.Name)
	lines = append(lines, "tag "+nameEscaped)

	// Tagger (optional)
	if meta.Author != "" {
		tz := meta.AuthorTimezone
		if tz == "" {
			tz = "+0000"
		}
		authorEscaped := escapeNewlines(meta.Author)
		lines = append(lines, fmt.Sprintf("tagger %s %d %s", authorEscaped, meta.AuthorTimestamp, tz))
	}

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
