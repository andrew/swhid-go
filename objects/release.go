package objects

import (
	"bytes"
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
func (t ReleaseTarget) GitType() (string, error) {
	switch t.Type {
	case TargetTypeContent:
		return "blob", nil
	case TargetTypeDirectory:
		return "tree", nil
	case TargetTypeRevision:
		return "commit", nil
	case TargetTypeRelease:
		return "tag", nil
	default:
		return "", fmt.Errorf("invalid release target type %q", t.Type)
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
	MessagePresent  bool
}

// ComputeReleaseHash computes the Git tag hash for a release.
func ComputeReleaseHash(meta ReleaseMetadata) (string, error) {
	serialized, err := serializeRelease(meta)
	if err != nil {
		return "", err
	}
	return computeObjectHash("tag", int64(len(serialized)), bytes.NewReader(serialized))
}

func serializeRelease(meta ReleaseMetadata) ([]byte, error) {
	if err := validateObjectID("target", meta.Target.Hash); err != nil {
		return nil, err
	}
	gitType, err := meta.Target.GitType()
	if err != nil {
		return nil, err
	}

	var lines []string

	// Object (target hash)
	lines = append(lines, "object "+meta.Target.Hash)

	// Type
	lines = append(lines, "type "+gitType)

	// Tag name
	nameEscaped := escapeNewlines(meta.Name)
	lines = append(lines, "tag "+nameEscaped)

	// Tagger (optional)
	if meta.Author != "" {
		authorEscaped := escapeNewlines(meta.Author)
		lines = append(lines, fmt.Sprintf("tagger %s %d %s", authorEscaped, meta.AuthorTimestamp, escapeNewlines(meta.AuthorTimezone)))
	}

	result := strings.Join(lines, "\n") + "\n"

	if meta.MessagePresent || meta.Message != "" {
		result += "\n" + meta.Message
	}

	return []byte(result), nil
}
