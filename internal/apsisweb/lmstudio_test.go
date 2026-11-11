package apsisweb

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// lmWebLog has a thread whose second request holds an unrecognised content
// type, and a singleton whose wall time does not exist.
const lmWebLog = `[2026-11-01 10:00:00][INFO] Received request: POST to /v1/chat/completions with body {"model":"example-model","messages":[{"role":"user","content":"first question"}],"client_metadata":{"thread_id":"thread-a"}}
[2026-11-01 10:00:01][WARN] warning text
[2026-11-01 10:00:02][INFO] Received request: POST to /v1/chat/completions with body {"model":"example-model","messages":[{"role":"user","content":[{"type":"mystery_part"}]}],"client_metadata":{"thread_id":"thread-a"}}
[2026-02-30 10:00:03][INFO] Received request: POST to /v1/chat/completions with body {"model":"example-model","messages":[{"role":"user","content":"hi"}]}
`

// lmWebRoot writes lmWebLog into a new LM Studio log directory under a fresh
// HOME, with TZ set to UTC.
func lmWebRoot(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("TZ", "")
	root := t.TempDir()
	path := filepath.Join(root, "2026-11", "2026-11-01.1.log")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(lmWebLog), 0o600))
	return root
}

var lmSessionLink = regexp.MustCompile(`href="/sessions/lmstudio/(lmstudio-thread-[0-9a-f]+)"`)

// The source is listed, and gets a filter pill, only in all-projects scope
// with a log directory (requirement 7.1).
func TestLMStudioListedOnlyAcrossProjects(t *testing.T) {
	root := lmWebRoot(t)
	for name, tc := range map[string]struct {
		config Config
		want   bool
	}{
		"project scope":            {config: Config{ProjectPath: t.TempDir(), LMStudioLogDir: root}},
		"all projects, disabled":   {config: Config{AllProjects: true}},
		"all projects, configured": {config: Config{AllProjects: true, LMStudioLogDir: root}, want: true},
	} {
		t.Run(name, func(t *testing.T) {
			srv, err := New(tc.config)
			require.NoError(t, err)
			_, page := fetch(t, srv, "/")
			assert.Equal(t, tc.want, strings.Contains(page, `data-source="lmstudio"`), "filter pill")
			assert.Equal(t, tc.want, strings.Contains(page, `href="/sessions/lmstudio/`), "session rows")
		})
	}
}

func TestLMStudioListingRows(t *testing.T) {
	srv, err := New(Config{AllProjects: true, LMStudioLogDir: lmWebRoot(t)})
	require.NoError(t, err)
	_, fragment := fetch(t, srv, "/api/sessions")

	assert.Contains(t, fragment, `<span class="session-size">2 requests</span>`, "a session's size is its request count")
	assert.Contains(t, fragment, `<span class="session-size">1 request</span>`)
	assert.Contains(t, fragment, `<span class="session-date">time unknown</span>`, "a nonexistent first wall time stays unknown")
	assert.Contains(t, fragment, `datetime="2026-11-01T10:00:00Z"`)
	assert.NotContains(t, fragment, "0001-01-01")
	assert.Contains(t, fragment, `<span class="session-details">example-model · thread thread-a · time assumed in UTC</span>`,
		"model, thread and assumed zone tell opaque identifiers apart")
	assert.Contains(t, fragment, `<span class="session-details">example-model · time assumed in UTC</span>`, "a singleton has no thread")
}

// A session page shows the session's parse warnings and thread ID, and the
// CLI instruction for inspecting whole log files (requirements 4.2, 6.2).
func TestLMStudioTranscriptView(t *testing.T) {
	srv, err := New(Config{AllProjects: true, LMStudioLogDir: lmWebRoot(t)})
	require.NoError(t, err)
	_, fragment := fetch(t, srv, "/api/sessions")
	m := lmSessionLink.FindStringSubmatch(fragment)
	require.NotNil(t, m, fragment)

	code, page := fetch(t, srv, "/sessions/lmstudio/"+m[1])
	require.Equal(t, http.StatusOK, code, page)
	assert.Contains(t, page, `class="warning-banner"`)
	assert.Contains(t, page, "line 3: 2026-11/2026-11-01.1.log:", "warnings name their file and line")
	assert.Contains(t, page, "Thread ID thread-a")
	assert.Contains(t, page, "first question")
	assert.Contains(t, page, "<code>apsis -a lmstudio &lt;file&gt;</code>")
	assert.NotContains(t, page, "warning text", "unattributed content stays out of the session view")
}

// Every browser instruction for reopening an LM Studio session names
// all-projects scope and the source (requirement 7.1).
func TestCLICommand(t *testing.T) {
	assert.Equal(t, "apsis --all-projects -s lmstudio lmstudio-thread-abc", cliCommand("lmstudio", "lmstudio-thread-abc"))
	assert.Equal(t, "apsis session-1", cliCommand("claude", "session-1"))
}

// The network warning adds the server-log disclosure only when LM Studio
// sessions are listed (requirement 7.5).
func TestNetworkWarning(t *testing.T) {
	const disclosure = "credential-bearing integration headers"
	for name, tc := range map[string]struct {
		config       Config
		wantWarning  bool
		wantDisclose bool
	}{
		"localhost":                 {config: Config{Bind: "localhost", AllProjects: true, LMStudioLogDir: "/logs"}},
		"network, project scope":    {config: Config{Bind: "0.0.0.0", LMStudioLogDir: "/logs"}, wantWarning: true},
		"network, LM Studio off":    {config: Config{Bind: "0.0.0.0", AllProjects: true}, wantWarning: true},
		"network, LM Studio listed": {config: Config{Bind: "0.0.0.0", AllProjects: true, LMStudioLogDir: "/logs"}, wantWarning: true, wantDisclose: true},
	} {
		t.Run(name, func(t *testing.T) {
			warning := tc.config.NetworkWarning()
			assert.Equal(t, tc.wantWarning, strings.Contains(warning, "accessible from the network"))
			assert.Equal(t, tc.wantDisclose, strings.Contains(warning, disclosure))
		})
	}
}

func TestLMStudioBadgeStyles(t *testing.T) {
	css, err := os.ReadFile(filepath.Join("static", "style.css"))
	require.NoError(t, err)
	assert.Equal(t, 2, strings.Count(string(css), ".source-lmstudio {"), "light and dark badge rules")
	assert.Equal(t, "5", CSSVersion, "the stylesheet changed, so its cache-busting version rises")
}

// BenchmarkLMStudioListFragment renders the HTMX session list for a generated
// log directory of 30 files holding 30,000 unthreaded requests (one row each)
// plus 3,000 threaded ones in 10 threads, and records the rendered fragment
// size beside the workload (requirement 8.5): a cold server, then a refresh
// of the unchanged directory through one server's cached lister.
func BenchmarkLMStudioListFragment(b *testing.B) {
	b.Setenv("HOME", b.TempDir())
	b.Setenv("TZ", "")
	root := b.TempDir()
	var inputBytes int64
	requests := 0
	for f := range 30 {
		var sb strings.Builder
		for i := range 1100 {
			meta := ""
			if i%11 == 10 {
				meta = fmt.Sprintf(`,"client_metadata":{"thread_id":"thread-%d"}`, (f+i)%10)
			}
			fmt.Fprintf(&sb, `[2026-11-%02d 09:%02d:%02d][INFO] Received request: POST to /v1/chat/completions with body {"model":"example-model","messages":[{"role":"user","content":"hi"}]%s}`+"\n",
				1+f%28, i/60%60, i%60, meta)
			requests++
		}
		inputBytes += int64(sb.Len())
		path := filepath.Join(root, "2026-11", fmt.Sprintf("2026-11-%02d.%d.log", 1+f%28, f/28+1))
		require.NoError(b, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(b, os.WriteFile(path, []byte(sb.String()), 0o600))
	}
	config := Config{AllProjects: true, LMStudioLogDir: root}
	report := func(b *testing.B, fragment string) {
		b.ReportMetric(float64(inputBytes), "input-bytes")
		b.ReportMetric(float64(requests), "requests")
		b.ReportMetric(float64(strings.Count(fragment, `href="/sessions/lmstudio/`)), "rows")
		b.ReportMetric(float64(len(fragment)), "fragment-bytes")
	}

	b.Run("cold", func(b *testing.B) {
		var fragment string
		for b.Loop() {
			srv, err := New(config)
			require.NoError(b, err)
			_, fragment = fetch(b, srv, "/api/sessions")
		}
		report(b, fragment)
	})
	b.Run("unchanged refresh", func(b *testing.B) {
		srv, err := New(config)
		require.NoError(b, err)
		_, fragment := fetch(b, srv, "/api/sessions")
		for b.Loop() {
			_, fragment = fetch(b, srv, "/api/sessions")
		}
		report(b, fragment)
	})
}
