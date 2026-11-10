package sessions

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/arjenschwarz/orbit/internal/transcript"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// lmResolver opens LM Studio sessions under root, in all-projects scope
// unless a project is given.
func lmResolver(t *testing.T, root, project string) *Resolver {
	t.Helper()
	r, err := NewResolver(project, WithLMStudioLogDir(root, true))
	require.NoError(t, err)
	return r
}

// lmResolveTree has a thread t in two files, singletons beside it, and a
// later file holding only thread u. Each request record is one line.
func lmResolveTree(t *testing.T) *execTree {
	t.Helper()
	tree := newLMTree(t)
	tree.write(lmReq("2026-11-01 10:00:00", "t", "model-a")+
		"[2026-11-01 10:00:01][WARN] warning text\n"+
		lmReq("2026-11-01 10:00:02", "", "model-b"), "root", "2026-11", "2026-11-01.1.log")
	tree.write(lmReq("2026-11-01 11:00:00", "", "model-c")+
		lmReq("2026-11-01 11:00:01", "t", "model-d"), "root", "2026-11", "2026-11-01.2.log")
	tree.write(lmReq("2026-11-02 09:00:00", "u", "model-e"), "root", "2026-11", "2026-11-02.1.log")
	return tree
}

func lmRequestKeys(res *transcript.ParseResult) []string {
	var out []string
	for _, req := range res.LMStudioRequests {
		out = append(out, req.File+":"+req.Model)
	}
	return out
}

// A path-form identifier opens only the file it names, after a directory
// listing that also names the next file, whose body is never read
// (requirements 1.5, 4.2).
func TestResolveLMStudio_SingletonReadsOneFile(t *testing.T) {
	tree := lmResolveTree(t)
	key := "2026-11/2026-11-01.2.log"
	id := lmRequestID(key, 1)
	reads := countLMReads(t)

	rs, err := lmResolver(t, tree.path("root"), "").Resolve(SourceLMStudio, id)
	require.NoError(t, err)

	assert.Equal(t, []string{tree.path("root", "2026-11", "2026-11-01.2.log")}, reads.opens())
	res := rs.Metadata.PreParsed
	require.NotNil(t, res)
	require.Len(t, res.LMStudioRequests, 1)
	req := res.LMStudioRequests[0]
	assert.Equal(t, id, req.ID)
	assert.Equal(t, id, req.SessionID)
	assert.Equal(t, key, req.File)
	assert.Equal(t, []transcript.LMStudioEventSummary{{File: key, NextFile: "2026-11/2026-11-02.1.log", Lines: []int{}}}, res.LMStudioEvents)

	assert.Equal(t, SourceLMStudio, rs.Metadata.Source)
	assert.Equal(t, id, rs.Metadata.ID)
	assert.Empty(t, rs.Metadata.Path, "no single file backs a session")
	assert.Empty(t, rs.Metadata.ThreadID)
	assert.Equal(t, int64(len(lmReq("2026-11-01 11:00:00", "", "model-c"))), rs.Metadata.Size)
	assert.Equal(t, time.Date(2026, 11, 1, 11, 0, 0, 0, time.UTC), rs.Metadata.CreatedAt.UTC())
	assert.False(t, rs.Metadata.TimeFromFilesystem)
	body, err := io.ReadAll(rs.Reader)
	require.NoError(t, err)
	assert.Empty(t, body)
}

// A hash-form identifier is matched against the discovered file names, so it
// opens only its file too (Q39).
func TestResolveLMStudio_HashFormReadsOneFile(t *testing.T) {
	tree := newLMTree(t)
	name := "2026-11-01." + strings.Repeat("0", 180) + "1.log"
	path := tree.write(lmReq("2026-11-01 10:00:00", "", "model-a"), "root", "2026-11", name)
	tree.write(lmReq("2026-11-01 09:00:00", "", "model-b"), "root", "2026-11", "2026-11-01.1.log")
	id := lmRequestID("2026-11/"+name, 1)
	require.True(t, strings.HasPrefix(id, lmRequestHashPrefix), id)
	reads := countLMReads(t)

	rs, err := lmResolver(t, tree.path("root"), "").Resolve(SourceLMStudio, id)
	require.NoError(t, err)
	assert.Equal(t, []string{path}, reads.opens())
	assert.Equal(t, []string{"2026-11/" + name + ":model-a"}, lmRequestKeys(rs.Metadata.PreParsed))
}

// A thread identifier selects the thread's requests across files, in
// discovery order, with summaries for the files holding them only. Every file
// is read once (requirements 1.5, 3.1, Q39).
func TestResolveLMStudio_Thread(t *testing.T) {
	tree := lmResolveTree(t)
	id := lmThreadSessionID("t")
	reads := countLMReads(t)

	rs, err := lmResolver(t, tree.path("root"), "").Resolve(SourceLMStudio, id)
	require.NoError(t, err)

	assert.ElementsMatch(t, []string{
		tree.path("root", "2026-11", "2026-11-01.1.log"),
		tree.path("root", "2026-11", "2026-11-01.2.log"),
		tree.path("root", "2026-11", "2026-11-02.1.log"),
	}, reads.opens())
	res := rs.Metadata.PreParsed
	assert.Equal(t, []string{"2026-11/2026-11-01.1.log:model-a", "2026-11/2026-11-01.2.log:model-d"}, lmRequestKeys(res))
	for _, req := range res.LMStudioRequests {
		assert.Equal(t, id, req.SessionID)
	}
	assert.Equal(t, 1, res.LMStudioRequests[1].PriorPrefixCount, "compared across files")
	for _, e := range res.Entries {
		assert.Contains(t, []string{res.LMStudioRequests[0].ID, res.LMStudioRequests[1].ID}, e.RequestID)
	}
	assert.Equal(t, []transcript.LMStudioEventSummary{
		{File: "2026-11/2026-11-01.1.log", NextFile: "2026-11/2026-11-01.2.log", Counts: transcript.LMStudioEventCounts{Warn: 1}, Lines: []int{2}},
		{File: "2026-11/2026-11-01.2.log", NextFile: "2026-11/2026-11-02.1.log", Lines: []int{}},
	}, res.LMStudioEvents)
	assert.Equal(t, "t", rs.Metadata.ThreadID)
	assert.Equal(t, "t", res.Metadata.ThreadID)
	assert.Equal(t, int64(len(lmReq("2026-11-01 10:00:00", "t", "model-a"))+len(lmReq("2026-11-01 11:00:01", "t", "model-d"))), rs.Metadata.Size)
	assert.Equal(t, time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC), rs.Metadata.CreatedAt.UTC())
}

// A threaded request's own identifier opens that request alone, still naming
// its logical session (requirement 3.3).
func TestResolveLMStudio_ThreadedRequest(t *testing.T) {
	tree := lmResolveTree(t)
	id := lmRequestID("2026-11/2026-11-01.2.log", 2)

	rs, err := lmResolver(t, tree.path("root"), "").Resolve(SourceLMStudio, id)
	require.NoError(t, err)
	res := rs.Metadata.PreParsed
	assert.Equal(t, []string{"2026-11/2026-11-01.2.log:model-d"}, lmRequestKeys(res))
	assert.Equal(t, lmThreadSessionID("t"), res.LMStudioRequests[0].SessionID)
}

func TestResolveLMStudio_NotFound(t *testing.T) {
	tree := lmResolveTree(t)
	outside := tree.write(lmReq("2026-11-01 10:00:00", "", "model-x"), "outside", "2026-11", "2026-11-01.1.log")
	root := tree.path("root")
	tests := map[string]struct {
		root, project, id string
		readsBodies       bool
	}{
		"project scope":            {root: root, project: tree.mkdir("project"), id: lmThreadSessionID("t")},
		"source disabled":          {root: "", id: lmThreadSessionID("t")},
		"missing root":             {root: tree.path("missing"), id: lmThreadSessionID("t")},
		"unknown form":             {root: root, id: "lmstudio-other-1"},
		"malformed thread digest":  {root: root, id: lmThreadPrefix + strings.Repeat("g", lmDigestLength)},
		"upper-case thread digest": {root: root, id: strings.ToUpper(lmThreadSessionID("t"))},
		"absent thread":            {root: root, id: lmThreadSessionID("absent"), readsBodies: true},
		"absent request line":      {root: root, id: lmRequestID("2026-11/2026-11-01.2.log", 9), readsBodies: true},
		"absent file":              {root: root, id: lmRequestID("2026-11/2026-11-09.1.log", 1)},
		"non-canonical line":       {root: root, id: strings.TrimSuffix(lmRequestID("2026-11/2026-11-01.2.log", 1), "-1") + "-01"},
		"escaping file":            {root: root, id: lmRequestID("../outside/2026-11/2026-11-01.1.log", 1)},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			reads := countLMReads(t)
			_, err := lmResolver(t, tc.root, tc.project).Resolve(SourceLMStudio, tc.id)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "session not found")
			assert.NotContains(t, reads.opens(), outside)
			if !tc.readsBodies {
				assert.Empty(t, reads.opens(), "no log body read")
			}
		})
	}
}

// A file of the thread that cannot be read is reported, and the requests of
// the other files are still shown.
func TestResolveLMStudio_UnreadableFileInThread(t *testing.T) {
	tree := lmResolveTree(t)
	broken := tree.path("root", "2026-11", "2026-11-01.2.log")
	open := lmOpen
	lmOpen = func(path string) (io.ReadCloser, error) {
		if path == broken {
			return nil, errors.New("permission denied")
		}
		return open(path)
	}
	t.Cleanup(func() { lmOpen = open })

	rs, err := lmResolver(t, tree.path("root"), "").Resolve(SourceLMStudio, lmThreadSessionID("t"))
	require.NoError(t, err)
	res := rs.Metadata.PreParsed
	assert.Equal(t, []string{"2026-11/2026-11-01.1.log:model-a"}, lmRequestKeys(res))
	require.Len(t, res.Warnings, 1)
	assert.Equal(t, WarningPathUnreadable, res.Warnings[0].Code)
	assert.Contains(t, res.Warnings[0].Message, "2026-11/2026-11-01.2.log")

	_, err = lmResolver(t, tree.path("root"), "").Resolve(SourceLMStudio, lmRequestID("2026-11/2026-11-01.2.log", 1))
	assert.ErrorContains(t, err, "permission denied", "a request's only file must be readable")
}

// A discovered file that has become a symlink is not read through
// (requirement 1.3).
func TestReadLMStudioFile_RechecksContainment(t *testing.T) {
	tree := newLMTree(t)
	target := tree.write(lmReq("2026-11-01 10:00:00", "", "model-a"), "outside.log")
	link := tree.path("root", "2026-11", "2026-11-01.1.log")
	tree.mkdir("root", "2026-11")
	require.NoError(t, os.Symlink(target, link))

	_, err := readLMStudioFile(tree.path("root"), lmLogFile{path: link, key: "2026-11/2026-11-01.1.log"})
	assert.ErrorContains(t, err, "no longer a file inside the log directory")
}

// ParseResolved returns an assembled session without reading its reader.
func TestParseResolved_PreParsed(t *testing.T) {
	pre := &transcript.ParseResult{Format: transcript.FormatLMStudio}
	rs := &ResolvedSession{Reader: io.NopCloser(failingReader{}), Metadata: SessionMetadata{Source: SourceLMStudio, PreParsed: pre}}
	got, err := ParseResolved(rs)
	require.NoError(t, err)
	assert.Same(t, pre, got)
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("reader must not be read") }

func TestResolvePath_LMStudio(t *testing.T) {
	tree := lmResolveTree(t)
	_, err := lmResolver(t, tree.path("root"), "").ResolvePath(SourceLMStudio, lmThreadSessionID("t"))
	assert.ErrorContains(t, err, "no single file path")
}

// Direct conversion keys a file as discovery does when the log directory
// lists it, including through a file symlink, and otherwise by its canonical
// path or the stdin marker (requirement 3.4).
func TestAssignLMStudioInputIDs(t *testing.T) {
	tree := newLMTree(t)
	log := lmReq("2026-11-01 10:00:00", "", "model-a") + lmReq("2026-11-01 10:00:01", "t", "model-b")
	listed := tree.write(log, "root", "2026-11", "2026-11-01.1.log")
	unlisted := tree.write(log, "root", "notes.log")
	outside := tree.write(log, "elsewhere", "2026-11-01.1.log")
	link := tree.path("root", "2026-11", "2026-11-02.1.log")
	require.NoError(t, os.Symlink(listed, link))
	root := tree.path("root")

	for name, tc := range map[string]struct {
		path, logDir, wantKey string
	}{
		"listed file":           {path: listed, logDir: root, wantKey: "2026-11/2026-11-01.1.log"},
		"symlink to it":         {path: link, logDir: root, wantKey: "2026-11/2026-11-01.1.log"},
		"unlisted file in root": {path: unlisted, logDir: root, wantKey: filepath.ToSlash(unlisted)},
		"file outside root":     {path: outside, logDir: root, wantKey: filepath.ToSlash(outside)},
		"source disabled":       {path: listed, wantKey: filepath.ToSlash(listed)},
		"stdin":                 {logDir: root, wantKey: lmStdinKey},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := transcript.ParseLMStudio(strings.NewReader(log))
			require.NoError(t, err)
			AssignLMStudioInputIDs(res, tc.path, tc.logDir)

			require.Len(t, res.LMStudioRequests, 2)
			single, threaded := res.LMStudioRequests[0], res.LMStudioRequests[1]
			assert.Equal(t, lmRequestID(tc.wantKey, 1), single.ID)
			assert.Equal(t, single.ID, single.SessionID)
			assert.Equal(t, lmRequestID(tc.wantKey, 2), threaded.ID)
			assert.Equal(t, lmThreadSessionID("t"), threaded.SessionID)
		})
	}
}
