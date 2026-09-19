package swhid

import (
	"testing"

	"github.com/andrew/swhid-go/objects"
)

func TestFromContent(t *testing.T) {
	id, err := FromContent([]byte("hello\n"))
	if err != nil {
		t.Fatalf("FromContent() error = %v", err)
	}

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

	id, err := FromDirectory(entries)
	if err != nil {
		t.Fatalf("FromDirectory() error = %v", err)
	}

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
		Directory:          emptyTreeHash,
		Author:             "Test <test@example.com>",
		AuthorTimestamp:    1000000000,
		AuthorTimezone:     "+0000",
		Committer:          "Test <test@example.com>",
		CommitterTimestamp: 1000000000,
		CommitterTimezone:  "+0000",
		Message:            "Test\n",
	}

	id, err := FromRevisionMetadata(meta)
	if err != nil {
		t.Fatalf("FromRevisionMetadata() error = %v", err)
	}

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
			Hash: emptyTreeHash,
			Type: objects.TargetTypeRevision,
		},
		Message: "Release\n",
	}

	id, err := FromReleaseMetadata(meta)
	if err != nil {
		t.Fatalf("FromReleaseMetadata() error = %v", err)
	}

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
			Name:       mainBranchRef,
			TargetType: objects.BranchTargetRevision,
			Target:     emptyTreeHash,
		},
	}

	id, err := FromSnapshotBranches(branches)
	if err != nil {
		t.Fatalf("FromSnapshotBranches() error = %v", err)
	}

	if id.ObjectType != ObjectTypeSnapshot {
		t.Errorf("FromSnapshotBranches() type = %v, want %v", id.ObjectType, ObjectTypeSnapshot)
	}

	if len(id.ObjectHash) != 40 {
		t.Errorf("FromSnapshotBranches() hash length = %d, want 40", len(id.ObjectHash))
	}
}
