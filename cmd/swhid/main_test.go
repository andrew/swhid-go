package main

import (
	"bytes"
	"encoding/json"
	"runtime/debug"
	"strings"
	"testing"
)

const (
	contentCommand    = "content"
	parseCommand      = "parse"
	longFormatFlag    = "-format"
	helloContentSWHID = "swh:1:cnt:3b18e512dba79e4c8300dd08aeb37f8e728b8dad"
	exampleOriginFlag = "origin=https://example.com"
)

func TestRunContentText(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	status := run([]string{contentCommand}, strings.NewReader("hello world\n"), &stdout, &stderr)

	if status != 0 {
		t.Fatalf("run returned %d, stderr: %s", status, stderr.String())
	}
	if !strings.Contains(stdout.String(), "SWHID: "+helloContentSWHID+"\n") {
		t.Fatalf("output did not contain content SWHID: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestRunContentJSONWithQualifier(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	status := run(
		[]string{contentCommand, longFormatFlag, formatJSON, "-qualifier", exampleOriginFlag + "/a+b"},
		strings.NewReader("hello world\n"),
		&stdout,
		&stderr,
	)

	if status != 0 {
		t.Fatalf("run returned %d, stderr: %s", status, stderr.String())
	}
	var output struct {
		SWHID      string            `json:"swhid"`
		Core       string            `json:"core"`
		Qualifiers map[string]string `json:"qualifiers"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if output.Core != helloContentSWHID {
		t.Fatalf("core = %q, want %q", output.Core, helloContentSWHID)
	}
	if output.Qualifiers["origin"] != "https://example.com/a+b" {
		t.Fatalf("origin = %q", output.Qualifiers["origin"])
	}
	if !strings.Contains(output.SWHID, "origin=https://example.com/a+b") {
		t.Fatalf("qualified SWHID did not preserve plus: %s", output.SWHID)
	}
}

func TestRunJSONLIsOneLine(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	status := run(
		[]string{contentCommand, longFormatFlag, formatJSONL},
		strings.NewReader("hello world\n"),
		&stdout,
		&stderr,
	)

	if status != exitSuccess {
		t.Fatalf("run returned %d, stderr: %s", status, stderr.String())
	}
	if lines := strings.Count(stdout.String(), "\n"); lines != 1 {
		t.Fatalf("JSONL output contains %d lines: %q", lines, stdout.String())
	}
	var output identifierOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if output.SWHID != helloContentSWHID {
		t.Errorf("swhid = %q, want %q", output.SWHID, helloContentSWHID)
	}
}

func TestRunRawWithOptionsAfterArgument(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	status := run(
		[]string{parseCommand, helloContentSWHID, longFormatFlag, formatRaw},
		strings.NewReader(""),
		&stdout,
		&stderr,
	)

	if status != exitSuccess {
		t.Fatalf("run returned %d, stderr: %s", status, stderr.String())
	}
	if got, want := stdout.String(), helloContentSWHID+"\n"; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestRunDoesNotRetainOptions(t *testing.T) {
	var firstOutput bytes.Buffer
	var firstError bytes.Buffer
	status := run(
		[]string{contentCommand, "-q", exampleOriginFlag},
		strings.NewReader("hello world\n"),
		&firstOutput,
		&firstError,
	)
	if status != 0 {
		t.Fatalf("first run returned %d, stderr: %s", status, firstError.String())
	}

	var secondOutput bytes.Buffer
	var secondError bytes.Buffer
	status = run([]string{contentCommand}, strings.NewReader("hello world\n"), &secondOutput, &secondError)
	if status != 0 {
		t.Fatalf("second run returned %d, stderr: %s", status, secondError.String())
	}
	if strings.Contains(secondOutput.String(), "origin=") {
		t.Fatalf("second run retained the first run's qualifier: %s", secondOutput.String())
	}
}

func TestRunVersion(t *testing.T) {
	previousVersion := version
	version = "1.2.3"
	t.Cleanup(func() { version = previousVersion })

	for _, args := range [][]string{{"version"}, {"--version"}} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer

		status := run(args, strings.NewReader(""), &stdout, &stderr)

		if status != exitSuccess {
			t.Fatalf("run(%q) returned %d, stderr: %s", args, status, stderr.String())
		}
		if got, want := stdout.String(), "swhid 1.2.3\n"; got != want {
			t.Errorf("run(%q) output = %q, want %q", args, got, want)
		}
		if stderr.Len() != 0 {
			t.Errorf("run(%q) stderr = %q", args, stderr.String())
		}
	}
}

func TestRunVersionUsesModuleVersion(t *testing.T) {
	previousVersion := version
	previousReadBuildInfo := readBuildInfo
	version = "devel"
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Main: debug.Module{Version: "v0.1.0"}}, true
	}
	t.Cleanup(func() {
		version = previousVersion
		readBuildInfo = previousReadBuildInfo
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	status := run([]string{"--version"}, strings.NewReader(""), &stdout, &stderr)

	if status != exitSuccess {
		t.Fatalf("run() returned %d, stderr: %s", status, stderr.String())
	}
	if got, want := stdout.String(), "swhid v0.1.0\n"; got != want {
		t.Errorf("run() output = %q, want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Errorf("run() stderr = %q", stderr.String())
	}
}

func TestRunRejectsInvalidOptionsAndArguments(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "invalid format", args: []string{contentCommand, longFormatFlag, "yaml"}},
		{name: "duplicate qualifier", args: []string{contentCommand, "-q", exampleOriginFlag, "-q", "origin=https://other.example"}},
		{name: "qualifier on parse", args: []string{parseCommand, "-q", exampleOriginFlag, helloContentSWHID}},
		{name: "unknown command", args: []string{"unknown"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			status := run(test.args, strings.NewReader("hello world\n"), &stdout, &stderr)
			if status != 2 {
				t.Fatalf("run returned %d, want 2; stderr: %s", status, stderr.String())
			}
			if stderr.Len() == 0 {
				t.Fatal("expected an error message")
			}
		})
	}
}

func TestRunRejectsInvalidQualifierValue(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	status := run(
		[]string{contentCommand, "-q", "path=relative"},
		strings.NewReader("hello world\n"),
		&stdout,
		&stderr,
	)

	if status != exitCommandError {
		t.Fatalf("run returned %d, want %d; stderr: %s", status, exitCommandError, stderr.String())
	}
	if !strings.Contains(stderr.String(), "path must be absolute") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestRunParseRequiresOneSWHID(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	status := run(
		[]string{parseCommand, helloContentSWHID, helloContentSWHID},
		strings.NewReader(""),
		&stdout,
		&stderr,
	)

	if status != exitUsageError {
		t.Fatalf("run returned %d, want %d; stderr: %s", status, exitUsageError, stderr.String())
	}
	if !strings.Contains(stderr.String(), "exactly one SWHID") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestOptionsFirstPreservesOptionTerminator(t *testing.T) {
	got := optionsFirst([]string{"--", "-value"})
	if len(got) != 2 || got[0] != "--" || got[1] != "-value" {
		t.Fatalf("optionsFirst() = %q, want [-- -value]", got)
	}
}
