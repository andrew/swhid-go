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
			hash, err := ComputeSnapshotHash(tt.branches)
			if err != nil {
				t.Fatalf("ComputeSnapshotHash() error = %v", err)
			}
			if hash != tt.wantHash {
				t.Errorf("ComputeSnapshotHash() = %v, want %v", hash, tt.wantHash)
			}
		})
	}
}

func TestSnapshotWithBranches(t *testing.T) {
	branches := []Branch{
		{
			Name:       testBranchName,
			TargetType: BranchTargetRevision,
			Target:     emptyTreeHash,
		},
	}

	hash, err := ComputeSnapshotHash(branches)
	if err != nil {
		t.Fatalf("ComputeSnapshotHash() error = %v", err)
	}

	if len(hash) != 40 {
		t.Errorf("ComputeSnapshotHash() hash length = %d, want 40", len(hash))
	}
}

func TestSnapshotBranchSorting(t *testing.T) {
	// Branches should be sorted by name
	branches1 := []Branch{
		{Name: testBranchName, TargetType: BranchTargetRevision, Target: emptyTreeHash},
		{Name: "refs/heads/dev", TargetType: BranchTargetRevision, Target: emptyTreeHash},
	}

	branches2 := []Branch{
		{Name: "refs/heads/dev", TargetType: BranchTargetRevision, Target: emptyTreeHash},
		{Name: testBranchName, TargetType: BranchTargetRevision, Target: emptyTreeHash},
	}

	hash1, err := ComputeSnapshotHash(branches1)
	if err != nil {
		t.Fatalf("ComputeSnapshotHash() error = %v", err)
	}
	hash2, err := ComputeSnapshotHash(branches2)
	if err != nil {
		t.Fatalf("ComputeSnapshotHash() error = %v", err)
	}

	if hash1 != hash2 {
		t.Errorf("Hash should be deterministic regardless of input order: %v != %v", hash1, hash2)
	}
}

func TestSnapshotWithAlias(t *testing.T) {
	branches := []Branch{
		{
			Name:       "HEAD",
			TargetType: BranchTargetAlias,
			Target:     testBranchName,
		},
		{
			Name:       testBranchName,
			TargetType: BranchTargetRevision,
			Target:     emptyTreeHash,
		},
	}

	hash, err := ComputeSnapshotHash(branches)
	if err != nil {
		t.Fatalf("ComputeSnapshotHash() error = %v", err)
	}

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

	hash, err := ComputeSnapshotHash(branches)
	if err != nil {
		t.Fatalf("ComputeSnapshotHash() error = %v", err)
	}

	const want = "4643cc976f3c35dba499513ec8dd2724000719d7"
	if hash != want {
		t.Errorf("ComputeSnapshotHash() = %s, want %s", hash, want)
	}
}

func TestComputeSnapshotHashRejectsInvalidBranches(t *testing.T) {
	tests := []struct {
		name     string
		branches []Branch
	}{
		{name: "duplicate name", branches: []Branch{{Name: testShortBranch, TargetType: BranchTargetRevision, Target: emptyTreeHash}, {Name: testShortBranch, TargetType: BranchTargetRevision, Target: emptyTreeHash}}},
		{name: "invalid target", branches: []Branch{{Name: testShortBranch, TargetType: BranchTargetRevision, Target: invalidObjectHash}}},
		{name: "missing target", branches: []Branch{{Name: testShortBranch, TargetType: BranchTargetRevision}}},
		{name: "empty alias", branches: []Branch{{Name: testShortBranch, TargetType: BranchTargetAlias}}},
		{name: "nul in alias", branches: []Branch{{Name: testShortBranch, TargetType: BranchTargetAlias, Target: "refs/heads/main\x00other"}}},
		{name: "target on dangling branch", branches: []Branch{{Name: testShortBranch, TargetType: BranchTargetDangling, Target: emptyTreeHash}}},
		{name: "invalid type", branches: []Branch{{Name: testShortBranch, TargetType: "unknown", Target: emptyTreeHash}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ComputeSnapshotHash(tt.branches); err == nil {
				t.Fatal("ComputeSnapshotHash() expected error")
			}
		})
	}
}
