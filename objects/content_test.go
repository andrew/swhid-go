package objects

import (
	"strings"
	"testing"
)

const (
	emptyBlobHash     = "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391"
	emptyTreeHash     = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"
	invalidObjectHash = "invalid"
	testAuthor        = "Test Author <test@example.com>"
	testIdentity      = "Test <test@example.com>"
	testBranchName    = "refs/heads/main"
	testShortBranch   = "main"
	testEntryName     = "file"
)

func TestComputeContentHash(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		wantHash string
	}{
		{
			name:     "empty content",
			data:     []byte{},
			wantHash: emptyBlobHash,
		},
		{
			name:     "hello world",
			data:     []byte("Hello, World!"),
			wantHash: "b45ef6fec89518d314f546fd6c3025367b721684",
		},
		{
			name:     "newline",
			data:     []byte("\n"),
			wantHash: "8b137891791fe96927ad78e64b0aad7bded08bdc",
		},
		{
			name:     "hello with newline",
			data:     []byte("hello\n"),
			wantHash: "ce013625030ba8dba906f756967f9e9ca394464a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := ComputeContentHash(tt.data)
			if err != nil {
				t.Fatalf("ComputeContentHash() error = %v", err)
			}
			if hash != tt.wantHash {
				t.Errorf("ComputeContentHash() = %v, want %v", hash, tt.wantHash)
			}
		})
	}
}

func TestComputeContentHashReaderRejectsWrongSize(t *testing.T) {
	if _, err := ComputeContentHashReader(strings.NewReader("content"), 3); err == nil {
		t.Fatal("ComputeContentHashReader() expected size error")
	}
}
