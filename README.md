# swhid-go

A Go library and CLI for computing Software Heritage Identifiers (SWHIDs).

SWHIDs are intrinsic identifiers for digital objects based on cryptographic hashes. They're used by Software Heritage to uniquely identify source code artifacts.

## Installation

```bash
go get github.com/andrew/swhid-go
```

For the CLI:

```bash
go install github.com/andrew/swhid-go/cmd/swhid@latest
```

## Library Usage

```go
package main

import (
    "fmt"
    "github.com/andrew/swhid-go"
    "github.com/andrew/swhid-go/objects"
)

func main() {
    // Compute SWHID for content
    id := swhid.FromContent([]byte("hello\n"))
    fmt.Println(id) // swh:1:cnt:ce013625030ba8dba906f756967f9e9ca394464a

    // Parse an existing SWHID
    parsed, _ := swhid.Parse("swh:1:cnt:ce013625030ba8dba906f756967f9e9ca394464a")
    fmt.Println(parsed.ObjectType) // cnt
    fmt.Println(parsed.ObjectHash) // ce013625030ba8dba906f756967f9e9ca394464a

    // Compute SWHID for a directory
    entries := []objects.DirectoryEntry{
        {Name: "hello.txt", Type: objects.EntryTypeFile, Target: "ce013625030ba8dba906f756967f9e9ca394464a"},
    }
    dirID := swhid.FromDirectory(entries)
    fmt.Println(dirID) // swh:1:dir:...

    // Hash a directory from the filesystem
    fsID, _ := swhid.FromDirectoryPath("/path/to/dir")
    fmt.Println(fsID)

    // Hash a git commit
    revID, _ := swhid.FromRevision("/path/to/repo", "HEAD")
    fmt.Println(revID)
}
```

## CLI Usage

```bash
# Parse and validate a SWHID
swhid parse swh:1:cnt:ce013625030ba8dba906f756967f9e9ca394464a

# Generate SWHID from file content (stdin)
echo "hello" | swhid content

# Generate SWHID from directory
swhid directory /path/to/dir

# Generate SWHID from git commit
swhid revision /path/to/repo
swhid revision /path/to/repo main
swhid revision /path/to/repo abc123

# Generate SWHID from annotated git tag
swhid release /path/to/repo v1.0.0

# Generate SWHID for repository snapshot
swhid snapshot /path/to/repo

# JSON output (flag before positional args)
swhid parse -f json swh:1:cnt:ce013625030ba8dba906f756967f9e9ca394464a

# Add qualifiers
echo "hello" | swhid content -q origin=https://github.com/example/repo
```

## Object Types

| Type | Code | Description |
|------|------|-------------|
| Content | `cnt` | File content (blob) |
| Directory | `dir` | Directory tree |
| Revision | `rev` | Git commit |
| Release | `rel` | Annotated tag |
| Snapshot | `snp` | Repository state |

## SWHID Format

```
swh:1:<type>:<hash>[;<qualifier>=<value>...]
```

- `swh` - scheme
- `1` - version
- `<type>` - object type (cnt, dir, rev, rel, snp)
- `<hash>` - 40-character SHA1 hex digest
- `<qualifier>` - optional qualifiers (origin, visit, anchor, path, lines, bytes)

## Links

- [SWHID Specification](https://www.swhid.org/)
- [Software Heritage](https://www.softwareheritage.org/)
- [Ruby implementation](https://github.com/swhid/swhid)

## License

MIT
