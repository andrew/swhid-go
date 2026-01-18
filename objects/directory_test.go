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
			wantHash: "4b825dc642cb6eb9a060e54bf8d69288fbee4904",
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
			hash := ComputeDirectoryHash(tt.entries)
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
		{Name: "z", Type: EntryTypeFile, Target: "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391"},
		{Name: "a", Type: EntryTypeFile, Target: "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391"},
		{Name: "m", Type: EntryTypeDirectory, Target: "4b825dc642cb6eb9a060e54bf8d69288fbee4904"},
	}

	// Computing should produce a deterministic hash regardless of input order
	hash1 := ComputeDirectoryHash(entries)

	// Reverse order
	entries2 := []DirectoryEntry{
		{Name: "m", Type: EntryTypeDirectory, Target: "4b825dc642cb6eb9a060e54bf8d69288fbee4904"},
		{Name: "a", Type: EntryTypeFile, Target: "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391"},
		{Name: "z", Type: EntryTypeFile, Target: "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391"},
	}

	hash2 := ComputeDirectoryHash(entries2)

	if hash1 != hash2 {
		t.Errorf("Hash should be deterministic regardless of input order: %v != %v", hash1, hash2)
	}
}
