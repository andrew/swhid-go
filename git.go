package swhid

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/andrew/swhid-go/objects"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
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

	commit, err := repo.CommitObject(*hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	meta := objects.RevisionMetadata{
		Directory:          commit.TreeHash.String(),
		Author:             formatPerson(commit.Author),
		AuthorTimestamp:    commit.Author.When.Unix(),
		AuthorTimezone:     formatTimezone(commit.Author.When),
		Committer:          formatPerson(commit.Committer),
		CommitterTimestamp: commit.Committer.When.Unix(),
		CommitterTimezone:  formatTimezone(commit.Committer.When),
		Message:            commit.Message,
	}

	// Get parent hashes
	for _, parentHash := range commit.ParentHashes {
		meta.Parents = append(meta.Parents, parentHash.String())
	}

	// Extract extra headers from raw commit
	extraHeaders := extractCommitExtraHeaders(repo, commit)
	if len(extraHeaders) > 0 {
		meta.ExtraHeaders = extraHeaders
	}

	return FromRevisionMetadata(meta), nil
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

	// Try to get the tag object
	tagObj, err := repo.TagObject(ref.Hash())
	if err != nil {
		// Lightweight tag - not supported
		return nil, fmt.Errorf("lightweight tags are not supported for release SWHIDs")
	}

	// Determine target type
	targetType := objects.TargetTypeRevision
	if _, err := repo.CommitObject(tagObj.Target); err == nil {
		targetType = objects.TargetTypeRevision
	} else if _, err := repo.TagObject(tagObj.Target); err == nil {
		targetType = objects.TargetTypeRelease
	} else if _, err := repo.TreeObject(tagObj.Target); err == nil {
		targetType = objects.TargetTypeDirectory
	} else if _, err := repo.BlobObject(tagObj.Target); err == nil {
		targetType = objects.TargetTypeContent
	}

	meta := objects.ReleaseMetadata{
		Name: tagObj.Name,
		Target: objects.ReleaseTarget{
			Hash: tagObj.Target.String(),
			Type: targetType,
		},
		Message: tagObj.Message,
	}

	if !tagObj.Tagger.When.IsZero() {
		meta.Author = formatPerson(tagObj.Tagger)
		meta.AuthorTimestamp = tagObj.Tagger.When.Unix()
		meta.AuthorTimezone = formatTimezone(tagObj.Tagger.When)
	}

	// Extract extra headers (like gpgsig for signed tags)
	extraHeaders := extractTagExtraHeaders(repo, tagObj)
	if len(extraHeaders) > 0 {
		meta.ExtraHeaders = extraHeaders
	}

	return FromReleaseMetadata(meta), nil
}

// FromSnapshot computes the SWHID for a Git repository snapshot.
func FromSnapshot(repoPath string) (*Identifier, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	var branches []objects.Branch

	// Check for HEAD first
	gitDir := filepath.Join(repoPath, ".git")
	if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
		headPath := filepath.Join(gitDir, "HEAD")
		if content, err := os.ReadFile(headPath); err == nil {
			headContent := strings.TrimSpace(string(content))
			if strings.HasPrefix(headContent, "ref:") {
				targetRef := strings.TrimSpace(strings.TrimPrefix(headContent, "ref:"))
				branches = append(branches, objects.Branch{
					Name:       "HEAD",
					TargetType: objects.BranchTargetAlias,
					Target:     targetRef,
				})
			}
		}
	}

	// Iterate all references
	refs, err := repo.References()
	if err != nil {
		return nil, fmt.Errorf("failed to get references: %w", err)
	}

	err = refs.ForEach(func(ref *plumbing.Reference) error {
		refName := ref.Name().String()

		if ref.Type() == plumbing.SymbolicReference {
			// Symbolic reference (alias)
			branches = append(branches, objects.Branch{
				Name:       refName,
				TargetType: objects.BranchTargetAlias,
				Target:     ref.Target().String(),
			})
		} else {
			// Direct reference
			targetType, targetHash := resolveRefTarget(repo, ref.Hash())
			branches = append(branches, objects.Branch{
				Name:       refName,
				TargetType: targetType,
				Target:     targetHash,
			})
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to iterate references: %w", err)
	}

	return FromSnapshotBranches(branches), nil
}

func resolveRefTarget(repo *git.Repository, hash plumbing.Hash) (objects.BranchTargetType, string) {
	// Try commit
	if _, err := repo.CommitObject(hash); err == nil {
		return objects.BranchTargetRevision, hash.String()
	}

	// Try tag
	if _, err := repo.TagObject(hash); err == nil {
		return objects.BranchTargetRelease, hash.String()
	}

	// Try tree
	if _, err := repo.TreeObject(hash); err == nil {
		return objects.BranchTargetDirectory, hash.String()
	}

	// Try blob
	if _, err := repo.BlobObject(hash); err == nil {
		return objects.BranchTargetContent, hash.String()
	}

	// Default to revision
	return objects.BranchTargetRevision, hash.String()
}

func formatPerson(sig object.Signature) string {
	return fmt.Sprintf("%s <%s>", sig.Name, sig.Email)
}

func formatTimezone(t interface{ Zone() (string, int) }) string {
	_, offset := t.Zone()
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	hours := offset / 3600
	minutes := (offset % 3600) / 60
	return fmt.Sprintf("%s%02d%02d", sign, hours, minutes)
}

func extractCommitExtraHeaders(repo *git.Repository, commit *object.Commit) [][2]string {
	// Get raw commit data
	obj, err := repo.Storer.EncodedObject(plumbing.CommitObject, commit.Hash)
	if err != nil {
		return nil
	}

	reader, err := obj.Reader()
	if err != nil {
		return nil
	}
	defer reader.Close()

	var buf bytes.Buffer
	buf.ReadFrom(reader)
	rawData := buf.String()

	return parseExtraHeaders(rawData, []string{"tree", "parent", "author", "committer"})
}

func extractTagExtraHeaders(repo *git.Repository, tag *object.Tag) [][2]string {
	obj, err := repo.Storer.EncodedObject(plumbing.TagObject, tag.Hash)
	if err != nil {
		return nil
	}

	reader, err := obj.Reader()
	if err != nil {
		return nil
	}
	defer reader.Close()

	var buf bytes.Buffer
	buf.ReadFrom(reader)
	rawData := buf.String()

	return parseExtraHeaders(rawData, []string{"object", "type", "tag", "tagger"})
}

func parseExtraHeaders(rawData string, standardHeaders []string) [][2]string {
	var extraHeaders [][2]string

	scanner := bufio.NewScanner(strings.NewReader(rawData))
	inHeaders := true

	for scanner.Scan() {
		line := scanner.Text()

		// Stop at blank line (start of message)
		if line == "" {
			inHeaders = false
			continue
		}

		if !inHeaders {
			continue
		}

		// Check for continuation line
		if strings.HasPrefix(line, " ") {
			if len(extraHeaders) > 0 {
				extraHeaders[len(extraHeaders)-1][1] += "\n" + line[1:]
			}
			continue
		}

		// Parse header
		idx := strings.Index(line, " ")
		if idx == -1 {
			continue
		}

		key := line[:idx]
		value := line[idx+1:]

		// Skip standard headers
		isStandard := false
		for _, sh := range standardHeaders {
			if key == sh {
				isStandard = true
				break
			}
		}
		if isStandard {
			continue
		}

		extraHeaders = append(extraHeaders, [2]string{key, value})
	}

	return extraHeaders
}
