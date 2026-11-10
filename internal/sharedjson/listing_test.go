package sharedjson

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/arjenschwarz/orbit/internal/sessions"
)

func TestListing_EmptyProjectScope(t *testing.T) {
	doc := Listing(nil, nil, Scope{Project: "/work/example-repo"})
	got := encode(t, doc)

	want, err := os.ReadFile("testdata/handwritten/listing-project-empty.json")
	require.NoError(t, err)
	assert.JSONEq(t, string(want), string(got))
	requireValid(t, loadSchemas(t), decodeJSON(t, string(got)))
}

func TestListing_RowsAndWarnings(t *testing.T) {
	list := []sessions.SessionInfo{
		{Source: sessions.SourceKiroIDE, ID: "no-time", Size: 1024},
		{Source: sessions.SourceCopilot, ID: "copilot-session", Size: 512, Project: "/work/copilot-repo",
			CreatedAt: time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC), TimeFromFilesystem: true},
		{Source: sessions.SourceCodex, ID: "0199bbbb-0000-7000-8000-000000000002", Size: 40960, Project: "/work/other",
			CreatedAt: time.Date(2026, 10, 2, 20, 0, 0, 0, time.FixedZone("example", 10*60*60)),
			Model:     "example-model", Provider: "example-provider", Origin: "codex_exec"},
		{Source: sessions.SourceClaude, ID: "claude-session", Size: 2048, Project: "/work/example-repo", Origin: "sdk-cli",
			CreatedAt: time.Date(2026, 10, 3, 8, 0, 0, 123_000_000, time.UTC)},
	}
	warnings := []sessions.ListWarning{
		{Source: sessions.SourceClaude, Code: "duplicate_session", Err: errors.New("session claude-session listed once; another copy exists")},
		{Source: sessions.SourceKiroCLI, Code: sessions.WarningSourceFailed, Err: errors.New("discover kiro sessions: database is locked")},
	}

	got := encode(t, Listing(list, warnings, Scope{AllProjects: true}))

	assert.JSONEq(t, `{
	  "schema_version": 1,
	  "kind": "listing",
	  "scope": {"all_projects": true},
	  "sessions": [
	    {"source": "kiro-ide", "id": "no-time", "size_bytes": 1024},
	    {"source": "copilot", "id": "copilot-session", "size_bytes": 512, "project": "/work/copilot-repo",
	     "file_modified_at": "2026-10-01T09:00:00Z"},
	    {"source": "codex", "id": "0199bbbb-0000-7000-8000-000000000002", "size_bytes": 40960, "project": "/work/other",
	     "created_at": "2026-10-02T10:00:00Z", "model": "example-model", "provider": "example-provider", "origin": "codex_exec"},
	    {"source": "claude", "id": "claude-session", "size_bytes": 2048, "project": "/work/example-repo", "origin": "sdk-cli",
	     "created_at": "2026-10-03T08:00:00.123Z"}
	  ],
	  "warnings": [
	    {"source": "claude", "code": "duplicate_session", "message": "session claude-session listed once; another copy exists"},
	    {"source": "kiro-cli", "code": "source_failed", "message": "discover kiro sessions: database is locked"}
	  ]
	}`, string(got))

	schemas := loadSchemas(t)
	requireValid(t, schemas, decodeJSON(t, string(got)))
	codes := publishedWarningCodes(t)
	for _, w := range warnings {
		assert.Contains(t, codes, w.Code, "every listing warning carries a published code")
	}
}

func TestListing_KeepsRowOrder(t *testing.T) {
	newer := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)
	older := newer.Add(-time.Hour)
	doc := Listing([]sessions.SessionInfo{
		{Source: sessions.SourceClaude, ID: "first", CreatedAt: newer},
		{Source: sessions.SourceClaude, ID: "second", CreatedAt: older},
	}, nil, Scope{Project: "/work/example-repo"})

	require.Len(t, doc.Sessions, 2)
	assert.Equal(t, "first", doc.Sessions[0].ID, "the projection keeps the lister's order")
	assert.Equal(t, "second", doc.Sessions[1].ID)
}

// LM Studio rows report their request count instead of a byte size; other
// rows keep size_bytes even when it is zero (requirement 7.2, Q19).
func TestListing_LMStudioRows(t *testing.T) {
	list := []sessions.SessionInfo{
		{Source: sessions.SourceLMStudio, ID: "lmstudio-req-p-MjAyNi0wMy8yMDI2LTAzLTA1LjEubG9n-44", Size: 900,
			Model: "example-model-b", Provider: "lmstudio", RequestCount: 1},
		{Source: sessions.SourceClaude, ID: "0b5e1a40-0000-4000-8000-000000000010", Project: "/work/example-repo",
			CreatedAt: time.Date(2026, 3, 5, 2, 0, 0, 0, time.UTC)},
		{Source: sessions.SourceLMStudio, ID: "lmstudio-thread-a2ba3f64e66545724423033a46a31b5fa9f6db429bfffb60",
			ThreadID: "thread-alpha", CreatedAt: time.Date(2026, 3, 5, 14, 0, 0, 0, time.FixedZone("AEDT", 11*60*60)),
			Size: 4096, Model: "example-model-a", Provider: "lmstudio", RequestCount: 3},
	}
	warnings := []sessions.ListWarning{{Source: sessions.SourceLMStudio, Code: sessions.WarningPathUnreadable,
		Err: errors.New("read 2026-03/2026-03-04.1.log: permission denied")}}

	got := encode(t, Listing(list, warnings, Scope{AllProjects: true}))

	want, err := os.ReadFile("testdata/handwritten/listing-lmstudio.json")
	require.NoError(t, err)
	assert.JSONEq(t, string(want), string(got))
	requireValid(t, loadSchemas(t), decodeJSON(t, string(got)))
}
