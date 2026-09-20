# swhid-go

A Go library and CLI for computing and parsing SoftWare Hash IDentifiers (SWHIDs). It targets the [SWHID v1.2 specification](https://www.swhid.org/specification/v1.2/), published as ISO/IEC 18670:2025.

The package handles content, directories, revisions, releases, snapshots, and qualified identifiers. SHA-1 calculations use collision detection and return an error if a collision is found.

## Install

The module requires Go 1.26 or later.

```bash
go get github.com/andrew/swhid-go
go install github.com/andrew/swhid-go/cmd/swhid@latest
```

## Go API

Go applications should import the package directly. Every hashing function returns an error because malformed metadata, input failures, and detected SHA-1 collisions cannot produce a valid SWHID.

```go
package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/andrew/swhid-go"
	"github.com/andrew/swhid-go/objects"
)

func main() {
	contentID, err := swhid.FromContent([]byte("hello\n"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(contentID)

	parsed, err := swhid.Parse("swh:1:cnt:ce013625030ba8dba906f756967f9e9ca394464a")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(parsed.ObjectType)

	directoryID, err := swhid.FromDirectory([]objects.DirectoryEntry{
		{Name: "hello.txt", Type: objects.EntryTypeFile, Target: contentID.ObjectHash},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(directoryID)

	data, err := json.Marshal(contentID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
```

`FromContentReader` hashes a stream with a known byte length. It is the better choice for large files. `FromDirectoryPath` walks a filesystem tree. `FromRevision`, `FromRelease`, and `FromSnapshot` read Git repositories through go-git.

Qualifiers are decoded in memory and emitted in canonical order. Construction validates the rules for `origin`, `visit`, `anchor`, `path`, `lines`, and `bytes`.

```go
qualified, err := contentID.WithQualifiers(map[string]string{
	"origin": "https://example.com/source.git",
	"path":   "/src/main.go",
	"lines":  "10-20",
})
if err != nil {
	log.Fatal(err)
}
fmt.Println(qualified)
```

`Identifier` implements `encoding.TextMarshaler` and `encoding.TextUnmarshaler`. JSON encoders therefore store an identifier as its canonical string when it appears inside another Go value.

The JSON value in the example is `"swh:1:cnt:ce013625030ba8dba906f756967f9e9ca394464a"`.

`objects.RevisionMetadata` and `objects.ReleaseMetadata` use `MessagePresent` to distinguish a missing message from a present empty message. Set it when the serialized object has a message separator but no message bytes.

## CLI

```text
swhid parse [options] <swhid>
swhid content [options]
swhid directory [options] <path>
swhid revision [options] <repo> [ref]
swhid release [options] <repo> <tag>
swhid snapshot [options] <repo>
```

Content is read from standard input. Release identifiers require annotated Git tags. Snapshot identifiers include local branches, tags, and one symbolic `HEAD`; remote-tracking references are excluded.

```bash
printf 'hello\n' | swhid content -f raw
swhid directory /path/to/tree --format json
swhid revision /path/to/repo HEAD --format raw
swhid release /path/to/repo v1.0.0 --format jsonl
swhid snapshot /path/to/repo --qualifier origin=https://example.com/repo.git
```

Options may appear before or after positional arguments. The output formats are:

- `text`: labelled output for a terminal. This is the default.
- `raw`: the canonical SWHID followed by a newline.
- `json`: an indented object with `swhid`, `core`, `object_type`, `object_hash`, and `qualifiers` fields.
- `jsonl`: the same object on one line. Each invocation emits one record.

Exit status `0` means success. Status `1` reports invalid input, hashing failures, or filesystem and Git errors. Status `2` reports command-line usage errors. Machine output goes to standard output and diagnostics go to standard error.

## Ruby subprocess use

Ruby applications can call the CLI with `Open3` and parse the JSON result. Pass command arguments as an array so paths and qualifier values do not pass through a shell.

```ruby
require "json"
require "open3"

stdout, stderr, status = Open3.capture3(
  "swhid", "parse", value, "--format", "json"
)
raise stderr unless status.success?

result = JSON.parse(stdout)
result.fetch("swhid")
```

Binary content can be sent through standard input without interpolation:

```ruby
stdout, stderr, status = Open3.capture3(
  "swhid", "content", "--format", "json",
  stdin_data: bytes,
  binmode: true
)
raise stderr unless status.success?

result = JSON.parse(stdout)
```

Use `jsonl` when a caller collects records from several invocations into one stream. A long-running batch protocol would need a separate request schema for binary content and filesystem paths; the current CLI processes one object per invocation.

## Filesystem behavior

`FromDirectoryPath` and `swhid directory` apply these rules:

- Files are streamed into the collision-detecting hasher.
- Symlinks hash their link target and are not followed.
- Git index modes determine executable files when an index is available.
- Gitlinks use mode `160000` and the revision stored in the index.
- `.git` entries are excluded at every directory depth.
- Sockets, devices, FIFOs, and other special files return an error.

## References

- [SWHID specification v1.2](https://www.swhid.org/specification/v1.2/)
- [Software Heritage](https://www.softwareheritage.org/)
- [Rust reference implementation](https://github.com/swhid/swhid-rs)
- [Software Heritage Python model](https://gitlab.softwareheritage.org/swh/devel/swh-model)

## License

MIT
