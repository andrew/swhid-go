package swhid

import (
	"testing"
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
			input:    "swh:1:cnt:94a9ed024d3859793618152ea559a168bbcbb5e2",
			wantType: ObjectTypeContent,
			wantHash: "94a9ed024d3859793618152ea559a168bbcbb5e2",
		},
		{
			name:     "valid directory SWHID",
			input:    "swh:1:dir:d198bc9d7a6bcf6db04f476d29314f157507d505",
			wantType: ObjectTypeDirectory,
			wantHash: "d198bc9d7a6bcf6db04f476d29314f157507d505",
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
			input:    "swh:1:cnt:94a9ed024d3859793618152ea559a168bbcbb5e2;origin=https://github.com/example/repo",
			wantType: ObjectTypeContent,
			wantHash: "94a9ed024d3859793618152ea559a168bbcbb5e2",
			wantQuals: map[string]string{
				"origin": "https://github.com/example/repo",
			},
		},
		{
			name:     "SWHID with multiple qualifiers",
			input:    "swh:1:cnt:94a9ed024d3859793618152ea559a168bbcbb5e2;origin=https://example.com;path=/src/main.go",
			wantType: ObjectTypeContent,
			wantHash: "94a9ed024d3859793618152ea559a168bbcbb5e2",
			wantQuals: map[string]string{
				"origin": "https://example.com",
				"path":   "/src/main.go",
			},
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid scheme",
			input:   "swx:1:cnt:94a9ed024d3859793618152ea559a168bbcbb5e2",
			wantErr: true,
		},
		{
			name:    "invalid version",
			input:   "swh:2:cnt:94a9ed024d3859793618152ea559a168bbcbb5e2",
			wantErr: true,
		},
		{
			name:    "invalid object type",
			input:   "swh:1:foo:94a9ed024d3859793618152ea559a168bbcbb5e2",
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
			objectHash: "94a9ed024d3859793618152ea559a168bbcbb5e2",
			want:       "swh:1:cnt:94a9ed024d3859793618152ea559a168bbcbb5e2",
		},
		{
			name:       "directory without qualifiers",
			objectType: ObjectTypeDirectory,
			objectHash: "d198bc9d7a6bcf6db04f476d29314f157507d505",
			want:       "swh:1:dir:d198bc9d7a6bcf6db04f476d29314f157507d505",
		},
		{
			name:       "content with origin qualifier",
			objectType: ObjectTypeContent,
			objectHash: "94a9ed024d3859793618152ea559a168bbcbb5e2",
			qualifiers: map[string]string{
				"origin": "https://example.com",
			},
			want: "swh:1:cnt:94a9ed024d3859793618152ea559a168bbcbb5e2;origin=https://example.com",
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
	id, _ := NewIdentifier(ObjectTypeContent, "94a9ed024d3859793618152ea559a168bbcbb5e2", map[string]string{
		"origin": "https://example.com",
	})

	core := id.CoreSWHID()
	want := "swh:1:cnt:94a9ed024d3859793618152ea559a168bbcbb5e2"

	if core != want {
		t.Errorf("CoreSWHID() = %v, want %v", core, want)
	}
}

func TestIdentifierEqual(t *testing.T) {
	id1, _ := NewIdentifier(ObjectTypeContent, "94a9ed024d3859793618152ea559a168bbcbb5e2", nil)
	id2, _ := NewIdentifier(ObjectTypeContent, "94a9ed024d3859793618152ea559a168bbcbb5e2", nil)
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
			objectHash: "94a9ed024d3859793618152ea559a168bbcbb5e2",
			wantErr:    false,
		},
		{
			name:       "invalid object type",
			objectType: "foo",
			objectHash: "94a9ed024d3859793618152ea559a168bbcbb5e2",
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
		"swh:1:cnt:94a9ed024d3859793618152ea559a168bbcbb5e2",
		"swh:1:dir:d198bc9d7a6bcf6db04f476d29314f157507d505",
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
