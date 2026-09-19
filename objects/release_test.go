package objects

import (
	"testing"
)

func TestComputeReleaseHash(t *testing.T) {
	meta := ReleaseMetadata{
		Name: "v1.0.0",
		Target: ReleaseTarget{
			Hash: emptyTreeHash,
			Type: TargetTypeRevision,
		},
		Author:          testAuthor,
		AuthorTimestamp: 1234567890,
		AuthorTimezone:  "+0000",
		Message:         "Release v1.0.0\n",
	}

	hash, err := ComputeReleaseHash(meta)
	if err != nil {
		t.Fatalf("ComputeReleaseHash() error = %v", err)
	}

	// Just verify it produces a 40-char hex hash
	if len(hash) != 40 {
		t.Errorf("ComputeReleaseHash() hash length = %d, want 40", len(hash))
	}

	// Verify determinism
	hash2, err := ComputeReleaseHash(meta)
	if err != nil {
		t.Fatalf("ComputeReleaseHash() error = %v", err)
	}
	if hash != hash2 {
		t.Errorf("ComputeReleaseHash() not deterministic: %v != %v", hash, hash2)
	}
}

func TestReleaseWithoutTagger(t *testing.T) {
	meta := ReleaseMetadata{
		Name: "v0.1.0",
		Target: ReleaseTarget{
			Hash: emptyTreeHash,
			Type: TargetTypeRevision,
		},
		Message: "Early release\n",
	}

	hash, err := ComputeReleaseHash(meta)
	if err != nil {
		t.Fatalf("ComputeReleaseHash() error = %v", err)
	}

	// Just verify it produces a 40-char hex hash
	if len(hash) != 40 {
		t.Errorf("ComputeReleaseHash() hash length = %d, want 40", len(hash))
	}
}

func TestReleasePresentEmptyMessage(t *testing.T) {
	meta := ReleaseMetadata{
		Name:           "v0.1.0",
		Target:         ReleaseTarget{Hash: emptyTreeHash, Type: TargetTypeRevision},
		MessagePresent: true,
	}

	presentHash, err := ComputeReleaseHash(meta)
	if err != nil {
		t.Fatalf("ComputeReleaseHash() error = %v", err)
	}
	meta.MessagePresent = false
	absentHash, err := ComputeReleaseHash(meta)
	if err != nil {
		t.Fatalf("ComputeReleaseHash() without message error = %v", err)
	}
	if presentHash == absentHash {
		t.Error("present empty message hashes the same as an absent message")
	}
}

func TestReleaseTargetGitType(t *testing.T) {
	tests := []struct {
		targetType TargetType
		wantGit    string
	}{
		{TargetTypeContent, "blob"},
		{TargetTypeDirectory, "tree"},
		{TargetTypeRevision, "commit"},
		{TargetTypeRelease, "tag"},
	}

	for _, tt := range tests {
		target := ReleaseTarget{Type: tt.targetType}
		got, err := target.GitType()
		if err != nil {
			t.Fatalf("GitType() error = %v", err)
		}
		if got != tt.wantGit {
			t.Errorf("GitType() for %v = %v, want %v", tt.targetType, got, tt.wantGit)
		}
	}
}

func TestReleaseRejectsInvalidTarget(t *testing.T) {
	meta := ReleaseMetadata{
		Name:   "v1.0.0",
		Target: ReleaseTarget{Hash: invalidObjectHash, Type: TargetTypeSnapshot},
	}
	if _, err := ComputeReleaseHash(meta); err == nil {
		t.Fatal("ComputeReleaseHash() expected error")
	}
}
