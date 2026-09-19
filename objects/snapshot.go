package objects

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// BranchTargetType represents the type of target a branch points to.
type BranchTargetType string

const (
	BranchTargetContent   BranchTargetType = "content"
	BranchTargetDirectory BranchTargetType = "directory"
	BranchTargetRevision  BranchTargetType = "revision"
	BranchTargetRelease   BranchTargetType = "release"
	BranchTargetSnapshot  BranchTargetType = "snapshot"
	BranchTargetAlias     BranchTargetType = "alias"
	BranchTargetDangling  BranchTargetType = "dangling"
)

// Branch represents a branch in a snapshot.
type Branch struct {
	Name       string
	TargetType BranchTargetType
	Target     string // 40-char hex hash, or branch name for alias, or empty for dangling
}

// ComputeSnapshotHash computes the hash for a snapshot.
func ComputeSnapshotHash(branches []Branch) (string, error) {
	serialized, err := serializeBranches(branches)
	if err != nil {
		return "", err
	}
	return computeObjectHash("snapshot", int64(len(serialized)), bytes.NewReader(serialized))
}

func serializeBranches(branches []Branch) ([]byte, error) {
	sorted := make([]Branch, len(branches))
	copy(sorted, branches)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})

	var result []byte
	seen := make(map[string]struct{}, len(sorted))
	for _, branch := range sorted {
		if branch.Name == "" || strings.ContainsRune(branch.Name, 0) {
			return nil, fmt.Errorf("invalid branch name %q", branch.Name)
		}
		if _, exists := seen[branch.Name]; exists {
			return nil, fmt.Errorf("duplicate branch name %q", branch.Name)
		}
		seen[branch.Name] = struct{}{}
		serialized, err := serializeBranch(branch)
		if err != nil {
			return nil, err
		}
		result = append(result, serialized...)
	}
	return result, nil
}

func serializeBranch(branch Branch) ([]byte, error) {
	targetType := branch.TargetType
	if targetType == BranchTargetDangling {
		targetType = BranchTargetRevision
	}
	targetIdentifier, err := computeTargetIdentifier(branch)
	if err != nil {
		return nil, err
	}
	targetLength := len(targetIdentifier)

	var result []byte
	result = append(result, []byte(targetType)...)
	result = append(result, ' ')
	result = append(result, []byte(branch.Name)...)
	result = append(result, 0)
	result = append(result, []byte(fmt.Sprintf("%d:", targetLength))...)
	result = append(result, targetIdentifier...)

	return result, nil
}

func computeTargetIdentifier(branch Branch) ([]byte, error) {
	switch branch.TargetType {
	case BranchTargetContent, BranchTargetDirectory, BranchTargetRevision, BranchTargetRelease, BranchTargetSnapshot:
		if branch.Target == "" {
			return nil, fmt.Errorf("missing target hash for branch %q", branch.Name)
		}
		hashBytes, err := hex.DecodeString(branch.Target)
		if err != nil || len(hashBytes) != 20 {
			return nil, fmt.Errorf("invalid target hash for branch %q", branch.Name)
		}
		return hashBytes, nil
	case BranchTargetAlias:
		if branch.Target == "" || strings.ContainsRune(branch.Target, 0) {
			return nil, fmt.Errorf("invalid alias target for branch %q", branch.Name)
		}
		return []byte(branch.Target), nil
	case BranchTargetDangling:
		if branch.Target != "" {
			return nil, fmt.Errorf("dangling branch %q has a target", branch.Name)
		}
		return nil, nil
	default:
		return nil, fmt.Errorf("invalid target type %q for branch %q", branch.TargetType, branch.Name)
	}
}
