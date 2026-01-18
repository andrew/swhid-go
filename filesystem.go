package swhid

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/andrew/swhid-go/objects"
	"github.com/go-git/go-git/v5"
)

// FromDirectoryPath computes the SWHID for a directory on the filesystem.
// It recursively hashes all files and subdirectories.
// If the directory is within a Git repository, it uses the Git index for file permissions.
func FromDirectoryPath(path string) (*Identifier, error) {
	return FromDirectoryPathWithOptions(path, nil, nil)
}

// FromDirectoryPathWithOptions computes the SWHID with custom options.
// gitRepo can be provided to use Git index for permissions.
// permissions can be provided as a map of path -> mode for explicit permissions.
func FromDirectoryPathWithOptions(path string, gitRepo *git.Repository, permissions map[string]os.FileMode) (*Identifier, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, &os.PathError{Op: "swhid", Path: path, Err: os.ErrInvalid}
	}

	// Try to discover Git repo if not provided
	if gitRepo == nil {
		gitRepo = discoverGitRepo(path)
	}

	entries, err := buildEntries(path, gitRepo, permissions)
	if err != nil {
		return nil, err
	}

	return FromDirectory(entries), nil
}

func discoverGitRepo(path string) *git.Repository {
	// Walk up the directory tree looking for .git
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil
	}

	for {
		repo, err := git.PlainOpen(absPath)
		if err == nil {
			return repo
		}

		parent := filepath.Dir(absPath)
		if parent == absPath {
			break
		}
		absPath = parent
	}

	return nil
}

func buildEntries(dirPath string, gitRepo *git.Repository, permissions map[string]os.FileMode) ([]objects.DirectoryEntry, error) {
	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var entries []objects.DirectoryEntry

	for _, de := range dirEntries {
		name := de.Name()

		// Skip .git directory
		if name == ".git" {
			continue
		}

		fullPath := filepath.Join(dirPath, name)
		info, err := de.Info()
		if err != nil {
			return nil, err
		}

		var entry objects.DirectoryEntry

		// Check if it's a symlink
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(fullPath)
			if err != nil {
				return nil, err
			}
			targetHash := objects.ComputeContentHash([]byte(target))
			entry = objects.DirectoryEntry{
				Name:   name,
				Type:   objects.EntryTypeSymlink,
				Target: targetHash,
			}
		} else if info.IsDir() {
			// Recurse into subdirectory
			subID, err := FromDirectoryPathWithOptions(fullPath, gitRepo, permissions)
			if err != nil {
				return nil, err
			}
			entry = objects.DirectoryEntry{
				Name:   name,
				Type:   objects.EntryTypeDirectory,
				Target: subID.ObjectHash,
			}
		} else {
			// Regular file
			content, err := os.ReadFile(fullPath)
			if err != nil {
				return nil, err
			}
			targetHash := objects.ComputeContentHash(content)

			entryType := objects.EntryTypeFile
			if isExecutable(fullPath, info, gitRepo, permissions) {
				entryType = objects.EntryTypeExecutable
			}

			entry = objects.DirectoryEntry{
				Name:   name,
				Type:   entryType,
				Target: targetHash,
			}
		}

		entries = append(entries, entry)
	}

	// Sort for deterministic output
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].SortKey() < entries[j].SortKey()
	})

	return entries, nil
}

func isExecutable(fullPath string, info os.FileInfo, gitRepo *git.Repository, permissions map[string]os.FileMode) bool {
	// Check explicit permissions map first
	if permissions != nil {
		if mode, ok := permissions[fullPath]; ok {
			return mode&0111 != 0
		}
		// Try with resolved path
		absPath, err := filepath.Abs(fullPath)
		if err == nil {
			if mode, ok := permissions[absPath]; ok {
				return mode&0111 != 0
			}
		}
	}

	// Check Git index for tracked files
	if gitRepo != nil {
		relPath := relativePathInRepo(fullPath, gitRepo)
		if relPath != "" {
			// Try to get mode from index
			idx, err := gitRepo.Storer.Index()
			if err == nil {
				for _, entry := range idx.Entries {
					if entry.Name == relPath {
						return entry.Mode&0111 != 0
					}
				}
			}
		}
	}

	// Fall back to filesystem
	return info.Mode()&0111 != 0
}

func relativePathInRepo(fullPath string, gitRepo *git.Repository) string {
	worktree, err := gitRepo.Worktree()
	if err != nil {
		return ""
	}

	repoRoot := worktree.Filesystem.Root()
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return ""
	}

	// Resolve symlinks
	absPath, _ = filepath.EvalSymlinks(absPath)
	repoRoot, _ = filepath.EvalSymlinks(repoRoot)

	// Normalize separators
	absPath = filepath.ToSlash(absPath)
	repoRoot = filepath.ToSlash(repoRoot)

	if !hasPrefix(absPath, repoRoot) {
		return ""
	}

	rel := absPath[len(repoRoot):]
	if len(rel) > 0 && rel[0] == '/' {
		rel = rel[1:]
	}
	return rel
}

func hasPrefix(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	return s[:len(prefix)] == prefix
}
