package objects

import (
	"errors"
	"strings"
	"testing"
)

func TestVerifyObjectHash(t *testing.T) {
	const content = "hello\n"
	if err := VerifyObjectHash("blob", int64(len(content)), strings.NewReader(content), "ce013625030ba8dba906f756967f9e9ca394464a"); err != nil {
		t.Fatalf("VerifyObjectHash() error = %v", err)
	}
}

func TestVerifyObjectHashRejectsMismatch(t *testing.T) {
	const content = "hello\n"
	err := VerifyObjectHash("blob", int64(len(content)), strings.NewReader(content), emptyBlobHash)
	if !errors.Is(err, ErrObjectHashMismatch) {
		t.Fatalf("VerifyObjectHash() error = %v, want ErrObjectHashMismatch", err)
	}
}

func TestVerifyObjectHashRejectsNegativeSize(t *testing.T) {
	err := VerifyObjectHash("blob", -1, strings.NewReader(""), emptyBlobHash)
	if err == nil {
		t.Fatal("VerifyObjectHash() expected error")
	}
}
