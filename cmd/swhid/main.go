package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/andrew/swhid-go"
)

var (
	formatFlag     string
	qualifierFlags qualifierList
)

type qualifierList map[string]string

func (q *qualifierList) String() string {
	return fmt.Sprintf("%v", *q)
}

func (q *qualifierList) Set(value string) error {
	parts := strings.SplitN(value, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid qualifier format: %s (expected KEY=VALUE)", value)
	}
	(*q)[parts[0]] = parts[1]
	return nil
}

func init() {
	qualifierFlags = make(qualifierList)
}

func main() {
	if len(os.Args) < 2 {
		showHelp()
		os.Exit(0)
	}

	command := os.Args[1]

	// Parse flags after command
	fs := flag.NewFlagSet(command, flag.ExitOnError)
	fs.StringVar(&formatFlag, "f", "text", "Output format (text, json)")
	fs.StringVar(&formatFlag, "format", "text", "Output format (text, json)")
	fs.Var(&qualifierFlags, "q", "Add qualifier (KEY=VALUE)")
	fs.Var(&qualifierFlags, "qualifier", "Add qualifier (KEY=VALUE)")

	// Skip the command name when parsing
	if len(os.Args) > 2 {
		fs.Parse(os.Args[2:])
	}

	args := fs.Args()

	var err error
	switch command {
	case "parse":
		err = runParse(args)
	case "content":
		err = runContent()
	case "directory":
		err = runDirectory(args)
	case "revision":
		err = runRevision(args)
	case "release":
		err = runRelease(args)
	case "snapshot":
		err = runSnapshot(args)
	case "help", "-h", "--help":
		showHelp()
	default:
		showHelp()
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runParse(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("SWHID string required")
	}

	id, err := swhid.Parse(args[0])
	if err != nil {
		return err
	}

	outputIdentifier(id)
	return nil
}

func runContent() error {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read stdin: %w", err)
	}

	id := swhid.FromContent(data)
	id = applyQualifiers(id)
	outputIdentifier(id)
	return nil
}

func runDirectory(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("directory path required")
	}

	path := args[0]

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("path does not exist: %s", path)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", path)
	}

	id, err := swhid.FromDirectoryPath(path)
	if err != nil {
		return err
	}

	id = applyQualifiers(id)
	outputIdentifier(id)
	return nil
}

func runRevision(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("repository path required")
	}

	repoPath := args[0]
	ref := "HEAD"
	if len(args) > 1 {
		ref = args[1]
	}

	id, err := swhid.FromRevision(repoPath, ref)
	if err != nil {
		return err
	}

	id = applyQualifiers(id)
	outputIdentifier(id)
	return nil
}

func runRelease(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("repository path and tag name required")
	}

	repoPath := args[0]
	tagName := args[1]

	id, err := swhid.FromRelease(repoPath, tagName)
	if err != nil {
		return err
	}

	id = applyQualifiers(id)
	outputIdentifier(id)
	return nil
}

func runSnapshot(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("repository path required")
	}

	repoPath := args[0]

	id, err := swhid.FromSnapshot(repoPath)
	if err != nil {
		return err
	}

	id = applyQualifiers(id)
	outputIdentifier(id)
	return nil
}

func applyQualifiers(id *swhid.Identifier) *swhid.Identifier {
	if len(qualifierFlags) == 0 {
		return id
	}

	quals := make(map[string]string)
	for k, v := range qualifierFlags {
		quals[k] = v
	}
	return id.WithQualifiers(quals)
}

func outputIdentifier(id *swhid.Identifier) {
	switch formatFlag {
	case "json":
		outputJSON(id)
	default:
		outputText(id)
	}
}

func outputText(id *swhid.Identifier) {
	fmt.Printf("SWHID: %s\n", id.String())
	fmt.Printf("Core:  %s\n", id.CoreSWHID())
	fmt.Printf("Type:  %s\n", id.ObjectType)
	fmt.Printf("Hash:  %s\n", id.ObjectHash)

	if len(id.Qualifiers) > 0 {
		fmt.Println("Qualifiers:")
		for key, value := range id.Qualifiers {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}
}

func outputJSON(id *swhid.Identifier) {
	data := map[string]interface{}{
		"swhid":       id.String(),
		"core":        id.CoreSWHID(),
		"object_type": id.ObjectType,
		"object_hash": id.ObjectHash,
		"qualifiers":  id.Qualifiers,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.Encode(data)
}

func showHelp() {
	fmt.Print(`swhid - Generate and parse SoftWare Hash IDentifiers

Usage:
  swhid parse <swhid>                   Parse and validate a SWHID
  swhid content [options]               Generate SWHID for content from stdin
  swhid directory <path> [options]      Generate SWHID for directory
  swhid revision <repo> [ref] [options] Generate SWHID for git revision/commit
  swhid release <repo> <tag> [options]  Generate SWHID for git release/tag
  swhid snapshot <repo> [options]       Generate SWHID for git snapshot

Options:
  -f, --format FORMAT              Output format (text, json)
  -q, --qualifier KEY=VALUE        Add qualifier to generated SWHID
  -h, --help                       Show this help

Examples:
  # Parse a SWHID
  swhid parse swh:1:cnt:94a9ed024d3859793618152ea559a168bbcbb5e2

  # Generate SWHID from file content
  cat file.txt | swhid content

  # Generate SWHID from directory
  swhid directory /path/to/dir

  # Generate SWHID from git commit
  swhid revision /path/to/repo
  swhid revision /path/to/repo main
  swhid revision /path/to/repo abc123

  # Generate SWHID from git tag
  swhid release /path/to/repo v1.0.0

  # Generate SWHID from git snapshot
  swhid snapshot /path/to/repo

  # Generate SWHID with qualifiers
  cat file.txt | swhid content -q origin=https://github.com/example/repo

  # Output as JSON
  swhid parse swh:1:cnt:94a9ed024d3859793618152ea559a168bbcbb5e2 -f json

For more information, visit: https://www.swhid.org/
`)
}
