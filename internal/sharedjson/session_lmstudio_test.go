package sharedjson

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/arjenschwarz/orbit/internal/transcript"
)

// lmZone is the zone LM Studio fixtures are interpreted in, so assumed times
// carry a non-zero offset.
const lmZone = "Australia/Melbourne"

func parseLMStudio(t *testing.T, input string) *transcript.ParseResult {
	t.Helper()
	t.Setenv("TZ", lmZone)
	res, err := transcript.ParseLMStudio(strings.NewReader(input))
	require.NoError(t, err)
	return res
}

func parseLMStudioFile(t *testing.T, name string) *transcript.ParseResult {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "transcript", "testdata", "lmstudio", name))
	require.NoError(t, err)
	return parseLMStudio(t, string(data))
}

func TestSession_LMStudioGolden(t *testing.T) {
	got := encode(t, Session(parseLMStudioFile(t, "threads.log"), SessionContext{}))

	goldenPath := filepath.Join("testdata", "golden", "lmstudio.json")
	if *update {
		require.NoError(t, os.WriteFile(goldenPath, got, 0o600))
	}
	want, err := os.ReadFile(goldenPath)
	require.NoError(t, err)
	assert.Equal(t, string(want), string(got))
	requireValid(t, loadSchemas(t), decodeJSON(t, string(got)))
}

// Every fixture's document validates, and its entries join requests only
// through request IDs the document declares (requirement 8.2).
func TestSession_LMStudioFixturesValidate(t *testing.T) {
	schemas := loadSchemas(t)
	codes := publishedWarningCodes(t)
	paths, err := filepath.Glob(filepath.Join("..", "transcript", "testdata", "lmstudio", "*.log"))
	require.NoError(t, err)
	require.NotEmpty(t, paths)
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			doc := Session(parseLMStudioFile(t, filepath.Base(path)), SessionContext{})
			requireValid(t, schemas, decodeJSON(t, string(encode(t, doc))))
			for _, w := range doc.Warnings {
				assert.Contains(t, codes, w.Code, "every projected warning carries a published code")
			}

			ids := map[string]bool{}
			for _, req := range doc.Requests {
				assert.False(t, ids[req.ID], "request %s listed once", req.ID)
				ids[req.ID] = true
				assert.Equal(t, "unknown", req.ResponseAttribution)
			}
			for _, e := range doc.Entries {
				if e.RequestID != "" {
					assert.True(t, ids[e.RequestID], "entry names declared request %s", e.RequestID)
					assert.False(t, e.Unattributed)
					continue
				}
				assert.Positive(t, e.SourceLine, "a log-level entry keeps its source position")
				assert.NotEmpty(t, e.WallTime, "a log-level entry keeps its logged wall time")
			}
		})
	}
}

func TestSession_LMStudioFields(t *testing.T) {
	out := string(encode(t, Session(parseLMStudioFile(t, "threads.log"), SessionContext{ID: "threads"})))
	doc := decodeJSON(t, out).(map[string]any)

	assert.Equal(t, map[string]any{
		"source": "lmstudio", "id": "threads", "created_at": "2026-03-05T14:00:00+11:00",
		"time_zone": lmZone, "model": "example-model-a", "provider": "lmstudio",
	}, doc["session"], "a direct view has no session-level thread: it can hold several")

	requests := doc["requests"].([]any)
	require.Len(t, requests, 8)
	first := requests[0].(map[string]any)
	assert.Equal(t, "1", first["id"], "the parser's temporary key until the sessions layer assigns IDs")
	assert.Equal(t, json.Number("1"), first["start_line"])
	assert.Equal(t, json.Number("42"), first["end_line"])
	assert.Equal(t, "/v1/responses", first["endpoint"])
	assert.Equal(t, "thread-alpha", first["thread_id"])
	assert.Equal(t, "turn", first["request_kind"])
	assert.Equal(t, "2026-03-05 14:00:00", first["wall_time"])
	assert.Equal(t, "2026-03-05T14:00:00+11:00", first["timestamp"], "the assumed instant keeps its offset")
	assert.Equal(t, "unknown", first["response_attribution"])
	assert.NotContains(t, first, "session_id", "assigned by the sessions layer, not the parser")
	params := first["parameters"].(map[string]any)
	assert.Equal(t, "auto", params["tool_choice"])
	assert.Equal(t, "thread-alpha", params["client_metadata"].(map[string]any)["thread_id"], "client metadata stays a parameter")

	assert.Equal(t, json.Number("2"), requests[2].(map[string]any)["prior_prefix_count"])
	assert.Equal(t, true, requests[3].(map[string]any)["context_discontinuity"])
	assert.Equal(t, "compaction", requests[3].(map[string]any)["request_kind"])
	assert.Equal(t, "thread-alpha", requests[5].(map[string]any)["parent_thread_id"])
	assert.NotContains(t, requests[6].(map[string]any), "thread_id", "a truncated thread ID is no identity")
	assert.Contains(t, out, `"thread_id": "thread-gam ... <Truncated in logs> ... mma-long"`,
		"parameters keep the truncated value as logged, without HTML escaping")

	entries := doc["entries"].([]any)
	assert.Equal(t, map[string]any{
		"role": "log", "timestamp": "2026-03-05T14:00:01+11:00", "model": "example-model-a", "provider": "lmstudio",
		"content": []any{map[string]any{"type": "text", "status": "WARN",
			"text": "Field 'tool_choice' is not supported and will be ignored."}},
		"unattributed": true, "source_line": json.Number("43"), "wall_time": "2026-03-05 14:00:01",
	}, entries[2])
	var supplied map[string]any
	for _, e := range entries {
		if m := e.(map[string]any); m["supplied_context"] == true {
			supplied = m
			break
		}
	}
	require.NotNil(t, supplied, "a resent assistant message is labelled supplied context")
	assert.Equal(t, "49", supplied["request_id"])
	assert.Equal(t, "assistant", supplied["role"])
}

func TestSession_LMStudioTruncatedContent(t *testing.T) {
	doc := decodeJSON(t, string(encode(t, Session(parseLMStudioFile(t, "tools.log"), SessionContext{})))).(map[string]any)

	var truncated []map[string]any
	for _, e := range doc["entries"].([]any) {
		for _, b := range e.(map[string]any)["content"].([]any) {
			if block := b.(map[string]any); block["truncated"] == true {
				truncated = append(truncated, block)
			}
		}
	}
	require.Len(t, truncated, 1)
	assert.Equal(t, "text", truncated[0]["type"])
	assert.Contains(t, truncated[0]["text"], "<Truncated in logs>", "the available text is kept as logged")
}

// Requests whose body was not translated keep their boundary and state, and
// no entry claims them (requirements 2.5, 6.1).
func TestSession_LMStudioBodyStates(t *testing.T) {
	doc := Session(parseLMStudioFile(t, "recovery.log"), SessionContext{})

	states := map[int]string{}
	for _, req := range doc.Requests {
		states[req.StartLine] = req.BodyState
	}
	assert.Equal(t, map[int]string{3: "not_logged", 4: "malformed", 12: "malformed", 21: "", 25: "", 28: "incomplete"}, states)
	for _, e := range doc.Entries {
		assert.Contains(t, []string{"21", "25"}, e.RequestID, "only translated bodies produce entries")
	}
}

// A session's creation time is its first request's own time, or absent; a
// later request's time never stands in (requirement 5.4).
func TestSession_LMStudioCreatedAtFromFirstRequestOnly(t *testing.T) {
	// 02:30 does not exist in Melbourne on 4 October 2026: clocks skip from 02:00 to 03:00.
	res := parseLMStudio(t, `[2026-10-04 02:30:00][DEBUG] Received request: POST to /v1/completions with body {
  "model": "example-model-a",
  "prompt": "First."
}
[2026-10-04 04:00:00][DEBUG] Received request: POST to /v1/completions with body {
  "model": "example-model-a",
  "prompt": "Second."
}
`)
	doc := Session(res, SessionContext{})

	assert.Empty(t, doc.Session.CreatedAt)
	require.Len(t, doc.Requests, 2)
	assert.Equal(t, "2026-10-04 02:30:00", doc.Requests[0].WallTime)
	assert.Empty(t, doc.Requests[0].Timestamp, "a nonexistent wall time has no instant")
	assert.Equal(t, "2026-10-04T04:00:00+11:00", doc.Requests[1].Timestamp)
	require.Len(t, doc.Entries, 2)
	assert.Equal(t, "2026-10-04T04:00:00+11:00", doc.Entries[1].Timestamp)
}

func TestSession_LMStudioNoRequestsOmitsArray(t *testing.T) {
	out := string(encode(t, Session(parseLMStudioFile(t, "notice_only.log"), SessionContext{})))
	doc := decodeJSON(t, out).(map[string]any)
	assert.NotContains(t, doc, "requests")
	assert.NotContains(t, doc, "unattributed_summary", "a direct view keeps events as entries")
	assert.Equal(t, []any{}, doc["entries"])
}

// lmSessionView is the parse result session assembly will produce for the
// handwritten session-view document: a thread's requests from two files,
// with each file's unattributed events reduced to a summary.
func lmSessionView(t *testing.T) *transcript.ParseResult {
	t.Helper()
	zone, err := time.LoadLocation(lmZone)
	require.NoError(t, err)
	const (
		session = "lmstudio-thread-a2ba3f64e66545724423033a46a31b5fa9f6db429bfffb60"
		first   = "lmstudio-req-p-MjAyNi0wMy8yMDI2LTAzLTA1LjEubG9n-1"
		second  = "lmstudio-req-p-MjAyNi0wMy8yMDI2LTAzLTA1LjIubG9n-3"
		third   = "lmstudio-req-p-MjAyNi0wMy8yMDI2LTAzLTA1LjIubG9n-30"
	)
	params := json.RawMessage(`{"stream":true,"client_metadata":{"thread_id":"thread-alpha"}}`)
	request := func(id string, start, end int, wall string) transcript.LMStudioRequest {
		resolved, err := time.ParseInLocation(time.DateTime, wall, zone)
		require.NoError(t, err)
		return transcript.LMStudioRequest{
			ID: id, SessionID: session, StartLine: start, EndLine: end, Endpoint: "/v1/responses",
			ThreadID: "thread-alpha", RequestKind: "turn", Model: "example-model-a",
			Timestamp: wall, ResolvedTime: resolved, Parameters: params,
		}
	}
	text := func(entryType, requestID, timestamp, text string) transcript.Entry {
		return transcript.Entry{
			Type:      entryType,
			Message:   &transcript.Message{Role: entryType, Content: []transcript.ContentItem{{Type: "text", Text: text}}},
			Timestamp: timestamp, Model: "example-model-a", Provider: "lmstudio", RequestID: requestID,
		}
	}

	resent := request(second, 3, 20, "2026-03-05 14:05:00")
	resent.PriorPrefixCount = 1
	incomplete := request(third, 30, 31, "2026-03-05 14:06:00")
	incomplete.RequestKind, incomplete.Parameters, incomplete.BodyState = "", nil, transcript.BodyStateIncomplete
	assistant := text("assistant", second, "2026-03-05T14:05:00+11:00", "The test expects a sorted list ... <Truncated in logs> ... order.")
	assistant.SuppliedContext = true
	assistant.Message.Content[0].Truncated = true

	return &transcript.ParseResult{
		Format: transcript.FormatLMStudio,
		Metadata: &transcript.ParseResultMetadata{
			StartedAt: "2026-03-05T14:00:00+11:00", FirstModel: "example-model-a", FirstProvider: "lmstudio",
			ThreadID: "thread-alpha", TimeZone: lmZone,
		},
		LMStudioRequests: []transcript.LMStudioRequest{request(first, 1, 12, "2026-03-05 14:00:00"), resent, incomplete},
		Entries: []transcript.Entry{
			text("user", first, "2026-03-05T14:00:00+11:00", "Fix the failing test."),
			text("user", second, "2026-03-05T14:05:00+11:00", "Fix the failing test."),
			assistant,
		},
		LMStudioEvents: []transcript.LMStudioEventSummary{
			{File: "2026-03/2026-03-05.1.log", NextFile: "2026-03/2026-03-05.2.log",
				Counts: transcript.LMStudioEventCounts{Response: 1, Warn: 1}, Lines: []int{13, 15}},
			{File: "2026-03/2026-03-05.2.log"},
		},
	}
}

// A logical-session view carries its requests' session ID and body-free
// event summaries in place of the events (requirements 1.5, 4.2).
func TestSession_LMStudioSessionView(t *testing.T) {
	got := encode(t, Session(lmSessionView(t), SessionContext{
		Source: "lmstudio", ID: "lmstudio-thread-a2ba3f64e66545724423033a46a31b5fa9f6db429bfffb60",
	}))

	want, err := os.ReadFile(filepath.Join("testdata", "handwritten", "lmstudio-session-view.json"))
	require.NoError(t, err)
	assert.JSONEq(t, string(want), string(got))
	assert.Contains(t, string(got), `"lines": []`, "a file without events still lists an empty array")
}

func TestSchema_LMStudioRejectsInvalidFields(t *testing.T) {
	schemas := loadSchemas(t)
	sessionView := func(change func(doc, request, entry, summary map[string]any)) any {
		doc := readJSON(t, filepath.Join("testdata", "handwritten", "lmstudio-session-view.json")).(map[string]any)
		change(doc,
			doc["requests"].([]any)[0].(map[string]any),
			doc["entries"].([]any)[2].(map[string]any),
			doc["unattributed_summary"].([]any)[0].(map[string]any))
		return doc
	}
	for name, change := range map[string]func(doc, request, entry, summary map[string]any){
		"request without id":                   func(_, request, _, _ map[string]any) { delete(request, "id") },
		"request without response_attribution": func(_, request, _, _ map[string]any) { delete(request, "response_attribution") },
		"request without wall_time":            func(_, request, _, _ map[string]any) { delete(request, "wall_time") },
		"zero start_line":                      func(_, request, _, _ map[string]any) { request["start_line"] = json.Number("0") },
		"string end_line":                      func(_, request, _, _ map[string]any) { request["end_line"] = "12" },
		"wall time as timestamp":               func(_, request, _, _ map[string]any) { request["timestamp"] = "2026-03-05 14:00:00" },
		"array parameters":                     func(_, request, _, _ map[string]any) { request["parameters"] = []any{"stream"} },
		"zero prior_prefix_count":              func(_, request, _, _ map[string]any) { request["prior_prefix_count"] = json.Number("0") },
		"string context_discontinuity":         func(_, request, _, _ map[string]any) { request["context_discontinuity"] = "true" },
		"empty request_id":                     func(_, _, entry, _ map[string]any) { entry["request_id"] = "" },
		"string supplied_context":              func(_, _, entry, _ map[string]any) { entry["supplied_context"] = "true" },
		"zero source_line":                     func(_, _, entry, _ map[string]any) { entry["source_line"] = json.Number("0") },
		"string truncated": func(_, _, entry, _ map[string]any) {
			entry["content"].([]any)[0].(map[string]any)["truncated"] = "yes"
		},
		"summary without lines": func(_, _, _, summary map[string]any) { delete(summary, "lines") },
		"summary without file":  func(_, _, _, summary map[string]any) { delete(summary, "file") },
		"counts without warn":   func(_, _, _, summary map[string]any) { delete(summary["counts"].(map[string]any), "warn") },
		"negative response count": func(_, _, _, summary map[string]any) {
			summary["counts"].(map[string]any)["response"] = json.Number("-1")
		},
		"zero line":                   func(_, _, _, summary map[string]any) { summary["lines"] = []any{json.Number("0")} },
		"empty time_zone":             func(doc, _, _, _ map[string]any) { doc["session"].(map[string]any)["time_zone"] = "" },
		"object unattributed_summary": func(doc, _, _, _ map[string]any) { doc["unattributed_summary"] = map[string]any{} },
	} {
		t.Run(name, func(t *testing.T) {
			doc := sessionView(change)
			assert.Error(t, schemas.published.Validate(doc))
			assert.Error(t, schemas.closed.Validate(doc))
		})
	}

	listing := func(change func(row map[string]any)) any {
		doc := readJSON(t, filepath.Join("testdata", "handwritten", "listing-lmstudio.json")).(map[string]any)
		change(doc["sessions"].([]any)[2].(map[string]any))
		return doc
	}
	for name, change := range map[string]func(row map[string]any){
		"zero request_count":   func(row map[string]any) { row["request_count"] = json.Number("0") },
		"string request_count": func(row map[string]any) { row["request_count"] = "3" },
	} {
		t.Run(name, func(t *testing.T) {
			assert.Error(t, schemas.published.Validate(listing(change)))
		})
	}
}

func TestSchema_LMStudioClosedCopyRejectsUndeclaredFields(t *testing.T) {
	schemas := loadSchemas(t)
	for name, change := range map[string]func(doc map[string]any){
		"request": func(doc map[string]any) { doc["requests"].([]any)[0].(map[string]any)["undeclared"] = true },
		"summary": func(doc map[string]any) { doc["unattributed_summary"].([]any)[0].(map[string]any)["undeclared"] = true },
		"counts": func(doc map[string]any) {
			doc["unattributed_summary"].([]any)[0].(map[string]any)["counts"].(map[string]any)["undeclared"] = json.Number("1")
		},
	} {
		t.Run(name, func(t *testing.T) {
			doc := readJSON(t, filepath.Join("testdata", "handwritten", "lmstudio-session-view.json")).(map[string]any)
			change(doc)
			assert.NoError(t, schemas.published.Validate(doc))
			assert.Error(t, schemas.closed.Validate(doc))
		})
	}

	doc := readJSON(t, filepath.Join("testdata", "handwritten", "lmstudio-session-view.json")).(map[string]any)
	doc["requests"].([]any)[0].(map[string]any)["parameters"].(map[string]any)["any_field"] = "kept"
	assert.NoError(t, schemas.closed.Validate(doc), "parameters hold whatever the body logged")
}
