package transcript

import (
	"bufio"
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// LM Studio's disk server log is a physical-line text grammar: a header line
// `[YYYY-MM-DD HH:MM:SS][LEVEL][tag]... message` starts a record, and every
// following line up to the next header continues it. Request records carry
// a pretty-printed JSON body after ` with body`.

// lmMaxRecordBytes is the largest LM Studio log record apsis parses, matching
// the 10 MiB line limit of the other scanners. A larger record keeps its
// header metadata, and its body is skipped with an oversized_record warning
// naming the limit and the record's size (requirement 6.4).
const lmMaxRecordBytes = 10 * 1024 * 1024

const (
	lmProvider           = "lmstudio"
	lmTruncationMarker   = "<Truncated in logs>"
	lmRequestPrefix      = "Received request: POST to "
	lmBodySuffix         = " with body"
	lmPredictionMarker   = "Generated prediction:"
	lmSyncResponseMarker = "Generated synchronous v1/responses response:"
)

// lmIncludedEndpoints are the POST paths translated as inference requests
// (requirement 2.1). Matching is exact.
var lmIncludedEndpoints = map[string]bool{
	"/v1/responses":            true,
	"/v1/chat/completions":     true,
	"/v1/completions":          true,
	"/v1/messages":             true,
	"/api/v0/chat/completions": true,
	"/api/v0/completions":      true,
	"/api/v1/chat":             true,
}

// lmExcludedEndpoints are documented embedding, token-counting and model
// operation paths. Their records never become requests or unsupported-endpoint
// warnings; WARN and ERROR ones are still retained (requirements 4.4, 4.6).
var lmExcludedEndpoints = map[string]bool{
	"/v1/embeddings":             true,
	"/api/v0/embeddings":         true,
	"/v1/responses/input_tokens": true,
	"/v1/messages/count_tokens":  true,
	"/v1/models":                 true,
	"/api/v0/models":             true,
	"/api/v1/models":             true,
	"/api/v1/models/load":        true,
	"/api/v1/models/unload":      true,
	"/api/v1/models/download":    true,
}

var (
	lmHeaderPattern     = regexp.MustCompile(`^\[(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})\]\[(\w+)\]((?:\[[^\]]*\])*)\s*(.*)$`)
	lmHeaderLikePattern = regexp.MustCompile(`^\[\d{4}-\d{2}-\d{2}`)
	lmTagPattern        = regexp.MustCompile(`\[([^\]]*)\]`)
	lmPostPathPattern   = regexp.MustCompile(`\bPOST (?:to )?(/[^\s)"']*)`)
)

// lmHeader is a parsed header line.
type lmHeader struct {
	wall    string   // "2006-01-02 15:04:05", as logged
	level   string   // as logged
	tags    []string // bracketed tags after the level, brackets removed
	message string   // the text after the tags
}

func parseLMHeader(line []byte) (lmHeader, bool) {
	if len(line) < 2 || line[0] != '[' || !lmHeaderLikePattern.Match(line) {
		return lmHeader{}, false
	}
	m := lmHeaderPattern.FindSubmatch(line)
	if m == nil {
		return lmHeader{}, false
	}
	h := lmHeader{wall: string(m[1]), level: string(m[2]), message: string(m[4])}
	for _, tag := range lmTagPattern.FindAllSubmatch(m[3], -1) {
		h.tags = append(h.tags, string(tag[1]))
	}
	return h, true
}

// lmRequestLine reads a `Received request: POST to <path>...` message: the
// path and the text after it.
func lmRequestLine(message string) (path, rest string, ok bool) {
	after, ok := strings.CutPrefix(message, lmRequestPrefix)
	if !ok {
		return "", "", false
	}
	if i := strings.IndexAny(after, " \t"); i >= 0 {
		return after[:i], after[i:], true
	}
	return after, "", true
}

// ProbeLMStudio reports whether chunk, the start of an input, holds an LM
// Studio header recording a request to an included endpoint (requirement
// 1.1). Body text never matches, because a header must start a line.
func ProbeLMStudio(chunk []byte) bool {
	request, _ := probeLMStudio(chunk, true)
	return request
}

// probeLMStudio also reports whether chunk holds any LM Studio header. When
// complete is false, chunk is cut at a read limit, so a final line without a
// newline counts only when text after its path shows the path is whole.
func probeLMStudio(chunk []byte, complete bool) (request, headers bool) {
	for len(chunk) > 0 {
		line, rest, terminated := bytes.Cut(chunk, []byte("\n"))
		chunk = rest
		h, ok := parseLMHeader(bytes.TrimSuffix(line, []byte("\r")))
		if !ok {
			continue
		}
		headers = true
		path, after, ok := lmRequestLine(h.message)
		if !ok || !lmIncludedEndpoints[path] || (!terminated && !complete && after == "") {
			continue
		}
		return true, true
	}
	return false, headers
}

// lmRecord is one header and its continuation lines.
type lmRecord struct {
	lmHeader
	start, end int    // one-based physical line range
	cont       []byte // each continuation line preceded by "\n"; nil once oversized
	size       int    // record bytes, line terminators included
	oversized  bool   // size exceeds lmMaxRecordBytes
	headerLike []int  // continuation lines that look like a malformed header
	last       bool   // the record ends the input
}

func (rec *lmRecord) lines() string {
	if rec.start == rec.end {
		return fmt.Sprintf("line %d", rec.start)
	}
	return fmt.Sprintf("lines %d-%d", rec.start, rec.end)
}

// lmUnsupported counts the records reporting one unsupported POST path.
type lmUnsupported struct {
	path  string
	line  int // first record's start line
	count int
}

type lmParser struct {
	zone        zoneResult
	requests    []LMStudioRequest
	entries     []Entry
	warnings    []ParseWarning
	unsupported []*lmUnsupported // first-seen order
}

// ParseLMStudioRecords parses one LM Studio log file or stream: included
// requests with their translated entries, logged responses and WARN/ERROR
// records as unattributed entries, and warnings. Request IDs, entry request
// IDs and request-scoped tool IDs hold the temporary key strconv.Itoa(StartLine)
// until the sessions layer assigns final identifiers. It returns an error only
// when reading fails.
func ParseLMStudioRecords(r io.Reader, zone zoneResult) (requests []LMStudioRequest, entries []Entry, warnings []ParseWarning, err error) {
	p := &lmParser{zone: zone, requests: []LMStudioRequest{}, entries: []Entry{}, warnings: []ParseWarning{}}
	if err := p.lex(r); err != nil {
		return nil, nil, nil, err
	}
	for _, u := range p.unsupported {
		noun := "records"
		if u.count == 1 {
			noun = "record"
		}
		p.warn(u.line, WarningUnsupportedEndpoint, fmt.Sprintf("unsupported POST path %s (%d %s)", u.path, u.count, noun))
	}
	for i := range p.entries {
		if p.entries[i].Message == nil {
			continue
		}
		for j := range p.entries[i].Message.Content {
			markTruncated(&p.entries[i].Message.Content[j])
		}
	}
	slices.SortStableFunc(p.warnings, func(a, b ParseWarning) int { return a.Line - b.Line })
	return p.requests, p.entries, p.warnings, nil
}

// ParseLMStudio parses an LM Studio log file or stream into one direct-view
// document: every record in source order, with each request's thread
// membership and history-prefix result annotated rather than split out
// (Decision 5). Wall times are interpreted in the zone TZ selects.
func ParseLMStudio(r io.Reader) (*ParseResult, error) {
	zone := zoneFromEnvironment()
	requests, entries, warnings, err := ParseLMStudioRecords(r, zone)
	if err != nil {
		return nil, err
	}
	annotateExtensions(requests, lmEntriesByRequest(entries))
	if zone.warning != "" {
		// Once per transcript, not per request (requirement 5.5).
		warnings = append([]ParseWarning{{Code: WarningTimeZone, Message: zone.warning}}, warnings...)
	}

	meta := &ParseResultMetadata{FirstProvider: lmProvider, TimeZone: zone.label}
	if i := slices.IndexFunc(requests, func(req LMStudioRequest) bool { return req.Model != "" }); i >= 0 {
		meta.FirstModel = requests[i].Model
	}
	if len(requests) > 0 && !requests[0].ResolvedTime.IsZero() {
		meta.StartedAt = requests[0].ResolvedTime.Format(time.RFC3339)
	}
	return &ParseResult{
		Entries:          entries,
		Warnings:         warnings,
		Metadata:         meta,
		Format:           FormatLMStudio,
		LMStudioRequests: requests,
	}, nil
}

// lex splits the input into records and classifies each as it completes.
func (p *lmParser) lex(r io.Reader) error {
	br := bufio.NewReaderSize(r, 64*1024)
	var rec *lmRecord
	textStart, textEnd := 0, 0 // non-blank text before the first header
	for lineNum := 1; ; lineNum++ {
		line, size, err := readLMLine(br)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("read LM Studio log: %w", err)
		}
		if lineNum == 1 {
			line = stripBOM(line)
		}
		if h, ok := parseLMHeader(line); ok {
			if rec != nil {
				p.record(rec)
			}
			rec = &lmRecord{lmHeader: h, start: lineNum, end: lineNum, size: size, oversized: size > lmMaxRecordBytes}
			continue
		}
		if rec == nil {
			if len(bytes.TrimSpace(line)) > 0 {
				textStart = cmp.Or(textStart, lineNum)
				textEnd = lineNum
			}
			continue
		}
		rec.end = lineNum
		rec.size += size
		if lmHeaderLikePattern.Match(line) {
			rec.headerLike = append(rec.headerLike, lineNum)
		}
		if rec.oversized {
			continue
		}
		if rec.size > lmMaxRecordBytes {
			rec.oversized, rec.cont = true, nil
			continue
		}
		rec.cont = append(append(rec.cont, '\n'), line...)
	}
	if rec != nil {
		rec.last = true
		p.record(rec)
	}

	// Requirement 6.3: report the range, never invent a record for it.
	switch {
	case textStart == 0:
	case rec == nil:
		p.warn(textStart, WarningUnparsedText, fmt.Sprintf("no LM Studio log records found (%s)", lineRange(textStart, textEnd)))
	default:
		p.warn(textStart, WarningUnparsedText, fmt.Sprintf("text before the first log header (%s)", lineRange(textStart, textEnd)))
	}
	return nil
}

func lineRange(start, end int) string {
	if start == end {
		return fmt.Sprintf("line %d", start)
	}
	return fmt.Sprintf("lines %d-%d", start, end)
}

// readLMLine returns the next physical line without its terminator and the
// line's size in bytes. A line over lmMaxRecordBytes keeps only enough bytes
// to show it is oversized. It returns io.EOF only when no bytes remain.
func readLMLine(br *bufio.Reader) (line []byte, size int, err error) {
	for {
		chunk, err := br.ReadSlice('\n')
		size += len(chunk)
		if keep := lmMaxRecordBytes + 1 - len(line); keep > 0 {
			line = append(line, chunk[:min(len(chunk), keep)]...)
		}
		switch {
		case errors.Is(err, bufio.ErrBufferFull):
			continue
		case errors.Is(err, io.EOF):
			if size == 0 {
				return nil, 0, io.EOF
			}
			return bytes.TrimSuffix(line, []byte("\r")), size, nil
		case err != nil:
			return nil, size, err
		}
		return bytes.TrimSuffix(bytes.TrimSuffix(line, []byte("\n")), []byte("\r")), size, nil
	}
}

func (p *lmParser) warn(line int, code, message string) {
	p.warnings = append(p.warnings, ParseWarning{Line: line, Code: code, Message: message})
}

// record classifies one record. The reported path is resolved first,
// independently of level; level decides only what is left (design table rows
// 1-5): an included request, a logged response, an unsupported POST path, a
// retained WARN/ERROR event, or an ignored routine record.
func (p *lmParser) record(rec *lmRecord) {
	if path, rest, ok := lmRequestLine(rec.message); ok && lmIncludedEndpoints[path] {
		tail, hasBody := strings.CutPrefix(rest, lmBodySuffix)
		if hasBody || strings.TrimSpace(rest) == "" {
			p.request(rec, path, hasBody, tail)
			return
		}
		p.warn(rec.start, WarningUnrecognizedRecord, fmt.Sprintf("record names %s but is not a recognised request record (%s)", path, rec.lines()))
	}
	if tail, ok := lmResponseBody(rec.message); ok {
		p.response(rec, tail)
		return
	}
	if strings.HasPrefix(rec.message, "Generated prediction") || strings.HasPrefix(rec.message, "Generated synchronous") {
		p.warn(rec.start, WarningUnrecognizedRecord, fmt.Sprintf("record looks like a logged response but has no recognised marker (%s)", rec.lines()))
	}
	if m := lmPostPathPattern.FindStringSubmatch(rec.message); m != nil && !lmIncludedEndpoints[m[1]] && !lmExcludedEndpoints[m[1]] {
		p.unsupportedEndpoint(rec, m[1])
		return
	}
	switch strings.ToUpper(rec.level) {
	case "WARN", "WARNING", "ERROR":
		p.diagnostic(rec)
	default:
		p.headerLikeLines(rec)
	}
}

// lmResponseBody returns the text after a logged-response marker.
func lmResponseBody(message string) (string, bool) {
	if tail, ok := strings.CutPrefix(message, lmPredictionMarker); ok {
		return tail, true
	}
	return strings.CutPrefix(message, lmSyncResponseMarker)
}

// headerLikeLines warns about continuation lines that look like a malformed
// header in a record without a JSON body (requirement 6.3).
func (p *lmParser) headerLikeLines(rec *lmRecord) {
	for _, line := range rec.headerLike {
		p.warn(line, WarningUnparsedText, "malformed log header")
	}
}

func (p *lmParser) request(rec *lmRecord, endpoint string, hasBody bool, tail string) {
	req := LMStudioRequest{
		ID:        strconv.Itoa(rec.start),
		StartLine: rec.start,
		EndLine:   rec.end,
		Size:      rec.size,
		Endpoint:  endpoint,
		Timestamp: rec.wall,
	}
	if t, ok := p.zone.interpret(rec.wall); ok {
		req.ResolvedTime = t
	}

	var body json.RawMessage
	switch {
	case rec.oversized:
		req.BodyState = BodyStateOversized
		p.warnOversized(rec)
	case !hasBody:
		req.BodyState = BodyStateNotLogged
		if len(bytes.TrimSpace(rec.cont)) > 0 {
			p.warn(rec.start+1, WarningUnparsedText, fmt.Sprintf("text after a request without a body (%s)", lineRange(rec.start+1, rec.end)))
		}
	default:
		body, req.BodyState = p.decodeBody(rec, tail, "request")
	}

	if body != nil {
		tr := translateLMRequest(endpoint, body, "req:"+req.ID+":")
		req.Model = tr.model
		req.ThreadID, req.ParentThreadID, req.RequestKind = tr.threadID, tr.parentThreadID, tr.requestKind
		req.Parameters = tr.parameters
		timestamp := ""
		if !req.ResolvedTime.IsZero() {
			timestamp = req.ResolvedTime.Format(time.RFC3339)
		}
		for _, item := range tr.items {
			p.entries = append(p.entries, Entry{
				Type:            item.entryType,
				Message:         &Message{Role: item.role, Content: item.content},
				Timestamp:       timestamp,
				Model:           req.Model,
				Provider:        lmProvider,
				RequestID:       req.ID,
				SuppliedContext: item.entryType == "assistant",
				ItemDigest:      item.digest,
			})
		}
		for _, w := range tr.warnings {
			p.warn(rec.start, w.code, w.message)
		}
	}
	p.requests = append(p.requests, req)
}

// response records a logged response as an unattributed assistant entry.
func (p *lmParser) response(rec *lmRecord, tail string) {
	e := p.logEntry(rec, "assistant")
	e.Unattributed = true
	var body json.RawMessage
	state := BodyStateOversized
	if rec.oversized {
		p.warnOversized(rec)
	} else {
		body, state = p.decodeBody(rec, tail, "response")
	}
	if body == nil {
		e.Message.Content = []ContentItem{{Type: "event", EventType: "response", Incomplete: state == BodyStateIncomplete}}
		p.entries = append(p.entries, e)
		return
	}
	content, model, warnings := translateLMResponse(body, "evt:"+strconv.Itoa(rec.start)+":")
	e.Message.Content = content
	if e.Model == "" {
		e.Model = model
	}
	for _, w := range warnings {
		p.warn(rec.start, w.code, w.message)
	}
	p.entries = append(p.entries, e)
}

// diagnostic retains a WARN or ERROR record as an unattributed entry: its
// level as the item status and its text as logged (requirements 4.1, 4.2).
func (p *lmParser) diagnostic(rec *lmRecord) {
	e := p.logEntry(rec, "log")
	e.Unattributed = true
	item := ContentItem{Type: "text", Status: rec.level}
	if strings.EqualFold(rec.level, "ERROR") {
		item.Type = "error"
	}
	if rec.oversized {
		p.warnOversized(rec)
	} else {
		item.Text = rec.message + string(rec.cont)
		p.headerLikeLines(rec)
	}
	e.Message.Content = []ContentItem{item}
	p.entries = append(p.entries, e)
}

// unsupportedEndpoint keeps a body-free record of a POST path that is neither
// translated nor excluded, counted into one warning per path (requirement 4.6).
func (p *lmParser) unsupportedEndpoint(rec *lmRecord, path string) {
	i := slices.IndexFunc(p.unsupported, func(u *lmUnsupported) bool { return u.path == path })
	if i < 0 {
		p.unsupported = append(p.unsupported, &lmUnsupported{path: path, line: rec.start})
		i = len(p.unsupported) - 1
	}
	p.unsupported[i].count++

	e := p.logEntry(rec, "log")
	payload, _ := json.Marshal(map[string]string{"path": path})
	e.Message.Content = []ContentItem{{Type: "event", EventType: "unsupported_endpoint", Payload: payload}}
	p.entries = append(p.entries, e)
	p.headerLikeLines(rec)
}

// logEntry starts an entry for a log-level record. Its first tag is kept as
// the recorded model tag, informational only and never an attribution key.
func (p *lmParser) logEntry(rec *lmRecord, entryType string) Entry {
	e := Entry{
		Type:       entryType,
		Message:    &Message{Role: entryType},
		Provider:   lmProvider,
		SourceLine: rec.start,
		WallTime:   rec.wall,
	}
	if t, ok := p.zone.interpret(rec.wall); ok {
		e.Timestamp = t.Format(time.RFC3339)
	}
	if len(rec.tags) > 0 {
		e.Model = rec.tags[0]
	}
	return e
}

func (p *lmParser) warnOversized(rec *lmRecord) {
	p.warn(rec.start, WarningOversizedRecord, fmt.Sprintf(
		"record of %d bytes exceeds the %d-byte parser limit; body skipped (%s)", rec.size, lmMaxRecordBytes, rec.lines()))
}

// decodeBody decodes the JSON value that starts after a request's ` with body`
// or a response marker. A value unterminated at the end of the input is
// incomplete and may still be growing (requirement 6.1), so it has no warning;
// any other decode failure is malformed.
func (p *lmParser) decodeBody(rec *lmRecord, tail, what string) (json.RawMessage, string) {
	body := append([]byte(tail), rec.cont...)
	dec := json.NewDecoder(bytes.NewReader(body))
	var raw json.RawMessage
	err := dec.Decode(&raw)
	switch {
	case err == nil:
	case rec.last && (errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF)):
		return nil, BodyStateIncomplete
	default:
		p.warn(rec.start, WarningInvalidRecord, fmt.Sprintf("%s body is not valid JSON (%s)", what, rec.lines()))
		return nil, BodyStateMalformed
	}

	offset := int(dec.InputOffset())
	if i := bytes.IndexFunc(body[offset:], func(r rune) bool { return !strings.ContainsRune(" \t\r\n", r) }); i >= 0 {
		line := rec.start + bytes.Count(body[:offset+i], []byte("\n"))
		p.warn(line, WarningUnparsedText, fmt.Sprintf("text after the %s body (%s)", what, lineRange(line, rec.end)))
	}
	return raw, ""
}

// markTruncated flags content holding LM Studio's truncation marker anywhere
// in its text or values (requirement 2.6). The content itself is unchanged.
func markTruncated(item *ContentItem) {
	item.Truncated = item.Truncated ||
		strings.Contains(item.Text, lmTruncationMarker) ||
		strings.Contains(item.Thinking, lmTruncationMarker) ||
		strings.Contains(item.Content, lmTruncationMarker) ||
		strings.Contains(item.Name, lmTruncationMarker) ||
		bytes.Contains(item.Payload, []byte(lmTruncationMarker)) ||
		valueHasTruncation(item.Input)
}

func valueHasTruncation(v any) bool {
	switch v := v.(type) {
	case string:
		return strings.Contains(v, lmTruncationMarker)
	case []any:
		return slices.ContainsFunc(v, valueHasTruncation)
	case map[string]any:
		for key, value := range v {
			if strings.Contains(key, lmTruncationMarker) || valueHasTruncation(value) {
				return true
			}
		}
	}
	return false
}

// lmEntriesByRequest groups request entries by their request ID, in order.
func lmEntriesByRequest(entries []Entry) map[string][]Entry {
	byRequest := make(map[string][]Entry)
	for _, e := range entries {
		if e.RequestID != "" {
			byRequest[e.RequestID] = append(byRequest[e.RequestID], e)
		}
	}
	return byRequest
}

// annotateExtensions compares each threaded request's logged items with its
// thread predecessor's, in the order given (requirements 3.5, 3.6). When the
// predecessor's complete item sequence is an unbroken leading prefix of the
// request's, PriorPrefixCount is its length; otherwise ContextDiscontinuity
// is set. Items are equal when their logged representations are, ignoring
// object-key order (Entry.ItemDigest). Singleton requests are never compared,
// and nothing is claimed when either body was not translated. Both the direct
// view and session assembly call this on their own merged order.
func annotateExtensions(requests []LMStudioRequest, entriesByRequest map[string][]Entry) {
	previous := make(map[string]int) // thread ID -> index of its latest request
	for i := range requests {
		req := &requests[i]
		req.ContextDiscontinuity, req.PriorPrefixCount = false, 0
		if req.ThreadID == "" {
			continue
		}
		j, seen := previous[req.ThreadID]
		previous[req.ThreadID] = i
		if !seen || requests[j].BodyState != "" || req.BodyState != "" {
			continue
		}
		prior, current := entriesByRequest[requests[j].ID], entriesByRequest[req.ID]
		if len(prior) <= len(current) && slices.EqualFunc(prior, current[:len(prior)], func(a, b Entry) bool {
			return a.ItemDigest == b.ItemDigest
		}) {
			req.PriorPrefixCount = len(prior)
		} else {
			req.ContextDiscontinuity = true
		}
	}
}
