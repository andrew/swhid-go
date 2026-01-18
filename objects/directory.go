package objects

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"sort"
)

// EntryType represents the type of a directory entry.
type EntryType int

const (
	EntryTypeFile EntryType = iota
	EntryTypeExecutable
	EntryTypeDirectory
	EntryTypeSymlink
	EntryTypeRevision // submodule
)

// DirectoryEntry represents an entry in a directory.
type DirectoryEntry struct {
	Name   string
	Type   EntryType
	Target string // 40-char hex hash
	Perms  string // optional, uses default if empty
}

// DefaultPerms returns the default Git permissions for an entry type.
func (e *DirectoryEntry) DefaultPerms() string {
	switch e.Type {
	case EntryTypeDirectory:
		return "40000"
	case EntryTypeFile:
		return "100644"
	case EntryTypeExecutable:
		return "100755"
	case EntryTypeSymlink:
		return "120000"
	case EntryTypeRevision:
		return "160000"
	default:
		return "100644"
	}
}

// Permissions returns the permissions string, using default if not set.
func (e *DirectoryEntry) Permissions() string {
	if e.Perms != "" {
		return e.Perms
	}
	return e.DefaultPerms()
}

// SortKey returns the key used for sorting entries.
// Directories are sorted as if they have a trailing slash.
func (e *DirectoryEntry) SortKey() string {
	if e.Type == EntryTypeDirectory {
		return e.Name + "/"
	}
	return e.Name
}

// ComputeDirectoryHash computes the Git tree hash for a directory.
func ComputeDirectoryHash(entries []DirectoryEntry) string {
	serialized := serializeEntries(entries)
	header := fmt.Sprintf("tree %d\x00", len(serialized))

	h := sha1.New()
	h.Write([]byte(header))
	h.Write(serialized)
	return hex.EncodeToString(h.Sum(nil))
}

func serializeEntries(entries []DirectoryEntry) []byte {
	// Sort entries by sort key
	sorted := make([]DirectoryEntry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].SortKey() < sorted[j].SortKey()
	})

	var result []byte
	for _, entry := range sorted {
		// Format: "<perms> <name>\0<binary_hash>"
		perms := entry.Permissions()
		result = append(result, []byte(perms)...)
		result = append(result, ' ')
		result = append(result, []byte(entry.Name)...)
		result = append(result, 0)

		// Convert hex hash to binary
		hashBytes, _ := hex.DecodeString(entry.Target)
		result = append(result, hashBytes...)
	}

	return result
}
