package objects

import (
	"testing"
)

func TestComputeRevisionHash(t *testing.T) {
	meta := RevisionMetadata{
		Directory:          "4b825dc642cb6eb9a060e54bf8d69288fbee4904",
		Author:             "Test Author <test@example.com>",
		AuthorTimestamp:    1234567890,
		AuthorTimezone:     "+0000",
		Committer:          "Test Author <test@example.com>",
		CommitterTimestamp: 1234567890,
		CommitterTimezone:  "+0000",
		Message:            "Initial commit\n",
	}

	hash := ComputeRevisionHash(meta)

	// Just verify it produces a 40-char hex hash
	if len(hash) != 40 {
		t.Errorf("ComputeRevisionHash() hash length = %d, want 40", len(hash))
	}

	// Verify determinism
	hash2 := ComputeRevisionHash(meta)
	if hash != hash2 {
		t.Errorf("ComputeRevisionHash() not deterministic: %v != %v", hash, hash2)
	}
}

func TestRevisionDefaultTimezone(t *testing.T) {
	meta := RevisionMetadata{
		Directory:          "4b825dc642cb6eb9a060e54bf8d69288fbee4904",
		Author:             "Test <test@example.com>",
		AuthorTimestamp:    1234567890,
		AuthorTimezone:     "", // Empty should default to +0000
		Committer:          "Test <test@example.com>",
		CommitterTimestamp: 1234567890,
		CommitterTimezone:  "", // Empty should default to +0000
		Message:            "Test\n",
	}

	hash1 := ComputeRevisionHash(meta)

	// Should produce same hash as explicit +0000
	meta2 := meta
	meta2.AuthorTimezone = "+0000"
	meta2.CommitterTimezone = "+0000"
	hash2 := ComputeRevisionHash(meta2)

	if hash1 != hash2 {
		t.Errorf("Empty timezone should default to +0000: %v != %v", hash1, hash2)
	}
}

func TestRevisionWithParent(t *testing.T) {
	// First commit
	meta1 := RevisionMetadata{
		Directory:          "4b825dc642cb6eb9a060e54bf8d69288fbee4904",
		Author:             "Test <test@example.com>",
		AuthorTimestamp:    1000000000,
		AuthorTimezone:     "+0000",
		Committer:          "Test <test@example.com>",
		CommitterTimestamp: 1000000000,
		CommitterTimezone:  "+0000",
		Message:            "First\n",
	}
	hash1 := ComputeRevisionHash(meta1)

	// Second commit with parent
	meta2 := RevisionMetadata{
		Directory:          "4b825dc642cb6eb9a060e54bf8d69288fbee4904",
		Parents:            []string{hash1},
		Author:             "Test <test@example.com>",
		AuthorTimestamp:    1000000001,
		AuthorTimezone:     "+0000",
		Committer:          "Test <test@example.com>",
		CommitterTimestamp: 1000000001,
		CommitterTimezone:  "+0000",
		Message:            "Second\n",
	}
	hash2 := ComputeRevisionHash(meta2)

	// Commits should be different
	if hash1 == hash2 {
		t.Errorf("Different commits should have different hashes")
	}
}
