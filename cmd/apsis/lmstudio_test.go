package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/arjenschwarz/orbit/internal/sharedjson"
)

// lmRecord is a one-line LM Studio request record. An empty thread records
// no client thread ID.
func lmRecord(wall, thread string) string {
	meta := ""
	if thread != "" {
		meta = fmt.Sprintf(`,"client_metadata":{"thread_id":%q}`, thread)
	}
	return fmt.Sprintf(`[%s][INFO] Received request: POST to /v1/chat/completions with body {"model":"example-model","messages":[{"role":"user","content":"hi"}]%s}`+"\n",
		wall, meta)
}

// lmLog is a log holding a threaded request, a WARN record and a singleton.
var lmLog = lmRecord("2026-11-01 10:00:00", "thread-a") +
	"[2026-11-01 10:00:01][WARN] warning text\n" +
	lmRecord("2026-11-01 10:00:02", "")

// lmRoot writes log into a new LM Studio log directory, under a fresh HOME
// with TZ set to UTC, and returns the directory and the file.
func lmRoot(t *testing.T, log string) (root, file string) {
	t.Helper()
	fakeHome(t)
	t.Setenv("TZ", "")
	root, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	return root, writeFile(t, filepath.Join(root, "2026-11", "2026-11-01.1.log"), log)
}

func listedIDs(t *testing.T, root string) []string {
	t.Helper()
	stdout, _, err := runCLIStdout(t, &Config{List: true, AllProjects: true, Format: "shared-json", LMStudioLogDir: &root})
	require.NoError(t, err)
	var ids []string
	for _, s := range decodeListing(t, stdout).Sessions {
		ids = append(ids, s.ID)
	}
	return ids
}

func requestIDs(doc sharedjson.SessionDocument) (ids, sessionIDs []string) {
	for _, req := range doc.Requests {
		ids = append(ids, req.ID)
		sessionIDs = append(sessionIDs, req.SessionID)
	}
	return ids, sessionIDs
}

func TestLMStudioLogDir(t *testing.T) {
	home := fakeHome(t)
	cwd, err := os.Getwd()
	require.NoError(t, err)
	unset := func(t *testing.T) {
		t.Setenv(lmStudioLogDirEnv, "")
		require.NoError(t, os.Unsetenv(lmStudioLogDirEnv))
	}
	setEnv := func(v string) func(*testing.T) { return func(t *testing.T) { t.Setenv(lmStudioLogDirEnv, v) } }
	flagOf := func(v string) *string { return &v }

	tests := map[string]struct {
		env          func(*testing.T)
		flag         *string
		wantDir      string
		wantExplicit bool
	}{
		"default":                   {env: unset, wantDir: filepath.Join(home, ".lmstudio", "server-logs")},
		"variable":                  {env: setEnv("/logs/env"), wantDir: filepath.Clean("/logs/env"), wantExplicit: true},
		"empty variable disables":   {env: setEnv(""), wantExplicit: true},
		"flag replaces variable":    {env: setEnv("/logs/env"), flag: flagOf("/logs/flag"), wantDir: filepath.Clean("/logs/flag"), wantExplicit: true},
		"empty flag disables":       {env: setEnv("/logs/env"), flag: flagOf(""), wantExplicit: true},
		"relative flag is resolved": {env: unset, flag: flagOf("logs"), wantDir: filepath.Join(cwd, "logs"), wantExplicit: true},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tc.env(t)
			dir, explicit, err := lmStudioLogDir(tc.flag)
			require.NoError(t, err)
			assert.Equal(t, tc.wantDir, dir)
			assert.Equal(t, tc.wantExplicit, explicit)
		})
	}
}

// A file the log directory lists converts with the listed identifiers,
// reading no other log (requirement 3.4).
func TestLMStudioFile_InRootIDsMatchListing(t *testing.T) {
	root, file := lmRoot(t, lmLog)
	writeFile(t, filepath.Join(root, "2026-11", "2026-11-02.1.log"), lmRecord("2026-11-02 09:00:00", "thread-a"))
	listed := listedIDs(t, root)
	require.Len(t, listed, 2, "one thread and one singleton")

	stdout, _, err := runCLI(t, &Config{Input: file, Format: "shared-json", LMStudioLogDir: &root}, "")
	require.NoError(t, err)
	doc := decodeSession(t, stdout)
	assert.Equal(t, "lmstudio", doc.Session.Source)
	assert.Empty(t, doc.Session.ID, "a log holds several sessions, so the document is none of them")
	ids, sessionIDs := requestIDs(doc)
	require.Len(t, ids, 2)
	assert.ElementsMatch(t, listed, slices.Compact(slices.Sorted(slices.Values(sessionIDs))))
	assert.Equal(t, ids[1], sessionIDs[1], "a singleton's session is its request")
	for _, entry := range doc.Entries {
		if entry.RequestID != "" {
			assert.Contains(t, ids, entry.RequestID, "entries name final request identifiers")
		}
	}

	out, _, err := runCLI(t, &Config{AllProjects: true, Source: "lmstudio", Input: ids[1], Format: "shared-json", LMStudioLogDir: &root}, "")
	require.NoError(t, err)
	assert.Equal(t, ids[1], decodeSession(t, out).Session.ID, "the singleton's identifier opens it")
}

// Outside the log directory and on stdin the identifiers are reproducible
// but input-scoped: they are not the listed ones and do not open.
func TestLMStudioInput_OutOfRootIDsAreInputScoped(t *testing.T) {
	root, file := lmRoot(t, lmLog)
	copied := writeFile(t, filepath.Join(t.TempDir(), "copy.log"), lmLog)
	listed := listedIDs(t, root)

	convert := func(cfg *Config, stdin string) []string {
		t.Helper()
		cfg.Format, cfg.LMStudioLogDir = "shared-json", &root
		stdout, _, err := runCLI(t, cfg, stdin)
		require.NoError(t, err)
		ids, _ := requestIDs(decodeSession(t, stdout))
		require.Len(t, ids, 2)
		return ids
	}
	inRoot := convert(&Config{Input: file}, "")
	for name, ids := range map[string][]string{
		"copied file": convert(&Config{Input: copied}, ""),
		"stdin":       convert(&Config{}, lmLog),
	} {
		t.Run(name, func(t *testing.T) {
			assert.NotEqual(t, inRoot, ids)
			assert.NotContains(t, listed, ids[1])
			_, _, err := runCLI(t, &Config{AllProjects: true, Source: "lmstudio", Input: ids[1], Format: "md", LMStudioLogDir: &root}, "")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "session not found")
		})
	}
	assert.Equal(t, convert(&Config{}, lmLog), convert(&Config{}, lmLog), "stdin identifiers are reproducible")
}

// A direct view's request meta lines name the identifiers that open the
// listed logical session and, inside a thread, the request (requirement 1.4).
func TestLMStudioFile_MetaLineShowsListedIDs(t *testing.T) {
	root, file := lmRoot(t, lmLog)
	stdout, _, err := runCLI(t, &Config{Input: file, Format: "shared-json", LMStudioLogDir: &root}, "")
	require.NoError(t, err)
	ids, sessionIDs := requestIDs(decodeSession(t, stdout))
	require.Len(t, ids, 2)

	md, _, err := runCLI(t, &Config{Input: file, Format: "md", LMStudioLogDir: &root}, "")
	require.NoError(t, err)
	assert.Contains(t, md, "session `"+sessionIDs[0]+"` · request `"+ids[0]+"`")
	assert.Contains(t, md, "session `"+ids[1]+"`")
	assert.NotContains(t, md, "request `"+ids[1]+"`", "a singleton's session is its request")
}

// Record warnings name the file or stdin they came from, as well as their
// lines; the time-zone warning has neither (requirement 6.2).
func TestLMStudioInput_WarningsNameTheInput(t *testing.T) {
	log := `[2026-11-01 10:00:00][INFO] Received request: POST to /v1/chat/completions with body {not json}` + "\n" +
		lmRecord("2026-11-01 10:00:01", "")
	_, file := lmRoot(t, log)
	t.Setenv("TZ", "Example/Nowhere")

	for name, tc := range map[string]struct {
		cfg   Config
		stdin string
		label string
	}{
		"file":  {cfg: Config{Input: file}, label: file},
		"stdin": {stdin: log, label: "stdin"},
	} {
		t.Run(name, func(t *testing.T) {
			cfg := tc.cfg
			cfg.Format = "md"
			_, stderr, err := runCLI(t, &cfg, tc.stdin)
			require.NoError(t, err)
			assert.Contains(t, stderr, "Warning: line 1: "+tc.label+": request body is not valid JSON (line 1)")
			assert.Contains(t, stderr, `Warning: TZ="Example/Nowhere" could not be loaded`, "the zone warning names no input")

			cfg = tc.cfg
			cfg.Format = "shared-json"
			stdout, _, err := runCLI(t, &cfg, tc.stdin)
			require.NoError(t, err)
			var messages []string
			for _, w := range decodeSession(t, stdout).Warnings {
				messages = append(messages, w.Message)
			}
			assert.Contains(t, messages, tc.label+": request body is not valid JSON (line 1)")
		})
	}
}

func TestLMStudioFile_ForcedFormat(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "internal", "transcript", "testdata", "lmstudio", "notice_only.log"))
	require.NoError(t, err)
	_, file := lmRoot(t, string(data))

	_, _, err = runCLI(t, &Config{Input: file, Format: "md"}, "")
	require.Error(t, err, "headers alone do not identify the format")
	assert.Contains(t, err.Error(), "-a lmstudio")

	stdout, _, err := runCLI(t, &Config{Input: file, Format: "shared-json", Agent: "lmstudio"}, "")
	require.NoError(t, err)
	assert.Equal(t, "lmstudio", decodeSession(t, stdout).Session.Source)
}

// Follow mode and raw JSON reject LM Studio input however it is chosen
// (requirement 8.1).
func TestLMStudioUnsupportedModes(t *testing.T) {
	root, file := lmRoot(t, lmLog)
	ids := listedIDs(t, root)

	tests := map[string]struct {
		cfg   Config
		stdin string
		want  error
	}{
		"follow forced format":     {cfg: Config{Input: file, Follow: true, Agent: "lmstudio"}, want: errFollowLMStudio},
		"follow detected file":     {cfg: Config{Input: file, Follow: true}, want: errFollowLMStudio},
		"follow source identifier": {cfg: Config{AllProjects: true, Source: "lmstudio", Input: ids[0], Follow: true}, want: errFollowLMStudio},
		"follow bare identifier":   {cfg: Config{AllProjects: true, Input: ids[0], Follow: true}, want: errFollowLMStudio},
		"raw JSON forced format":   {cfg: Config{Input: file, Format: "json", Agent: "lmstudio"}, want: errRawJSONLMStudio},
		"raw JSON detected file":   {cfg: Config{Input: file, Format: "json"}, want: errRawJSONLMStudio},
		"raw JSON stdin":           {cfg: Config{Format: "json"}, stdin: lmLog, want: errRawJSONLMStudio},
		"raw JSON identifier":      {cfg: Config{AllProjects: true, Source: "lmstudio", Input: ids[0], Format: "json"}, want: errRawJSONLMStudio},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := tc.cfg
			cfg.LMStudioLogDir = &root
			if cfg.Format == "" {
				cfg.Format = "md"
			}
			if cfg.Follow {
				_, err := run(&cfg)
				require.ErrorIs(t, err, tc.want)
				return
			}
			_, _, err := runCLI(t, &cfg, tc.stdin)
			require.ErrorIs(t, err, tc.want)
		})
	}
	assert.Contains(t, errRawJSONLMStudio.Error(), "-f shared-json", "raw JSON users are sent to shared JSON")
}

// latest can pick an LM Studio session; an unsupported mode then names it
// and the remedies, without choosing an older session (requirement 7.6).
func TestLMStudioLatestUnsupportedModes(t *testing.T) {
	root, _ := lmRoot(t, lmLog)
	ids := listedIDs(t, root)

	for name, tc := range map[string]struct {
		cfg  Config
		want error
	}{
		"follow":   {cfg: Config{Follow: true, Format: "md"}, want: errFollowLMStudio},
		"raw JSON": {cfg: Config{Format: "json"}, want: errRawJSONLMStudio},
	} {
		t.Run(name, func(t *testing.T) {
			cfg := tc.cfg
			cfg.AllProjects, cfg.Input, cfg.LMStudioLogDir = true, "latest", &root
			_, err := run(&cfg)
			require.ErrorIs(t, err, tc.want)
			assert.Contains(t, err.Error(), "LM Studio session "+ids[len(ids)-1])
			assert.Contains(t, err.Error(), "--lmstudio-log-dir ''")
			assert.Contains(t, err.Error(), "open a session explicitly")
		})
	}

	stdout, _, err := runCLI(t, &Config{AllProjects: true, Input: "latest", Format: "md", LMStudioLogDir: &root}, "")
	require.NoError(t, err, "Markdown shows the session")
	assert.Contains(t, stdout, "singleton session")
}

func TestLMStudioTextListing(t *testing.T) {
	root, _ := lmRoot(t, lmRecord("2026-02-30 10:00:00", "thread-a")+lmRecord("2026-11-01 10:00:00", "thread-a"))

	stdout, _, err := runCLIStdout(t, &Config{List: true, AllProjects: true, Format: "md", LMStudioLogDir: &root})
	require.NoError(t, err)
	assert.Contains(t, stdout, "[lmstudio]\tlmstudio-thread-")
	assert.Contains(t, stdout, "\ttime unknown\t2 requests\t-", "an invalid first wall time leaves the time unknown")

	empty := ""
	stdout, _, err = runCLIStdout(t, &Config{List: true, AllProjects: true, Format: "md", LMStudioLogDir: &empty})
	require.NoError(t, err)
	assert.NotContains(t, stdout, "lmstudio", "an empty directory disables the source")

	stdout, _, err = runCLIStdout(t, &Config{List: true, Format: "md", LMStudioLogDir: &root})
	require.NoError(t, err)
	assert.NotContains(t, stdout, "lmstudio", "project scope never lists the source")
}

func TestLMStudioMissingRootWarning(t *testing.T) {
	fakeHome(t)
	missing := filepath.Join(t.TempDir(), "missing")

	_, stderr, err := runCLIStdout(t, &Config{List: true, AllProjects: true, Format: "md", LMStudioLogDir: &missing})
	require.NoError(t, err)
	assert.Contains(t, stderr, "Warning: could not list lmstudio sessions: LM Studio log directory "+missing)

	_, stderr, err = runCLIStdout(t, &Config{List: true, AllProjects: true, Format: "md"})
	require.NoError(t, err)
	assert.NotContains(t, stderr, "lmstudio", "the absent default directory is silent")
}

// Help names what server logs expose and how they grow listings and affect
// latest (requirements 7.5, 7.6).
func TestLMStudioHelp(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err)
	old := os.Stderr
	os.Stderr = w
	printUsage()
	os.Stderr = old
	require.NoError(t, w.Close())
	data, err := io.ReadAll(r)
	require.NoError(t, err)
	help := strings.Join(strings.Fields(string(data)), " ")

	for _, want := range []string{
		"full prompts, request parameters including credential-bearing integration headers, and traffic from every client of the LM Studio server",
		"listing rows grow one-for-one with unthreaded requests",
		"--all-projects latest can pick an LM Studio session, which -F and -f json cannot show",
		"--lmstudio-log-dir '' or open another session explicitly",
		"for other files and stdin they are input-scoped",
		"--source <source> Open the argument as a session ID of this source: claude, codex, codex-exec, copilot, kiro-cli, kiro-ide, lmstudio",
	} {
		assert.Contains(t, help, want)
	}
}
