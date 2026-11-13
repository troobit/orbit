package transcript

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// utcZone interprets wall times in UTC, independently of the test host.
var utcZone = resolveZone("", true)

func parseLMRecords(t *testing.T, input string) ([]LMStudioRequest, []Entry, []ParseWarning) {
	t.Helper()
	requests, entries, warnings, err := ParseLMStudioRecords(strings.NewReader(input), utcZone)
	require.NoError(t, err)
	return requests, entries, warnings
}

// lmEntrySummary is the part of an entry that classification decides.
type lmEntrySummary struct {
	Type         string
	RequestID    string
	Unattributed bool
	SourceLine   int
	Kinds        []string // content item type, with its status or event type
}

func summarizeLMEntries(entries []Entry) []lmEntrySummary {
	out := make([]lmEntrySummary, 0, len(entries))
	for _, e := range entries {
		s := lmEntrySummary{Type: e.Type, RequestID: e.RequestID, Unattributed: e.Unattributed, SourceLine: e.SourceLine}
		for _, item := range e.Message.Content {
			kind := item.Type
			if detail := item.Status + item.EventType; detail != "" {
				kind += "/" + detail
			}
			s.Kinds = append(s.Kinds, kind)
		}
		out = append(out, s)
	}
	return out
}

func TestParseLMHeader(t *testing.T) {
	tests := map[string]struct {
		line string
		want lmHeader
		ok   bool
	}{
		"level only": {
			line: "[2026-03-02 09:15:01][DEBUG] Received request: POST to /v1/responses",
			want: lmHeader{wall: "2026-03-02 09:15:01", level: "DEBUG", message: "Received request: POST to /v1/responses"},
			ok:   true,
		},
		"tags": {
			line: "[2026-03-02 09:15:01][INFO][LM STUDIO SERVER][slot-1] Ready",
			want: lmHeader{wall: "2026-03-02 09:15:01", level: "INFO", tags: []string{"LM STUDIO SERVER", "slot-1"}, message: "Ready"},
			ok:   true,
		},
		"bracketed message after a space": {
			line: "[2026-03-02 09:15:01][INFO] [not a tag] text",
			want: lmHeader{wall: "2026-03-02 09:15:01", level: "INFO", message: "[not a tag] text"},
			ok:   true,
		},
		"empty message": {
			line: "[2026-03-02 09:15:01][WARN]",
			want: lmHeader{wall: "2026-03-02 09:15:01", level: "WARN"},
			ok:   true,
		},
		"invalid calendar values still lex": {
			line: "[2026-13-45 25:61:61][INFO] text",
			want: lmHeader{wall: "2026-13-45 25:61:61", level: "INFO", message: "text"},
			ok:   true,
		},
		"missing level":        {line: "[2026-03-02 09:15:01] text"},
		"missing seconds":      {line: "[2026-03-02 09:15][INFO] text"},
		"ISO separator":        {line: "[2026-03-02T09:15:01][INFO] text"},
		"indented":             {line: " [2026-03-02 09:15:01][INFO] text"},
		"JSON body line":       {line: `  "role": "user",`},
		"array start":          {line: "["},
		"level with a space":   {line: "[2026-03-02 09:15:01][IN FO] text"},
		"unterminated bracket": {line: "[2026-03-02 09:15:01"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, ok := parseLMHeader([]byte(tc.line))
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

// classification.log exercises the five rows of the design's classification
// table, including the precedence of a reported path over level.
func TestParseLMStudioRecords_Classification(t *testing.T) {
	requests, entries, warnings := parseLMRecords(t, readLMFixture(t, "classification.log"))

	require.Len(t, requests, 1, "only the included endpoint is a request, whatever its level")
	assert.Equal(t, LMStudioRequest{
		ID: "12", StartLine: 12, EndLine: 20, Size: 208, Endpoint: "/v1/chat/completions", Model: "example-model-a",
		Timestamp: "2026-03-03 10:00:03", ResolvedTime: time.Date(2026, 3, 3, 10, 0, 3, 0, time.UTC),
	}, requests[0])

	assert.Equal(t, []lmEntrySummary{
		{Type: "user", RequestID: "12", Kinds: []string{"text"}},                          // row 1
		{Type: "assistant", Unattributed: true, SourceLine: 21, Kinds: []string{"text"}},  // row 2
		{Type: "log", SourceLine: 36, Kinds: []string{"event/unsupported_endpoint"}},      // row 3: /v1/decisions
		{Type: "log", SourceLine: 40, Kinds: []string{"event/unsupported_endpoint"}},      // row 3 at ERROR, not row 4
		{Type: "log", SourceLine: 41, Kinds: []string{"event/unsupported_endpoint"}},      // row 3 again
		{Type: "log", Unattributed: true, SourceLine: 46, Kinds: []string{"text/WARN"}},   // row 4: excluded path at WARN
		{Type: "log", Unattributed: true, SourceLine: 54, Kinds: []string{"text/WARN"}},   // row 4
		{Type: "log", Unattributed: true, SourceLine: 55, Kinds: []string{"error/ERROR"}}, // row 4
	}, summarizeLMEntries(entries), "routine records, excluded paths below WARN and GET requests are ignored (row 5)")

	assert.Equal(t, `{"path":"/v1/decisions"}`, string(entries[2].Message.Content[0].Payload))
	assert.Equal(t, "Model crashed while generating.\nContinuation of the error report.", entries[7].Message.Content[0].Text)
	assert.Equal(t, "example-model-a", entries[7].Model, "the record's tag, informational only")
	assert.Equal(t, "2026-03-03 10:00:12", entries[7].WallTime)
	assert.Equal(t, "2026-03-03T10:00:12Z", entries[7].Timestamp)

	assert.Equal(t, []ParseWarning{
		{Line: 36, Code: WarningUnsupportedEndpoint, Message: "unsupported POST path /v1/decisions (2 records)"},
		{Line: 40, Code: WarningUnsupportedEndpoint, Message: "unsupported POST path /v2/future/endpoint (1 record)"},
	}, warnings)
}

func TestParseLMStudioRecords_Recovery(t *testing.T) {
	requests, entries, warnings := parseLMRecords(t, readLMFixture(t, "recovery.log"))

	type got struct {
		start, end int
		endpoint   string
		state      string
		model      string
	}
	var states []got
	for _, r := range requests {
		states = append(states, got{r.StartLine, r.EndLine, r.Endpoint, r.BodyState, r.Model})
		assert.NotEmpty(t, r.Timestamp, "metadata is kept whatever the body state")
	}
	assert.Equal(t, []got{
		{3, 3, "/v1/chat/completions", BodyStateNotLogged, ""},
		{4, 9, "/v1/chat/completions", BodyStateMalformed, ""}, // unterminated, but another header follows
		{12, 20, "/v1/messages", BodyStateMalformed, ""},
		{21, 24, "/v1/completions", "", "example-model-b"},
		{25, 27, "/v1/completions", "", ""},
		{28, 30, "/v1/responses", BodyStateIncomplete, ""}, // unterminated at end of input
	}, states)

	assert.Equal(t, []lmEntrySummary{
		{Type: "user", RequestID: "21", Kinds: []string{"text"}},
		{Type: "user", RequestID: "25", Kinds: []string{"event/request_body"}},
	}, summarizeLMEntries(entries), "requests without a translated body have no entries")

	assert.Equal(t, []ParseWarning{
		{Line: 1, Code: WarningUnparsedText, Message: "text before the first log header (lines 1-2)"},
		{Line: 4, Code: WarningInvalidRecord, Message: "request body is not valid JSON (lines 4-9)"},
		{Line: 11, Code: WarningUnparsedText, Message: "malformed log header (line 11)"},
		{Line: 12, Code: WarningInvalidRecord, Message: "request body is not valid JSON (lines 12-20)"},
		{Line: 25, Code: WarningUnrecognizedRecord, Message: "/v1/completions request body is not a JSON object (lines 25-27)"},
		{Line: 28, Code: WarningInvalidRecord, Message: "request body is incomplete at the end of the input (lines 28-30)"},
	}, warnings, "an incomplete final body has a state and a body-free warning")
}

func TestParseLMStudioRecords_Recovery_NoBodyText(t *testing.T) {
	for _, name := range []string{"recovery.log", "classification.log", "tools.log"} {
		t.Run(name, func(t *testing.T) {
			_, _, warnings := parseLMRecords(t, readLMFixture(t, name))
			for _, w := range warnings {
				for _, body := range []string{"stops before", "still being written", "Recovered", "not an object", "copied before", "Proceed", "AAAA"} {
					assert.NotContains(t, w.Message, body, "warning %q leaks body text", w.Message)
				}
			}
		})
	}
}

func TestParseLMStudioRecords_Inputs(t *testing.T) {
	const header = "[2026-03-02 09:00:00][DEBUG] Received request: POST to /v1/completions"
	const valid = header + ` with body {"prompt": "ok"}` + "\n"
	tests := map[string]struct {
		input    string
		states   []string
		kinds    []lmEntrySummary
		warnings []ParseWarning
	}{
		"empty input": {input: ""},
		"blank lines only": {
			input: "\n  \n\r\n",
		},
		"no recognised records": {
			input:    "plain text\nmore text\n",
			warnings: []ParseWarning{{Line: 1, Code: WarningUnparsedText, Message: "no LM Studio log records found (lines 1-2)"}},
		},
		"leading BOM and CRLF": {
			input:  bom + strings.ReplaceAll(valid, "\n", "\r\n"),
			states: []string{""},
			kinds:  []lmEntrySummary{{Type: "user", RequestID: "1", Kinds: []string{"text"}}},
		},
		"no newline at end of input": {
			input:  strings.TrimSuffix(valid, "\n"),
			states: []string{""},
			kinds:  []lmEntrySummary{{Type: "user", RequestID: "1", Kinds: []string{"text"}}},
		},
		"unterminated body at end of input": {
			input:    header + ` with body {"prompt": "o`,
			states:   []string{BodyStateIncomplete},
			warnings: []ParseWarning{{Line: 1, Code: WarningInvalidRecord, Message: "request body is incomplete at the end of the input (line 1)"}},
		},
		"empty body at end of input": {
			input:    header + " with body\n",
			states:   []string{BodyStateIncomplete},
			warnings: []ParseWarning{{Line: 1, Code: WarningInvalidRecord, Message: "request body is incomplete at the end of the input (line 1)"}},
		},
		"empty body before another record": {
			input:    header + " with body\n" + valid,
			states:   []string{BodyStateMalformed, ""},
			kinds:    []lmEntrySummary{{Type: "user", RequestID: "2", Kinds: []string{"text"}}},
			warnings: []ParseWarning{{Line: 1, Code: WarningInvalidRecord, Message: "request body is not valid JSON (line 1)"}},
		},
		"text after a complete body": {
			input:    header + " with body {\n  \"prompt\": \"ok\"\n}\nstray\nlines\n",
			states:   []string{""},
			kinds:    []lmEntrySummary{{Type: "user", RequestID: "1", Kinds: []string{"text"}}},
			warnings: []ParseWarning{{Line: 4, Code: WarningUnparsedText, Message: "text after the request body (lines 4-5)"}},
		},
		"text after a request without a body": {
			input:    header + "\nstray\n",
			states:   []string{BodyStateNotLogged},
			warnings: []ParseWarning{{Line: 2, Code: WarningUnparsedText, Message: "text after a request without a body (line 2)"}},
		},
		"included path with an unrecognised suffix at WARN": {
			input:  "[2026-03-02 09:00:00][WARN] Received request: POST to /v1/completions from client\n",
			states: nil,
			kinds:  []lmEntrySummary{{Type: "log", Unattributed: true, SourceLine: 1, Kinds: []string{"text/WARN"}}},
			warnings: []ParseWarning{{Line: 1, Code: WarningUnrecognizedRecord,
				Message: "record names /v1/completions but is not a recognised request record (line 1)"}},
		},
		"included path with an unrecognised suffix at INFO": {
			input: "[2026-03-02 09:00:00][INFO] Received request: POST to /v1/completions from client\n",
			warnings: []ParseWarning{{Line: 1, Code: WarningUnrecognizedRecord,
				Message: "record names /v1/completions but is not a recognised request record (line 1)"}},
		},
		"response claim without a recognised marker": {
			input: "[2026-03-02 09:00:00][INFO] Generated prediction for slot 1\n",
			warnings: []ParseWarning{{Line: 1, Code: WarningUnrecognizedRecord,
				Message: "record looks like a logged response but has no recognised marker (line 1)"}},
		},
		"POST path in an error message": {
			input:    "[2026-03-02 09:00:00][ERROR] Unexpected endpoint or method. (POST /v1/unknown). Returning 200 anyway\n",
			kinds:    []lmEntrySummary{{Type: "log", SourceLine: 1, Kinds: []string{"event/unsupported_endpoint"}}},
			warnings: []ParseWarning{{Line: 1, Code: WarningUnsupportedEndpoint, Message: "unsupported POST path /v1/unknown (1 record)"}},
		},
		"response before the first request": {
			input:  "[2026-03-02 09:00:00][INFO] Generated prediction: {\"choices\": [{\"text\": \"late\"}]}\n" + valid,
			states: []string{""},
			kinds: []lmEntrySummary{
				{Type: "assistant", Unattributed: true, SourceLine: 1, Kinds: []string{"text"}},
				{Type: "user", RequestID: "2", Kinds: []string{"text"}},
			},
		},
		"malformed response body": {
			input:    "[2026-03-02 09:00:00][INFO] Generated prediction: {not json}\n" + valid,
			states:   []string{""},
			kinds:    []lmEntrySummary{{Type: "assistant", Unattributed: true, SourceLine: 1, Kinds: []string{"event/response"}}, {Type: "user", RequestID: "2", Kinds: []string{"text"}}},
			warnings: []ParseWarning{{Line: 1, Code: WarningInvalidRecord, Message: "response body is not valid JSON (line 1)"}},
		},
		"header-like line in a warning": {
			input:    "[2026-03-02 09:00:00][WARN] Something\n[2026-03-02 09:00] broken\n",
			kinds:    []lmEntrySummary{{Type: "log", Unattributed: true, SourceLine: 1, Kinds: []string{"text/WARN"}}},
			warnings: []ParseWarning{{Line: 2, Code: WarningUnparsedText, Message: "malformed log header (line 2)"}},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			requests, entries, warnings := parseLMRecords(t, tc.input)
			var states []string
			for _, r := range requests {
				states = append(states, r.BodyState)
			}
			assert.Equal(t, tc.states, states)
			assert.Equal(t, tc.kinds, nilIfEmpty(summarizeLMEntries(entries)))
			assert.Equal(t, tc.warnings, nilIfEmpty(warnings))
		})
	}
}

func nilIfEmpty[T any](s []T) []T {
	if len(s) == 0 {
		return nil
	}
	return s
}

func TestParseLMStudioRecords_Oversized(t *testing.T) {
	big := strings.Repeat("x", lmMaxRecordBytes)
	next := "[2026-03-02 09:00:05][DEBUG] Received request: POST to /v1/completions with body {\"prompt\": \"after\"}\n"
	tests := map[string]struct {
		input    string
		size     int
		lines    string
		endLine  int
		hasEntry bool // an unattributed entry keeps the record's metadata
	}{
		"request body over many lines": {
			input: "[2026-03-02 09:00:00][DEBUG] Received request: POST to /v1/completions with body {\n" +
				`  "prompt": "` + big + "\"\n}\n" + next,
			lines:   "lines 1-3",
			endLine: 3,
		},
		"request body on its header line": {
			input:   `[2026-03-02 09:00:00][DEBUG] Received request: POST to /v1/completions with body {"prompt": "` + big + "\"}\n" + next,
			lines:   "line 1",
			endLine: 1,
		},
		"logged response": {
			input:    "[2026-03-02 09:00:00][INFO] Generated prediction: {\n" + `  "x": "` + big + "\"\n}\n" + next,
			lines:    "lines 1-3",
			endLine:  3,
			hasEntry: true,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			requests, entries, warnings := parseLMRecords(t, tc.input)
			size := strings.Index(tc.input, next)

			require.Len(t, warnings, 1)
			assert.Equal(t, WarningOversizedRecord, warnings[0].Code)
			assert.Equal(t, fmt.Sprintf("record of %d bytes exceeds the %d-byte parser limit; body skipped (%s)",
				size, lmMaxRecordBytes, tc.lines), warnings[0].Message, "the diagnostic names the size and the limit")

			last := requests[len(requests)-1]
			assert.Equal(t, "", last.BodyState, "the next record is still read")
			assert.Equal(t, tc.endLine+1, last.StartLine)
			if tc.hasEntry {
				require.Len(t, requests, 1)
				assert.Equal(t, 1, entries[0].SourceLine)
				assert.True(t, entries[0].Unattributed)
				return
			}
			require.Len(t, requests, 2)
			assert.Equal(t, BodyStateOversized, requests[0].BodyState)
			assert.Equal(t, "/v1/completions", requests[0].Endpoint, "header metadata survives")
			assert.Equal(t, tc.endLine, requests[0].EndLine)
		})
	}
}

// Wall times keep their logged text; only a unique interpretation becomes an
// instant, and source order never follows the clock (requirements 5.1-5.3).
func TestParseLMStudioRecords_WallTimes(t *testing.T) {
	newYork := resolveZone("America/New_York", true)
	require.Empty(t, newYork.warning)
	input := "" +
		"[2026-11-01 01:30:00][DEBUG] Received request: POST to /v1/completions with body {\"prompt\": \"repeated hour\"}\n" +
		"[2026-03-08 02:30:00][DEBUG] Received request: POST to /v1/completions with body {\"prompt\": \"skipped hour\"}\n" +
		"[2026-01-15 09:00:00][DEBUG] Received request: POST to /v1/completions with body {\"prompt\": \"clock moved back\"}\n" +
		"[2026-02-30 09:00:00][WARN] invalid date\n"

	t.Run("America/New_York", func(t *testing.T) {
		requests, entries, _, err := ParseLMStudioRecords(strings.NewReader(input), newYork)
		require.NoError(t, err)
		require.Len(t, requests, 3)
		assert.Equal(t, []int{1, 2, 3}, []int{requests[0].StartLine, requests[1].StartLine, requests[2].StartLine})
		assert.Equal(t, "2026-11-01 01:30:00", requests[0].Timestamp)
		assert.True(t, requests[0].ResolvedTime.IsZero(), "ambiguous")
		assert.True(t, requests[1].ResolvedTime.IsZero(), "nonexistent")
		assert.Equal(t, "2026-01-15T09:00:00-05:00", requests[2].ResolvedTime.Format(time.RFC3339))
		assert.Equal(t, "", entries[0].Timestamp)
		assert.Equal(t, "2026-01-15T09:00:00-05:00", entries[2].Timestamp)
		warn := entries[len(entries)-1]
		assert.Equal(t, "2026-02-30 09:00:00", warn.WallTime)
		assert.Equal(t, "", warn.Timestamp, "invalid")
	})

	t.Run("unloadable zone", func(t *testing.T) {
		requests, entries, _, err := ParseLMStudioRecords(strings.NewReader(input), resolveZone("Not/AZone", true))
		require.NoError(t, err)
		for _, r := range requests {
			assert.True(t, r.ResolvedTime.IsZero())
			assert.NotEmpty(t, r.Timestamp)
		}
		for _, e := range entries {
			assert.Empty(t, e.Timestamp)
		}
	})
}

func TestParseLMStudioRecords_ReadError(t *testing.T) {
	_, _, _, err := ParseLMStudioRecords(failingReader{strings.NewReader("[2026-03-02 09:00:00][INFO] x\n")}, utcZone)
	require.Error(t, err)
}

// Listing reads files through SummarizeLMStudio, so it must report what the
// full parse reports for every fixture.
func TestSummarizeLMStudioMatchesFullParse(t *testing.T) {
	t.Parallel()
	paths, err := filepath.Glob("testdata/lmstudio/*.log")
	require.NoError(t, err)
	require.NotEmpty(t, paths)
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			t.Parallel()
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			full, err := ParseLMStudio(bytes.NewReader(data))
			require.NoError(t, err)
			got, err := SummarizeLMStudio(bytes.NewReader(data))
			require.NoError(t, err)

			var want []LMStudioRequestSummary
			for _, req := range full.LMStudioRequests {
				want = append(want, LMStudioRequestSummary{
					StartLine: req.StartLine, Size: req.Size, ResolvedTime: req.ResolvedTime, Model: req.Model, ThreadID: req.ThreadID,
				})
			}
			assert.Equal(t, want, got.Requests)
			assert.Equal(t, slices.ContainsFunc(full.Entries, func(e Entry) bool { return e.Unattributed }), got.HasLogEntries)
			assert.Equal(t, full.Metadata.TimeZone, got.TimeZone)
		})
	}
}
