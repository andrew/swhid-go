package objects

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
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

	serializedSize := 0
	for i := range sorted {
		serializedSize += serializedBranchSize(sorted[i])
	}
	result := make([]byte, 0, serializedSize)
	seen := make(map[string]struct{}, len(sorted))
	for _, branch := range sorted {
		if branch.Name == "" || strings.ContainsRune(branch.Name, 0) {
			return nil, fmt.Errorf("invalid branch name %q", branch.Name)
		}
		if _, exists := seen[branch.Name]; exists {
			return nil, fmt.Errorf("duplicate branch name %q", branch.Name)
		}
		seen[branch.Name] = struct{}{}
		var err error
		result, err = appendSerializedBranch(result, branch)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func serializedBranchSize(branch Branch) int {
	targetTypeLength := len(branch.TargetType)
	targetLength := len(branch.Target)
	switch branch.TargetType {
	case BranchTargetContent, BranchTargetDirectory, BranchTargetRevision, BranchTargetRelease, BranchTargetSnapshot:
		targetLength = objectHashBytes
	case BranchTargetDangling:
		targetLength = 0
	}
	return targetTypeLength + 1 + len(branch.Name) + 1 + decimalDigits(targetLength) + 1 + targetLength
}

func decimalDigits(value int) int {
	digits := 1
	for value >= decimalBase {
		value /= decimalBase
		digits++
	}
	return digits
}

func appendSerializedBranch(result []byte, branch Branch) ([]byte, error) {
	var hashBytes [objectHashBytes]byte
	targetLength := 0
	switch branch.TargetType {
	case BranchTargetContent, BranchTargetDirectory, BranchTargetRevision, BranchTargetRelease, BranchTargetSnapshot:
		if branch.Target == "" {
			return nil, fmt.Errorf("missing target hash for branch %q", branch.Name)
		}
		var ok bool
		hashBytes, ok = decodeObjectID(branch.Target)
		if !ok {
			return nil, fmt.Errorf("invalid target hash for branch %q", branch.Name)
		}
		targetLength = len(hashBytes)
	case BranchTargetAlias:
		if branch.Target == "" || strings.ContainsRune(branch.Target, 0) {
			return nil, fmt.Errorf("invalid alias target for branch %q", branch.Name)
		}
		targetLength = len(branch.Target)
	case BranchTargetDangling:
		if branch.Target != "" {
			return nil, fmt.Errorf("dangling branch %q has a target", branch.Name)
		}
	default:
		return nil, fmt.Errorf("invalid target type %q for branch %q", branch.TargetType, branch.Name)
	}

	result = append(result, []byte(branch.TargetType)...)
	result = append(result, ' ')
	result = append(result, []byte(branch.Name)...)
	result = append(result, 0)
	result = strconv.AppendInt(result, int64(targetLength), decimalBase)
	result = append(result, ':')
	switch branch.TargetType {
	case BranchTargetContent, BranchTargetDirectory, BranchTargetRevision, BranchTargetRelease, BranchTargetSnapshot:
		result = append(result, hashBytes[:]...)
	case BranchTargetAlias:
		result = append(result, branch.Target...)
	}
	return result, nil
}
