package objects

import (
	"testing"
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
			wantHash: "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391",
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
			hash := ComputeContentHash(tt.data)
			if hash != tt.wantHash {
				t.Errorf("ComputeContentHash() = %v, want %v", hash, tt.wantHash)
			}
		})
	}
}
