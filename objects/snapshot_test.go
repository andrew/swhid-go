package objects

import (
	"testing"
)

func TestComputeSnapshotHash(t *testing.T) {
	tests := []struct {
		name     string
		branches []Branch
		wantHash string
	}{
		{
			name:     "empty snapshot",
			branches: []Branch{},
			wantHash: "1a8893e6a86f444e8be8e7bda6cb34fb1735a00e",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := ComputeSnapshotHash(tt.branches)
			if hash != tt.wantHash {
				t.Errorf("ComputeSnapshotHash() = %v, want %v", hash, tt.wantHash)
			}
		})
	}
}

func TestSnapshotWithBranches(t *testing.T) {
	branches := []Branch{
		{
			Name:       "refs/heads/main",
			TargetType: BranchTargetRevision,
			Target:     "4b825dc642cb6eb9a060e54bf8d69288fbee4904",
		},
	}

	hash := ComputeSnapshotHash(branches)

	// Just verify it produces a 40-char hex hash
	if len(hash) != 40 {
		t.Errorf("ComputeSnapshotHash() hash length = %d, want 40", len(hash))
	}
}

func TestSnapshotBranchSorting(t *testing.T) {
	// Branches should be sorted by name
	branches1 := []Branch{
		{Name: "refs/heads/main", TargetType: BranchTargetRevision, Target: "4b825dc642cb6eb9a060e54bf8d69288fbee4904"},
		{Name: "refs/heads/dev", TargetType: BranchTargetRevision, Target: "4b825dc642cb6eb9a060e54bf8d69288fbee4904"},
	}

	branches2 := []Branch{
		{Name: "refs/heads/dev", TargetType: BranchTargetRevision, Target: "4b825dc642cb6eb9a060e54bf8d69288fbee4904"},
		{Name: "refs/heads/main", TargetType: BranchTargetRevision, Target: "4b825dc642cb6eb9a060e54bf8d69288fbee4904"},
	}

	hash1 := ComputeSnapshotHash(branches1)
	hash2 := ComputeSnapshotHash(branches2)

	if hash1 != hash2 {
		t.Errorf("Hash should be deterministic regardless of input order: %v != %v", hash1, hash2)
	}
}

func TestSnapshotWithAlias(t *testing.T) {
	branches := []Branch{
		{
			Name:       "HEAD",
			TargetType: BranchTargetAlias,
			Target:     "refs/heads/main",
		},
		{
			Name:       "refs/heads/main",
			TargetType: BranchTargetRevision,
			Target:     "4b825dc642cb6eb9a060e54bf8d69288fbee4904",
		},
	}

	hash := ComputeSnapshotHash(branches)

	// Just verify it produces a 40-char hex hash
	if len(hash) != 40 {
		t.Errorf("ComputeSnapshotHash() hash length = %d, want 40", len(hash))
	}
}

func TestSnapshotWithDangling(t *testing.T) {
	branches := []Branch{
		{
			Name:       "refs/heads/broken",
			TargetType: BranchTargetDangling,
			Target:     "",
		},
	}

	hash := ComputeSnapshotHash(branches)

	// Just verify it produces a 40-char hex hash
	if len(hash) != 40 {
		t.Errorf("ComputeSnapshotHash() hash length = %d, want 40", len(hash))
	}
}
