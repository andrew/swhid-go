package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/andrew/swhid-go"
)

const (
	exitSuccess      = 0
	exitCommandError = 1
	exitUsageError   = 2
	qualifierParts   = 2
	optionalRefArgs  = 2
	releaseArgs      = 2
	formatText       = "text"
	formatRaw        = "raw"
	formatJSON       = "json"
	formatJSONL      = "jsonl"
	supportedFormats = "text, raw, json, jsonl"
)

var (
	version       = "devel"
	readBuildInfo = debug.ReadBuildInfo
)

type commandOptions struct {
	format     string
	qualifiers qualifierList
}

type qualifierList map[string]string

type usageError struct {
	message string
}

func (e *usageError) Error() string {
	return e.message
}

type identifierOutput struct {
	SWHID      string            `json:"swhid"`
	Core       string            `json:"core"`
	ObjectType swhid.ObjectType  `json:"object_type"`
	ObjectHash string            `json:"object_hash"`
	Qualifiers map[string]string `json:"qualifiers"`
}

func (q *qualifierList) String() string {
	keys := make([]string, 0, len(*q))
	for key := range *q {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+(*q)[key])
	}
	return strings.Join(parts, ",")
}

func (q *qualifierList) Set(value string) error {
	parts := strings.SplitN(value, "=", qualifierParts)
	if len(parts) != 2 || parts[0] == "" {
		return fmt.Errorf("invalid qualifier format: %s (expected KEY=VALUE)", value)
	}
	if _, exists := (*q)[parts[0]]; exists {
		return fmt.Errorf("duplicate qualifier: %s", parts[0])
	}
	(*q)[parts[0]] = parts[1]
	return nil
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		showHelp(stdout)
		return exitSuccess
	}
	if args[0] == "version" || args[0] == "-version" || args[0] == "--version" {
		_, _ = fmt.Fprintf(stdout, "swhid %s\n", reportedVersion())
		return exitSuccess
	}

	command := args[0]
	options := commandOptions{qualifiers: make(qualifierList)}
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&options.format, "f", formatText, "Output format ("+supportedFormats+")")
	fs.StringVar(&options.format, "format", formatText, "Output format ("+supportedFormats+")")
	fs.Var(&options.qualifiers, "q", "Add qualifier (KEY=VALUE)")
	fs.Var(&options.qualifiers, "qualifier", "Add qualifier (KEY=VALUE)")
	if err := fs.Parse(optionsFirst(args[1:])); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitSuccess
		}
		return exitUsageError
	}
	if !validFormat(options.format) {
		_, _ = fmt.Fprintf(stderr, "Error: unsupported output format %q; supported formats: %s\n", options.format, supportedFormats)
		return exitUsageError
	}
	if command == "parse" && len(options.qualifiers) != 0 {
		_, _ = fmt.Fprintln(stderr, "Error: qualifiers cannot be added when parsing a SWHID")
		return exitUsageError
	}

	var err error
	switch command {
	case "parse":
		err = runParse(fs.Args(), options, stdout)
	case "content":
		err = runContent(stdin, options, stdout)
	case "directory":
		err = runDirectory(fs.Args(), options, stdout)
	case "revision":
		err = runRevision(fs.Args(), options, stdout)
	case "release":
		err = runRelease(fs.Args(), options, stdout)
	case "snapshot":
		err = runSnapshot(fs.Args(), options, stdout)
	case "help", "-h", "--help":
		showHelp(stdout)
		return exitSuccess
	default:
		_, _ = fmt.Fprintf(stderr, "Error: unknown command %q\n", command)
		showHelp(stderr)
		return exitUsageError
	}

	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)
		var usageErr *usageError
		if errors.As(err, &usageErr) {
			return exitUsageError
		}
		return exitCommandError
	}
	return exitSuccess
}

func reportedVersion() string {
	if version != "devel" {
		return version
	}
	info, ok := readBuildInfo()
	if !ok || info.Main.Version == "" || info.Main.Version == "(devel)" {
		return version
	}
	return info.Main.Version
}

func validFormat(format string) bool {
	switch format {
	case formatText, formatRaw, formatJSON, formatJSONL:
		return true
	default:
		return false
	}
}

func optionsFirst(args []string) []string {
	options := make([]string, 0, len(args))
	positional := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positional = append(positional, arg)
			positional = append(positional, args[i+1:]...)
			break
		}
		switch arg {
		case "-f", "-format", "--format", "-q", "-qualifier", "--qualifier":
			options = append(options, arg)
			if i+1 < len(args) {
				i++
				options = append(options, args[i])
			}
		case "-h", "--help":
			options = append(options, arg)
		default:
			if strings.HasPrefix(arg, "-") {
				options = append(options, arg)
			} else {
				positional = append(positional, arg)
			}
		}
	}
	return append(options, positional...)
}

func newUsageError(message string) error {
	return &usageError{message: message}
}

func runParse(args []string, options commandOptions, output io.Writer) error {
	if len(args) != 1 {
		return newUsageError("exactly one SWHID string is required")
	}
	id, err := swhid.Parse(args[0])
	if err != nil {
		return err
	}
	return outputIdentifier(output, id, options.format)
}

func runContent(input io.Reader, options commandOptions, output io.Writer) error {
	id, err := contentIdentifier(input)
	if err != nil {
		return err
	}
	id, err = applyQualifiers(id, options.qualifiers)
	if err != nil {
		return err
	}
	return outputIdentifier(output, id, options.format)
}

func contentIdentifier(input io.Reader) (*swhid.Identifier, error) {
	temp, err := os.CreateTemp("", "swhid-content-*")
	if err != nil {
		return nil, fmt.Errorf("create content spool: %w", err)
	}
	name := temp.Name()
	defer func() { _ = os.Remove(name) }()

	size, copyErr := io.Copy(temp, input)
	if copyErr != nil {
		_ = temp.Close()
		return nil, fmt.Errorf("read content: %w", copyErr)
	}
	if _, err := temp.Seek(0, io.SeekStart); err != nil {
		_ = temp.Close()
		return nil, fmt.Errorf("rewind content: %w", err)
	}
	id, hashErr := swhid.FromContentReader(temp, size)
	closeErr := temp.Close()
	if hashErr != nil {
		return nil, hashErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return id, nil
}

func runDirectory(args []string, options commandOptions, output io.Writer) error {
	if len(args) != 1 {
		return newUsageError("exactly one directory path is required")
	}
	info, err := os.Stat(args[0])
	if err != nil {
		return fmt.Errorf("stat %s: %w", args[0], err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", args[0])
	}
	id, err := swhid.FromDirectoryPath(args[0])
	if err != nil {
		return err
	}
	id, err = applyQualifiers(id, options.qualifiers)
	if err != nil {
		return err
	}
	return outputIdentifier(output, id, options.format)
}

func runRevision(args []string, options commandOptions, output io.Writer) error {
	if len(args) < 1 || len(args) > optionalRefArgs {
		return newUsageError("repository path and at most one reference are required")
	}
	ref := "HEAD"
	if len(args) == optionalRefArgs {
		ref = args[1]
	}
	id, err := swhid.FromRevision(args[0], ref)
	if err != nil {
		return err
	}
	id, err = applyQualifiers(id, options.qualifiers)
	if err != nil {
		return err
	}
	return outputIdentifier(output, id, options.format)
}

func runRelease(args []string, options commandOptions, output io.Writer) error {
	if len(args) != releaseArgs {
		return newUsageError("repository path and tag name are required")
	}
	id, err := swhid.FromRelease(args[0], args[1])
	if err != nil {
		return err
	}
	id, err = applyQualifiers(id, options.qualifiers)
	if err != nil {
		return err
	}
	return outputIdentifier(output, id, options.format)
}

func runSnapshot(args []string, options commandOptions, output io.Writer) error {
	if len(args) != 1 {
		return newUsageError("exactly one repository path is required")
	}
	id, err := swhid.FromSnapshot(args[0])
	if err != nil {
		return err
	}
	id, err = applyQualifiers(id, options.qualifiers)
	if err != nil {
		return err
	}
	return outputIdentifier(output, id, options.format)
}

func applyQualifiers(id *swhid.Identifier, qualifiers qualifierList) (*swhid.Identifier, error) {
	if len(qualifiers) == 0 {
		return id, nil
	}
	values := make(map[string]string, len(qualifiers))
	for key, value := range qualifiers {
		values[key] = value
	}
	return id.WithQualifiers(values)
}

func outputIdentifier(output io.Writer, id *swhid.Identifier, format string) error {
	switch format {
	case formatRaw:
		_, err := fmt.Fprintln(output, id.String())
		return err
	case formatJSON:
		return outputJSON(output, id, true)
	case formatJSONL:
		return outputJSON(output, id, false)
	default:
		return outputText(output, id)
	}
}

func outputText(output io.Writer, id *swhid.Identifier) error {
	if _, err := fmt.Fprintf(output, "SWHID: %s\nCore:  %s\nType:  %s\nHash:  %s\n", id.String(), id.CoreSWHID(), id.ObjectType, id.ObjectHash); err != nil {
		return err
	}
	if len(id.Qualifiers) == 0 {
		return nil
	}
	if _, err := fmt.Fprintln(output, "Qualifiers:"); err != nil {
		return err
	}
	keys := make([]string, 0, len(id.Qualifiers))
	for key := range id.Qualifiers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if _, err := fmt.Fprintf(output, "  %s: %s\n", key, id.Qualifiers[key]); err != nil {
			return err
		}
	}
	return nil
}

func outputJSON(output io.Writer, id *swhid.Identifier, indent bool) error {
	data := identifierOutput{
		SWHID:      id.String(),
		Core:       id.CoreSWHID(),
		ObjectType: id.ObjectType,
		ObjectHash: id.ObjectHash,
		Qualifiers: id.Qualifiers,
	}
	encoder := json.NewEncoder(output)
	if indent {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(data)
}

func showHelp(output io.Writer) {
	_, _ = fmt.Fprint(output, `swhid - Generate and parse SoftWare Hash IDentifiers

Usage:
  swhid parse [options] <swhid>             Parse and validate a SWHID
  swhid content [options]                   Generate SWHID for content from stdin
  swhid directory [options] <path>          Generate SWHID for a directory
  swhid revision [options] <repo> [ref]     Generate SWHID for a Git revision
  swhid release [options] <repo> <tag>      Generate SWHID for a Git release
  swhid snapshot [options] <repo>           Generate SWHID for a Git snapshot
  swhid version                             Show the swhid version

Options:
  -f, --format FORMAT              Output format (text, raw, json, jsonl)
  -q, --qualifier KEY=VALUE        Add qualifier to a generated SWHID
  -h, --help                       Show command help
      --version                    Show the swhid version

Examples:
  swhid parse swh:1:cnt:94a9ed024d3859793618152ea559a168bbcbb5e2
  cat file.txt | swhid content
  swhid directory /path/to/dir
  swhid revision /path/to/repo main
  swhid release /path/to/repo v1.0.0
  swhid snapshot /path/to/repo
  cat file.txt | swhid content -q origin=https://github.com/example/repo
  swhid parse -f json swh:1:cnt:94a9ed024d3859793618152ea559a168bbcbb5e2
  swhid parse swh:1:cnt:94a9ed024d3859793618152ea559a168bbcbb5e2 -f raw

For more information, visit: https://www.swhid.org/
`)
}
