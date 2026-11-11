package transcript

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Final identifiers replace every temporary request key: request IDs, entry
// request IDs and request-scoped tool IDs. Event-scoped tool IDs and every
// recorded identifier stay as they were.
func TestAssignLMStudioIDs(t *testing.T) {
	before := parseLMFixture(t, "tools.log")
	res := parseLMFixture(t, "tools.log")
	final := func(key string) string { return "lmstudio-req-p-x-" + key }
	AssignLMStudioIDs(res, func(req LMStudioRequest) (string, string) {
		return final(strconv.Itoa(req.StartLine)), "session-" + strconv.Itoa(req.StartLine)
	})

	require.NotEmpty(t, res.LMStudioRequests)
	for i, req := range res.LMStudioRequests {
		key := before.LMStudioRequests[i].ID
		assert.Equal(t, final(key), req.ID)
		assert.Equal(t, "session-"+key, req.SessionID)
	}
	rewritten := func(id string) string {
		rest, ok := strings.CutPrefix(id, "req:")
		if !ok {
			return id
		}
		key, recorded, _ := strings.Cut(rest, ":")
		return "req:" + final(key) + ":" + recorded
	}
	toolIDs := 0
	for i, e := range res.Entries {
		was := before.Entries[i]
		if was.RequestID == "" {
			assert.Empty(t, e.RequestID)
		} else {
			assert.Equal(t, final(was.RequestID), e.RequestID)
		}
		if e.Message == nil {
			continue
		}
		for j, item := range e.Message.Content {
			old := was.Message.Content[j]
			assert.Equal(t, rewritten(old.ID), item.ID)
			assert.Equal(t, rewritten(old.ToolUseID), item.ToolUseID)
			assert.Equal(t, old.ItemID, item.ItemID, "recorded identifier")
			if strings.HasPrefix(old.ID, "req:") || strings.HasPrefix(old.ToolUseID, "req:") {
				toolIDs++
			}
		}
	}
	assert.Positive(t, toolIDs, "the fixture has request-scoped tool IDs")
}

// lmSessionFiles are two log files sharing thread t, with unattributed events,
// a request of another session and a warning in each kind of record.
var lmSessionFiles = []struct{ key, content string }{
	{"2026-11/2026-11-01.1.log", strings.Join([]string{
		`[2026-11-01 10:00:00][INFO] Received request: POST to /v1/chat/completions with body {"model":"m","messages":[{"role":"user","content":"one"}],"client_metadata":{"thread_id":"t"}}`,
		`[2026-11-01 10:00:01][INFO] Generated prediction: {"choices":[{"message":{"role":"assistant","content":"logged answer"}}]}`,
		`[2026-11-01 10:00:02][INFO] Received request: POST to /v1/chat/completions with body {"model":"m","messages":[{"role":"user","content":"other client"}]}`,
		`[2026-11-01 10:00:03][WARN] warning text`,
	}, "\n") + "\n"},
	{"2026-11/2026-11-01.2.log", strings.Join([]string{
		`[2026-11-01 11:00:00][ERROR] error text`,
		`[2026-11-01 11:00:01][INFO] Received request: POST to /v1/chat/completions with body {"model":"m2","messages":[{"role":"user","content":"one"},{"role":"assistant","content":"answer"},{"role":"user","content":[{"type":"mystery"}]}],"client_metadata":{"thread_id":"t"}}`,
		`[2026-11-01 11:00:02][INFO] Received request: POST to /v1/chat/completions with body {"model":`,
		`[2026-11-01 11:00:03][INFO] Received request: POST to /v1/chat/completions with body {"model":"m","messages":[]}`,
	}, "\n") + "\n"},
	{"2026-11/2026-11-02.1.log", `[2026-11-02 09:00:00][ERROR] unrelated file` + "\n"},
}

// lmSessionParts parses lmSessionFiles with identifiers that name the file,
// line and session, as the sessions layer assigns them.
func lmSessionParts(t *testing.T, tz string) []LMStudioSessionPart {
	t.Helper()
	t.Setenv("TZ", tz)
	parts := make([]LMStudioSessionPart, len(lmSessionFiles))
	for i, f := range lmSessionFiles {
		res, err := ParseLMStudio(strings.NewReader(f.content))
		require.NoError(t, err)
		AssignLMStudioIDs(res, func(req LMStudioRequest) (string, string) {
			id := f.key + "#" + strconv.Itoa(req.StartLine)
			if req.ThreadID != "" {
				return id, "thread-" + req.ThreadID
			}
			return id, id
		})
		parts[i] = LMStudioSessionPart{File: f.key, Result: res}
		if i+1 < len(lmSessionFiles) {
			parts[i].NextFile = lmSessionFiles[i+1].key
		}
	}
	return parts
}

func inLMThread(req LMStudioRequest) bool { return req.SessionID == "thread-t" }

// A session view keeps the selected requests of each file with their entries,
// replaces unattributed events with body-free summaries and keeps only the
// selected records' warnings, labelled with their file (requirements 1.5, 4.2,
// 6.2). History prefixes are compared across files (3.5).
func TestAssembleLMStudioSession(t *testing.T) {
	res := AssembleLMStudioSession(lmSessionParts(t, ""), inLMThread)

	type request struct {
		id, file    string
		priorPrefix int
	}
	var requests []request
	for _, req := range res.LMStudioRequests {
		requests = append(requests, request{req.ID, req.File, req.PriorPrefixCount})
	}
	assert.Equal(t, []request{
		{"2026-11/2026-11-01.1.log#1", "2026-11/2026-11-01.1.log", 0},
		{"2026-11/2026-11-01.2.log#2", "2026-11/2026-11-01.2.log", 1},
	}, requests)

	var entries []string
	for _, e := range res.Entries {
		assert.False(t, e.Unattributed, "no unattributed entry in a session view")
		entries = append(entries, e.RequestID+" "+e.Type)
	}
	assert.Equal(t, []string{
		"2026-11/2026-11-01.1.log#1 user",
		"2026-11/2026-11-01.2.log#2 user", "2026-11/2026-11-01.2.log#2 assistant", "2026-11/2026-11-01.2.log#2 user",
	}, entries)

	assert.Equal(t, []LMStudioEventSummary{
		{File: "2026-11/2026-11-01.1.log", NextFile: "2026-11/2026-11-01.2.log", Counts: LMStudioEventCounts{Response: 1, Warn: 1}, Lines: []int{2, 4}},
		{File: "2026-11/2026-11-01.2.log", NextFile: "2026-11/2026-11-02.1.log", Counts: LMStudioEventCounts{Error: 1}, Lines: []int{1}},
	}, res.LMStudioEvents, "the file without the session contributes nothing")

	require.Len(t, res.Warnings, 1, "the other request's invalid_record is not the session's")
	assert.Equal(t, WarningUnrecognizedRecord, res.Warnings[0].Code)
	assert.Equal(t, 2, res.Warnings[0].Line)
	assert.True(t, strings.HasPrefix(res.Warnings[0].Message, "2026-11/2026-11-01.2.log: "), res.Warnings[0].Message)

	assert.Equal(t, FormatLMStudio, res.Format)
	assert.Equal(t, "2026-11-01T10:00:00Z", res.Metadata.StartedAt)
	assert.Equal(t, "m", res.Metadata.FirstModel)
	assert.Equal(t, "t", res.Metadata.ThreadID)
	assert.Equal(t, lmSessionParts(t, "")[0].Result.Metadata.TimeZone, res.Metadata.TimeZone)
}

// An unloadable zone is reported once per session view, not once per file
// (requirement 5.5).
func TestAssembleLMStudioSession_ZoneWarningOnce(t *testing.T) {
	res := AssembleLMStudioSession(lmSessionParts(t, "Nowhere/Invalid"), inLMThread)
	require.NotEmpty(t, res.Warnings)
	assert.Equal(t, WarningTimeZone, res.Warnings[0].Code)
	assert.Equal(t, 1, countWarnings(res.Warnings, WarningTimeZone))
	assert.Empty(t, res.Metadata.StartedAt, "no instant without a zone")
}

// Markdown, standalone HTML and web HTML show each request's file and the
// same event summaries, without any event's content (requirements 4.2, 4.5).
func TestLMStudioRender_SessionView(t *testing.T) {
	res := AssembleLMStudioSession(lmSessionParts(t, ""), inLMThread)
	view := NewLMStudioView(res)
	out := renderLM(res)
	assert.Contains(t, out.html, out.fragment)

	var mdParts, htmlParts []string
	for _, req := range res.LMStudioRequests {
		mdParts = append(mdParts, lmPartsMarkdown(view.requestParts(&req)))
		htmlParts = append(htmlParts, lmPartsHTML(view.requestParts(&req)))
	}
	mdParts = append(mdParts, lmEventsHeading)
	htmlParts = append(htmlParts, lmEventsHeading)
	for _, s := range res.LMStudioEvents {
		mdParts = append(mdParts, lmPartsMarkdown(eventSummaryParts(s)))
		htmlParts = append(htmlParts, lmPartsHTML(eventSummaryParts(s)))
	}
	assertInOrder(t, out.md, mdParts)
	assertInOrder(t, out.fragment, htmlParts)
	for _, r := range []string{out.md, out.fragment} {
		assert.Contains(t, r, "2026-11/2026-11-01.1.log")
		assert.Contains(t, r, "1 response, 1 warning, 0 errors")
		assert.Contains(t, r, "lines 2, 4")
		assert.NotContains(t, r, "last discovered file", "a later file was discovered")
		for _, content := range []string{"logged answer", "warning text", "error text", "other client", "Unattributed · "} {
			assert.NotContains(t, r, content)
		}
	}
	assert.Contains(t, out.md, "`apsis -a lmstudio <file>`")
	assert.Contains(t, out.fragment, "<code>apsis -a lmstudio &lt;file&gt;</code>")
	assert.Contains(t, out.md, "`apsis --all-projects -s lmstudio <session-id>`", "the reopen route names scope and source (7.1)")
	assert.Contains(t, out.fragment, "<code>apsis --all-projects -s lmstudio &lt;session-id&gt;</code>")
	assert.Equal(t, strings.Count(out.md, "📄"), strings.Count(out.fragment, "📄"))
}
