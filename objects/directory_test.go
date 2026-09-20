package objects

import (
	"testing"
)

func TestComputeDirectoryHash(t *testing.T) {
	tests := []struct {
		name     string
		entries  []DirectoryEntry
		wantHash string
	}{
		{
			name:     "empty directory",
			entries:  []DirectoryEntry{},
			wantHash: emptyTreeHash,
		},
		{
			name: "single file with hello content",
			entries: []DirectoryEntry{
				{
					Name:   "hello.txt",
					Type:   EntryTypeFile,
					Target: "ce013625030ba8dba906f756967f9e9ca394464a", // "hello\n"
				},
			},
			// Verified against Git and Ruby implementation
			wantHash: "aaa96ced2d9a1c8e72c56b253a0e2fe78393feb7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := ComputeDirectoryHash(tt.entries)
			if err != nil {
				t.Fatalf("ComputeDirectoryHash() error = %v", err)
			}
			if hash != tt.wantHash {
				t.Errorf("ComputeDirectoryHash() = %v, want %v", hash, tt.wantHash)
			}
		})
	}
}

func TestDirectoryEntrySortKey(t *testing.T) {
	tests := []struct {
		name  string
		entry DirectoryEntry
		want  string
	}{
		{
			name:  "regular file",
			entry: DirectoryEntry{Name: "file.txt", Type: EntryTypeFile},
			want:  "file.txt",
		},
		{
			name:  "directory",
			entry: DirectoryEntry{Name: "dir", Type: EntryTypeDirectory},
			want:  "dir/",
		},
		{
			name:  "executable",
			entry: DirectoryEntry{Name: "script.sh", Type: EntryTypeExecutable},
			want:  "script.sh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.entry.SortKey(); got != tt.want {
				t.Errorf("SortKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDirectoryEntryDefaultPerms(t *testing.T) {
	tests := []struct {
		entryType EntryType
		wantPerms string
	}{
		{EntryTypeFile, "100644"},
		{EntryTypeExecutable, "100755"},
		{EntryTypeDirectory, "40000"},
		{EntryTypeSymlink, "120000"},
		{EntryTypeRevision, "160000"},
	}

	for _, tt := range tests {
		entry := DirectoryEntry{Type: tt.entryType}
		if got := entry.DefaultPerms(); got != tt.wantPerms {
			t.Errorf("DefaultPerms() for %v = %v, want %v", tt.entryType, got, tt.wantPerms)
		}
	}
}

func TestDirectoryEntrySorting(t *testing.T) {
	// Entries should be sorted by name, with directories having trailing /
	entries := []DirectoryEntry{
		{Name: "z", Type: EntryTypeFile, Target: emptyBlobHash},
		{Name: "a", Type: EntryTypeFile, Target: emptyBlobHash},
		{Name: "m", Type: EntryTypeDirectory, Target: emptyTreeHash},
	}

	// Computing should produce a deterministic hash regardless of input order
	hash1, err := ComputeDirectoryHash(entries)
	if err != nil {
		t.Fatalf("ComputeDirectoryHash() error = %v", err)
	}

	// Reverse order
	entries2 := []DirectoryEntry{
		{Name: "m", Type: EntryTypeDirectory, Target: emptyTreeHash},
		{Name: "a", Type: EntryTypeFile, Target: emptyBlobHash},
		{Name: "z", Type: EntryTypeFile, Target: emptyBlobHash},
	}

	hash2, err := ComputeDirectoryHash(entries2)
	if err != nil {
		t.Fatalf("ComputeDirectoryHash() error = %v", err)
	}

	if hash1 != hash2 {
		t.Errorf("Hash should be deterministic regardless of input order: %v != %v", hash1, hash2)
	}
}

func TestComputeDirectoryHashPreservesExplicitPermissions(t *testing.T) {
	entries := []DirectoryEntry{{
		Name:   "file",
		Type:   EntryTypeFile,
		Target: emptyBlobHash,
		Perms:  "100664",
	}}

	hash, err := ComputeDirectoryHash(entries)
	if err != nil {
		t.Fatalf("ComputeDirectoryHash() error = %v", err)
	}
	const want = "df143af729209e32ee1bcfb027177885e78eac09"
	if hash != want {
		t.Errorf("ComputeDirectoryHash() = %s, want %s", hash, want)
	}
}

func TestComputeDirectoryHashRejectsInvalidEntries(t *testing.T) {
	tests := []struct {
		name    string
		entries []DirectoryEntry
	}{
		{name: "duplicate name", entries: []DirectoryEntry{{Name: testEntryName, Target: emptyBlobHash}, {Name: testEntryName, Target: emptyBlobHash}}},
		{name: "slash in name", entries: []DirectoryEntry{{Name: "dir/file", Target: emptyBlobHash}}},
		{name: "invalid target", entries: []DirectoryEntry{{Name: testEntryName, Target: invalidObjectHash}}},
		{name: "invalid type", entries: []DirectoryEntry{{Name: testEntryName, Type: EntryType(99), Target: emptyBlobHash}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ComputeDirectoryHash(tt.entries); err == nil {
				t.Fatal("ComputeDirectoryHash() expected error")
			}
		})
	}
}
