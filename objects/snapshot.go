package objects

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"sort"
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
func ComputeSnapshotHash(branches []Branch) string {
	serialized := serializeBranches(branches)
	header := fmt.Sprintf("snapshot %d\x00", len(serialized))

	h := sha1.New()
	h.Write([]byte(header))
	h.Write(serialized)
	return hex.EncodeToString(h.Sum(nil))
}

func serializeBranches(branches []Branch) []byte {
	// Sort branches by name
	sorted := make([]Branch, len(branches))
	copy(sorted, branches)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})

	var result []byte
	for _, branch := range sorted {
		result = append(result, serializeBranch(branch)...)
	}
	return result
}

func serializeBranch(branch Branch) []byte {
	targetIdentifier := computeTargetIdentifier(branch)
	targetLength := len(targetIdentifier)

	// Format: "<target_type> <name>\0<target_length>:<target_identifier>"
	var result []byte
	result = append(result, []byte(branch.TargetType)...)
	result = append(result, ' ')
	result = append(result, []byte(branch.Name)...)
	result = append(result, 0)
	result = append(result, []byte(fmt.Sprintf("%d:", targetLength))...)
	result = append(result, targetIdentifier...)

	return result
}

func computeTargetIdentifier(branch Branch) []byte {
	switch branch.TargetType {
	case BranchTargetContent, BranchTargetDirectory, BranchTargetRevision, BranchTargetRelease, BranchTargetSnapshot:
		// Convert hex hash to binary
		hashBytes, _ := hex.DecodeString(branch.Target)
		return hashBytes
	case BranchTargetAlias:
		// Alias target is the branch name as bytes
		return []byte(branch.Target)
	case BranchTargetDangling:
		// Dangling has no target
		return []byte{}
	default:
		return []byte{}
	}
}
