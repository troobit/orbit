package sessions

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arjenschwarz/orbit/internal/transcript"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// lmReq is a single-line LM Studio request record. An empty thread records
// no client thread ID.
func lmReq(wall, thread, model string) string {
	meta := ""
	if thread != "" {
		meta = fmt.Sprintf(`,"client_metadata":{"thread_id":%q}`, thread)
	}
	return fmt.Sprintf(`[%s][INFO] Received request: POST to /v1/chat/completions with body {"model":%q,"messages":[{"role":"user","content":"hi"}]%s}`+"\n",
		wall, model, meta)
}

// newLMTree is an exec tree with TZ set to UTC, so request times interpret.
func newLMTree(t *testing.T) *execTree {
	t.Helper()
	t.Setenv("TZ", "")
	return newExecTree(t)
}

// lmLister lists only the LM Studio source, in all-projects scope.
func lmLister(t *testing.T, root string, explicit bool) *Lister {
	t.Helper()
	l, err := NewLister(WithLMStudioLogDir(root, explicit))
	require.NoError(t, err)
	return l
}

func listLM(t *testing.T, l *Lister) ([]SessionInfo, []ListWarning) {
	t.Helper()
	return l.listLMStudio("")
}

// lmReads records the files LM Studio listing opens and fingerprints.
type lmReads struct {
	mu           sync.Mutex
	opened       []string
	fingerprints int
}

func (r *lmReads) opens() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.opened)
}

func (r *lmReads) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.opened, r.fingerprints = nil, 0
}

// countLMReads replaces lmOpen and fingerprintFile with counting wrappers.
func countLMReads(t testing.TB) *lmReads {
	t.Helper()
	reads := &lmReads{}
	open, fingerprint := lmOpen, fingerprintFile
	lmOpen = func(path string) (io.ReadCloser, error) {
		reads.mu.Lock()
		reads.opened = append(reads.opened, path)
		reads.mu.Unlock()
		return open(path)
	}
	fingerprintFile = func(path string) (fileFingerprint, bool, error) {
		reads.mu.Lock()
		reads.fingerprints++
		reads.mu.Unlock()
		return fingerprint(path)
	}
	t.Cleanup(func() { lmOpen, fingerprintFile = open, fingerprint })
	return reads
}

func lmKeys(files []lmLogFile) []string {
	keys := make([]string, 0, len(files))
	for _, f := range files {
		keys = append(keys, f.key)
	}
	return keys
}

func lmWarningCodes(warnings []ListWarning) []string {
	codes := make([]string, 0, len(warnings))
	for _, w := range warnings {
		codes = append(codes, w.Code)
	}
	slices.Sort(codes)
	return codes
}

func TestFindLMStudioFiles_NamesAndOrder(t *testing.T) {
	tree := newLMTree(t)
	for _, key := range []string{
		"2026-11/2026-11-10.10.log",
		"2026-11/2026-11-10.2.log",
		"2026-11/2026-11-09.1.log",
		"2026-10/2026-10-31.1.log",
		"2026-11/2026-11-10.log",   // no suffix
		"2026-11/2026-11-10.1.txt", // other extension
		"2026-11/notes.log",
		"2026-1/2026-01-01.1.log", // not a month directory
		"logs/2026-11-10.1.log",
		"2026-11-10.1.log", // not inside a month directory
		"2026-11/nested/2026-11-10.1.log",
	} {
		tree.write("x\n", "root", filepath.FromSlash(key))
	}
	tree.mkdir("root", "2026-11", "2026-11-10.3.log") // a directory with a log name

	files, warnings, err := findLMStudioFiles(tree.path("root"))
	require.NoError(t, err)
	assert.Empty(t, warnings)
	assert.Equal(t, []string{
		"2026-10/2026-10-31.1.log",
		"2026-11/2026-11-09.1.log",
		"2026-11/2026-11-10.2.log",
		"2026-11/2026-11-10.10.log",
	}, lmKeys(files), "filename date, then numeric suffix")
	assert.Equal(t, tree.path("root", "2026-11", "2026-11-10.10.log"), files[3].path)
}

func TestFindLMStudioFiles_Symlinks(t *testing.T) {
	tree := newLMTree(t)
	target := tree.write("x\n", "root", "2026-11", "2026-11-10.2.log")
	tree.write("x\n", "root", "archive", "2026-11-09.1.log")
	tree.write("x\n", "outside", "2026-11-08.1.log")
	tree.write("x\n", "outside", "2026-12", "2026-12-01.1.log")
	link := func(target string, parts ...string) {
		require.NoError(t, os.Symlink(target, tree.path(parts...)))
	}
	link(target, "root", "2026-11", "2026-11-10.1.log")                                           // second name for a root file
	link(tree.path("root", "archive", "2026-11-09.1.log"), "root", "2026-11", "2026-11-09.1.log") // into a non-month directory of the root
	link(tree.path("outside", "2026-11-08.1.log"), "root", "2026-11", "2026-11-08.1.log")         // escapes the root
	link(tree.path("root", "2026-11", "missing.log"), "root", "2026-11", "2026-11-07.1.log")      // dangling
	link(tree.path("outside", "2026-12"), "root", "2026-12")                                      // directory symlink
	link(tree.path("root"), "rootlink")

	for name, root := range map[string]string{"root": tree.path("root"), "symlinked root": tree.path("rootlink")} {
		t.Run(name, func(t *testing.T) {
			canonical, err := lmStudioRoot(root)
			require.NoError(t, err)
			files, warnings, err := findLMStudioFiles(canonical)
			require.NoError(t, err)
			assert.Equal(t, []string{WarningPathUnreadable}, lmWarningCodes(warnings), "the dangling link")
			assert.Equal(t, []string{"archive/2026-11-09.1.log", "2026-11/2026-11-10.2.log"}, lmKeys(files),
				"a file symlink joins as its target, keyed by the target's path and listed once; escaping links and directory links are not followed")
			assert.Equal(t, "2026-11-10", files[1].date)
			assert.Equal(t, 1, files[1].seq, "ordered by the earliest name the file was found under")
		})
	}
}

func TestListLMStudio_Grouping(t *testing.T) {
	tree := newLMTree(t)
	first := lmReq("2026-11-10 09:00:00", "", "model-a") +
		lmReq("2026-11-10 09:01:00", "thread-alpha", "") +
		"[2026-11-10 09:01:30][INFO] Server started.\n" +
		lmReq("2026-11-10 09:02:00", "thread-alpha", "model-b")
	second := lmReq("2026-11-10 10:00:00", "thread-alpha", "model-c") +
		lmReq("2026-11-10 10:01:00", "<Truncated in logs>", "model-d")
	tree.write(second, "root", "2026-11", "2026-11-10.2.log")
	tree.write(first, "root", "2026-11", "2026-11-10.1.log")

	rows, warnings := listLM(t, lmLister(t, tree.path("root"), true))
	assert.Empty(t, warnings)
	firstKey, secondKey := "2026-11/2026-11-10.1.log", "2026-11/2026-11-10.2.log"
	lines := strings.SplitAfter(first, "\n")
	threadSize := int64(len(lines[1]) + len(lines[3]) + len(strings.SplitAfter(second, "\n")[0]))
	assert.Equal(t, []SessionInfo{
		{
			ID: lmRequestID(firstKey, 1), CreatedAt: time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC),
			Size: int64(len(lines[0])), Source: SourceLMStudio, Model: "model-a", Provider: "lmstudio", RequestCount: 1, TimeZone: "UTC",
		},
		{
			ID: lmThreadSessionID("thread-alpha"), CreatedAt: time.Date(2026, 11, 10, 9, 1, 0, 0, time.UTC),
			Size: threadSize, Source: SourceLMStudio, Model: "model-b", Provider: "lmstudio", ThreadID: "thread-alpha", RequestCount: 3, TimeZone: "UTC",
		},
		{
			ID: lmRequestID(secondKey, 2), CreatedAt: time.Date(2026, 11, 10, 10, 1, 0, 0, time.UTC),
			Size: int64(len(strings.SplitAfter(second, "\n")[1])), Source: SourceLMStudio, Model: "model-d", Provider: "lmstudio", RequestCount: 1, TimeZone: "UTC",
		},
	}, rows, "a thread groups across files with its first-known model; a truncated thread ID is a singleton; size counts only the session's records; every row names the assumed zone")
}

func TestListLMStudio_CreatedAtFromFirstRequestOnly(t *testing.T) {
	tree := newLMTree(t)
	t.Setenv("TZ", "America/New_York")
	// 02:30 does not exist on 2026-03-08 in New York; the later request's
	// time must not stand in for it (requirement 5.4).
	tree.write(lmReq("2026-03-08 02:30:00", "thread-alpha", "m")+lmReq("2026-03-08 04:00:00", "thread-alpha", "m"),
		"root", "2026-03", "2026-03-08.1.log")

	rows, warnings := listLM(t, lmLister(t, tree.path("root"), true))
	assert.Empty(t, warnings)
	require.Len(t, rows, 1)
	assert.True(t, rows[0].CreatedAt.IsZero())
	assert.False(t, rows[0].TimeFromFilesystem)
}

func TestListLMStudio_InvalidTimeZone(t *testing.T) {
	tree := newLMTree(t)
	t.Setenv("TZ", "Not/AZone")
	tree.write(lmReq("2026-11-10 09:00:00", "", "m"), "root", "2026-11", "2026-11-10.1.log")
	tree.write(lmReq("2026-11-11 09:00:00", "", "m"), "root", "2026-11", "2026-11-11.1.log")

	rows, warnings := listLM(t, lmLister(t, tree.path("root"), true))
	require.Len(t, rows, 2)
	assert.True(t, rows[0].CreatedAt.IsZero(), "no other zone is substituted")
	assert.Equal(t, []string{transcript.WarningTimeZone}, lmWarningCodes(warnings), "once per pass, not per file")
}

func TestListLMStudio_RootStates(t *testing.T) {
	tree := newLMTree(t)
	file := tree.write("x\n", "file")
	tests := map[string]struct {
		dir      string
		explicit bool
		want     []string
	}{
		"default root absent":   {dir: tree.path("missing"), want: []string{}},
		"explicit root absent":  {dir: tree.path("missing"), explicit: true, want: []string{WarningRootUnreadable}},
		"explicit root is file": {dir: file, explicit: true, want: []string{WarningRootUnreadable}},
		"default root is file":  {dir: file, want: []string{}},
		"disabled":              {dir: "", explicit: true, want: []string{}},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			rows, warnings := listLM(t, lmLister(t, tc.dir, tc.explicit))
			assert.Empty(t, rows)
			assert.Equal(t, tc.want, lmWarningCodes(warnings))
		})
	}
}

func TestListLMStudio_FileWarnings(t *testing.T) {
	tree := newLMTree(t)
	tree.write("[2026-11-10 09:00:00][INFO] Server started.\n[2026-11-10 09:00:01][INFO] Loaded model.\n", "root", "2026-11", "2026-11-10.1.log")
	tree.write("not a log\nat all\n", "root", "2026-11", "2026-11-10.2.log")
	tree.write("", "root", "2026-11", "2026-11-10.3.log")
	tree.write(lmReq("2026-11-10 09:00:00", "", "m")+`[2026-11-10 09:00:01][INFO] Received request: POST to /v1/chat/completions with body {"model":`+"\n"+
		"[2026-11-10 09:00:02][INFO] next\n", "root", "2026-11", "2026-11-10.4.log")

	rows, warnings := listLM(t, lmLister(t, tree.path("root"), true))
	assert.Len(t, rows, 2, "the malformed request is still listed")
	require.Equal(t, []string{transcript.WarningUnparsedText}, lmWarningCodes(warnings),
		"only the non-empty file without records warns; notice-only and empty files are silent, and record warnings stay with the transcript")
	assert.Equal(t, "2026-11/2026-11-10.2.log: no LM Studio log records found (lines 1-2)", warnings[0].Err.Error())
}

func TestListLMStudio_UnreadableFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permissions do not restrict root")
	}
	tree := newLMTree(t)
	tree.write(lmReq("2026-11-10 09:00:00", "", "m"), "root", "2026-11", "2026-11-10.1.log")
	locked := tree.write(lmReq("2026-11-10 09:00:00", "", "m"), "root", "2026-11", "2026-11-10.2.log")
	dir := tree.mkdir("root", "2026-12")
	require.NoError(t, os.Chmod(locked, 0))
	require.NoError(t, os.Chmod(dir, 0))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755); _ = os.Chmod(locked, 0o600) })

	rows, warnings := listLM(t, lmLister(t, tree.path("root"), false))
	assert.Len(t, rows, 1, "other files remain")
	assert.Equal(t, []string{WarningPathUnreadable, WarningPathUnreadable}, lmWarningCodes(warnings),
		"unreadable files and month directories warn even under the default root")
}

func TestListAll_LMStudioProjectScopeReadsNothing(t *testing.T) {
	tree := newLMTree(t)
	tree.write(lmReq("2026-11-10 09:00:00", "", "m"), "root", "2026-11", "2026-11-10.1.log")
	reads := countLMReads(t)

	for name, dir := range map[string]string{"populated root": tree.path("root"), "missing root": tree.path("missing")} {
		t.Run(name, func(t *testing.T) {
			reads.reset()
			l := lmLister(t, dir, true)
			list, warnings, err := l.ListAll(tree.mkdir("project"))
			require.NoError(t, err)
			for _, s := range list {
				assert.NotEqual(t, SourceLMStudio, s.Source)
			}
			for _, w := range warnings {
				assert.NotEqual(t, SourceLMStudio, w.Source)
			}
			assert.Empty(t, reads.opens())
			assert.Zero(t, reads.fingerprints)
		})
	}

	reads.reset()
	list, _, err := lmLister(t, tree.path("root"), true).ListAll("")
	require.NoError(t, err)
	assert.True(t, slices.ContainsFunc(list, func(s SessionInfo) bool { return s.Source == SourceLMStudio }),
		"all-projects scope lists the source")
}

func TestLister_SourcesWithLMStudio(t *testing.T) {
	newLMTree(t)
	assert.NotContains(t, lmLister(t, "", false).Sources(), SourceLMStudio)
	with := lmLister(t, "/somewhere", false)
	assert.Contains(t, with.Sources(), SourceLMStudio)
	assert.False(t, with.opts.ExtendedOrdering, "the default-on root leaves project-scope ordering alone")
	assert.True(t, IsValidSource(SourceLMStudio))
}

// requireUnixCache skips tests of the zero-read refresh on platforms whose
// fingerprint is never cacheable (Q20).
func requireUnixCache(t testing.TB) {
	t.Helper()
	if runtime.GOOS == "windows" || runtime.GOOS == "plan9" || runtime.GOOS == "js" || runtime.GOOS == "wasip1" {
		t.Skip("the LM Studio cache is Unix-only")
	}
}

func TestListLMStudio_CacheRefresh(t *testing.T) {
	requireUnixCache(t)
	tree := newLMTree(t)
	a := tree.write(lmReq("2026-11-10 09:00:00", "", "m"), "root", "2026-11", "2026-11-10.1.log")
	b := tree.write(lmReq("2026-11-10 10:00:00", "thread-b1", "m"), "root", "2026-11", "2026-11-10.2.log")
	c := tree.write(lmReq("2026-11-10 11:00:00", "thread-c1", "m"), "root", "2026-11", "2026-11-10.3.log")
	reads := countLMReads(t)
	l := lmLister(t, tree.path("root"), true)
	refresh := func() []string {
		t.Helper()
		reads.reset()
		rows, warnings := listLM(t, l)
		require.Empty(t, warnings)
		return rowIDs(rows)
	}
	key := func(path string) string {
		rel, err := filepath.Rel(tree.path("root"), path)
		require.NoError(t, err)
		return filepath.ToSlash(rel)
	}

	initial := refresh()
	assert.Len(t, reads.opens(), 3)

	assert.Equal(t, initial, refresh())
	assert.Empty(t, reads.opens(), "an unchanged tree reads no log bodies")

	f, err := os.OpenFile(a, os.O_APPEND|os.O_WRONLY, 0)
	require.NoError(t, err)
	_, err = f.WriteString(lmReq("2026-11-10 09:30:00", "", "m"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	assert.Contains(t, refresh(), lmRequestID(key(a), 2), "an append is seen")
	assert.Equal(t, []string{a}, reads.opens(), "only the changed file is read")

	// Same size and modification time: only the inode and change time differ.
	info, err := os.Stat(b)
	require.NoError(t, err)
	replacement := tree.write(lmReq("2026-11-10 10:00:00", "thread-b2", "m"), "root", "replacement.tmp")
	require.NoError(t, os.Chtimes(replacement, info.ModTime(), info.ModTime()))
	require.NoError(t, os.Rename(replacement, b))
	ids := refresh()
	assert.Contains(t, ids, lmThreadSessionID("thread-b2"), "a same-size replacement keeping the modification time is seen")
	assert.NotContains(t, ids, lmThreadSessionID("thread-b1"))
	assert.Equal(t, []string{b}, reads.opens())

	// Same inode, size and modification time: only the change time differs.
	info, err = os.Stat(c)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(c, []byte(lmReq("2026-11-10 11:00:00", "thread-c2", "m")), 0o600))
	require.NoError(t, os.Chtimes(c, info.ModTime(), info.ModTime()))
	ids = refresh()
	assert.Contains(t, ids, lmThreadSessionID("thread-c2"), "an in-place rewrite keeping size and modification time is seen")
	assert.Equal(t, []string{c}, reads.opens())

	require.NoError(t, os.Truncate(c, 0))
	assert.NotContains(t, refresh(), lmThreadSessionID("thread-c2"), "a truncated file's sessions disappear")

	require.NoError(t, os.Remove(a))
	ids = refresh()
	assert.NotContains(t, ids, lmRequestID(key(a), 1), "a removed file's sessions disappear")
	assert.Empty(t, reads.opens())
	l.lmCache.mu.Lock()
	assert.NotContains(t, l.lmCache.entries, a, "a removed file leaves the cache")
	l.lmCache.mu.Unlock()
}

func TestListLMStudio_IncompleteRequestCompletes(t *testing.T) {
	tree := newLMTree(t)
	path := tree.write("[2026-11-10 09:00:00][INFO] Received request: POST to /v1/chat/completions with body {\n  \"model\": \"m\",\n",
		"root", "2026-11", "2026-11-10.1.log")
	l := lmLister(t, tree.path("root"), true)
	reqID := lmRequestID("2026-11/2026-11-10.1.log", 1)

	rows, _ := listLM(t, l)
	assert.Equal(t, []string{reqID}, rowIDs(rows), "an incomplete body has no visible thread yet")

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	require.NoError(t, err)
	_, err = f.WriteString("  \"messages\": [{\"role\": \"user\", \"content\": \"hi\"}],\n  \"client_metadata\": {\"thread_id\": \"thread-alpha\"}\n}\n")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	root, err := lmStudioRoot(tree.path("root"))
	require.NoError(t, err)
	idx, _, err := buildLMStudioIndex(root, l.lmCache)
	require.NoError(t, err)
	assert.Equal(t, []string{lmThreadSessionID("thread-alpha")}, idx.order, "the singleton row gives way to the thread")
	assert.Equal(t, reqID, idx.sessions[idx.order[0]][0].requests[0].id, "the request keeps its identifier")
}

func TestLMCache_NotCacheableNeverHits(t *testing.T) {
	tree := newLMTree(t)
	tree.write(lmReq("2026-11-10 09:00:00", "", "m"), "root", "2026-11", "2026-11-10.1.log")
	tree.write(lmReq("2026-11-10 10:00:00", "", "m"), "root", "2026-11", "2026-11-10.2.log")
	reads := countLMReads(t)
	// Identical fingerprints on every call: only the flag may stop a hit.
	fingerprintFile = func(string) (fileFingerprint, bool, error) { return fileFingerprint{}, false, nil }
	l := lmLister(t, tree.path("root"), true)

	for range 2 {
		reads.reset()
		rows, _ := listLM(t, l)
		assert.Len(t, rows, 2)
		assert.Len(t, reads.opens(), 2, "every refresh reads every file")
	}
}

func TestBuildLMStudioIndex_OverlappingPasses(t *testing.T) {
	tree := newLMTree(t)
	for i := range 6 {
		tree.write(lmReq("2026-11-10 09:00:00", fmt.Sprintf("thread-%d", i%3), "m"), "root", "2026-11", fmt.Sprintf("2026-11-10.%d.log", i+1))
	}
	root, err := lmStudioRoot(tree.path("root"))
	require.NoError(t, err)
	cache := &lmCache{}

	var wg sync.WaitGroup
	results := make([][]string, 8)
	for i := range results {
		wg.Go(func() {
			idx, _, err := buildLMStudioIndex(root, cache)
			if assert.NoError(t, err) {
				results[i] = slices.Sorted(slices.Values(idx.order))
			}
		})
	}
	wg.Wait()
	for _, got := range results {
		assert.Equal(t, results[0], got)
	}
	assert.Len(t, results[0], 3)
}

// BenchmarkListLMStudio lists a generated root of 30 files holding 30,000
// unthreaded requests (one listing row each) plus 3,000 threaded ones in 10
// threads: a cold pass with an empty cache, and a refresh of the unchanged
// tree, which must read no log body (requirements 7.3, 8.5).
// BenchmarkLMStudioListFragment in apsisweb records the rendered list size.
func BenchmarkListLMStudio(b *testing.B) {
	b.Setenv("TZ", "")
	root, err := filepath.EvalSymlinks(b.TempDir())
	require.NoError(b, err)
	b.Setenv("HOME", filepath.Join(root, "home"))
	var inputBytes int64
	requests := 0
	for f := range 30 {
		var sb strings.Builder
		for i := range 1100 {
			thread := ""
			if i%11 == 10 {
				thread = fmt.Sprintf("thread-%d", (f+i)%10)
			}
			sb.WriteString(lmReq(fmt.Sprintf("2026-11-%02d 09:%02d:%02d", 1+f%28, i/60%60, i%60), thread, "example-model"))
			requests++
		}
		inputBytes += int64(sb.Len())
		dir := filepath.Join(root, "logs", "2026-11")
		require.NoError(b, os.MkdirAll(dir, 0o755))
		require.NoError(b, os.WriteFile(filepath.Join(dir, fmt.Sprintf("2026-11-%02d.%d.log", 1+f%28, f/28+1)), []byte(sb.String()), 0o600))
	}
	report := func(b *testing.B, rows int) {
		b.ReportMetric(float64(inputBytes), "input-bytes")
		b.ReportMetric(float64(requests), "requests")
		b.ReportMetric(float64(rows), "rows")
	}

	b.Run("cold", func(b *testing.B) {
		rows := 0
		for b.Loop() {
			l, err := NewLister(WithLMStudioLogDir(filepath.Join(root, "logs"), true))
			require.NoError(b, err)
			list, _ := l.listLMStudio("")
			rows = len(list)
		}
		report(b, rows)
	})
	b.Run("unchanged refresh", func(b *testing.B) {
		requireUnixCache(b)
		l, err := NewLister(WithLMStudioLogDir(filepath.Join(root, "logs"), true))
		require.NoError(b, err)
		list, _ := l.listLMStudio("")
		reads := countLMReads(b)
		for b.Loop() {
			list, _ = l.listLMStudio("")
		}
		if opened := reads.opens(); len(opened) != 0 {
			b.Fatalf("unchanged refresh read %d log bodies, want 0", len(opened))
		}
		report(b, len(list))
	})
}
