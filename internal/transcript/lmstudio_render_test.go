package transcript

import (
	stdhtml "html"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// lmRenderFixtures cover threads and repeated history, tool calls and
// truncation, classification and unsupported endpoints, and body states.
var lmRenderFixtures = []string{"threads", "tools", "classification", "recovery"}

// lmRendered holds one parse result in every human-readable output.
type lmRendered struct {
	md, html, fragment string
}

func renderLM(res *ParseResult) lmRendered {
	opts := RenderOptions{SessionID: "example-log", LMStudio: NewLMStudioView(res)}
	return lmRendered{
		md:       RenderMarkdown(res.Entries, opts),
		html:     RenderHTML(res.Entries, opts),
		fragment: RenderHTMLFragment(res.Entries, opts),
	}
}

func renderLMInput(t *testing.T, tz, input string) lmRendered {
	t.Helper()
	t.Setenv("TZ", tz)
	res, err := ParseLMStudio(strings.NewReader(input))
	require.NoError(t, err)
	return renderLM(res)
}

func TestLMStudioRender_Goldens(t *testing.T) {
	for _, name := range lmRenderFixtures {
		t.Run(name, func(t *testing.T) {
			out := renderLM(parseLMFixture(t, name+".log"))
			for file, got := range map[string]string{
				name + ".md.golden":            out.md,
				name + ".html.golden":          styleBlock.ReplaceAllString(out.html, "<style></style>"),
				name + ".fragment.html.golden": out.fragment,
			} {
				path := filepath.Join("testdata", "lmstudio", file)
				if *updateGoldens {
					require.NoError(t, os.WriteFile(path, []byte(got), 0o600))
				}
				want, err := os.ReadFile(path)
				require.NoError(t, err)
				assert.Equal(t, string(want), got, file)
			}
		})
	}
}

// Markdown, standalone HTML and web HTML show the same boundaries in the same
// order, with the same content, states and markers (requirements 4.5, 8.2).
func TestLMStudioRender_Parity(t *testing.T) {
	for _, name := range lmRenderFixtures {
		t.Run(name, func(t *testing.T) {
			res := parseLMFixture(t, name+".log")
			view := NewLMStudioView(res)
			out := renderLM(res)
			assert.Contains(t, out.html, out.fragment, "standalone and web HTML share one body")

			// Boundaries: each request and log-level entry, in order.
			var mdParts, htmlParts []string
			for _, seg := range view.segments(res.Entries) {
				if seg.request != nil {
					parts := view.requestParts(seg.request)
					mdParts = append(mdParts, lmPartsMarkdown(parts))
					htmlParts = append(htmlParts, lmPartsHTML(parts))
					continue
				}
				parts := view.eventParts(&seg.entries[0])
				mdParts = append(mdParts, lmPartsHTML(parts))
				htmlParts = append(htmlParts, lmPartsHTML(parts))
			}
			require.NotEmpty(t, mdParts)
			assertInOrder(t, out.md, mdParts)
			assertInOrder(t, out.fragment, htmlParts)

			// Content: every shown text, thinking and log line.
			for _, s := range lmShownTexts(view, res.Entries) {
				assert.Contains(t, out.md, lmMarkdownSource(s), "markdown")
				assert.Contains(t, stdhtml.UnescapeString(out.fragment), s, "html")
			}

			// States and markers occur equally often.
			for _, marker := range []string{
				"📨", "Unattributed · ", "Unsupported endpoint", "supplied context", "⤷ repeats",
				"🎛️", "🔧", "💭 Thinking", "Event: ", "Token usage", "✂️",
				"body not logged", "body malformed", "body incomplete", "context discontinuity", "extends prior request",
			} {
				assert.Equal(t, strings.Count(out.md, marker), strings.Count(out.fragment, marker), marker)
			}
		})
	}
}

// lmShownTexts returns the text, thinking and log content a render shows:
// everything outside a request's repeated prefix.
func lmShownTexts(view *LMStudioView, entries []Entry) []string {
	var texts []string
	for _, seg := range view.segments(entries) {
		shown := seg.entries
		if seg.request != nil {
			shown, _ = seg.shown()
		}
		for _, e := range shown {
			if e.Message == nil {
				continue
			}
			for _, item := range e.Message.Content {
				for _, s := range []string{item.Text, item.Thinking} {
					if strings.TrimSpace(s) != "" {
						texts = append(texts, s)
					}
				}
			}
		}
	}
	return texts
}

func assertInOrder(t *testing.T, out string, parts []string) {
	t.Helper()
	pos := 0
	for _, p := range parts {
		i := strings.Index(out[pos:], p)
		if i < 0 {
			t.Errorf("%q missing or out of order", p)
			return
		}
		pos += i + len(p)
	}
}

func TestLMStudioRender_Notices(t *testing.T) {
	out := renderLM(parseLMFixture(t, "threads.log"))
	for name, got := range map[string]string{"markdown": out.md, "html": out.fragment} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, 1, strings.Count(got, "Request attribution unknown:"), "attribution is explained once (Q16)")
			assert.Equal(t, 1, strings.Count(got, "UTC, assumed for every logged wall time; the log records no zone"), "the zone is stated once")
			assert.Contains(t, got, "2026-03-05T14:00:00Z (assumed UTC)")
		})
	}
}

// A repeated prefix is referenced, not shown again, and only within one
// thread (requirement 3.5).
func TestLMStudioRender_RepeatedPrefix(t *testing.T) {
	out := renderLM(parseLMFixture(t, "threads.log"))
	for name, got := range map[string]string{"markdown": out.md, "html": out.fragment} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, 1, strings.Count(got, "repeats the first 2 items of the thread"))
			assert.Equal(t, 1, strings.Count(got, "Fix the failing test."), "the repeated user message")
			assert.Equal(t, 2, strings.Count(got, "You are a coding agent."), "the compaction request is a discontinuity, so it shows its instructions")
		})
	}
}

// Tool calls pair with results inside their own request or event only, even
// when the recorded IDs collide (requirement 2.8).
func TestLMStudioRender_ToolScopes(t *testing.T) {
	out := renderLM(parseLMFixture(t, "tools.log"))
	for name, got := range map[string]string{"markdown": out.md, "html": out.fragment} {
		t.Run(name, func(t *testing.T) {
			first := blockContaining(t, got, "<details", "🔧 list_files")
			assert.Contains(t, first, "b.txt")
			second := blockContaining(t, got, "<details", "not json")
			assert.Contains(t, second, "file contents")
			assert.NotContains(t, second, "b.txt")

			response := blockContaining(t, got, "<details", "Tool call: read_file")
			assert.Contains(t, response, "a.txt", "the response's call keeps its own input")
			assert.NotContains(t, response, "file contents", "and pairs with no request's result")

			orphan := blockContaining(t, got, "<details", "orphan")
			assert.Contains(t, orphan, "Tool Error", "an unmatched result is kept")
		})
	}

	// The call renders under the assistant message that made it.
	section := blockContaining(t, out.fragment, "<section", "not json")
	section, _, _ = strings.Cut(section, "</section>")
	assert.Contains(t, section, "supplied context")
	assert.Contains(t, section, "not json")
	assert.Contains(t, section, "file contents")
	heading := blockContaining(t, out.md, "\n### ", "not json")
	assert.True(t, strings.HasPrefix(heading, "🤖 Assistant · supplied context"), heading)
}

// Parameters are labelled outside the conversation, summarised, and keep
// their values and truncation markers as logged (requirement 2.9).
func TestLMStudioRender_Parameters(t *testing.T) {
	out := renderLM(parseLMFixture(t, "threads.log"))
	assert.Contains(t, out.md, "<summary>🎛️ Request parameters · stream true · tools: shell</summary>")
	assert.Contains(t, out.fragment, "<span class=\"icon\">🎛️</span> Request parameters · stream true · tools: shell</summary>")
	assert.Contains(t, out.md, `"x-codex-turn-metadata": "{\"request_kind\":\"turn\",\"thread_id\":\"thread-beta\",\"sandbox\":\"wor ... <Truncated in logs> ... \"}"`)
	assert.Contains(t, out.fragment, `&#34;thread_id&#34;: &#34;thread-gam ... &lt;Truncated in logs&gt; ... mma-long&#34;`)
}

func TestLMParameterSummary(t *testing.T) {
	tests := map[string]struct {
		params string
		want   string
	}{
		"no parameters":       {``, "Request parameters"},
		"not an object":       {`[1]`, "Request parameters"},
		"stream only":         {`{"stream":false}`, "Request parameters · stream false"},
		"chat function tools": {`{"tools":[{"type":"function","function":{"name":"a"}},{"type":"function","function":{"name":"b"}}]}`, "Request parameters · tools: a, b"},
		"responses and types": {`{"stream":true,"tools":[{"type":"function","name":"shell"},{"type":"web_search"}]}`, "Request parameters · stream true · tools: shell, web_search"},
		"tools not an array":  {`{"tools":"all"}`, "Request parameters"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, lmParameterSummary([]byte(tc.params)))
		})
	}
}

const lmAmbiguousLog = `[2026-04-05 02:30:00][DEBUG] Received request: POST to /v1/completions with body {
  "model": "example-model-a",
  "prompt": "During the repeated hour."
}
[2026-04-05 02:30:01][WARN][example-model-a] A warning in the repeated hour.
[2026-04-05 04:00:00][DEBUG] Received request: POST to /v1/completions with body {
  "model": "example-model-a",
  "prompt": "After it."
}
`

// Times show their assumed zone, or the wall time and why no instant is
// shown (requirements 5.1, 5.2, 5.5).
func TestLMStudioRender_Times(t *testing.T) {
	t.Run("ambiguous", func(t *testing.T) {
		out := renderLMInput(t, "Australia/Melbourne", lmAmbiguousLog)
		for name, got := range map[string]string{"markdown": out.md, "html": out.fragment} {
			assert.Contains(t, got, "Australia/Melbourne, assumed for every logged wall time", name)
			assert.Contains(t, got, "2026-04-05 02:30:00 (wall time; no unique time in Australia/Melbourne)", name)
			assert.Contains(t, got, "2026-04-05 02:30:01 (wall time; no unique time in Australia/Melbourne)", name)
			assert.Contains(t, got, "2026-04-05T04:00:00+10:00 (assumed Australia/Melbourne)", name)
		}
	})
	t.Run("zone not loaded", func(t *testing.T) {
		out := renderLMInput(t, "Example/Nowhere", lmAmbiguousLog)
		for name, got := range map[string]string{"markdown": out.md, "html": out.fragment} {
			assert.Contains(t, got, "could not be loaded", name)
			assert.NotContains(t, got, "assumed", name)
			assert.Contains(t, got, "2026-04-05 04:00:00 (wall time; zone not loaded)", name)
		}
		assert.Contains(t, out.md, "⚠️ **Time zone:**")
		assert.Contains(t, out.fragment, `<p class="lm-note warning">⚠️ <strong>Time zone:</strong>`)
	})
}

// A request without translated content keeps its boundary and body state
// (requirement 2.5).
func TestLMStudioRender_BodyStates(t *testing.T) {
	out := renderLM(parseLMFixture(t, "recovery.log"))
	for name, got := range map[string]string{"markdown": out.md, "html": out.fragment} {
		t.Run(name, func(t *testing.T) {
			assert.Contains(t, got, "line 3 · 2026-03-04T08:00:00Z (assumed UTC) · singleton session (no valid thread ID) · body not logged")
			assert.Contains(t, got, "lines 4-9 · 2026-03-04T08:00:01Z (assumed UTC) · singleton session (no valid thread ID) · ⚠ body malformed")
			assert.Contains(t, got, "lines 28-30 · 2026-03-04T08:00:06Z (assumed UTC) · singleton session (no valid thread ID) · ⏳ body incomplete")
			assert.NotContains(t, got, "this body", "untranslated bodies are not shown")
		})
	}
}

// The truncation marker stays visible as text in rendered Markdown and HTML,
// and the item is marked (requirement 2.6).
func TestLMStudioRender_Truncation(t *testing.T) {
	out := renderLM(parseLMFixture(t, "tools.log"))
	assert.Contains(t, out.md, `from the ... \<Truncated in logs> ... to the end`)
	assert.Contains(t, out.md, "✂️ *(truncated in logs)*")
	assert.Contains(t, out.fragment, "from the ... &lt;Truncated in logs&gt; ... to the end")
	assert.Contains(t, out.fragment, `<p class="truncated">✂️ (truncated in logs)</p>`)

	// Markdown-rendered content keeps the marker as text too.
	rendered := markdownToHTML(lmMarkdownSource("cut ... " + lmTruncationMarker + " ... here"))
	assert.Contains(t, rendered, "cut ... &lt;Truncated in logs&gt; ... here")
}
