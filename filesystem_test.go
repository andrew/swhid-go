package swhid

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andrew/swhid-go/objects"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/format/index"
)

func TestFromDirectoryPath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tmpDir, "hello.txt")
	if err := os.WriteFile(testFile, []byte("hello\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	id, err := FromDirectoryPath(tmpDir)
	if err != nil {
		t.Fatalf("FromDirectoryPath() error = %v", err)
	}

	if id.ObjectType != ObjectTypeDirectory {
		t.Errorf("FromDirectoryPath() type = %v, want %v", id.ObjectType, ObjectTypeDirectory)
	}

	// Should match verified hash
	wantHash := "aaa96ced2d9a1c8e72c56b253a0e2fe78393feb7"
	if id.ObjectHash != wantHash {
		t.Errorf("FromDirectoryPath() hash = %v, want %v", id.ObjectHash, wantHash)
	}
}

func TestFromDirectoryPathEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	id, err := FromDirectoryPath(tmpDir)
	if err != nil {
		t.Fatalf("FromDirectoryPath() error = %v", err)
	}

	// Empty tree hash
	wantHash := emptyTreeHash
	if id.ObjectHash != wantHash {
		t.Errorf("FromDirectoryPath() hash = %v, want %v", id.ObjectHash, wantHash)
	}
}

func TestFromDirectoryPathNested(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested structure
	subDir := filepath.Join(tmpDir, "sub")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	testFile := filepath.Join(subDir, "file.txt")
	if err := os.WriteFile(testFile, []byte("test\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	id, err := FromDirectoryPath(tmpDir)
	if err != nil {
		t.Fatalf("FromDirectoryPath() error = %v", err)
	}

	if id.ObjectType != ObjectTypeDirectory {
		t.Errorf("FromDirectoryPath() type = %v, want %v", id.ObjectType, ObjectTypeDirectory)
	}

	if len(id.ObjectHash) != 40 {
		t.Errorf("FromDirectoryPath() hash length = %d, want 40", len(id.ObjectHash))
	}
}

func TestFromDirectoryPathNotExists(t *testing.T) {
	_, err := FromDirectoryPath("/nonexistent/path/that/should/not/exist")
	if err == nil {
		t.Error("FromDirectoryPath() expected error for nonexistent path")
	}
}

func TestFromDirectoryPathFile(t *testing.T) {
	// Create a temp file
	tmpFile, err := os.CreateTemp("", "swhid-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.Remove(tmpFile.Name()); err != nil && !os.IsNotExist(err) {
			t.Errorf("Remove() error = %v", err)
		}
	})

	_, err = FromDirectoryPath(tmpFile.Name())
	if err == nil {
		t.Error("FromDirectoryPath() expected error for file path")
	}
}

func TestFromDirectoryPathSkipsDotGitEntries(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".git"), []byte("ordinary file"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	nested := filepath.Join(dir, "nested")
	if err := os.MkdirAll(filepath.Join(nested, ".git"), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(nested, ".git", "ignored"), []byte("ignored"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	want, err := FromDirectory([]objects.DirectoryEntry{{Name: "nested", Type: objects.EntryTypeDirectory, Target: emptyTreeHash}})
	if err != nil {
		t.Fatalf("FromDirectory() error = %v", err)
	}
	got, err := FromDirectoryPath(dir)
	if err != nil {
		t.Fatalf("FromDirectoryPath() error = %v", err)
	}
	if got.ObjectHash != want.ObjectHash {
		t.Errorf("FromDirectoryPath() hash = %s, want %s", got.ObjectHash, want.ObjectHash)
	}
}

func TestFromDirectoryPathUsesGitlinkFromIndex(t *testing.T) {
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("PlainInit() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "vendor", "module"), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "vendor", "module", "checkout.txt"), []byte("ignored"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	idx := &index.Index{Version: 2, Entries: []*index.Entry{{
		Name: "vendor/module",
		Mode: filemode.Submodule,
		Hash: plumbing.NewHash(emptyTreeHash),
	}}}
	if err := repo.Storer.SetIndex(idx); err != nil {
		t.Fatalf("SetIndex() error = %v", err)
	}

	vendorID, err := FromDirectory([]objects.DirectoryEntry{{Name: "module", Type: objects.EntryTypeRevision, Target: emptyTreeHash}})
	if err != nil {
		t.Fatalf("FromDirectory() error = %v", err)
	}
	want, err := FromDirectory([]objects.DirectoryEntry{{Name: "vendor", Type: objects.EntryTypeDirectory, Target: vendorID.ObjectHash}})
	if err != nil {
		t.Fatalf("FromDirectory() error = %v", err)
	}
	got, err := FromDirectoryPath(dir)
	if err != nil {
		t.Fatalf("FromDirectoryPath() error = %v", err)
	}
	if got.ObjectHash != want.ObjectHash {
		t.Errorf("FromDirectoryPath() hash = %s, want %s", got.ObjectHash, want.ObjectHash)
	}
}

func TestFromDirectoryPathReportsInvalidGitIndex(t *testing.T) {
	dir := t.TempDir()
	if _, err := git.PlainInit(dir, false); err != nil {
		t.Fatalf("PlainInit() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".git", "index"), []byte("invalid index"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := FromDirectoryPath(dir); err == nil {
		t.Fatal("FromDirectoryPath() expected invalid index error")
	}
}

func TestFromDirectoryPathUsesIndexModeFromRepositorySubdirectory(t *testing.T) {
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("PlainInit() error = %v", err)
	}
	sourceDir := filepath.Join(dir, "src")
	if err := os.Mkdir(sourceDir, 0755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "tool"), nil, 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	idx := &index.Index{Version: 2, Entries: []*index.Entry{{
		Name: "src/tool",
		Mode: filemode.Executable,
		Hash: plumbing.NewHash(emptyBlobHash),
	}}}
	if err := repo.Storer.SetIndex(idx); err != nil {
		t.Fatalf("SetIndex() error = %v", err)
	}

	want, err := FromDirectory([]objects.DirectoryEntry{{Name: "tool", Type: objects.EntryTypeExecutable, Target: emptyBlobHash}})
	if err != nil {
		t.Fatalf("FromDirectory() error = %v", err)
	}
	got, err := FromDirectoryPath(sourceDir)
	if err != nil {
		t.Fatalf("FromDirectoryPath() error = %v", err)
	}
	if got.ObjectHash != want.ObjectHash {
		t.Errorf("FromDirectoryPath() hash = %s, want %s", got.ObjectHash, want.ObjectHash)
	}
}
