package swhid

import (
	"testing"

	"github.com/andrew/swhid-go/objects"
)

func TestFromContent(t *testing.T) {
	id := FromContent([]byte("hello\n"))

	if id.ObjectType != ObjectTypeContent {
		t.Errorf("FromContent() type = %v, want %v", id.ObjectType, ObjectTypeContent)
	}

	// Verified against Git: echo "hello" | git hash-object --stdin
	wantHash := "ce013625030ba8dba906f756967f9e9ca394464a"
	if id.ObjectHash != wantHash {
		t.Errorf("FromContent() hash = %v, want %v", id.ObjectHash, wantHash)
	}
}

func TestFromDirectory(t *testing.T) {
	entries := []objects.DirectoryEntry{
		{
			Name:   "hello.txt",
			Type:   objects.EntryTypeFile,
			Target: "ce013625030ba8dba906f756967f9e9ca394464a",
		},
	}

	id := FromDirectory(entries)

	if id.ObjectType != ObjectTypeDirectory {
		t.Errorf("FromDirectory() type = %v, want %v", id.ObjectType, ObjectTypeDirectory)
	}

	// Verified against Git and Ruby implementation
	wantHash := "aaa96ced2d9a1c8e72c56b253a0e2fe78393feb7"
	if id.ObjectHash != wantHash {
		t.Errorf("FromDirectory() hash = %v, want %v", id.ObjectHash, wantHash)
	}
}

func TestFromRevisionMetadata(t *testing.T) {
	meta := objects.RevisionMetadata{
		Directory:          "4b825dc642cb6eb9a060e54bf8d69288fbee4904",
		Author:             "Test <test@example.com>",
		AuthorTimestamp:    1000000000,
		AuthorTimezone:     "+0000",
		Committer:          "Test <test@example.com>",
		CommitterTimestamp: 1000000000,
		CommitterTimezone:  "+0000",
		Message:            "Test\n",
	}

	id := FromRevisionMetadata(meta)

	if id.ObjectType != ObjectTypeRevision {
		t.Errorf("FromRevisionMetadata() type = %v, want %v", id.ObjectType, ObjectTypeRevision)
	}

	if len(id.ObjectHash) != 40 {
		t.Errorf("FromRevisionMetadata() hash length = %d, want 40", len(id.ObjectHash))
	}
}

func TestFromReleaseMetadata(t *testing.T) {
	meta := objects.ReleaseMetadata{
		Name: "v1.0.0",
		Target: objects.ReleaseTarget{
			Hash: "4b825dc642cb6eb9a060e54bf8d69288fbee4904",
			Type: objects.TargetTypeRevision,
		},
		Message: "Release\n",
	}

	id := FromReleaseMetadata(meta)

	if id.ObjectType != ObjectTypeRelease {
		t.Errorf("FromReleaseMetadata() type = %v, want %v", id.ObjectType, ObjectTypeRelease)
	}

	if len(id.ObjectHash) != 40 {
		t.Errorf("FromReleaseMetadata() hash length = %d, want 40", len(id.ObjectHash))
	}
}

func TestFromSnapshotBranches(t *testing.T) {
	branches := []objects.Branch{
		{
			Name:       "refs/heads/main",
			TargetType: objects.BranchTargetRevision,
			Target:     "4b825dc642cb6eb9a060e54bf8d69288fbee4904",
		},
	}

	id := FromSnapshotBranches(branches)

	if id.ObjectType != ObjectTypeSnapshot {
		t.Errorf("FromSnapshotBranches() type = %v, want %v", id.ObjectType, ObjectTypeSnapshot)
	}

	if len(id.ObjectHash) != 40 {
		t.Errorf("FromSnapshotBranches() hash length = %d, want 40", len(id.ObjectHash))
	}
}
