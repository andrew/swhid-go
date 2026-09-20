package swhid

import (
	"io"

	"github.com/andrew/swhid-go/objects"
)

// FromContent computes the SWHID for file content.
func FromContent(data []byte) (*Identifier, error) {
	hash, err := objects.ComputeContentHash(data)
	if err != nil {
		return nil, err
	}
	return NewIdentifier(ObjectTypeContent, hash, nil)
}

// FromContentReader computes the SWHID for content with a known byte length.
func FromContentReader(content io.Reader, size int64) (*Identifier, error) {
	hash, err := objects.ComputeContentHashReader(content, size)
	if err != nil {
		return nil, err
	}
	return NewIdentifier(ObjectTypeContent, hash, nil)
}

// FromDirectory computes the SWHID for a directory with the given entries.
func FromDirectory(entries []objects.DirectoryEntry) (*Identifier, error) {
	hash, err := objects.ComputeDirectoryHash(entries)
	if err != nil {
		return nil, err
	}
	return NewIdentifier(ObjectTypeDirectory, hash, nil)
}

// FromRevisionMetadata computes the SWHID for a revision with the given metadata.
func FromRevisionMetadata(meta objects.RevisionMetadata) (*Identifier, error) {
	hash, err := objects.ComputeRevisionHash(meta)
	if err != nil {
		return nil, err
	}
	return NewIdentifier(ObjectTypeRevision, hash, nil)
}

// FromReleaseMetadata computes the SWHID for a release with the given metadata.
func FromReleaseMetadata(meta objects.ReleaseMetadata) (*Identifier, error) {
	hash, err := objects.ComputeReleaseHash(meta)
	if err != nil {
		return nil, err
	}
	return NewIdentifier(ObjectTypeRelease, hash, nil)
}

// FromSnapshotBranches computes the SWHID for a snapshot with the given branches.
func FromSnapshotBranches(branches []objects.Branch) (*Identifier, error) {
	hash, err := objects.ComputeSnapshotHash(branches)
	if err != nil {
		return nil, err
	}
	return NewIdentifier(ObjectTypeSnapshot, hash, nil)
}
