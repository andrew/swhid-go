package objects

import (
	"fmt"
	"testing"
)

const (
	benchmarkDirectoryEntries = 1000
	benchmarkSnapshotBranches = 1000
	benchmarkContentSize      = 1024 * 1024
)

var benchmarkHashResult string

func BenchmarkComputeContentHash(b *testing.B) {
	data := make([]byte, benchmarkContentSize)
	for i := range data {
		data[i] = byte(i)
	}
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		hash, err := ComputeContentHash(data)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkHashResult = hash
	}
}

func BenchmarkComputeDirectoryHash(b *testing.B) {
	entries := make([]DirectoryEntry, benchmarkDirectoryEntries)
	for i := range entries {
		entryType := EntryTypeFile
		target := emptyBlobHash
		if i%10 == 0 {
			entryType = EntryTypeDirectory
			target = emptyTreeHash
		}
		entries[i] = DirectoryEntry{
			Name:   fmt.Sprintf("entry-%08d", benchmarkDirectoryEntries-i),
			Type:   entryType,
			Target: target,
		}
	}
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		hash, err := ComputeDirectoryHash(entries)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkHashResult = hash
	}
}

func BenchmarkComputeSnapshotHash(b *testing.B) {
	branches := make([]Branch, benchmarkSnapshotBranches)
	for i := range branches {
		branches[i] = Branch{
			Name:       fmt.Sprintf("refs/heads/branch-%08d", benchmarkSnapshotBranches-i),
			TargetType: BranchTargetRevision,
			Target:     emptyTreeHash,
		}
	}
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		hash, err := ComputeSnapshotHash(branches)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkHashResult = hash
	}
}
