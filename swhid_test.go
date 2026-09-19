package swhid

import (
	"encoding/json"
	"testing"
)

const (
	testContentHash   = "94a9ed024d3859793618152ea559a168bbcbb5e2"
	testContentCore   = "swh:1:cnt:" + testContentHash
	testDirectoryHash = "d198bc9d7a6bcf6db04f476d29314f157507d505"
	testDirectoryCore = "swh:1:dir:" + testDirectoryHash
	testOriginKey     = qualifierOrigin
	testOriginURL     = "https://example.com"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantType  ObjectType
		wantHash  string
		wantQuals map[string]string
		wantErr   bool
	}{
		{
			name:     "valid content SWHID",
			input:    testContentCore,
			wantType: ObjectTypeContent,
			wantHash: testContentHash,
		},
		{
			name:     "valid directory SWHID",
			input:    testDirectoryCore,
			wantType: ObjectTypeDirectory,
			wantHash: testDirectoryHash,
		},
		{
			name:     "valid revision SWHID",
			input:    "swh:1:rev:309cf2674ee7a0749978cf8265ab91a60aea0f7d",
			wantType: ObjectTypeRevision,
			wantHash: "309cf2674ee7a0749978cf8265ab91a60aea0f7d",
		},
		{
			name:     "valid release SWHID",
			input:    "swh:1:rel:22ece559cc7cc2364edc5e5593d63ae8bd229f9f",
			wantType: ObjectTypeRelease,
			wantHash: "22ece559cc7cc2364edc5e5593d63ae8bd229f9f",
		},
		{
			name:     "valid snapshot SWHID",
			input:    "swh:1:snp:c7c108084bc0bf3d81436bf980b46e98bd338453",
			wantType: ObjectTypeSnapshot,
			wantHash: "c7c108084bc0bf3d81436bf980b46e98bd338453",
		},
		{
			name:     "SWHID with origin qualifier",
			input:    testContentCore + ";origin=https://github.com/example/repo",
			wantType: ObjectTypeContent,
			wantHash: testContentHash,
			wantQuals: map[string]string{
				testOriginKey: "https://github.com/example/repo",
			},
		},
		{
			name:     "SWHID with multiple qualifiers",
			input:    testContentCore + ";origin=" + testOriginURL + ";path=/src/main.go",
			wantType: ObjectTypeContent,
			wantHash: testContentHash,
			wantQuals: map[string]string{
				testOriginKey: testOriginURL,
				qualifierPath: "/src/main.go",
			},
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid scheme",
			input:   "swx:1:cnt:" + testContentHash,
			wantErr: true,
		},
		{
			name:    "invalid version",
			input:   "swh:2:cnt:" + testContentHash,
			wantErr: true,
		},
		{
			name:    "invalid object type",
			input:   "swh:1:foo:" + testContentHash,
			wantErr: true,
		},
		{
			name:    "invalid hash length",
			input:   "swh:1:cnt:94a9ed024d3859793618152ea559a168bbcbb5e",
			wantErr: true,
		},
		{
			name:    "invalid hash characters",
			input:   "swh:1:cnt:94a9ed024d3859793618152ea559a168bbcbb5ez",
			wantErr: true,
		},
		{
			name:    "missing parts",
			input:   "swh:1:cnt",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := Parse(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Parse() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Parse() unexpected error: %v", err)
				return
			}

			if id.ObjectType != tt.wantType {
				t.Errorf("ObjectType = %v, want %v", id.ObjectType, tt.wantType)
			}

			if id.ObjectHash != tt.wantHash {
				t.Errorf("ObjectHash = %v, want %v", id.ObjectHash, tt.wantHash)
			}

			if tt.wantQuals != nil {
				for k, v := range tt.wantQuals {
					if id.Qualifiers[k] != v {
						t.Errorf("Qualifier[%s] = %v, want %v", k, id.Qualifiers[k], v)
					}
				}
			}
		})
	}
}

func TestIdentifierString(t *testing.T) {
	tests := []struct {
		name       string
		objectType ObjectType
		objectHash string
		qualifiers map[string]string
		want       string
	}{
		{
			name:       "content without qualifiers",
			objectType: ObjectTypeContent,
			objectHash: testContentHash,
			want:       testContentCore,
		},
		{
			name:       "directory without qualifiers",
			objectType: ObjectTypeDirectory,
			objectHash: testDirectoryHash,
			want:       testDirectoryCore,
		},
		{
			name:       "content with origin qualifier",
			objectType: ObjectTypeContent,
			objectHash: testContentHash,
			qualifiers: map[string]string{
				testOriginKey: testOriginURL,
			},
			want: testContentCore + ";origin=" + testOriginURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := NewIdentifier(tt.objectType, tt.objectHash, tt.qualifiers)
			if err != nil {
				t.Fatalf("NewIdentifier() error: %v", err)
			}

			got := id.String()
			if got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIdentifierCoreSWHID(t *testing.T) {
	id, _ := NewIdentifier(ObjectTypeContent, testContentHash, map[string]string{
		testOriginKey: testOriginURL,
	})

	core := id.CoreSWHID()
	want := testContentCore

	if core != want {
		t.Errorf("CoreSWHID() = %v, want %v", core, want)
	}
}

func TestIdentifierEqual(t *testing.T) {
	id1, _ := NewIdentifier(ObjectTypeContent, testContentHash, nil)
	id2, _ := NewIdentifier(ObjectTypeContent, testContentHash, nil)
	id3, _ := NewIdentifier(ObjectTypeContent, "0000000000000000000000000000000000000000", nil)

	if !id1.Equal(id2) {
		t.Error("Equal() should return true for identical identifiers")
	}

	if id1.Equal(id3) {
		t.Error("Equal() should return false for different identifiers")
	}

	if id1.Equal(nil) {
		t.Error("Equal() should return false when compared to nil")
	}
}

func TestNewIdentifierValidation(t *testing.T) {
	tests := []struct {
		name       string
		objectType ObjectType
		objectHash string
		wantErr    bool
	}{
		{
			name:       "valid",
			objectType: ObjectTypeContent,
			objectHash: testContentHash,
			wantErr:    false,
		},
		{
			name:       "invalid object type",
			objectType: "foo",
			objectHash: testContentHash,
			wantErr:    true,
		},
		{
			name:       "invalid hash length",
			objectType: ObjectTypeContent,
			objectHash: "94a9ed024d",
			wantErr:    true,
		},
		{
			name:       "invalid hash characters",
			objectType: ObjectTypeContent,
			objectHash: "94a9ed024d3859793618152ea559a168bbcbb5ez",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewIdentifier(tt.objectType, tt.objectHash, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewIdentifier() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	tests := []string{
		testContentCore,
		testDirectoryCore,
		"swh:1:rev:309cf2674ee7a0749978cf8265ab91a60aea0f7d",
		"swh:1:rel:22ece559cc7cc2364edc5e5593d63ae8bd229f9f",
		"swh:1:snp:c7c108084bc0bf3d81436bf980b46e98bd338453",
	}

	for _, swhidStr := range tests {
		t.Run(swhidStr, func(t *testing.T) {
			id, err := Parse(swhidStr)
			if err != nil {
				t.Fatalf("Parse() error: %v", err)
			}

			got := id.String()
			if got != swhidStr {
				t.Errorf("Round trip failed: got %v, want %v", got, swhidStr)
			}
		})
	}
}

func TestQualifierRoundTripPreservesLiteralPlus(t *testing.T) {
	const input = testContentCore + ";origin=" + testOriginURL + "/a+b"

	id, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got := id.String(); got != input {
		t.Errorf("String() = %q, want %q", got, input)
	}
}

func TestQualifierRoundTripEscapesReservedValues(t *testing.T) {
	id, err := NewIdentifier(ObjectTypeContent, testContentHash, map[string]string{
		"origin":      "https://example.com/a b;100%",
		qualifierPath: "/a b;100%",
	})
	if err != nil {
		t.Fatalf("NewIdentifier() error = %v", err)
	}
	const want = testContentCore + ";origin=https://example.com/a%20b%3B100%25;path=/a%20b%3B100%25"
	if got := id.String(); got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
	parsed, err := Parse(want)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if !id.Equal(parsed) {
		t.Errorf("Parse() = %q, want %q", parsed.String(), id.String())
	}
}

func TestParseRejectsInvalidQualifiers(t *testing.T) {
	tests := []string{
		testContentCore + ";invalid",
		testContentCore + ";",
		testContentCore + ";origin=" + testOriginURL + ";",
		testContentCore + ";unknown=value",
		testContentCore + ";lines=0",
		testContentCore + ";lines=20-10",
		testContentCore + ";bytes=20-10",
		testContentCore + ";lines=1-2;bytes=0-1",
		testContentCore + ";visit=swh:1:rev:" + testContentHash,
		testContentCore + ";anchor=" + testContentCore + ";path=/file",
		testContentCore + ";path=relative",
		testContentCore + ";path=/contains space",
		testContentCore + ";unknown=line%0Abreak",
		"swh:1:rev:" + testContentHash + ";path=/file",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			if _, err := Parse(input); err == nil {
				t.Errorf("Parse(%q) expected error", input)
			}
		})
	}
}

func TestIdentifierCopiesAndSortsQualifiers(t *testing.T) {
	qualifiers := map[string]string{qualifierPath: "/file", qualifierOrigin: testOriginURL}
	id, err := NewIdentifier(ObjectTypeContent, testContentHash, qualifiers)
	if err != nil {
		t.Fatalf("NewIdentifier() error = %v", err)
	}
	qualifiers[qualifierOrigin] = "https://changed.example.com"

	const want = testContentCore + ";origin=" + testOriginURL + ";path=/file"
	if got := id.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestPathQualifierEscapesPathDelimiters(t *testing.T) {
	id, err := NewIdentifier(ObjectTypeContent, testContentHash, map[string]string{qualifierPath: "/search?q#result"})
	if err != nil {
		t.Fatalf("NewIdentifier() error = %v", err)
	}
	const want = testContentCore + ";path=/search%3Fq%23result"
	if got := id.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestIdentifierTextAndJSONRoundTrip(t *testing.T) {
	id, err := Parse(testContentCore + ";origin=" + testOriginURL)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	text, err := id.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText() error = %v", err)
	}
	if got, want := string(text), id.String(); got != want {
		t.Errorf("MarshalText() = %q, want %q", got, want)
	}

	encoded, err := json.Marshal(id)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var decoded Identifier
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if !id.Equal(&decoded) {
		t.Errorf("JSON round trip = %s, want %s", decoded.String(), id.String())
	}
}

func TestIdentifierUnmarshalTextPreservesValueOnError(t *testing.T) {
	id, err := Parse(testContentCore)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if err := id.UnmarshalText([]byte("invalid")); err == nil {
		t.Fatal("UnmarshalText() expected error")
	}
	if got := id.String(); got != testContentCore {
		t.Errorf("Identifier changed to %q", got)
	}
}
