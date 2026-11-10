package sessions

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

// lmIDPattern is requirement 3.3's identifier shape.
var lmIDPattern = regexp.MustCompile(`^lmstudio-[A-Za-z0-9_-]+$`)

func assertLMID(t interface {
	Helper()
	Fatalf(string, ...any)
}, id string) {
	t.Helper()
	if !lmIDPattern.MatchString(id) || len(id) > lmMaxIDLength {
		t.Fatalf("identifier %q (%d characters) is not an lmstudio- identifier of at most %d ASCII letters, digits, _ or -", id, len(id), lmMaxIDLength)
	}
}

func TestLMRequestID(t *testing.T) {
	long := strings.Repeat("d/", 120) + "2026-11-10.1.log"
	tests := map[string]struct {
		key      string
		line     int
		want     string
		pathForm bool
	}{
		"path form": {
			key: "2026-11/2026-11-10.1.log", line: 12,
			want:     "lmstudio-req-p-MjAyNi0xMS8yMDI2LTExLTEwLjEubG9n-12",
			pathForm: true,
		},
		"hash form when the path form exceeds 256 characters": {
			key: long, line: 7,
			want: "lmstudio-req-h-" + sha256Hex(long)[:48] + "-7",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := lmRequestID(tc.key, tc.line)
			assert.Equal(t, tc.want, got)
			assertLMID(t, got)
			key, line, ok := parseLMRequestID(got)
			assert.Equal(t, tc.pathForm, ok)
			if tc.pathForm {
				assert.Equal(t, tc.key, key)
				assert.Equal(t, tc.line, line)
			}
		})
	}
}

func TestLMSessionID(t *testing.T) {
	req := lmRequestID("2026-11/2026-11-10.1.log", 3)
	assert.Equal(t, req, lmSessionID(req, ""), "a singleton session is its request")
	assert.Equal(t, "lmstudio-thread-"+sha256Hex("thread-alpha")[:48], lmSessionID(req, "thread-alpha"))
	assert.Equal(t, lmSessionID(req, "thread-alpha"), lmSessionID(lmRequestID("2026-11/2026-11-11.1.log", 9), "thread-alpha"),
		"requests of one thread share a session whatever file holds them")
}

func TestParseLMRequestID_Rejects(t *testing.T) {
	valid := lmRequestID("2026-11/2026-11-10.1.log", 12)
	for name, id := range map[string]string{
		"hash form":               lmRequestID(strings.Repeat("x", 300), 3),
		"thread form":             lmThreadSessionID("thread-alpha"),
		"no line":                 strings.TrimSuffix(valid, "-12"),
		"zero line":               strings.TrimSuffix(valid, "12") + "0",
		"signed line":             strings.TrimSuffix(valid, "12") + "+12",
		"padded line":             strings.TrimSuffix(valid, "12") + "012",
		"empty path":              "lmstudio-req-p--12",
		"not base64url":           "lmstudio-req-p-a.b-12",
		"padded base64":           "lmstudio-req-p-YQ==-12",
		"other prefix":            "lmstudio-req-x-YQ-12",
		"non-canonical base64url": "lmstudio-req-p-YR-1",
	} {
		t.Run(name, func(t *testing.T) {
			_, _, ok := parseLMRequestID(id)
			assert.False(t, ok, "%s", id)
		})
	}
}

func TestPropertyLMIdentifiers(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		key := rapid.StringN(1, 300, -1).Draw(rt, "key")
		line := rapid.IntRange(1, 10_000_000).Draw(rt, "line")
		thread := rapid.StringN(1, 400, -1).Draw(rt, "thread")

		id := lmRequestID(key, line)
		assertLMID(rt, id)
		if again := lmRequestID(key, line); again != id {
			rt.Fatalf("not deterministic: %q then %q", id, again)
		}
		gotKey, gotLine, ok := parseLMRequestID(id)
		switch {
		case strings.HasPrefix(id, lmRequestPathPrefix):
			if !ok || gotKey != key || gotLine != line {
				rt.Fatalf("path form %q decodes to (%q, %d, %v), want (%q, %d)", id, gotKey, gotLine, ok, key, line)
			}
		case strings.HasPrefix(id, lmRequestHashPrefix):
			if ok {
				rt.Fatalf("hash form %q decodes as a path form", id)
			}
		default:
			rt.Fatalf("identifier %q has neither request form", id)
		}

		otherLine := rapid.IntRange(1, 10_000_000).Draw(rt, "other line")
		if otherLine != line && lmRequestID(key, otherLine) == id {
			rt.Fatalf("lines %d and %d share identifier %q", line, otherLine, id)
		}
		otherKey := rapid.StringN(1, 300, -1).Draw(rt, "other key")
		if otherKey != key && lmRequestID(otherKey, line) == id {
			rt.Fatalf("keys %q and %q share identifier %q", key, otherKey, id)
		}

		session := lmSessionID(id, thread)
		assertLMID(rt, session)
		if session != lmThreadSessionID(thread) || session == id {
			rt.Fatalf("threaded session %q is not the thread digest", session)
		}
		if _, _, ok := parseLMRequestID(session); ok {
			rt.Fatalf("thread session %q decodes as a request", session)
		}
	})
}

// lmTestFile is a generated log file of single-line requests: request i
// starts on line i+1.
type lmTestFile struct {
	key     string   // root-relative path
	threads []string // per request; "" records no thread ID
}

func (f lmTestFile) content(from int) string {
	var sb strings.Builder
	for i, thread := range f.threads[from:] {
		sb.WriteString(lmReq(fmt.Sprintf("2026-11-10 10:00:%02d", (from+i)%60), thread, "example-model"))
	}
	return sb.String()
}

// lmSessionsByRequest maps every indexed request to its session and file.
func lmSessionsByRequest(idx *lmIndex) map[string][2]string {
	out := make(map[string][2]string)
	for sid, refs := range idx.sessions {
		for _, ref := range refs {
			for _, req := range ref.requests {
				out[req.id] = [2]string{sid, ref.file.key}
			}
		}
	}
	return out
}

// TestPropertyLMStudioIdentityStability checks requirement 3.3 through
// discovery: appending to a file, adding an earlier file and removing another
// file leave every surviving request's identifier and session unchanged, and
// a session without surviving requests disappears.
func TestPropertyLMStudioIdentityStability(t *testing.T) {
	t.Setenv("TZ", "")
	base := t.TempDir()
	thread := rapid.SampledFrom([]string{"", "", "thread-a", "thread-b"})
	rapid.Check(t, func(rt *rapid.T) {
		dir, err := os.MkdirTemp(base, "root")
		if err != nil {
			rt.Fatal(err)
		}
		root, err := filepath.EvalSymlinks(dir)
		if err != nil {
			rt.Fatal(err)
		}
		write := func(key, content string, flag int) {
			path := filepath.Join(root, filepath.FromSlash(key))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				rt.Fatal(err)
			}
			f, err := os.OpenFile(path, flag|os.O_WRONLY|os.O_CREATE, 0o600)
			if err != nil {
				rt.Fatal(err)
			}
			if _, err := f.WriteString(content); err != nil {
				rt.Fatal(err)
			}
			if err := f.Close(); err != nil {
				rt.Fatal(err)
			}
		}
		build := func() map[string][2]string {
			idx, warnings, err := buildLMStudioIndex(root, nil)
			if err != nil || len(warnings) > 0 {
				rt.Fatalf("index: %v, warnings %v", err, warnings)
			}
			return lmSessionsByRequest(idx)
		}

		n := rapid.IntRange(2, 4).Draw(rt, "files")
		files := make([]lmTestFile, n)
		for i := range files {
			files[i] = lmTestFile{
				key:     fmt.Sprintf("2026-11/2026-11-%02d.%d.log", 10+i/2, i%2+1),
				threads: rapid.SliceOfN(thread, 1, 4).Draw(rt, fmt.Sprintf("threads %d", i)),
			}
			write(files[i].key, files[i].content(0), os.O_TRUNC)
		}
		before := build()

		grown := rapid.IntRange(0, n-1).Draw(rt, "appended file")
		removed := (grown + rapid.IntRange(1, n-1).Draw(rt, "removed offset")) % n
		from := len(files[grown].threads)
		files[grown].threads = append(files[grown].threads, rapid.SliceOfN(thread, 1, 3).Draw(rt, "appended")...)
		write(files[grown].key, files[grown].content(from), os.O_APPEND)
		write("2026-10/2026-10-01.1.log", lmTestFile{threads: rapid.SliceOfN(thread, 1, 3).Draw(rt, "earlier")}.content(0), os.O_TRUNC)
		if err := os.Remove(filepath.Join(root, filepath.FromSlash(files[removed].key))); err != nil {
			rt.Fatal(err)
		}
		after := build()

		survivingSessions := make(map[string]bool)
		for _, v := range after {
			survivingSessions[v[0]] = true
		}
		for reqID, was := range before {
			now, ok := after[reqID]
			if was[1] == files[removed].key {
				if ok {
					rt.Fatalf("request %s of the removed file is still indexed", reqID)
				}
				if strings.HasPrefix(was[0], lmRequestPathPrefix) && survivingSessions[was[0]] {
					rt.Fatalf("singleton session %s of the removed file is still listed", was[0])
				}
				continue
			}
			if !ok || now != was {
				rt.Fatalf("request %s moved from %v to %v (present %v)", reqID, was, now, ok)
			}
		}
	})
}
