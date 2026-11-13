package transcript

import (
	"cmp"
	"encoding/json"
	"fmt"
	stdhtml "html"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Rendering of LM Studio log transcripts: request boundaries with their
// parameters, log-level events whose request attribution is unknown, and the
// time-zone and attribution notes each transcript states once. Markdown and
// HTML build every label from the same helpers, so both show the same
// content and states (requirement 4.5).

// LMStudioView is what an LM Studio transcript renders besides its entries.
type LMStudioView struct {
	Requests    []LMStudioRequest      // source order, or a session's merged discovery order
	Events      []LMStudioEventSummary // a session view's references to unattributed events; nil in a direct view
	TimeZone    string                 // label of the zone logged wall times are assumed to be in
	ZoneWarning string                 // set when a TZ override could not be loaded
}

// NewLMStudioView returns the render view of an LM Studio parse result, or
// nil for every other format.
func NewLMStudioView(res *ParseResult) *LMStudioView {
	if res == nil || res.Format != FormatLMStudio {
		return nil
	}
	view := &LMStudioView{Requests: res.LMStudioRequests, Events: res.LMStudioEvents}
	if res.Metadata != nil {
		view.TimeZone = res.Metadata.TimeZone
	}
	if i := slices.IndexFunc(res.Warnings, func(w ParseWarning) bool { return w.Code == WarningTimeZone }); i >= 0 {
		view.ZoneWarning = res.Warnings[i].Message
	}
	return view
}

const (
	lmAttributionNotice = "the log does not record which request produced a logged response, warning or error, so none is shown as a request's output."
	lmTruncatedLabel    = "✂️ (truncated in logs)"
)

// lmBodyStateLabels describe a request body that was not translated
// (requirements 2.5, 6.1, 6.4).
var lmBodyStateLabels = map[string]string{
	BodyStateNotLogged:  "body not logged",
	BodyStateMalformed:  "⚠ body malformed",
	BodyStateIncomplete: "⏳ body incomplete",
	BodyStateOversized:  "⚠ body over the parser size limit",
}

// lmNote is a statement made once per transcript rather than per request
// (requirements 2.4, 5.1, 5.5).
type lmNote struct {
	warning     bool
	label, text string
}

func (v *LMStudioView) notes() []lmNote {
	zone := lmNote{label: "Time zone:", text: v.TimeZone + ", assumed for every logged wall time; the log records no zone."}
	if v.ZoneWarning != "" {
		zone = lmNote{warning: true, label: "Time zone:", text: v.ZoneWarning}
	}
	return []lmNote{zone, {label: "Request attribution unknown:", text: lmAttributionNotice}}
}

// timeLabel shows a logged wall time as the assumed instant with its offset
// when the zone gives exactly one, otherwise as logged with the reason
// (requirements 5.1, 5.2).
func (v *LMStudioView) timeLabel(wall, assumed string) string {
	switch {
	case assumed != "":
		return assumed + " (assumed " + v.TimeZone + ")"
	case v.ZoneWarning != "":
		return wall + " (wall time; zone not loaded)"
	}
	return wall + " (wall time; no unique time in " + v.TimeZone + ")"
}

// lmPart is one part of a meta line: text, then an optional recorded value
// shown as code.
type lmPart struct{ text, code string }

func lmPartsMarkdown(parts []lmPart) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		s := p.text
		if p.code != "" {
			s = strings.TrimSpace(s + " " + lmCodeSpan(p.code))
		}
		out = append(out, s)
	}
	return strings.Join(out, " · ")
}

// lmCodeSpan wraps s in a Markdown code span whose delimiter is longer than
// any backtick run in s, so recorded text cannot end the span early.
func lmCodeSpan(s string) string {
	d := strings.Repeat("`", lmLongestBacktickRun(s)+1)
	if strings.HasPrefix(s, "`") || strings.HasSuffix(s, "`") {
		s = " " + s + " "
	}
	return d + s + d
}

// lmFence returns a code-fence marker longer than any backtick run in s, so
// recorded text cannot close the fence early.
func lmFence(s string) string {
	return strings.Repeat("`", max(3, lmLongestBacktickRun(s)+1))
}

func lmLongestBacktickRun(s string) int {
	longest, run := 0, 0
	for _, r := range s {
		if r == '`' {
			run++
			longest = max(longest, run)
		} else {
			run = 0
		}
	}
	return longest
}

// lmPartsHTML escapes every part, for HTML and for Markdown summary lines.
func lmPartsHTML(parts []lmPart) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		s := stdhtml.EscapeString(p.text)
		if p.code != "" {
			s = strings.TrimSpace(s + " <code>" + stdhtml.EscapeString(p.code) + "</code>")
		}
		out = append(out, s)
	}
	return strings.Join(out, " · ")
}

// requestParts describe a request boundary: source location, time, model,
// logical-session membership, body state and its relation to the thread's
// previous request (requirements 1.4, 2.2, 2.5, 3.6, 3.7).
func (v *LMStudioView) requestParts(req *LMStudioRequest) []lmPart {
	assumed := ""
	if !req.ResolvedTime.IsZero() {
		assumed = req.ResolvedTime.Format(time.RFC3339)
	}
	parts := []lmPart{{text: lineRange(req.StartLine, req.EndLine)}, {text: v.timeLabel(req.Timestamp, assumed)}}
	if req.File != "" {
		parts = append([]lmPart{{code: req.File}}, parts...)
	}
	if req.Model != "" {
		parts = append(parts, lmPart{"model", req.Model})
	}
	// Identifiers the sessions layer assigned, which open the logical session
	// and the request on their own; a singleton's session is its request.
	if req.SessionID != "" {
		parts = append(parts, lmPart{"session", req.SessionID})
		if req.ID != req.SessionID {
			parts = append(parts, lmPart{"request", req.ID})
		}
	}
	if req.ThreadID != "" {
		parts = append(parts, lmPart{"thread", req.ThreadID})
	} else {
		parts = append(parts, lmPart{text: "singleton session (no valid thread ID)"})
	}
	if req.ParentThreadID != "" {
		parts = append(parts, lmPart{"parent thread", req.ParentThreadID})
	}
	if req.RequestKind != "" {
		parts = append(parts, lmPart{"request kind", req.RequestKind})
	}
	if label := lmBodyStateLabels[req.BodyState]; label != "" {
		parts = append(parts, lmPart{text: label})
	}
	switch {
	case req.PriorPrefixCount > 0:
		parts = append(parts, lmPart{text: "extends prior request"})
	case req.ContextDiscontinuity:
		parts = append(parts, lmPart{text: "⚠ context discontinuity"})
	}
	return parts
}

// eventParts describe a log-level entry: its start line, time and recorded
// model tag, which is informational only (requirement 4.2).
func (v *LMStudioView) eventParts(e *Entry) []lmPart {
	parts := []lmPart{{text: lineRange(e.SourceLine, e.SourceLine)}, {text: v.timeLabel(e.WallTime, e.Timestamp)}}
	if e.Model != "" {
		parts = append(parts, lmPart{"model", e.Model})
	}
	return parts
}

const (
	lmEventsHeading = "Unattributed events in these log files"
	lmEventsNotice  = "Logged responses, warnings and errors in the files holding this session's requests " +
		"are counted here without content and are not claimed for this session. " +
		"This view, opened by source and session identifier with `apsis --all-projects -s lmstudio <session-id>`, shows only the session's requests; " +
		"converting a whole file with `apsis -a lmstudio <file>` shows every record it holds, from every client of the server, " +
		"and needs CLI access to the server's log files. Paths are relative to the LM Studio log directory."
)

// eventSummaryParts describe one file's unattributed events in a session
// view: the file, counts by kind, start lines and the next discovered file,
// which can hold later events (requirement 4.2).
func eventSummaryParts(s LMStudioEventSummary) []lmPart {
	parts := []lmPart{
		{code: s.File},
		{text: lmCount(s.Counts.Response, "response") + ", " + lmCount(s.Counts.Warn, "warning") + ", " + lmCount(s.Counts.Error, "error")},
	}
	if len(s.Lines) > 0 {
		lines := make([]string, len(s.Lines))
		for i, line := range s.Lines {
			lines[i] = strconv.Itoa(line)
		}
		label := "lines "
		if len(lines) == 1 {
			label = "line "
		}
		parts = append(parts, lmPart{text: label + strings.Join(lines, ", ")})
	}
	if s.NextFile != "" {
		parts = append(parts, lmPart{"next file", s.NextFile})
	} else {
		parts = append(parts, lmPart{text: "last discovered file"})
	}
	return parts
}

func lmCount(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return strconv.Itoa(n) + " " + noun + "s"
}

// lmPrefixText stands in for the logged items a request repeats from its
// thread's previous request (requirement 3.5).
func lmPrefixText(n int) string {
	if n == 1 {
		return "⤷ repeats the first item of the thread's previous request"
	}
	return fmt.Sprintf("⤷ repeats the first %d items of the thread's previous request", n)
}

// lmParameterSummary labels a request's parameters block with its stream
// setting and offered tool names where recorded (requirement 2.9).
func lmParameterSummary(params json.RawMessage) string {
	label := "Request parameters"
	var fields map[string]json.RawMessage
	if json.Unmarshal(params, &fields) != nil {
		return label
	}
	if stream, ok := fields["stream"]; ok {
		label += " · stream " + string(stream)
	}
	var tools []struct {
		Type     string `json:"type"`
		Name     string `json:"name"`
		Function struct {
			Name string `json:"name"`
		} `json:"function"`
	}
	if json.Unmarshal(fields["tools"], &tools) == nil {
		var names []string
		for _, t := range tools {
			if name := cmp.Or(t.Name, t.Function.Name, t.Type); name != "" {
				names = append(names, name)
			}
		}
		if len(names) > 0 {
			label += " · tools: " + strings.Join(names, ", ")
		}
	}
	return label
}

// lmEntryHeading names a request entry by its recorded role, marking an
// assistant message resent as input as supplied context (requirement 2.4).
// An entry holding only tool results has no heading, as tool output renders
// elsewhere.
func lmEntryHeading(e *Entry) (icon, role, note string, ok bool) {
	content := e.Message.Content
	if len(content) > 0 && !slices.ContainsFunc(content, func(c ContentItem) bool { return c.Type != "tool_result" }) {
		return "", "", "", false
	}
	if e.SuppliedContext {
		note = "supplied context"
	}
	switch e.Message.Role {
	case "user":
		return "👤", "User", note, true
	case "assistant":
		return "🤖", "Assistant", note, true
	case "system":
		return "⚙️", "System", note, true
	case "developer":
		return "⚙️", "Developer", note, true
	case "tool":
		return "🔧", "Tool output", note, true
	}
	return "👤", e.Message.Role, note, true
}

// lmEventLabel names a log-level entry: a logged response, or a retained
// WARN or ERROR record with its level as logged (requirement 4.2).
func lmEventLabel(e *Entry) (icon, label string) {
	if e.Type == "assistant" {
		return "🤖", "Unattributed · response"
	}
	level := ""
	if len(e.Message.Content) > 0 {
		level = e.Message.Content[0].Status
	}
	if strings.EqualFold(level, "ERROR") {
		return "❌", "Unattributed · " + level
	}
	return "⚠️", "Unattributed · " + level
}

// lmUnsupportedPath returns the path of a body-free unsupported-endpoint
// entry (requirement 4.6).
func lmUnsupportedPath(e *Entry) (string, bool) {
	if len(e.Message.Content) != 1 || e.Message.Content[0].EventType != "unsupported_endpoint" {
		return "", false
	}
	var payload struct {
		Path string `json:"path"`
	}
	_ = json.Unmarshal(e.Message.Content[0].Payload, &payload)
	return payload.Path, true
}

// lmSegment is one top-level block of a transcript: a request with its
// entries, or one log-level entry.
type lmSegment struct {
	request *LMStudioRequest
	entries []Entry
}

// shown returns the entries a request renders and how many leading entries
// it repeats from its thread's previous request instead.
func (s lmSegment) shown() (entries []Entry, repeated int) {
	n := min(s.request.PriorPrefixCount, len(s.entries))
	return s.entries[n:], n
}

// segments places requests and log-level entries in source order. Log-level
// entries occur only in the direct view of one input, where start lines are
// comparable; a session view holds requests alone, in its merged order. A
// request without entries still has its boundary (requirement 2.5).
func (v *LMStudioView) segments(entries []Entry) []lmSegment {
	byRequest := lmEntriesByRequest(entries)
	var segments []lmSegment
	next := 0
	requestsBefore := func(line int) {
		for ; next < len(v.Requests) && v.Requests[next].StartLine < line; next++ {
			req := &v.Requests[next]
			segments = append(segments, lmSegment{request: req, entries: byRequest[req.ID]})
		}
	}
	for i := range entries {
		if entries[i].RequestID == "" && entries[i].Message != nil {
			requestsBefore(entries[i].SourceLine)
			segments = append(segments, lmSegment{entries: entries[i : i+1]})
		}
	}
	requestsBefore(math.MaxInt)
	return segments
}

// lmRenderer holds the tool state one transcript render shares. Tool IDs are
// scoped to their request or event, so one map serves the whole transcript.
type lmRenderer struct {
	view     *LMStudioView
	toolMeta map[string]toolMetadata
	results  map[string]*ContentItem // tool_result by the tool_use ID it answers
	paired   map[string]bool         // tool_use IDs already rendered with their result
}

func newLMRenderer(view *LMStudioView, entries []Entry) *lmRenderer {
	results := make(map[string]*ContentItem)
	for i := range entries {
		if entries[i].Message == nil {
			continue
		}
		content := entries[i].Message.Content
		for j := range content {
			if content[j].Type == "tool_result" && content[j].ToolUseID != "" {
				results[content[j].ToolUseID] = &content[j]
			}
		}
	}
	// Prebuilt, so a result still finds a call that sits in a repeated prefix.
	return &lmRenderer{view: view, toolMeta: BuildToolMeta(entries), results: results, paired: make(map[string]bool)}
}

// pairedResult returns the result a tool call renders with at the call, so
// the call stays under the message that made it. A call without a result,
// such as a logged response's, renders on its own.
func (r *lmRenderer) pairedResult(item *ContentItem) *ContentItem {
	if item.ID == "" {
		return nil
	}
	return r.results[item.ID]
}

// renderLMStudioMarkdown renders an LM Studio transcript body as Markdown.
func renderLMStudioMarkdown(entries []Entry, view *LMStudioView) string {
	r := newLMRenderer(view, entries)
	var sb strings.Builder
	for _, n := range view.notes() {
		if n.warning {
			sb.WriteString("⚠️ ")
		}
		fmt.Fprintf(&sb, "**%s** %s\n\n", n.label, n.text)
	}
	sb.WriteString("---\n\n")
	for _, seg := range view.segments(entries) {
		if seg.request != nil {
			r.requestMarkdown(&sb, seg)
		} else {
			r.eventMarkdown(&sb, &seg.entries[0])
		}
		sb.WriteString("---\n\n")
	}
	if len(view.Events) > 0 {
		fmt.Fprintf(&sb, "## 📄 %s\n\n%s\n\n", lmEventsHeading, lmEventsNotice)
		for _, s := range view.Events {
			sb.WriteString("- " + lmPartsMarkdown(eventSummaryParts(s)) + "\n")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func (r *lmRenderer) requestMarkdown(sb *strings.Builder, seg lmSegment) {
	req := seg.request
	fmt.Fprintf(sb, "## 📨 Request · `POST %s`\n\n%s\n\n", req.Endpoint, lmPartsMarkdown(r.view.requestParts(req)))
	shown, repeated := seg.shown()
	if repeated > 0 {
		sb.WriteString(lmPrefixText(repeated) + "\n\n")
	}
	if len(req.Parameters) > 0 {
		payload := indentedPayload(req.Parameters)
		fence := lmFence(payload)
		fmt.Fprintf(sb, "<details>\n<summary>🎛️ %s</summary>\n\n%sjson\n%s\n%s\n\n</details>\n\n",
			escapeSummary(lmParameterSummary(req.Parameters)), fence, payload, fence)
	}
	for i := range shown {
		e := &shown[i]
		if e.Message == nil {
			continue
		}
		if icon, role, note, ok := lmEntryHeading(e); ok {
			fmt.Fprintf(sb, "### %s %s", icon, role)
			if note != "" {
				sb.WriteString(" · " + note)
			}
			sb.WriteString("\n\n")
		}
		sb.WriteString(r.itemsMarkdown(e.Message.Content))
	}
}

func (r *lmRenderer) eventMarkdown(sb *strings.Builder, e *Entry) {
	if path, ok := lmUnsupportedPath(e); ok {
		parts := append([]lmPart{{code: "POST " + path}}, r.view.eventParts(e)...)
		fmt.Fprintf(sb, "⚠️ **Unsupported endpoint** · %s\n\n", lmPartsMarkdown(parts))
		return
	}
	icon, label := lmEventLabel(e)
	parts := append([]lmPart{{text: label}}, r.view.eventParts(e)...)
	fmt.Fprintf(sb, "<details>\n<summary>%s %s</summary>\n\n", icon, lmPartsHTML(parts))
	if e.Type == "log" {
		// Log text as logged, not as Markdown.
		for _, item := range e.Message.Content {
			if item.Text != "" {
				fence := lmFence(item.Text)
				fmt.Fprintf(sb, "%s\n%s\n%s\n\n", fence, item.Text, fence)
			}
			if item.Truncated {
				sb.WriteString(lmTruncatedMarkdown)
			}
		}
	} else {
		sb.WriteString(r.itemsMarkdown(e.Message.Content))
	}
	sb.WriteString("</details>\n\n")
}

const lmTruncatedMarkdown = "✂️ *(truncated in logs)*\n\n"

func (r *lmRenderer) itemsMarkdown(content []ContentItem) string {
	var sb strings.Builder
	for i := range content {
		item := &content[i]
		switch item.Type {
		case "text":
			if strings.TrimSpace(item.Text) != "" {
				sb.WriteString(lmMarkdownSource(item.Text))
				sb.WriteString("\n\n")
			}
		case "thinking":
			if item.Thinking != "" {
				fmt.Fprintf(&sb, "<details>\n<summary>💭 Thinking</summary>\n\n%s\n\n</details>\n\n", lmMarkdownSource(item.Thinking))
			}
		case "tool_use":
			result := r.pairedResult(item)
			if result == nil {
				fmt.Fprintf(&sb, "<details>\n<summary>🔧 Tool call: %s</summary>\n\n%s</details>\n\n",
					escapeSummary(item.Name), formatToolInput(item.Name, item.Input))
				break
			}
			out := formatToolUse(item, r.toolMeta, nil)
			if out == "" {
				out = formatToolResult(result, r.toolMeta)
				r.paired[item.ID] = true
			}
			sb.WriteString(out)
		case "tool_result":
			if !r.paired[item.ToolUseID] {
				sb.WriteString(formatToolResult(item, r.toolMeta))
			}
		default:
			if out, ok := formatExecContentMarkdown(item); ok {
				sb.WriteString(out)
			}
		}
		if item.Truncated {
			sb.WriteString(lmTruncatedMarkdown)
		}
	}
	return sb.String()
}

// renderLMStudioHTML writes an LM Studio transcript body as HTML, shared by
// standalone documents and web fragments.
func renderLMStudioHTML(sb *strings.Builder, entries []Entry, view *LMStudioView) {
	r := newLMRenderer(view, entries)
	for _, n := range view.notes() {
		class, icon := "lm-note", ""
		if n.warning {
			class, icon = "lm-note warning", "⚠️ "
		}
		fmt.Fprintf(sb, "        <p class=\"%s\">%s<strong>%s</strong> %s</p>\n",
			class, icon, stdhtml.EscapeString(n.label), stdhtml.EscapeString(n.text))
	}
	for _, seg := range view.segments(entries) {
		if seg.request != nil {
			r.requestHTML(sb, seg)
		} else {
			r.eventHTML(sb, &seg.entries[0])
		}
	}
	if len(view.Events) > 0 {
		sb.WriteString("        <section class=\"message lm-events\">\n")
		sb.WriteString("            <div class=\"message-header\">\n")
		sb.WriteString("                <span class=\"icon\">📄</span>\n")
		fmt.Fprintf(sb, "                <span class=\"message-role\">%s</span>\n", lmEventsHeading)
		sb.WriteString("            </div>\n")
		sb.WriteString("            <div class=\"message-content\">\n")
		fmt.Fprintf(sb, "                <p class=\"lm-meta\">%s</p>\n", lmInlineCodeHTML(lmEventsNotice))
		sb.WriteString("                <ul>\n")
		for _, s := range view.Events {
			fmt.Fprintf(sb, "                    <li class=\"lm-meta\">%s</li>\n", lmPartsHTML(eventSummaryParts(s)))
		}
		sb.WriteString("                </ul>\n")
		sb.WriteString("            </div>\n")
		sb.WriteString("        </section>\n")
	}
}

// lmInlineCodeHTML escapes a fixed notice and turns its backquoted spans into
// code elements, so Markdown and HTML show the same command.
func lmInlineCodeHTML(s string) string {
	var sb strings.Builder
	code := false
	for part := range strings.SplitSeq(s, "`") {
		if code {
			sb.WriteString("<code>" + stdhtml.EscapeString(part) + "</code>")
		} else {
			sb.WriteString(stdhtml.EscapeString(part))
		}
		code = !code
	}
	return sb.String()
}

func (r *lmRenderer) requestHTML(sb *strings.Builder, seg lmSegment) {
	req := seg.request
	sb.WriteString("        <section class=\"message lm-request\">\n")
	sb.WriteString("            <div class=\"message-header\">\n")
	sb.WriteString("                <span class=\"icon\">📨</span>\n")
	fmt.Fprintf(sb, "                <span class=\"message-role\">Request <code>POST %s</code></span>\n", stdhtml.EscapeString(req.Endpoint))
	sb.WriteString("            </div>\n")
	sb.WriteString("            <div class=\"message-content\">\n")
	fmt.Fprintf(sb, "                <p class=\"lm-meta\">%s</p>\n", lmPartsHTML(r.view.requestParts(req)))
	shown, repeated := seg.shown()
	if repeated > 0 {
		fmt.Fprintf(sb, "                <p class=\"lm-meta\">%s</p>\n", stdhtml.EscapeString(lmPrefixText(repeated)))
	}
	if len(req.Parameters) > 0 {
		sb.WriteString("                <details class=\"tool-collapsible\">\n")
		fmt.Fprintf(sb, "                    <summary><span class=\"icon\">🎛️</span> %s</summary>\n", stdhtml.EscapeString(lmParameterSummary(req.Parameters)))
		sb.WriteString("                    <div class=\"tool-content\">\n")
		fmt.Fprintf(sb, "                        <pre><code>%s</code></pre>\n", stdhtml.EscapeString(indentedPayload(req.Parameters)))
		sb.WriteString("                    </div>\n")
		sb.WriteString("                </details>\n")
	}
	sb.WriteString("            </div>\n")
	sb.WriteString("        </section>\n")

	for i := range shown {
		e := &shown[i]
		if e.Message == nil {
			continue
		}
		items := r.itemsHTML(e.Message.Content, e.Type == "assistant")
		icon, role, note, ok := lmEntryHeading(e)
		if !ok {
			sb.WriteString(items)
			continue
		}
		fmt.Fprintf(sb, "        <section class=\"message %s\">\n", stdhtml.EscapeString(e.Type))
		sb.WriteString("            <div class=\"message-header\">\n")
		fmt.Fprintf(sb, "                <span class=\"icon\">%s</span>\n", icon)
		fmt.Fprintf(sb, "                <span class=\"message-role\">%s</span>\n", stdhtml.EscapeString(role))
		writeMessageMetaHTML(sb, "", note)
		sb.WriteString("            </div>\n")
		sb.WriteString("            <div class=\"message-content\">\n")
		sb.WriteString(items)
		sb.WriteString("            </div>\n")
		sb.WriteString("        </section>\n")
	}
}

func (r *lmRenderer) eventHTML(sb *strings.Builder, e *Entry) {
	if path, ok := lmUnsupportedPath(e); ok {
		parts := append([]lmPart{{code: "POST " + path}}, r.view.eventParts(e)...)
		fmt.Fprintf(sb, "        <p class=\"lm-note\">⚠️ <strong>Unsupported endpoint</strong> · %s</p>\n", lmPartsHTML(parts))
		return
	}
	icon, label := lmEventLabel(e)
	class := "tool-collapsible lm-event"
	if icon == "❌" {
		class += " error"
	}
	parts := append([]lmPart{{text: label}}, r.view.eventParts(e)...)
	fmt.Fprintf(sb, "        <details class=\"%s\">\n", class)
	fmt.Fprintf(sb, "            <summary><span class=\"icon\">%s</span> %s</summary>\n", icon, lmPartsHTML(parts))
	sb.WriteString("            <div class=\"tool-content\">\n")
	if e.Type == "log" {
		for _, item := range e.Message.Content {
			if item.Text != "" {
				fmt.Fprintf(sb, "                <pre><code>%s</code></pre>\n", stdhtml.EscapeString(item.Text))
			}
			if item.Truncated {
				sb.WriteString(lmTruncatedHTML)
			}
		}
	} else {
		sb.WriteString(r.itemsHTML(e.Message.Content, true))
	}
	sb.WriteString("            </div>\n")
	sb.WriteString("        </details>\n")
}

const lmTruncatedHTML = "                <p class=\"truncated\">" + lmTruncatedLabel + "</p>\n"

// lmMarkdownSource escapes the truncation marker in text read as Markdown,
// which would otherwise take it for an HTML tag and hide it.
func lmMarkdownSource(s string) string {
	return strings.ReplaceAll(s, lmTruncationMarker, `\`+lmTruncationMarker)
}

// itemsHTML renders content items; markdown renders text as Markdown, as for
// other sources' assistant messages, rather than as escaped paragraphs.
func (r *lmRenderer) itemsHTML(content []ContentItem, markdown bool) string {
	var sb strings.Builder
	for i := range content {
		item := &content[i]
		switch item.Type {
		case "text":
			switch {
			case strings.TrimSpace(item.Text) == "":
			case markdown:
				sb.WriteString("                <div class=\"markdown-content\">\n")
				sb.WriteString(markdownToHTML(lmMarkdownSource(item.Text)))
				sb.WriteString("                </div>\n")
			default:
				fmt.Fprintf(&sb, "                <p>%s</p>\n", stdhtml.EscapeString(item.Text))
			}
		case "thinking":
			if item.Thinking != "" {
				sb.WriteString("                <div class=\"thinking-block\">\n")
				sb.WriteString("                    <div class=\"thinking-header\">💭 Thinking</div>\n")
				sb.WriteString("                    <div class=\"thinking-content markdown-content\">\n")
				sb.WriteString(markdownToHTML(lmMarkdownSource(item.Thinking)))
				sb.WriteString("                    </div>\n")
				sb.WriteString("                </div>\n")
			}
		case "tool_use":
			if result := r.pairedResult(item); result != nil {
				out := formatToolUseHTML(item, r.toolMeta, nil)
				if out == "" {
					out, _ = toolResultBlockHTML(result, r.toolMeta)
					r.paired[item.ID] = true
				}
				sb.WriteString(out)
				break
			}
			sb.WriteString("                <details class=\"tool-collapsible\">\n")
			fmt.Fprintf(&sb, "                    <summary><span class=\"icon\">🔧</span> Tool call: %s</summary>\n", stdhtml.EscapeString(item.Name))
			sb.WriteString("                    <div class=\"tool-content\">\n")
			sb.WriteString(formatToolInputHTML(item.Name, item.Input))
			sb.WriteString("                    </div>\n")
			sb.WriteString("                </details>\n")
		case "tool_result":
			if !r.paired[item.ToolUseID] {
				block, _ := toolResultBlockHTML(item, r.toolMeta)
				sb.WriteString(block)
			}
		default:
			if out, ok := formatExecContentHTML(item); ok {
				sb.WriteString(out)
			}
		}
		if item.Truncated {
			sb.WriteString(lmTruncatedHTML)
		}
	}
	return sb.String()
}
