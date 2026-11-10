package transcript

import (
	"slices"
	"strings"
)

// AssignLMStudioIDs replaces the parser's temporary request keys in one
// file's parse result with final identifiers: each request's ID and
// SessionID, its entries' RequestID, and its request-scoped tool IDs. assign
// returns a request's final ID and logical session. It must run before parse
// results from several files are merged, since temporary keys repeat across
// files.
func AssignLMStudioIDs(res *ParseResult, assign func(LMStudioRequest) (id, sessionID string)) {
	final := make(map[string]string, len(res.LMStudioRequests))
	for i := range res.LMStudioRequests {
		req := &res.LMStudioRequests[i]
		id, sessionID := assign(*req)
		final[req.ID] = id
		req.ID, req.SessionID = id, sessionID
	}
	for i := range res.Entries {
		e := &res.Entries[i]
		id, ok := final[e.RequestID]
		if e.RequestID == "" || !ok {
			continue
		}
		oldScope, newScope := "req:"+e.RequestID+":", "req:"+id+":"
		e.RequestID = id
		if e.Message == nil {
			continue
		}
		for j := range e.Message.Content {
			item := &e.Message.Content[j]
			if rest, ok := strings.CutPrefix(item.ID, oldScope); ok {
				item.ID = newScope + rest
			}
			if rest, ok := strings.CutPrefix(item.ToolUseID, oldScope); ok {
				item.ToolUseID = newScope + rest
			}
		}
	}
}

// LMStudioSessionPart is one file's contribution to a logical-session view:
// its ParseLMStudio result with final identifiers assigned, and where the
// file sits in discovery order.
type LMStudioSessionPart struct {
	File     string // root-relative path of the file
	NextFile string // root-relative path of the next discovered file; "" for the last
	Result   *ParseResult
}

// AssembleLMStudioSession builds a logical-session view from the files
// holding it, in discovery order (requirements 1.5, 4.2): the requests keep
// selects, labelled with their file, and their entries, with history prefixes
// compared again in merged order (3.5); each file's unattributed events
// replaced by a body-free summary; and only the warnings of the selected
// records, labelled with their file. A file whose parse has no selected
// request contributes nothing. keep must select one logical session or one
// request, as the metadata takes the first request's thread ID as the view's.
func AssembleLMStudioSession(parts []LMStudioSessionPart, keep func(LMStudioRequest) bool) *ParseResult {
	res := &ParseResult{
		Entries:          []Entry{},
		Warnings:         []ParseWarning{},
		Format:           FormatLMStudio,
		LMStudioRequests: []LMStudioRequest{},
		LMStudioEvents:   []LMStudioEventSummary{},
	}
	var zone *ParseWarning
	timeZone := ""
	for _, part := range parts {
		var requests []LMStudioRequest
		for _, req := range part.Result.LMStudioRequests {
			if keep(req) {
				req.File = part.File
				requests = append(requests, req)
			}
		}
		if len(requests) == 0 {
			continue
		}
		if part.Result.Metadata != nil && timeZone == "" {
			timeZone = part.Result.Metadata.TimeZone
		}
		res.LMStudioRequests = append(res.LMStudioRequests, requests...)

		kept := make(map[string]bool, len(requests))
		for _, req := range requests {
			kept[req.ID] = true
		}
		for _, e := range part.Result.Entries {
			if kept[e.RequestID] {
				res.Entries = append(res.Entries, e)
			}
		}
		counts, lines := CountLMStudioEvents(part.Result.Entries)
		res.LMStudioEvents = append(res.LMStudioEvents, LMStudioEventSummary{
			File: part.File, NextFile: part.NextFile, Counts: counts, Lines: lines,
		})

		for _, w := range part.Result.Warnings {
			switch {
			case w.Code == WarningTimeZone:
				if zone == nil {
					zone = &w
				}
			case slices.ContainsFunc(requests, func(req LMStudioRequest) bool {
				return req.StartLine <= w.Line && w.Line <= req.EndLine
			}):
				w.Message = part.File + ": " + w.Message
				res.Warnings = append(res.Warnings, w)
			}
		}
	}
	annotateExtensions(res.LMStudioRequests, lmEntriesByRequest(res.Entries))
	if zone != nil {
		// Once per transcript, not per file (requirement 5.5).
		res.Warnings = append([]ParseWarning{*zone}, res.Warnings...)
	}
	res.Metadata = lmMetadata(res.LMStudioRequests, timeZone)
	if len(res.LMStudioRequests) > 0 {
		// One thread, or one request: the view's correlation ID (3.7).
		res.Metadata.ThreadID = res.LMStudioRequests[0].ThreadID
	}
	return res
}

// CountLMStudioEvents counts a file's unattributed events by kind and returns
// their start lines in source order: a logged response, an ERROR record, or
// otherwise a WARN record (requirement 4.2).
func CountLMStudioEvents(entries []Entry) (LMStudioEventCounts, []int) {
	var counts LMStudioEventCounts
	lines := []int{}
	for _, e := range entries {
		if !e.Unattributed {
			continue
		}
		switch {
		case e.Type == "assistant":
			counts.Response++
		case e.Message != nil && len(e.Message.Content) > 0 && e.Message.Content[0].Type == "error":
			counts.Error++
		default:
			counts.Warn++
		}
		lines = append(lines, e.SourceLine)
	}
	return counts, lines
}
