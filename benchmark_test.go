package swhid

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

const (
	benchmarkFilesystemEntries = 1000
	benchmarkGitRefs           = 1000
)

var benchmarkIdentifierResult *Identifier

func BenchmarkParseQualified(b *testing.B) {
	const value = "swh:1:cnt:94a9ed024d3859793618152ea559a168bbcbb5e2;origin=https://example.com/repo.git;visit=swh:1:snp:c7c108084bc0bf3d81436bf980b46e98bd338453;anchor=swh:1:rev:309cf2674ee7a0749978cf8265ab91a60aea0f7d;path=/src/main.go;lines=10-20"
	b.ReportAllocs()

	for b.Loop() {
		id, err := Parse(value)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkIdentifierResult = id
	}
}

func BenchmarkFromDirectoryPath(b *testing.B) {
	dir := b.TempDir()
	if _, err := git.PlainInit(dir, false); err != nil {
		b.Fatal(err)
	}
	for i := range benchmarkFilesystemEntries {
		name := filepath.Join(dir, fmt.Sprintf("file-%08d", i))
		if err := os.WriteFile(name, nil, 0644); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		id, err := FromDirectoryPath(dir)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkIdentifierResult = id
	}
}

func BenchmarkFromSnapshot(b *testing.B) {
	dir := b.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		b.Fatal(err)
	}
	payload := "tree " + emptyTreeHash + "\n" +
		"author Test <test@example.com> 1000000000 +0000\n" +
		"committer Test <test@example.com> 1000000000 +0000\n\n"
	commitHash := storeBenchmarkObject(b, repo, plumbing.CommitObject, payload)
	mainRef := plumbing.NewBranchReferenceName("main")
	if err := repo.Storer.SetReference(plumbing.NewHashReference(mainRef, commitHash)); err != nil {
		b.Fatal(err)
	}
	if err := repo.Storer.SetReference(plumbing.NewSymbolicReference(plumbing.HEAD, mainRef)); err != nil {
		b.Fatal(err)
	}
	for i := range benchmarkGitRefs {
		name := plumbing.NewBranchReferenceName(fmt.Sprintf("benchmark-%08d", i))
		if err := repo.Storer.SetReference(plumbing.NewHashReference(name, commitHash)); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		id, err := FromSnapshot(dir)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkIdentifierResult = id
	}
}

func storeBenchmarkObject(b *testing.B, repo *git.Repository, objectType plumbing.ObjectType, payload string) plumbing.Hash {
	b.Helper()
	object := repo.Storer.NewEncodedObject()
	object.SetType(objectType)
	object.SetSize(int64(len(payload)))
	writer, err := object.Writer()
	if err != nil {
		b.Fatal(err)
	}
	if _, err := writer.Write([]byte(payload)); err != nil {
		b.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		b.Fatal(err)
	}
	hash, err := repo.Storer.SetEncodedObject(object)
	if err != nil {
		b.Fatal(err)
	}
	return hash
}
