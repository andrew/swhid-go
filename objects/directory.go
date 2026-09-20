package objects

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
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
func ComputeDirectoryHash(entries []DirectoryEntry) (string, error) {
	serialized, err := serializeEntries(entries)
	if err != nil {
		return "", err
	}
	return computeObjectHash("tree", int64(len(serialized)), bytes.NewReader(serialized))
}

func serializeEntries(entries []DirectoryEntry) ([]byte, error) {
	sorted := make([]DirectoryEntry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].SortKey() < sorted[j].SortKey()
	})

	var result []byte
	seen := make(map[string]struct{}, len(sorted))
	for _, entry := range sorted {
		if entry.Name == "" || strings.ContainsAny(entry.Name, "/\x00") {
			return nil, fmt.Errorf("invalid directory entry name %q", entry.Name)
		}
		switch entry.Type {
		case EntryTypeFile, EntryTypeExecutable, EntryTypeDirectory, EntryTypeSymlink, EntryTypeRevision:
		default:
			return nil, fmt.Errorf("invalid directory entry type %d for %q", entry.Type, entry.Name)
		}
		if _, exists := seen[entry.Name]; exists {
			return nil, fmt.Errorf("duplicate directory entry name %q", entry.Name)
		}
		seen[entry.Name] = struct{}{}

		perms := entry.Permissions()
		result = append(result, []byte(perms)...)
		result = append(result, ' ')
		result = append(result, []byte(entry.Name)...)
		result = append(result, 0)

		hashBytes, err := hex.DecodeString(entry.Target)
		if err != nil || len(hashBytes) != 20 {
			return nil, fmt.Errorf("invalid target hash for %q", entry.Name)
		}
		result = append(result, hashBytes...)
	}

	return result, nil
}
