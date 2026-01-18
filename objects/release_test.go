package objects

import (
	"testing"
)

func TestComputeReleaseHash(t *testing.T) {
	meta := ReleaseMetadata{
		Name: "v1.0.0",
		Target: ReleaseTarget{
			Hash: "4b825dc642cb6eb9a060e54bf8d69288fbee4904",
			Type: TargetTypeRevision,
		},
		Author:          "Test Author <test@example.com>",
		AuthorTimestamp: 1234567890,
		AuthorTimezone:  "+0000",
		Message:         "Release v1.0.0\n",
	}

	hash := ComputeReleaseHash(meta)

	// Just verify it produces a 40-char hex hash
	if len(hash) != 40 {
		t.Errorf("ComputeReleaseHash() hash length = %d, want 40", len(hash))
	}

	// Verify determinism
	hash2 := ComputeReleaseHash(meta)
	if hash != hash2 {
		t.Errorf("ComputeReleaseHash() not deterministic: %v != %v", hash, hash2)
	}
}

func TestReleaseWithoutTagger(t *testing.T) {
	meta := ReleaseMetadata{
		Name: "v0.1.0",
		Target: ReleaseTarget{
			Hash: "4b825dc642cb6eb9a060e54bf8d69288fbee4904",
			Type: TargetTypeRevision,
		},
		Message: "Early release\n",
	}

	hash := ComputeReleaseHash(meta)

	// Just verify it produces a 40-char hex hash
	if len(hash) != 40 {
		t.Errorf("ComputeReleaseHash() hash length = %d, want 40", len(hash))
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
		{TargetTypeSnapshot, "snapshot"},
	}

	for _, tt := range tests {
		target := ReleaseTarget{Type: tt.targetType}
		if got := target.GitType(); got != tt.wantGit {
			t.Errorf("GitType() for %v = %v, want %v", tt.targetType, got, tt.wantGit)
		}
	}
}
