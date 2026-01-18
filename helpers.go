package swhid

import "github.com/andrew/swhid-go/objects"

// FromContent computes the SWHID for file content.
func FromContent(data []byte) *Identifier {
	hash := objects.ComputeContentHash(data)
	id, _ := NewIdentifier(ObjectTypeContent, hash, nil)
	return id
}

// FromDirectory computes the SWHID for a directory with the given entries.
func FromDirectory(entries []objects.DirectoryEntry) *Identifier {
	hash := objects.ComputeDirectoryHash(entries)
	id, _ := NewIdentifier(ObjectTypeDirectory, hash, nil)
	return id
}

// FromRevisionMetadata computes the SWHID for a revision with the given metadata.
func FromRevisionMetadata(meta objects.RevisionMetadata) *Identifier {
	hash := objects.ComputeRevisionHash(meta)
	id, _ := NewIdentifier(ObjectTypeRevision, hash, nil)
	return id
}

// FromReleaseMetadata computes the SWHID for a release with the given metadata.
func FromReleaseMetadata(meta objects.ReleaseMetadata) *Identifier {
	hash := objects.ComputeReleaseHash(meta)
	id, _ := NewIdentifier(ObjectTypeRelease, hash, nil)
	return id
}

// FromSnapshotBranches computes the SWHID for a snapshot with the given branches.
func FromSnapshotBranches(branches []objects.Branch) *Identifier {
	hash := objects.ComputeSnapshotHash(branches)
	id, _ := NewIdentifier(ObjectTypeSnapshot, hash, nil)
	return id
}
