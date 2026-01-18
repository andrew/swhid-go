package swhid

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFromDirectoryPath(t *testing.T) {
	// Create a temp directory for testing
	tmpDir, err := os.MkdirTemp("", "swhid-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

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
	// Create a temp directory for testing
	tmpDir, err := os.MkdirTemp("", "swhid-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	id, err := FromDirectoryPath(tmpDir)
	if err != nil {
		t.Fatalf("FromDirectoryPath() error = %v", err)
	}

	// Empty tree hash
	wantHash := "4b825dc642cb6eb9a060e54bf8d69288fbee4904"
	if id.ObjectHash != wantHash {
		t.Errorf("FromDirectoryPath() hash = %v, want %v", id.ObjectHash, wantHash)
	}
}

func TestFromDirectoryPathNested(t *testing.T) {
	// Create a temp directory for testing
	tmpDir, err := os.MkdirTemp("", "swhid-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

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
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	_, err = FromDirectoryPath(tmpFile.Name())
	if err == nil {
		t.Error("FromDirectoryPath() expected error for file path")
	}
}
