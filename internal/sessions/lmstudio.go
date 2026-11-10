package sessions

import (
	"cmp"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/arjenschwarz/orbit/internal/transcript"
)

// LM Studio server logs are YYYY-MM/YYYY-MM-DD.N.log files under one
// configured root. Listing reads each file into a summary, which a Lister
// caches across refreshes, and groups requests into logical sessions: one
// per recorded client thread ID, otherwise one per request (Decision 1).

// Identifier forms (requirement 3.3). A path-form request identifier carries
// its file's root-relative path, so a singleton session can be opened without
// an index; the hash form keeps long paths within the length limit. The
// explicit -p-/-h- discriminator matters because a hex digest is also valid
// base64url.
const (
	lmRequestPathPrefix = "lmstudio-req-p-"
	lmRequestHashPrefix = "lmstudio-req-h-"
	lmThreadPrefix      = "lmstudio-thread-"
	lmMaxIDLength       = 256
	lmDigestLength      = 48 // hex characters of SHA-256 kept in hash-based identifiers
)

// lmRequestID derives a request's identifier from the slash-separated
// root-relative path of the file holding it and its start line. It reads
// nothing, so appends and other files never change it (requirements 3.3, 3.4).
func lmRequestID(fileKey string, startLine int) string {
	line := strconv.Itoa(startLine)
	if id := lmRequestPathPrefix + base64.RawURLEncoding.EncodeToString([]byte(fileKey)) + "-" + line; len(id) <= lmMaxIDLength {
		return id
	}
	return lmRequestHashPrefix + lmDigest(fileKey) + "-" + line
}

// lmThreadSessionID derives a threaded logical session's identifier from the
// recorded client thread ID.
func lmThreadSessionID(threadID string) string {
	return lmThreadPrefix + lmDigest(threadID)
}

// lmSessionID is the logical session a request belongs to: its thread's, or
// the request's own identifier when it records no usable thread ID.
func lmSessionID(requestID, threadID string) string {
	if threadID == "" {
		return requestID
	}
	return lmThreadSessionID(threadID)
}

// lmDigest hashes the full value, so distinct inputs never share an
// identifier through truncation (Q18).
func lmDigest(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])[:lmDigestLength]
}

// parseLMRequestID decodes a path-form request identifier into the file key
// and start line it was derived from. Every other form reports false, and so
// does any spelling lmRequestID would not produce, so each request has one
// identifier.
func parseLMRequestID(id string) (fileKey string, startLine int, ok bool) {
	rest, ok := strings.CutPrefix(id, lmRequestPathPrefix)
	sep := strings.LastIndexByte(rest, '-')
	if !ok || sep < 0 {
		return "", 0, false
	}
	line, lineErr := strconv.Atoi(rest[sep+1:])
	key, keyErr := base64.RawURLEncoding.DecodeString(rest[:sep])
	if lineErr != nil || keyErr != nil || len(key) == 0 || line < 1 || lmRequestID(string(key), line) != id {
		return "", 0, false
	}
	return string(key), line, true
}

var (
	lmMonthDirPattern = regexp.MustCompile(`^\d{4}-\d{2}$`)
	lmLogFilePattern  = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\.(\d+)\.log$`)
)

// lmLogFile is a discovered server-log file.
type lmLogFile struct {
	path string // canonical path
	key  string // slash-separated path relative to the root; the identifier key
	date string // filename date
	seq  int    // filename .N suffix
}

// compareLMFiles is discovery order: filename date, numeric suffix, then key.
func compareLMFiles(a, b lmLogFile) int {
	return cmp.Or(strings.Compare(a.date, b.date), cmp.Compare(a.seq, b.seq), strings.Compare(a.key, b.key))
}

// lmStudioRoot canonicalises the configured log root, which must be a directory.
func lmStudioRoot(dir string) (string, error) {
	root, err := CanonicalPath(dir)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(root)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a directory", root)
	}
	return root, nil
}

// findLMStudioFiles lists the YYYY-MM/YYYY-MM-DD.N.log files under a
// canonical root in discovery order, from directory metadata only
// (requirements 1.2, 1.3). Symlinked month directories are not descended. A
// file symlink joins as its target when that is a regular file inside the
// root; each canonical file is listed once, keyed by its own root-relative
// path and ordered by the earliest name it was found under. The error reports
// a root that cannot be read.
func findLMStudioFiles(root string) ([]lmLogFile, []ListWarning, error) {
	months, err := os.ReadDir(root)
	if err != nil {
		return nil, nil, err
	}
	var warnings []ListWarning
	unreadable := func(path string, err error) {
		warnings = append(warnings, ListWarning{Source: SourceLMStudio, Code: WarningPathUnreadable,
			Err: fmt.Errorf("%s: %w", path, err)})
	}
	found := make(map[string]lmLogFile)
	for _, month := range months {
		if !month.IsDir() || !lmMonthDirPattern.MatchString(month.Name()) {
			continue
		}
		dir := filepath.Join(root, month.Name())
		entries, err := os.ReadDir(dir)
		if err != nil {
			unreadable(dir, err)
			continue
		}
		for _, e := range entries {
			m := lmLogFilePattern.FindStringSubmatch(e.Name())
			if m == nil {
				continue
			}
			seq, err := strconv.Atoi(m[2])
			if err != nil {
				continue
			}
			path := filepath.Join(dir, e.Name())
			switch {
			case e.Type().IsRegular():
			case e.Type()&fs.ModeSymlink != 0:
				target, err := CanonicalPath(path)
				if err != nil {
					unreadable(path, err)
					continue
				}
				if !withinAnyRoot(target, []string{root}) {
					continue
				}
				info, err := os.Stat(target)
				if err != nil {
					unreadable(path, err)
					continue
				}
				if !info.Mode().IsRegular() {
					continue
				}
				path = target
			default:
				continue
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				continue
			}
			f := lmLogFile{path: path, key: filepath.ToSlash(rel), date: m[1], seq: seq}
			if prev, seen := found[path]; !seen || compareLMFiles(f, prev) < 0 {
				found[path] = f
			}
		}
	}
	files := slices.SortedFunc(maps.Values(found), compareLMFiles)
	return files, warnings, nil
}

// lmFileSummary is what listing keeps of one log file: request identity and
// listing metadata, unattributed-event counts and file-level warnings, never
// message content.
type lmFileSummary struct {
	requests    []lmRequestSummary // source order
	events      transcript.LMStudioEventCounts
	eventLines  []int // start line of each unattributed event, for session-view references (4.2)
	warnings    []ListWarning
	zoneWarning string // the TZ override failed; reported once per pass
}

// lmRequestSummary is one request's listing metadata.
type lmRequestSummary struct {
	id        string
	threadID  string    // "" = singleton
	size      int64     // the request record's bytes
	createdAt time.Time // the interpreted wall time; zero when not unique, never a filesystem time (5.4)
	model     string
}

// lmOpen opens log files for reading. Tests replace it to count body reads.
var lmOpen = func(path string) (io.ReadCloser, error) { return os.Open(path) }

// summarizeLMStudioFile reads one log file into its summary. A discovered file
// is parsed as a log without the detection probe, so a rotated file holding
// only notices still belongs to the root (1.2). Record warnings stay with the
// transcript; the summary keeps the parser's text-outside-records warnings
// only for a file that yielded nothing, which is how a discovered file with no
// recognised records is reported (7.4).
func summarizeLMStudioFile(file lmLogFile) (lmFileSummary, error) {
	f, err := lmOpen(file.path)
	if err != nil {
		return lmFileSummary{}, err
	}
	defer func() { _ = f.Close() }()
	res, err := transcript.ParseLMStudio(f)
	if err != nil {
		return lmFileSummary{}, err
	}

	var s lmFileSummary
	for _, req := range res.LMStudioRequests {
		s.requests = append(s.requests, lmRequestSummary{
			id:        lmRequestID(file.key, req.StartLine),
			threadID:  req.ThreadID,
			size:      int64(req.Size),
			createdAt: req.ResolvedTime,
			model:     req.Model,
		})
	}
	for _, e := range res.Entries {
		if !e.Unattributed {
			continue
		}
		switch {
		case e.Type == "assistant":
			s.events.Response++
		case e.Message != nil && len(e.Message.Content) > 0 && e.Message.Content[0].Type == "error":
			s.events.Error++
		default:
			s.events.Warn++
		}
		s.eventLines = append(s.eventLines, e.SourceLine)
	}
	empty := len(res.LMStudioRequests) == 0 && len(res.Entries) == 0
	for _, w := range res.Warnings {
		switch {
		case w.Code == transcript.WarningTimeZone:
			s.zoneWarning = w.Message
		case w.Code == transcript.WarningUnparsedText && empty:
			s.warnings = append(s.warnings, ListWarning{Source: SourceLMStudio, Code: w.Code,
				Err: fmt.Errorf("%s: %s", file.key, w.Message)})
		}
	}
	return s, nil
}

// lmFileFingerprint identifies a file's state. statFingerprint fills it on
// Unix; elsewhere it is never compared.
type lmFileFingerprint struct {
	size, mtime, ctime int64
	inode, device      uint64
}

// lmFingerprint fingerprints a file and reports whether the fingerprint can
// be trusted for a cache hit. Tests replace it.
var lmFingerprint = statFingerprint

// lmCache keeps file summaries across listing passes within one process.
type lmCache struct {
	mu      sync.Mutex
	entries map[string]lmCacheEntry // canonical path -> fingerprint and summary
}

type lmCacheEntry struct {
	fingerprint lmFileFingerprint
	summary     lmFileSummary
}

// summary returns a file's summary: the cached one while the file is
// cacheable and its fingerprint unchanged, otherwise a fresh read, which is
// stored when cacheable. The flag, not fingerprint equality, gates every hit,
// so a platform without fingerprints never reuses a summary. Callers hold mu.
func (c *lmCache) summary(file lmLogFile) (lmFileSummary, error) {
	fp, cacheable, err := lmFingerprint(file.path)
	if err != nil {
		delete(c.entries, file.path)
		return lmFileSummary{}, err
	}
	if e, ok := c.entries[file.path]; ok && cacheable && e.fingerprint == fp {
		return e.summary, nil
	}
	delete(c.entries, file.path)
	s, err := summarizeLMStudioFile(file)
	if err == nil && cacheable {
		if c.entries == nil {
			c.entries = make(map[string]lmCacheEntry)
		}
		c.entries[file.path] = lmCacheEntry{fingerprint: fp, summary: s}
	}
	return s, err
}

// lmFileRef is the part of a logical session one file holds.
type lmFileRef struct {
	file     lmLogFile
	requests []lmRequestSummary // the session's requests in the file, in source order
}

// lmIndex is one pass over a log root.
type lmIndex struct {
	files     []lmLogFile              // every discovered file, in discovery order
	summaries map[string]lmFileSummary // canonical path -> summary; absent when the file could not be read
	sessions  map[string][]lmFileRef   // session ID -> the files holding it, in discovery order
	order     []string                 // session IDs by first request, in discovery order
}

// buildLMStudioIndex reads every discovered file under a canonical root,
// reusing cache for unchanged files, and groups requests into logical
// sessions: exact thread IDs across files, otherwise one session per request
// (3.1, 3.2). A nil cache never hits, so every body is read. The cache's mutex
// is held for the whole pass, so overlapping passes serialise. The error
// reports a root that cannot be read.
func buildLMStudioIndex(root string, cache *lmCache) (*lmIndex, []ListWarning, error) {
	files, warnings, err := findLMStudioFiles(root)
	if err != nil {
		return nil, warnings, err
	}
	if cache == nil {
		cache = &lmCache{}
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()

	idx := &lmIndex{
		files:     files,
		summaries: make(map[string]lmFileSummary, len(files)),
		sessions:  make(map[string][]lmFileRef),
	}
	zoneWarning := ""
	for _, file := range files {
		summary, err := cache.summary(file)
		if err != nil {
			warnings = append(warnings, ListWarning{Source: SourceLMStudio, Code: WarningPathUnreadable,
				Err: fmt.Errorf("%s: %w", file.key, err)})
			continue
		}
		idx.summaries[file.path] = summary
		warnings = append(warnings, summary.warnings...)
		zoneWarning = cmp.Or(zoneWarning, summary.zoneWarning)
		for _, req := range summary.requests {
			id := lmSessionID(req.id, req.threadID)
			refs := idx.sessions[id]
			if len(refs) == 0 {
				idx.order = append(idx.order, id)
			}
			if len(refs) == 0 || refs[len(refs)-1].file.path != file.path {
				refs = append(refs, lmFileRef{file: file})
			}
			last := &refs[len(refs)-1]
			last.requests = append(last.requests, req)
			idx.sessions[id] = refs
		}
	}
	// Removed files drop out of the cache with their summaries.
	maps.DeleteFunc(cache.entries, func(path string, _ lmCacheEntry) bool {
		_, kept := idx.summaries[path]
		return !kept
	})
	if zoneWarning != "" {
		// Once per pass, as once per transcript (requirement 5.5).
		warnings = append(warnings, ListWarning{Source: SourceLMStudio, Code: transcript.WarningTimeZone, Err: errors.New(zoneWarning)})
	}
	return idx, warnings, nil
}

// listLMStudio lists the logical sessions under the LM Studio log root. The
// source records no project, so project scope returns before touching the
// root, whatever the configuration (requirement 7.1). A root that cannot be
// read is reported only when the user chose it (7.4).
func (l *Lister) listLMStudio(projectPath string) ([]SessionInfo, []ListWarning) {
	if projectPath != "" || l.opts.LMStudioLogDir == "" {
		return nil, nil
	}
	var idx *lmIndex
	var warnings []ListWarning
	root, err := lmStudioRoot(l.opts.LMStudioLogDir)
	if err == nil {
		idx, warnings, err = buildLMStudioIndex(root, l.lmCache)
	}
	if err != nil {
		if l.opts.LMStudioLogDirExplicit {
			warnings = append(warnings, ListWarning{Source: SourceLMStudio, Code: WarningRootUnreadable,
				Err: fmt.Errorf("LM Studio log directory %s: %w", l.opts.LMStudioLogDir, err)})
		}
		return nil, warnings
	}
	sessions := make([]SessionInfo, 0, len(idx.order))
	for _, id := range idx.order {
		sessions = append(sessions, lmSessionInfo(id, idx.sessions[id]))
	}
	return sessions, warnings
}

// lmSessionInfo is a logical session's listing row (requirement 7.2). Its
// time is its first request's in discovery order, or unknown when that one
// has no unique interpretation (5.4); its model is the first one recorded.
// Size counts only the session's own request records (Q7).
func lmSessionInfo(id string, refs []lmFileRef) SessionInfo {
	first := refs[0].requests[0]
	info := SessionInfo{
		ID:        id,
		CreatedAt: first.createdAt,
		Source:    SourceLMStudio,
		Provider:  SourceLMStudio,
		ThreadID:  first.threadID,
	}
	for _, ref := range refs {
		for _, req := range ref.requests {
			info.RequestCount++
			info.Size += req.size
			info.Model = cmp.Or(info.Model, req.model)
		}
	}
	return info
}
