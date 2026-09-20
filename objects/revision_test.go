package objects

import (
	"strings"
	"testing"
)

func TestComputeRevisionHash(t *testing.T) {
	meta := RevisionMetadata{
		Directory:          emptyTreeHash,
		Author:             testAuthor,
		AuthorTimestamp:    1234567890,
		AuthorTimezone:     "+0000",
		Committer:          testAuthor,
		CommitterTimestamp: 1234567890,
		CommitterTimezone:  "+0000",
		Message:            "Initial commit\n",
	}

	hash, err := ComputeRevisionHash(meta)
	if err != nil {
		t.Fatalf("ComputeRevisionHash() error = %v", err)
	}

	// Just verify it produces a 40-char hex hash
	if len(hash) != 40 {
		t.Errorf("ComputeRevisionHash() hash length = %d, want 40", len(hash))
	}

	// Verify determinism
	hash2, err := ComputeRevisionHash(meta)
	if err != nil {
		t.Fatalf("ComputeRevisionHash() error = %v", err)
	}
	if hash != hash2 {
		t.Errorf("ComputeRevisionHash() not deterministic: %v != %v", hash, hash2)
	}
}

func TestRevisionEmptyTimezoneIsPreserved(t *testing.T) {
	meta := RevisionMetadata{
		Directory:          emptyTreeHash,
		Author:             testIdentity,
		AuthorTimestamp:    1234567890,
		AuthorTimezone:     "", // Empty should default to +0000
		Committer:          testIdentity,
		CommitterTimestamp: 1234567890,
		CommitterTimezone:  "", // Empty should default to +0000
		Message:            "Test\n",
	}

	hash1, err := ComputeRevisionHash(meta)
	if err != nil {
		t.Fatalf("ComputeRevisionHash() error = %v", err)
	}

	// An absent timezone is an empty byte sequence, not UTC.
	meta2 := meta
	meta2.AuthorTimezone = "+0000"
	meta2.CommitterTimezone = "+0000"
	hash2, err := ComputeRevisionHash(meta2)
	if err != nil {
		t.Fatalf("ComputeRevisionHash() error = %v", err)
	}

	if hash1 == hash2 {
		t.Errorf("empty timezone and +0000 produced the same hash: %v", hash1)
	}
}

func TestRevisionWithParent(t *testing.T) {
	// First commit
	meta1 := RevisionMetadata{
		Directory:          emptyTreeHash,
		Author:             testIdentity,
		AuthorTimestamp:    1000000000,
		AuthorTimezone:     "+0000",
		Committer:          testIdentity,
		CommitterTimestamp: 1000000000,
		CommitterTimezone:  "+0000",
		Message:            "First\n",
	}
	hash1, err := ComputeRevisionHash(meta1)
	if err != nil {
		t.Fatalf("ComputeRevisionHash() error = %v", err)
	}

	// Second commit with parent
	meta2 := RevisionMetadata{
		Directory:          emptyTreeHash,
		Parents:            []string{hash1},
		Author:             testIdentity,
		AuthorTimestamp:    1000000001,
		AuthorTimezone:     "+0000",
		Committer:          testIdentity,
		CommitterTimestamp: 1000000001,
		CommitterTimezone:  "+0000",
		Message:            "Second\n",
	}
	hash2, err := ComputeRevisionHash(meta2)
	if err != nil {
		t.Fatalf("ComputeRevisionHash() error = %v", err)
	}

	// Commits should be different
	if hash1 == hash2 {
		t.Errorf("Different commits should have different hashes")
	}
}

func TestRevisionPresentEmptyMessage(t *testing.T) {
	meta := RevisionMetadata{
		Directory:          emptyTreeHash,
		Author:             testIdentity,
		AuthorTimestamp:    1000000000,
		AuthorTimezone:     "+0000",
		Committer:          testIdentity,
		CommitterTimestamp: 1000000000,
		CommitterTimezone:  "+0000",
		MessagePresent:     true,
	}

	hash, err := ComputeRevisionHash(meta)
	if err != nil {
		t.Fatalf("ComputeRevisionHash() error = %v", err)
	}
	const want = "f13d592bfeb89d8b699fcf328f613453652a2e8f"
	if hash != want {
		t.Errorf("ComputeRevisionHash() = %s, want %s", hash, want)
	}

	meta.MessagePresent = false
	absentHash, err := ComputeRevisionHash(meta)
	if err != nil {
		t.Fatalf("ComputeRevisionHash() without message error = %v", err)
	}
	if hash == absentHash {
		t.Error("present empty message hashes the same as an absent message")
	}
}

func TestRevisionRejectsInvalidMetadata(t *testing.T) {
	tests := []RevisionMetadata{
		{Directory: invalidObjectHash, Author: testIdentity, Committer: testIdentity},
		{Directory: strings.ToUpper(emptyTreeHash), Author: testIdentity, Committer: testIdentity},
		{Directory: emptyTreeHash, Author: testIdentity, Committer: testIdentity, ExtraHeaders: [][2]string{{"bad key", "value"}}},
	}

	for _, meta := range tests {
		if _, err := ComputeRevisionHash(meta); err == nil {
			t.Fatal("ComputeRevisionHash() expected error")
		}
	}
}
