package swhid

import (
	"io"
	"testing"

	"github.com/andrew/swhid-go/objects"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

const (
	emptyTreeHash = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"
	mainBranchRef = "refs/heads/main"
)

func TestFromRevisionPreservesGitObjectID(t *testing.T) {
	repoPath, _, commitHash := repositoryWithEmptyCommit(t)

	id, err := FromRevision(repoPath, "HEAD")
	if err != nil {
		t.Fatalf("FromRevision() error = %v", err)
	}
	if id.ObjectHash != commitHash.String() {
		t.Errorf("FromRevision() hash = %s, want %s", id.ObjectHash, commitHash)
	}
}

func TestFromRevisionPreservesMultilineHeaders(t *testing.T) {
	repoPath, repo, _ := repositoryWithEmptyCommit(t)
	payload := "tree " + emptyTreeHash + "\n" +
		"author Test <test@example.com> 1000000000 +0000\n" +
		"committer Test <test@example.com> 1000000000 +0000\n" +
		"gpgsig -----BEGIN PGP SIGNATURE-----\n" +
		" fake-signature-data\n" +
		" -----END PGP SIGNATURE-----\n\nSigned commit\n"
	commitHash := storeObject(t, repo, plumbing.CommitObject, payload)
	if err := repo.Storer.SetReference(plumbing.NewHashReference(plumbing.NewBranchReferenceName("signed"), commitHash)); err != nil {
		t.Fatalf("SetReference() error = %v", err)
	}

	id, err := FromRevision(repoPath, "signed")
	if err != nil {
		t.Fatalf("FromRevision() error = %v", err)
	}
	if id.ObjectHash != commitHash.String() {
		t.Errorf("FromRevision() hash = %s, want %s", id.ObjectHash, commitHash)
	}
}

func TestFromReleasePreservesGitObjectID(t *testing.T) {
	repoPath, repo, commitHash := repositoryWithEmptyCommit(t)
	payload := "object " + commitHash.String() + "\n" +
		"type commit\n" +
		"tag v1.0.0\n" +
		"tagger Test <test@example.com> 1000000000 +0000\n\n"
	tagHash := storeObject(t, repo, plumbing.TagObject, payload)
	if err := repo.Storer.SetReference(plumbing.NewHashReference(plumbing.NewTagReferenceName("v1.0.0"), tagHash)); err != nil {
		t.Fatalf("SetReference() error = %v", err)
	}

	id, err := FromRelease(repoPath, "v1.0.0")
	if err != nil {
		t.Fatalf("FromRelease() error = %v", err)
	}
	if id.ObjectHash != tagHash.String() {
		t.Errorf("FromRelease() hash = %s, want %s", id.ObjectHash, tagHash)
	}
}

func TestFromSnapshotUsesBranchesTagsAndSingleHEAD(t *testing.T) {
	repoPath, repo, commitHash := repositoryWithEmptyCommit(t)
	remoteRef := plumbing.ReferenceName("refs/remotes/origin/main")
	if err := repo.Storer.SetReference(plumbing.NewHashReference(remoteRef, commitHash)); err != nil {
		t.Fatalf("SetReference() error = %v", err)
	}

	id, err := FromSnapshot(repoPath)
	if err != nil {
		t.Fatalf("FromSnapshot() error = %v", err)
	}
	const want = "934c60fe7b2bc0e9c004729e9087bd52d552ef61"
	if id.ObjectHash != want {
		t.Errorf("FromSnapshot() hash = %s, want %s", id.ObjectHash, want)
	}
}

func TestFromSnapshotIncludesDanglingBranch(t *testing.T) {
	repoPath, repo, commitHash := repositoryWithEmptyCommit(t)
	missingHash := plumbing.NewHash("1111111111111111111111111111111111111111")
	if err := repo.Storer.SetReference(plumbing.NewHashReference(plumbing.NewBranchReferenceName("missing"), missingHash)); err != nil {
		t.Fatalf("SetReference() error = %v", err)
	}

	got, err := FromSnapshot(repoPath)
	if err != nil {
		t.Fatalf("FromSnapshot() error = %v", err)
	}
	want, err := FromSnapshotBranches([]objects.Branch{
		{Name: "HEAD", TargetType: objects.BranchTargetAlias, Target: mainBranchRef},
		{Name: mainBranchRef, TargetType: objects.BranchTargetRevision, Target: commitHash.String()},
		{Name: "refs/heads/missing", TargetType: objects.BranchTargetDangling},
	})
	if err != nil {
		t.Fatalf("FromSnapshotBranches() error = %v", err)
	}
	if got.ObjectHash != want.ObjectHash {
		t.Errorf("FromSnapshot() hash = %s, want %s", got.ObjectHash, want.ObjectHash)
	}
}

func repositoryWithEmptyCommit(t *testing.T) (string, *git.Repository, plumbing.Hash) {
	t.Helper()
	repoPath := t.TempDir()
	repo, err := git.PlainInit(repoPath, false)
	if err != nil {
		t.Fatalf("PlainInit() error = %v", err)
	}

	payload := "tree " + emptyTreeHash + "\n" +
		"author Test <test@example.com> 1000000000 +0000\n" +
		"committer Test <test@example.com> 1000000000 +0000\n\n"
	commitHash := storeObject(t, repo, plumbing.CommitObject, payload)
	mainRef := plumbing.NewBranchReferenceName("main")
	if err := repo.Storer.SetReference(plumbing.NewHashReference(mainRef, commitHash)); err != nil {
		t.Fatalf("SetReference() error = %v", err)
	}
	if err := repo.Storer.SetReference(plumbing.NewSymbolicReference(plumbing.HEAD, mainRef)); err != nil {
		t.Fatalf("SetReference() error = %v", err)
	}

	return repoPath, repo, commitHash
}

func storeObject(t *testing.T, repo *git.Repository, objectType plumbing.ObjectType, payload string) plumbing.Hash {
	t.Helper()
	obj := repo.Storer.NewEncodedObject()
	obj.SetType(objectType)
	obj.SetSize(int64(len(payload)))
	w, err := obj.Writer()
	if err != nil {
		t.Fatalf("Writer() error = %v", err)
	}
	if _, err := io.WriteString(w, payload); err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	hash, err := repo.Storer.SetEncodedObject(obj)
	if err != nil {
		t.Fatalf("SetEncodedObject() error = %v", err)
	}
	return hash
}
