package swhid

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/andrew/swhid-go/objects"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
)

type indexedEntry struct {
	mode filemode.FileMode
	hash plumbing.Hash
}

// FromDirectoryPath computes the SWHID for a directory on the filesystem.
// It recursively hashes all files and subdirectories.
// If the directory is within a Git repository, it uses the Git index for file permissions.
func FromDirectoryPath(dirPath string) (*Identifier, error) {
	return FromDirectoryPathWithOptions(dirPath, nil, nil)
}

// FromDirectoryPathWithOptions computes the SWHID with custom options.
// gitRepo can be provided to use Git index for permissions.
// permissions can be provided as a map of path -> mode for explicit permissions.
func FromDirectoryPathWithOptions(dirPath string, gitRepo *git.Repository, permissions map[string]os.FileMode) (*Identifier, error) {
	info, err := os.Stat(dirPath)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, &os.PathError{Op: "swhid", Path: dirPath, Err: os.ErrInvalid}
	}

	if gitRepo == nil {
		gitRepo = discoverGitRepo(dirPath)
	}

	indexEntries, err := loadIndexEntries(gitRepo)
	if err != nil {
		return nil, fmt.Errorf("read Git index: %w", err)
	}
	repoRoot, err := repositoryRoot(gitRepo)
	if err != nil {
		return nil, fmt.Errorf("find Git worktree: %w", err)
	}

	repoRelativePath, withinRepo := relativePathInRepo(dirPath, repoRoot)
	return fromDirectoryPath(dirPath, repoRelativePath, withinRepo, permissions, indexEntries)
}

func discoverGitRepo(dirPath string) *git.Repository {
	absPath, err := filepath.Abs(dirPath)
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

func loadIndexEntries(repo *git.Repository) (map[string]indexedEntry, error) {
	if repo == nil {
		return nil, nil
	}
	idx, err := repo.Storer.Index()
	if err != nil {
		return nil, err
	}
	entries := make(map[string]indexedEntry, len(idx.Entries))
	for _, entry := range idx.Entries {
		entries[entry.Name] = indexedEntry{mode: entry.Mode, hash: entry.Hash}
	}
	return entries, nil
}

func repositoryRoot(repo *git.Repository) (string, error) {
	if repo == nil {
		return "", nil
	}
	worktree, err := repo.Worktree()
	if err != nil {
		return "", err
	}
	root, err := filepath.Abs(worktree.Filesystem.Root())
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}
	return root, nil
}

func fromDirectoryPath(dirPath, repoRelativePath string, withinRepo bool, permissions map[string]os.FileMode, indexEntries map[string]indexedEntry) (*Identifier, error) {
	entries, err := buildEntries(dirPath, repoRelativePath, withinRepo, permissions, indexEntries)
	if err != nil {
		return nil, err
	}
	return FromDirectory(entries)
}

func buildEntries(dirPath, repoRelativePath string, withinRepo bool, permissions map[string]os.FileMode, indexEntries map[string]indexedEntry) ([]objects.DirectoryEntry, error) {
	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	entries := make([]objects.DirectoryEntry, 0, len(dirEntries))
	for _, de := range dirEntries {
		name := de.Name()
		if name == ".git" {
			continue
		}
		fullPath := filepath.Join(dirPath, name)
		relPath := ""
		if withinRepo {
			relPath = path.Join(repoRelativePath, name)
		}

		indexEntry, tracked := indexEntries[relPath]
		if tracked && indexEntry.mode == filemode.Submodule {
			entries = append(entries, objects.DirectoryEntry{
				Name:   name,
				Type:   objects.EntryTypeRevision,
				Target: indexEntry.hash.String(),
			})
			continue
		}

		info, err := de.Info()
		if err != nil {
			return nil, err
		}

		entry := objects.DirectoryEntry{Name: name}
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			target, err := os.Readlink(fullPath)
			if err != nil {
				return nil, err
			}
			entry.Type = objects.EntryTypeSymlink
			entry.Target, err = objects.ComputeContentHash([]byte(target))
			if err != nil {
				return nil, err
			}
		case info.IsDir():
			subID, err := fromDirectoryPath(fullPath, relPath, withinRepo, permissions, indexEntries)
			if err != nil {
				return nil, err
			}
			entry.Type = objects.EntryTypeDirectory
			entry.Target = subID.ObjectHash
		case info.Mode().IsRegular():
			entry.Type = objects.EntryTypeFile
			if isExecutable(fullPath, relPath, info, permissions, indexEntries) {
				entry.Type = objects.EntryTypeExecutable
			}
			entry.Target, err = hashFile(fullPath, info.Size())
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unsupported file type %s", fullPath)
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func hashFile(path string, size int64) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	hash, hashErr := objects.ComputeContentHashReader(file, size)
	closeErr := file.Close()
	if hashErr != nil {
		return "", hashErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	return hash, nil
}

func isExecutable(fullPath, relPath string, info os.FileInfo, permissions map[string]os.FileMode, indexEntries map[string]indexedEntry) bool {
	if mode, ok := permissions[fullPath]; ok {
		return mode&0111 != 0
	}
	absPath, err := filepath.Abs(fullPath)
	if err == nil {
		if mode, ok := permissions[absPath]; ok {
			return mode&0111 != 0
		}
	}
	if entry, ok := indexEntries[relPath]; ok {
		return entry.mode == filemode.Executable
	}
	return info.Mode()&0111 != 0
}

func relativePathInRepo(fullPath, repoRoot string) (string, bool) {
	if repoRoot == "" {
		return "", false
	}
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", false
	}
	if resolved, err := filepath.EvalSymlinks(absPath); err == nil {
		absPath = resolved
	}
	relPath, err := filepath.Rel(repoRoot, absPath)
	if err != nil || relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return "", false
	}
	if relPath == "." {
		return "", true
	}
	return filepath.ToSlash(relPath), true
}
