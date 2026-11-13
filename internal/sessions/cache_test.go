package sessions

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"github.com/arjenschwarz/orbit/internal/agents/kiro/logs"
)

var (
	cacheTime  = time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	cacheTime2 = time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC) // same length when formatted
)

// --- fingerprint and cached() ---

// counting returns a compute function that counts its calls.
func counting(calls *int, value string) func() (string, error) {
	return func() (string, error) {
		*calls++
		return value, nil
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

func TestCached_UnchangedFileHits(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a")
	writeFile(t, path, "one")
	c := &fileCache{}
	var calls int
	for range 3 {
		c.beginPass()
		got, err := cached(c, "k", []string{path}, counting(&calls, "v"))
		require.NoError(t, err)
		assert.Equal(t, "v", got)
		c.endPass()
	}
	assert.Equal(t, 1, calls, "only the first pass extracts")
}

func TestCached_ChangeDetection(t *testing.T) {
	old := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	tests := map[string]func(t *testing.T, path string){
		"append": func(t *testing.T, path string) {
			f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
			require.NoError(t, err)
			_, err = f.WriteString("x")
			require.NoError(t, err)
			require.NoError(t, f.Close())
		},
		"equal-size in-place rewrite with the old mtime restored": func(t *testing.T, path string) {
			writeFile(t, path, "two")
			require.NoError(t, os.Chtimes(path, old, old))
		},
		"equal-size replacement renamed over the file with the old mtime restored": func(t *testing.T, path string) {
			tmp := path + ".new"
			writeFile(t, tmp, "two")
			require.NoError(t, os.Chtimes(tmp, old, old))
			require.NoError(t, os.Rename(tmp, path))
		},
	}
	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "a")
			writeFile(t, path, "one")
			require.NoError(t, os.Chtimes(path, old, old))
			c := &fileCache{}
			var calls int
			run := func() {
				c.beginPass()
				_, err := cached(c, "k", []string{path}, counting(&calls, "v"))
				require.NoError(t, err)
				c.endPass()
			}
			run()
			run()
			require.Equal(t, 1, calls)
			change(t, path)
			run()
			assert.Equal(t, 2, calls, "the change invalidates the entry")
			run()
			assert.Equal(t, 2, calls, "and the new state is cached again")
		})
	}
}

func TestCached_CompanionPresenceChangesTheKey(t *testing.T) {
	dir := t.TempDir()
	main, companion := filepath.Join(dir, "main"), filepath.Join(dir, "main-wal")
	writeFile(t, main, "db")
	c := &fileCache{}
	var calls int
	run := func() {
		c.beginPass()
		_, err := cached(c, "k", []string{main, companion}, counting(&calls, "v"))
		require.NoError(t, err)
		c.endPass()
	}
	run()
	run()
	require.Equal(t, 1, calls, "an absent companion is part of the cached key")

	writeFile(t, companion, "wal")
	run()
	assert.Equal(t, 2, calls, "creating the companion changes the key")
	run()
	require.Equal(t, 2, calls)

	writeFile(t, companion, "wal2")
	run()
	assert.Equal(t, 3, calls, "changing the companion changes the key")

	require.NoError(t, os.Remove(companion))
	run()
	assert.Equal(t, 4, calls, "removing the companion changes the key")
}

func TestCached_FingerprintErrorNeverHits(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a")
	writeFile(t, path, "one")
	c := &fileCache{}
	var calls int
	run := func() {
		c.beginPass()
		_, err := cached(c, "k", []string{path}, counting(&calls, "v"))
		require.NoError(t, err)
		c.endPass()
	}
	run()
	require.Equal(t, 1, calls)

	orig := fingerprintFile
	t.Cleanup(func() { fingerprintFile = orig })
	fingerprintFile = func(string) (fileFingerprint, bool, error) {
		return fileFingerprint{}, false, fmt.Errorf("no metadata")
	}
	run()
	run()
	assert.Equal(t, 3, calls, "an unavailable fingerprint extracts every pass and stores nothing")
	assert.Empty(t, c.entries)
}

func TestCached_UncacheablePlatformNeverHits(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a")
	writeFile(t, path, "one")
	orig := fingerprintFile
	t.Cleanup(func() { fingerprintFile = orig })
	fingerprintFile = func(string) (fileFingerprint, bool, error) { return fileFingerprint{}, false, nil }
	c := &fileCache{}
	var calls int
	for range 2 {
		c.beginPass()
		_, err := cached(c, "k", []string{path}, counting(&calls, "v"))
		require.NoError(t, err)
		c.endPass()
	}
	assert.Equal(t, 2, calls, "without trusted fingerprints every pass extracts, never falling back to size/mtime")
}

func TestCached_KeyChangingDuringExtractionIsNotStored(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a")
	writeFile(t, path, "one")
	c := &fileCache{}
	var calls int
	c.beginPass()
	got, err := cached(c, "k", []string{path}, func() (string, error) {
		calls++
		writeFile(t, path, "two!") // the file changes while it is being read
		return "stale", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "stale", got, "the result is still used for this pass")
	c.endPass()
	assert.Empty(t, c.entries, "but it is not cached")

	c.beginPass()
	_, err = cached(c, "k", []string{path}, counting(&calls, "fresh"))
	require.NoError(t, err)
	assert.Equal(t, 2, calls, "and the next pass extracts again")
}

func TestCached_FailedExtractionIsNotStored(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a")
	writeFile(t, path, "one")
	c := &fileCache{}
	calls := 0
	for range 2 {
		c.beginPass()
		_, err := cached(c, "k", []string{path}, func() (string, error) {
			calls++
			return "", fmt.Errorf("boom")
		})
		require.Error(t, err)
		c.endPass()
	}
	assert.Equal(t, 2, calls)
}

func TestCached_SweepsEntriesAPassDidNotUse(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	writeFile(t, a, "a")
	writeFile(t, b, "b")
	c := &fileCache{}
	var calls int
	c.beginPass()
	_, _ = cached(c, "a", []string{a}, counting(&calls, "a"))
	_, _ = cached(c, "b", []string{b}, counting(&calls, "b"))
	c.endPass()
	require.Len(t, c.entries, 2)

	c.beginPass()
	_, _ = cached(c, "a", []string{a}, counting(&calls, "a"))
	c.endPass()
	assert.Equal(t, []string{"a"}, slices.Collect(mapsKeys(c.entries)), "the unseen entry is dropped")
}

func mapsKeys(m map[string]*cacheEntry) func(yield func(string) bool) {
	return func(yield func(string) bool) {
		for k := range m {
			if !yield(k) {
				return
			}
		}
	}
}

func TestCached_NilCacheComputes(t *testing.T) {
	var calls int
	for range 2 {
		got, err := cached[string](nil, "k", []string{"unused"}, counting(&calls, "v"))
		require.NoError(t, err)
		assert.Equal(t, "v", got)
	}
	assert.Equal(t, 2, calls)
}

// Design risk 2: each write through a WAL-mode connection changes the
// combined fingerprint of the database, -wal and -journal files, so a Kiro CLI
// database that changed is never served from cache.
func TestFingerprint_WALWritesChangeTheCombinedKey(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "data.sqlite3")
	conn, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)")
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	_, err = conn.Exec(`CREATE TABLE t (v INTEGER)`)
	require.NoError(t, err)

	c := &fileCache{}
	files := []string{dbPath, dbPath + "-wal", dbPath + "-journal"}
	key := func() []fileFingerprint {
		fps, ok := c.fingerprints(files)
		require.True(t, ok)
		return fps
	}
	before := key()
	assert.Equal(t, before, key(), "an untouched database keeps its key")

	for i := range 2 {
		_, err = conn.Exec(`INSERT INTO t VALUES (?)`, i)
		require.NoError(t, err)
		after := key()
		assert.NotEqual(t, before, after, "write %d changes the key", i+1)
		before = after
	}
}

// --- Lister integration ---

const (
	cacheClaudeID = "claude-1"
	cacheCodexID  = "11111111-1111-4111-8111-111111111111"
	cacheCopilot  = "copilot-1"
	cacheIDEExec  = "ide-exec-1"
)

// cacheFixture is a Lister over every cached source in all-projects scope,
// plus the fixture paths tests change.
type cacheFixture struct {
	t       *testing.T
	home    string
	lister  *Lister
	kiroDB  string
	chat    string
	stream  string
	claude  string
	rollout string
	copilot string

	mu           sync.Mutex
	computes     []string
	kiroCalls    int
	kiroSessions []logs.SessionMetadata
}

func newCacheFixture(t *testing.T) *cacheFixture {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	home := filepath.Join(root, "home")
	f := &cacheFixture{t: t, home: home}

	setupClaudeSession(t, home, "/work/claude", cacheClaudeID, cacheTime)
	setupCodexSession(t, home, "/work/codex", cacheCodexID, cacheTime)
	setupCopilotSession(t, home, "/work/copilot", cacheCopilot, cacheTime)
	f.claude = filepath.Join(home, ".claude", "projects", "-work-claude", cacheClaudeID+".jsonl")
	f.rollout = filepath.Join(home, ".codex", "sessions", "2026", "11", "01", "session-"+cacheCodexID+".jsonl")
	f.copilot = filepath.Join(home, ".copilot", "session-state", cacheCopilot, "workspace.yaml")

	ideBase := filepath.Join(root, "kiro-ide")
	require.NoError(t, os.MkdirAll(filepath.Join(ideBase, "ws1"), 0o755))
	f.chat = writeKiroIDEChatFile(t, filepath.Join(ideBase, "ws1"), "a.chat", cacheIDEExec, 2)

	execRoot := filepath.Join(root, "exec")
	require.NoError(t, os.MkdirAll(filepath.Join(execRoot, "repo"), 0o755))
	f.stream = filepath.Join(execRoot, "repo", "run.jsonl")
	writeFile(t, f.stream, `{"type":"thread.started","thread_id":"thread-1"}`+"\n")

	f.kiroDB = filepath.Join(root, "data.sqlite3")
	writeFile(t, f.kiroDB, "db")
	f.kiroSessions = []logs.SessionMetadata{{ConversationID: "kiro-1", Directory: "/work/kiro", CreatedAt: cacheTime, Size: 10}}

	f.lister = &Lister{
		homeDir:         home,
		opts:            Options{CodexExecRoots: []string{execRoot}, ExtendedOrdering: true},
		cache:           &fileCache{},
		kiroIDEBasePath: func() (string, error) { return ideBase, nil },
		kiroDBPath:      func() (string, error) { return f.kiroDB, nil },
		kiroDiscoverAll: func(context.Context) ([]logs.SessionMetadata, error) {
			f.kiroCalls++
			return slices.Clone(f.kiroSessions), nil
		},
	}
	f.lister.cache.onCompute = func(key string) {
		f.mu.Lock()
		f.computes = append(f.computes, key)
		f.mu.Unlock()
	}
	return f
}

// list runs one all-projects pass and returns its rows keyed "source/id", the
// warnings, and the extractions the pass ran.
func (f *cacheFixture) list() (map[string]SessionInfo, []ListWarning, []string) {
	f.t.Helper()
	f.mu.Lock()
	f.computes = nil
	f.mu.Unlock()
	rows, warnings, err := f.lister.ListAll("")
	require.NoError(f.t, err)
	byKey := make(map[string]SessionInfo)
	for _, r := range rows {
		byKey[r.Source+"/"+r.ID] = r
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return byKey, warnings, slices.Clone(f.computes)
}

func TestLister_UnchangedRefreshExtractsNothing(t *testing.T) {
	f := newCacheFixture(t)
	first, warnings, computes := f.list()
	require.Empty(t, warnings)
	require.Len(t, first, 6, "every source lists one row")
	assert.ElementsMatch(t, []string{
		"claude:" + f.claude, "codex:" + f.rollout, "copilot:" + f.copilot,
		"kiro-ide:" + f.chat, "codex-exec:" + f.stream, "kiro-cli:",
	}, computes, "a cold pass extracts once per file")
	require.Equal(t, 1, f.kiroCalls)

	for range 2 {
		again, warnings, computes := f.list()
		assert.Empty(t, warnings)
		assert.Empty(t, computes, "an unchanged pass reads no transcript body")
		assert.Equal(t, first, again, "and lists the same rows")
	}
	assert.Equal(t, 1, f.kiroCalls, "the Kiro database is not opened while its files are unchanged")
}

func TestLister_ChangesShowOnTheNextPass(t *testing.T) {
	t.Run("claude edit with equal size and restored mtime", func(t *testing.T) {
		f := newCacheFixture(t)
		f.list()
		info, err := os.Stat(f.claude)
		require.NoError(t, err)
		data, err := os.ReadFile(f.claude)
		require.NoError(t, err)
		edited := strings.Replace(string(data), cacheTime.Format(time.RFC3339), cacheTime2.Format(time.RFC3339), 1)
		require.Len(t, edited, len(data))
		writeFile(t, f.claude, edited)
		require.NoError(t, os.Chtimes(f.claude, info.ModTime(), info.ModTime()))

		rows, _, computes := f.list()
		assert.Equal(t, []string{"claude:" + f.claude}, computes)
		assert.True(t, rows["claude/"+cacheClaudeID].CreatedAt.Equal(cacheTime2))
	})

	t.Run("codex rollout edit", func(t *testing.T) {
		f := newCacheFixture(t)
		f.list()
		setupCodexSession(t, f.home, "/work/moved", cacheCodexID, cacheTime)
		rows, _, computes := f.list()
		assert.Equal(t, []string{"codex:" + f.rollout}, computes)
		assert.Equal(t, "/work/moved", rows["codex/"+cacheCodexID].Project)
	})

	t.Run("codex rollout that stops being valid is omitted with a warning", func(t *testing.T) {
		f := newCacheFixture(t)
		f.list()
		writeFile(t, f.rollout, "not json\n")
		rows, warnings, _ := f.list()
		assert.NotContains(t, rows, "codex/"+cacheCodexID, "stale metadata is not served")
		require.Len(t, warnings, 1)
		assert.Equal(t, SourceCodex, warnings[0].Source)
	})

	t.Run("copilot workspace.yaml is a companion-only change", func(t *testing.T) {
		f := newCacheFixture(t)
		f.list()
		setupCopilotSession(t, f.home, "/work/elsewhere", cacheCopilot, cacheTime)
		rows, _, computes := f.list()
		assert.Equal(t, []string{"copilot:" + f.copilot}, computes)
		assert.Equal(t, "/work/elsewhere", rows["copilot/"+cacheCopilot].Project)
	})

	t.Run("kiro ide chat file", func(t *testing.T) {
		f := newCacheFixture(t)
		f.list()
		writeKiroIDEChatFile(t, filepath.Dir(f.chat), "a.chat", cacheIDEExec, 5)
		_, _, computes := f.list()
		assert.Equal(t, []string{"kiro-ide:" + f.chat}, computes)
	})

	t.Run("kiro ide file added and removed", func(t *testing.T) {
		f := newCacheFixture(t)
		f.list()
		extra := writeKiroIDEChatFile(t, filepath.Dir(f.chat), "b.chat", "ide-exec-2", 1)
		rows, _, computes := f.list()
		assert.Equal(t, []string{"kiro-ide:" + extra}, computes, "only the new file is read")
		assert.Contains(t, rows, "kiro-ide/ide-exec-2")

		require.NoError(t, os.Remove(extra))
		rows, _, computes = f.list()
		assert.Empty(t, computes)
		assert.NotContains(t, rows, "kiro-ide/ide-exec-2")
		assert.NotContains(t, f.lister.cache.entries, "kiro-ide:"+extra, "the removed file's entry is swept")
	})

	t.Run("exec stream edit keeps unaffected identifiers", func(t *testing.T) {
		f := newCacheFixture(t)
		before, _, _ := f.list()
		writeFile(t, f.stream, `{"type":"thread.started","thread_id":"thread-1"}`+"\n"+`{"type":"turn.started"}`+"\n")
		after, _, computes := f.list()
		assert.Equal(t, []string{"codex-exec:" + f.stream}, computes)
		var beforeID, afterID string
		for k := range before {
			if strings.HasPrefix(k, "codex-exec/") {
				beforeID = k
			}
		}
		for k := range after {
			if strings.HasPrefix(k, "codex-exec/") {
				afterID = k
			}
		}
		assert.Equal(t, beforeID, afterID)
	})

	t.Run("a newly created nearer repository", func(t *testing.T) {
		f := newCacheFixture(t)
		rows, _, _ := f.list()
		var id string
		for k, r := range rows {
			if r.Source == SourceCodexExec {
				id = k
				assert.Empty(t, r.Project)
			}
		}
		require.NoError(t, os.Mkdir(filepath.Join(filepath.Dir(f.stream), ".git"), 0o755))
		rows, _, computes := f.list()
		assert.Empty(t, computes, "the stream itself is not read again")
		assert.Equal(t, filepath.Dir(f.stream), rows[id].Project)
	})

	t.Run("kiro database companion appears and the main file changes", func(t *testing.T) {
		f := newCacheFixture(t)
		f.list()
		writeFile(t, f.kiroDB+"-wal", "wal")
		f.kiroSessions = append(f.kiroSessions, logs.SessionMetadata{ConversationID: "kiro-2", Directory: "/work/kiro", CreatedAt: cacheTime2})
		rows, _, _ := f.list()
		assert.Equal(t, 2, f.kiroCalls, "a -wal file appearing re-runs discovery")
		assert.Contains(t, rows, "kiro-cli/kiro-2")

		f.list()
		assert.Equal(t, 2, f.kiroCalls)
		writeFile(t, f.kiroDB, "db-changed")
		f.list()
		assert.Equal(t, 3, f.kiroCalls)
	})
}

func TestLister_ReturnedSliceIsNotSharedWithTheNextPass(t *testing.T) {
	f := newCacheFixture(t)
	first, _, err := f.lister.ListAll("")
	require.NoError(t, err)
	want := slices.Clone(first)
	slices.Reverse(first)
	for i := range first {
		first[i].Project = "mutated"
	}
	second, _, err := f.lister.ListAll("")
	require.NoError(t, err)
	assert.Equal(t, want, second)
}

func TestLister_ConcurrentPassesAreSerialised(t *testing.T) {
	f := newCacheFixture(t)
	want, _, _ := f.list()
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			rows, _, err := f.lister.ListAll("")
			assert.NoError(t, err)
			assert.Len(t, rows, len(want))
		})
	}
	wg.Wait()
}

// BenchmarkListAll_KiroIDE lists generated Kiro IDE .chat files with a cold
// cache and again with a warm one (design risk 4).
func BenchmarkListAll_KiroIDE(b *testing.B) {
	const files, messages = 500, 200
	base := b.TempDir()
	for i := range files {
		dir := filepath.Join(base, fmt.Sprintf("ws%d", i%10))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			b.Fatal(err)
		}
		msgs := strings.Repeat(`{"role":"human","content":"`+strings.Repeat("x", 400)+`"},`, messages)
		data := fmt.Sprintf(`{"executionId":"exec-%d","chat":[%s{"role":"bot","content":"done"}],"metadata":{"startTime":1790812800000,"modelId":"m","modelProvider":"p"}}`, i, msgs)
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("%d.chat", i)), []byte(data), 0o600); err != nil {
			b.Fatal(err)
		}
	}
	newLister := func() *Lister {
		return &Lister{homeDir: b.TempDir(), cache: &fileCache{}, kiroIDEBasePath: func() (string, error) { return base, nil }}
	}
	b.Run("cold", func(b *testing.B) {
		for b.Loop() {
			if _, _, err := newLister().ListAll(""); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("warm", func(b *testing.B) {
		l := newLister()
		if _, _, err := l.ListAll(""); err != nil {
			b.Fatal(err)
		}
		for b.Loop() {
			if _, _, err := l.ListAll(""); err != nil {
				b.Fatal(err)
			}
		}
	})
}
