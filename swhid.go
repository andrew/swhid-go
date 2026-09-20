// Package swhid computes and parses SoftWare Hash IDentifiers (SWHIDs).
//
// The package targets SWHID specification v1.2. It handles content, directories,
// revisions, releases, snapshots, and qualified identifiers.
//
// Basic usage:
//
//	id, err := swhid.FromContent([]byte("hello world"))
//
//	parsed, err := swhid.Parse("swh:1:cnt:94a9ed024d3859793618152ea559a168bbcbb5e2")
package swhid

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	Scheme                  = "swh"
	SchemeVersion           = 1
	ObjectIDLen             = 40
	schemeVersionString     = "1"
	qualifierRangePartCount = 2
	qualifierOrigin         = "origin"
	qualifierVisit          = "visit"
	qualifierAnchor         = "anchor"
	qualifierPath           = "path"
	qualifierLines          = "lines"
	qualifierBytes          = "bytes"
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

// Qualifier keys in canonical order.
var canonicalQualifierOrder = []string{
	qualifierOrigin,
	qualifierVisit,
	qualifierAnchor,
	qualifierPath,
	qualifierLines,
	qualifierBytes,
}

// Error types
var (
	ErrEmptySWHID        = errors.New("SWHID string cannot be nil or empty")
	ErrInvalidFormat     = errors.New("invalid SWHID format")
	ErrInvalidScheme     = errors.New("invalid scheme")
	ErrInvalidVersion    = errors.New("invalid version")
	ErrInvalidObjectType = errors.New("invalid object type")
	ErrInvalidObjectHash = errors.New("invalid object hash")
	ErrInvalidQualifier  = errors.New("invalid qualifier")
)

// Identifier represents a parsed SWHID.
type Identifier struct {
	Scheme     string
	Version    int
	ObjectType ObjectType
	ObjectHash string
	Qualifiers map[string]string
}

// MarshalText returns the canonical string form of a valid Identifier.
func (id *Identifier) MarshalText() ([]byte, error) {
	if id == nil {
		return nil, ErrEmptySWHID
	}
	if id.Scheme != Scheme {
		return nil, fmt.Errorf("%w: %s", ErrInvalidScheme, id.Scheme)
	}
	if id.Version != SchemeVersion {
		return nil, fmt.Errorf("%w: %d", ErrInvalidVersion, id.Version)
	}
	validated, err := NewIdentifier(id.ObjectType, id.ObjectHash, id.Qualifiers)
	if err != nil {
		return nil, err
	}
	return []byte(validated.String()), nil
}

// UnmarshalText parses a SWHID into id.
func (id *Identifier) UnmarshalText(text []byte) error {
	if id == nil {
		return errors.New("cannot unmarshal SWHID into nil Identifier")
	}
	parsed, err := Parse(string(text))
	if err != nil {
		return err
	}
	*id = *parsed
	return nil
}

// NewIdentifier creates a new Identifier with validation.
func NewIdentifier(objectType ObjectType, objectHash string, qualifiers map[string]string) (*Identifier, error) {
	return newIdentifier(objectType, objectHash, qualifiers, true)
}

func newIdentifier(objectType ObjectType, objectHash string, qualifiers map[string]string, copyQualifiers bool) (*Identifier, error) {
	if !validObjectTypes[objectType] {
		return nil, fmt.Errorf("%w: %s", ErrInvalidObjectType, objectType)
	}

	if !validObjectHash(objectHash) {
		return nil, fmt.Errorf("%w: must be %d hex digits", ErrInvalidObjectHash, ObjectIDLen)
	}

	qualifiers, err := validateQualifiers(objectType, qualifiers, copyQualifiers)
	if err != nil {
		return nil, err
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
	if strings.IndexFunc(swhidString, unicode.IsSpace) >= 0 {
		return nil, fmt.Errorf("%w: whitespace is not allowed", ErrInvalidFormat)
	}

	corePart, qualifierString, hasQualifiers := strings.Cut(swhidString, ";")
	objectType, objectHash, err := parseCoreIdentifier(corePart)
	if err != nil {
		return nil, err
	}

	qualifiers := make(map[string]string, len(canonicalQualifierOrder))
	for hasQualifiers {
		part := qualifierString
		if nextPart, remaining, found := strings.Cut(qualifierString, ";"); found {
			part = nextPart
			qualifierString = remaining
		} else {
			hasQualifiers = false
		}
		if part == "" {
			return nil, fmt.Errorf("%w: empty qualifier", ErrInvalidQualifier)
		}
		idx := strings.Index(part, "=")
		if idx == -1 {
			return nil, fmt.Errorf("%w: %s", ErrInvalidQualifier, part)
		}
		key := part[:idx]
		if key == "" {
			return nil, fmt.Errorf("%w: empty key", ErrInvalidQualifier)
		}
		if _, exists := qualifiers[key]; exists {
			return nil, fmt.Errorf("%w: duplicate %s", ErrInvalidQualifier, key)
		}
		value, err := decodeQualifierValue(part[idx+1:])
		if err != nil {
			return nil, fmt.Errorf("%w %s: %v", ErrInvalidQualifier, key, err)
		}
		qualifiers[key] = value
	}

	return newIdentifier(objectType, objectHash, qualifiers, false)
}

func parseCoreIdentifier(value string) (ObjectType, string, error) {
	scheme, rest, ok := strings.Cut(value, ":")
	if !ok {
		return "", "", ErrInvalidFormat
	}
	version, rest, ok := strings.Cut(rest, ":")
	if !ok {
		return "", "", ErrInvalidFormat
	}
	objectTypeValue, objectHash, ok := strings.Cut(rest, ":")
	if !ok || strings.ContainsRune(objectHash, ':') {
		return "", "", ErrInvalidFormat
	}
	if scheme != Scheme {
		return "", "", fmt.Errorf("%w: %s", ErrInvalidScheme, scheme)
	}
	if version != schemeVersionString {
		return "", "", fmt.Errorf("%w: %s", ErrInvalidVersion, version)
	}
	objectType := ObjectType(objectTypeValue)
	if !validObjectTypes[objectType] {
		return "", "", fmt.Errorf("%w: %s", ErrInvalidObjectType, objectType)
	}
	if !validObjectHash(objectHash) {
		return "", "", fmt.Errorf("%w: must be %d hex digits", ErrInvalidObjectHash, ObjectIDLen)
	}
	return objectType, objectHash, nil
}

func validObjectHash(value string) bool {
	if len(value) != ObjectIDLen {
		return false
	}
	for i := range value {
		if value[i] < '0' || value[i] > '9' && (value[i] < 'a' || value[i] > 'f') {
			return false
		}
	}
	return true
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

// WithQualifiers returns a new validated Identifier with the given qualifiers.
func (id *Identifier) WithQualifiers(qualifiers map[string]string) (*Identifier, error) {
	return NewIdentifier(id.ObjectType, id.ObjectHash, qualifiers)
}

func formatQualifiers(quals map[string]string) string {
	var parts []string

	for _, key := range canonicalQualifierOrder {
		if value, ok := quals[key]; ok {
			parts = append(parts, key+"="+encodeQualifierValue(key, value))
		}
	}

	return strings.Join(parts, ";")
}

func encodeQualifierValue(key, value string) string {
	const hexDigits = "0123456789ABCDEF"
	needsEscaping := false
	for _, r := range value {
		if qualifierRuneNeedsEscaping(key, r) {
			needsEscaping = true
			break
		}
	}
	if !needsEscaping {
		return value
	}
	var encoded strings.Builder
	encoded.Grow(len(value))
	for len(value) > 0 {
		r, size := utf8.DecodeRuneInString(value)
		part := value[:size]
		if qualifierRuneNeedsEscaping(key, r) {
			for _, b := range []byte(part) {
				_ = encoded.WriteByte('%')
				_ = encoded.WriteByte(hexDigits[b>>4])
				_ = encoded.WriteByte(hexDigits[b&0x0f])
			}
		} else {
			_, _ = encoded.WriteString(part)
		}
		value = value[size:]
	}
	return encoded.String()
}

func qualifierRuneNeedsEscaping(key string, value rune) bool {
	return value == '%' || value == ';' || unicode.IsSpace(value) || key == qualifierPath && (value == '?' || value == '#')
}

func decodeQualifierValue(value string) (string, error) {
	return url.PathUnescape(value)
}

func validateQualifiers(objectType ObjectType, qualifiers map[string]string, copyInput bool) (map[string]string, error) {
	validated := qualifiers
	if copyInput || validated == nil {
		validated = make(map[string]string, len(qualifiers))
	}
	for key, value := range qualifiers {
		if !knownQualifier(key) {
			return nil, fmt.Errorf("%w: %q", ErrInvalidQualifier, key)
		}
		if !utf8.ValidString(value) || strings.IndexFunc(value, unicode.IsControl) >= 0 {
			return nil, fmt.Errorf("%w %s: value contains invalid characters", ErrInvalidQualifier, key)
		}
		if err := validateQualifier(key, value); err != nil {
			return nil, fmt.Errorf("%w %s: %v", ErrInvalidQualifier, key, err)
		}
		if copyInput {
			validated[key] = value
		}
	}

	_, hasLines := validated[qualifierLines]
	_, hasBytes := validated[qualifierBytes]
	_, hasPath := validated[qualifierPath]
	if hasLines && hasBytes {
		return nil, fmt.Errorf("%w: lines and bytes cannot be combined", ErrInvalidQualifier)
	}
	if objectType != ObjectTypeContent && (hasLines || hasBytes) {
		return nil, fmt.Errorf("%w: fragment qualifiers require content", ErrInvalidQualifier)
	}
	if objectType != ObjectTypeContent && objectType != ObjectTypeDirectory && hasPath {
		return nil, fmt.Errorf("%w: path requires content or directory", ErrInvalidQualifier)
	}
	return validated, nil
}

func validateQualifier(key, value string) error {
	switch key {
	case qualifierOrigin:
		parsed, err := url.ParseRequestURI(encodeQualifierValue(key, value))
		if err != nil || parsed.Scheme == "" {
			return errors.New("origin must be an absolute URI")
		}
	case qualifierVisit:
		return validateCoreQualifier(value, ObjectTypeSnapshot)
	case qualifierAnchor:
		objectType, _, err := parseCoreIdentifier(value)
		if err != nil {
			return errors.New("anchor must be a core SWHID")
		}
		if objectType == ObjectTypeContent {
			return errors.New("anchor cannot identify content")
		}
	case qualifierPath:
		if !strings.HasPrefix(value, "/") {
			return errors.New("path must be absolute")
		}
	case qualifierLines:
		return validateRange(value, 1)
	case qualifierBytes:
		return validateRange(value, 0)
	}
	return nil
}

func knownQualifier(key string) bool {
	for _, known := range canonicalQualifierOrder {
		if key == known {
			return true
		}
	}
	return false
}

func validateCoreQualifier(value string, objectType ObjectType) error {
	parsedType, _, err := parseCoreIdentifier(value)
	if err != nil || parsedType != objectType {
		return fmt.Errorf("must be a core %s SWHID", objectType)
	}
	return nil
}

func validateRange(value string, minimum uint64) error {
	parts := strings.Split(value, "-")
	if len(parts) > qualifierRangePartCount || len(parts) == 0 {
		return errors.New("invalid range")
	}
	start, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil || start < minimum {
		return errors.New("invalid range start")
	}
	if len(parts) == qualifierRangePartCount {
		end, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil || end < start {
			return errors.New("invalid range end")
		}
	}
	return nil
}
