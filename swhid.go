// Package swhid provides functionality for computing and parsing Software Heritage Identifiers (SWHIDs).
//
// SWHIDs are intrinsic identifiers for digital objects (source code files, directories, commits, etc.)
// based on cryptographic hashes. This package implements the SWHID specification v1.
//
// Basic usage:
//
//	// Compute SWHID for file content
//	id := swhid.FromContent([]byte("hello world"))
//	fmt.Println(id) // swh:1:cnt:...
//
//	// Parse an existing SWHID
//	id, err := swhid.Parse("swh:1:cnt:94a9ed024d3859793618152ea559a168bbcbb5e2")
package swhid

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

const (
	Scheme        = "swh"
	SchemeVersion = 1
	ObjectIDLen   = 40
)

// ObjectType represents the type of object identified by a SWHID.
type ObjectType string

const (
	ObjectTypeContent   ObjectType = "cnt"
	ObjectTypeDirectory ObjectType = "dir"
	ObjectTypeRevision  ObjectType = "rev"
	ObjectTypeRelease   ObjectType = "rel"
	ObjectTypeSnapshot  ObjectType = "snp"
)

var validObjectTypes = map[ObjectType]bool{
	ObjectTypeContent:   true,
	ObjectTypeDirectory: true,
	ObjectTypeRevision:  true,
	ObjectTypeRelease:   true,
	ObjectTypeSnapshot:  true,
}

var hashRegex = regexp.MustCompile(`^[0-9a-f]{40}$`)

// Qualifier keys in canonical order.
var canonicalQualifierOrder = []string{"origin", "visit", "anchor", "path", "lines", "bytes"}

// Error types
var (
	ErrEmptySWHID        = errors.New("SWHID string cannot be nil or empty")
	ErrInvalidFormat     = errors.New("invalid SWHID format")
	ErrInvalidScheme     = errors.New("invalid scheme")
	ErrInvalidVersion    = errors.New("invalid version")
	ErrInvalidObjectType = errors.New("invalid object type")
	ErrInvalidObjectHash = errors.New("invalid object hash")
)

// Identifier represents a parsed SWHID.
type Identifier struct {
	Scheme     string
	Version    int
	ObjectType ObjectType
	ObjectHash string
	Qualifiers map[string]string
}

// NewIdentifier creates a new Identifier with validation.
func NewIdentifier(objectType ObjectType, objectHash string, qualifiers map[string]string) (*Identifier, error) {
	if !validObjectTypes[objectType] {
		return nil, fmt.Errorf("%w: %s", ErrInvalidObjectType, objectType)
	}

	if !hashRegex.MatchString(objectHash) {
		return nil, fmt.Errorf("%w: must be %d hex digits", ErrInvalidObjectHash, ObjectIDLen)
	}

	if qualifiers == nil {
		qualifiers = make(map[string]string)
	}

	return &Identifier{
		Scheme:     Scheme,
		Version:    SchemeVersion,
		ObjectType: objectType,
		ObjectHash: objectHash,
		Qualifiers: qualifiers,
	}, nil
}

// Parse parses a SWHID string into an Identifier.
func Parse(swhidString string) (*Identifier, error) {
	if swhidString == "" {
		return nil, ErrEmptySWHID
	}

	// Split core part from qualifiers
	parts := strings.Split(swhidString, ";")
	corePart := parts[0]
	qualifierParts := parts[1:]

	// Parse core part
	coreParts := strings.Split(corePart, ":")
	if len(coreParts) != 4 {
		return nil, ErrInvalidFormat
	}

	scheme := coreParts[0]
	versionStr := coreParts[1]
	objectType := ObjectType(coreParts[2])
	objectHash := coreParts[3]

	if scheme != Scheme {
		return nil, fmt.Errorf("%w: %s", ErrInvalidScheme, scheme)
	}

	if versionStr != "1" {
		return nil, fmt.Errorf("%w: %s", ErrInvalidVersion, versionStr)
	}

	if !validObjectTypes[objectType] {
		return nil, fmt.Errorf("%w: %s", ErrInvalidObjectType, objectType)
	}

	if !hashRegex.MatchString(objectHash) {
		return nil, fmt.Errorf("%w: must be %d hex digits", ErrInvalidObjectHash, ObjectIDLen)
	}

	// Parse qualifiers
	qualifiers := make(map[string]string)
	for _, part := range qualifierParts {
		if part == "" {
			continue
		}
		idx := strings.Index(part, "=")
		if idx == -1 {
			continue
		}
		key := part[:idx]
		value := part[idx+1:]
		qualifiers[key] = decodeQualifierValue(value)
	}

	return &Identifier{
		Scheme:     Scheme,
		Version:    SchemeVersion,
		ObjectType: objectType,
		ObjectHash: objectHash,
		Qualifiers: qualifiers,
	}, nil
}

// String returns the canonical SWHID string representation.
func (id *Identifier) String() string {
	core := id.CoreSWHID()
	if len(id.Qualifiers) == 0 {
		return core
	}

	qualifierStr := formatQualifiers(id.Qualifiers)
	return core + ";" + qualifierStr
}

// CoreSWHID returns the core SWHID without qualifiers.
func (id *Identifier) CoreSWHID() string {
	return fmt.Sprintf("%s:%d:%s:%s", id.Scheme, id.Version, id.ObjectType, id.ObjectHash)
}

// Equal returns true if two identifiers are equal.
func (id *Identifier) Equal(other *Identifier) bool {
	if other == nil {
		return false
	}
	if id.CoreSWHID() != other.CoreSWHID() {
		return false
	}
	if len(id.Qualifiers) != len(other.Qualifiers) {
		return false
	}
	for k, v := range id.Qualifiers {
		if other.Qualifiers[k] != v {
			return false
		}
	}
	return true
}

// WithQualifiers returns a new Identifier with the given qualifiers.
func (id *Identifier) WithQualifiers(qualifiers map[string]string) *Identifier {
	return &Identifier{
		Scheme:     id.Scheme,
		Version:    id.Version,
		ObjectType: id.ObjectType,
		ObjectHash: id.ObjectHash,
		Qualifiers: qualifiers,
	}
}

func formatQualifiers(quals map[string]string) string {
	var parts []string

	// Add qualifiers in canonical order first
	for _, key := range canonicalQualifierOrder {
		if value, ok := quals[key]; ok {
			parts = append(parts, key+"="+encodeQualifierValue(value))
		}
	}

	// Add remaining qualifiers
	for key, value := range quals {
		isCanonical := false
		for _, ck := range canonicalQualifierOrder {
			if key == ck {
				isCanonical = true
				break
			}
		}
		if !isCanonical {
			parts = append(parts, key+"="+encodeQualifierValue(value))
		}
	}

	return strings.Join(parts, ";")
}

func encodeQualifierValue(value string) string {
	// Encode semicolons and percent signs
	value = strings.ReplaceAll(value, "%", "%25")
	value = strings.ReplaceAll(value, ";", "%3B")
	return value
}

func decodeQualifierValue(value string) string {
	// Decode URL-encoded values
	decoded, err := url.QueryUnescape(value)
	if err != nil {
		return value
	}
	return decoded
}
