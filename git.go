package swhid

import (
	"errors"
	"fmt"

	"github.com/andrew/swhid-go/objects"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// FromRevision computes the SWHID for a Git revision (commit).
func FromRevision(repoPath, ref string) (*Identifier, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	if ref == "" {
		ref = "HEAD"
	}

	hash, err := repo.ResolveRevision(plumbing.Revision(ref))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve reference %s: %w", ref, err)
	}

	encoded, err := repo.Storer.EncodedObject(plumbing.CommitObject, *hash)
	if err != nil {
		return nil, fmt.Errorf("failed to read commit: %w", err)
	}
	if err := verifyGitObject(encoded); err != nil {
		return nil, fmt.Errorf("failed to verify commit: %w", err)
	}

	return NewIdentifier(ObjectTypeRevision, hash.String(), nil)
}

// FromRelease computes the SWHID for a Git release (annotated tag).
func FromRelease(repoPath, tagName string) (*Identifier, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	refName := plumbing.NewTagReferenceName(tagName)
	ref, err := repo.Reference(refName, true)
	if err != nil {
		return nil, fmt.Errorf("tag %s not found: %w", tagName, err)
	}

	encoded, err := repo.Storer.EncodedObject(plumbing.TagObject, ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("lightweight tags are not supported for release SWHIDs: %w", err)
	}
	if err := verifyGitObject(encoded); err != nil {
		return nil, fmt.Errorf("failed to verify tag: %w", err)
	}

	return NewIdentifier(ObjectTypeRelease, ref.Hash().String(), nil)
}

// FromSnapshot computes the SWHID for a Git repository snapshot.
func FromSnapshot(repoPath string) (*Identifier, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	var branches []objects.Branch
	head, err := repo.Reference(plumbing.HEAD, false)
	if err != nil && !errors.Is(err, plumbing.ErrReferenceNotFound) {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}
	if err == nil && head.Type() == plumbing.SymbolicReference {
		branches = append(branches, objects.Branch{
			Name:       "HEAD",
			TargetType: objects.BranchTargetAlias,
			Target:     head.Target().String(),
		})
	}

	refs, err := repo.References()
	if err != nil {
		return nil, fmt.Errorf("failed to get references: %w", err)
	}

	err = refs.ForEach(func(ref *plumbing.Reference) error {
		if !ref.Name().IsBranch() && !ref.Name().IsTag() {
			return nil
		}

		branch := objects.Branch{Name: ref.Name().String()}
		if ref.Type() == plumbing.SymbolicReference {
			branch.TargetType = objects.BranchTargetAlias
			branch.Target = ref.Target().String()
		} else {
			branch.TargetType, branch.Target, err = resolveRefTarget(repo, ref.Hash())
			if err != nil {
				return fmt.Errorf("failed to resolve reference %s: %w", ref.Name(), err)
			}
		}
		branches = append(branches, branch)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to iterate references: %w", err)
	}

	return FromSnapshotBranches(branches)
}

func resolveRefTarget(repo *git.Repository, hash plumbing.Hash) (objects.BranchTargetType, string, error) {
	obj, err := repo.Storer.EncodedObject(plumbing.AnyObject, hash)
	if errors.Is(err, plumbing.ErrObjectNotFound) {
		return objects.BranchTargetDangling, "", nil
	}
	if err != nil {
		return "", "", err
	}

	switch obj.Type() {
	case plumbing.CommitObject:
		return objects.BranchTargetRevision, hash.String(), nil
	case plumbing.TagObject:
		return objects.BranchTargetRelease, hash.String(), nil
	case plumbing.TreeObject:
		return objects.BranchTargetDirectory, hash.String(), nil
	case plumbing.BlobObject:
		return objects.BranchTargetContent, hash.String(), nil
	default:
		return "", "", fmt.Errorf("unsupported Git object type %s", obj.Type())
	}
}

func verifyGitObject(object plumbing.EncodedObject) error {
	reader, err := object.Reader()
	if err != nil {
		return err
	}
	verifyErr := objects.VerifyObjectHash(object.Type().String(), object.Size(), reader, object.Hash().String())
	closeErr := reader.Close()
	if verifyErr != nil {
		return verifyErr
	}
	return closeErr
}
