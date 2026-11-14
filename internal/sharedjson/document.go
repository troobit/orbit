// Package sharedjson projects apsis's shared transcript representation into
// versioned, harness-neutral JSON documents described by
// docs/schemas/apsis-shared-json.v1.schema.json.
package sharedjson

import (
	"encoding/json"
	"io"

	"github.com/arjenschwarz/orbit/internal/transcript"
)

// SchemaVersion is the published schema version. Removing a field or changing
// its meaning raises it; additive fields and block types do not.
const SchemaVersion = 1

// SessionDocument is a session: metadata, ordered entries and parse warnings.
type SessionDocument struct {
	SchemaVersion int       `json:"schema_version"`
	Kind          string    `json:"kind"`
	Session       Metadata  `json:"session"`
	Requests      []Request `json:"requests,omitempty"` // LM Studio request boundaries, in source or discovery order
	Entries       []Entry   `json:"entries"`
	Warnings      []Warning `json:"warnings"`

	// UnattributedSummary stands in for the unattributed events of the files
	// an LM Studio logical-session view draws on; direct views keep the
	// events as entries instead.
	UnattributedSummary []UnattributedSummary `json:"unattributed_summary,omitempty"`
}

// Metadata describes a session. Unavailable values are omitted.
type Metadata struct {
	Source    string `json:"source,omitempty"`
	ID        string `json:"id,omitempty"`
	ThreadID  string `json:"thread_id,omitempty"` // Codex exec thread, or LM Studio client thread on listing rows
	CreatedAt string `json:"created_at,omitempty"`
	TimeZone  string `json:"time_zone,omitempty"` // LM Studio: zone the logged wall times are assumed to be in
	Project   string `json:"project,omitempty"`
	Model     string `json:"model,omitempty"`
	Provider  string `json:"provider,omitempty"`
	Origin    string `json:"origin,omitempty"`
	Cost      *Cost  `json:"cost,omitempty"`
}

// Cost is a session's total cost in its unit.
type Cost struct {
	Amount float64 `json:"amount"`
	Unit   string  `json:"unit"`
}

// Entry is one entry in transcript order.
type Entry struct {
	ID        string `json:"id,omitempty"` // source record identifier, not a tool link
	Role      string `json:"role"`
	Timestamp string `json:"timestamp,omitempty"`
	Model     string `json:"model,omitempty"`
	Provider  string `json:"provider,omitempty"`
	Turn      int    `json:"turn,omitempty"` // 1-based Codex exec turn
	Content   []any  `json:"content"`        // one of the *Block types

	// LM Studio only.
	RequestID       string `json:"request_id,omitempty"`       // the request the entry was logged in; empty on a log-level entry
	SuppliedContext bool   `json:"supplied_context,omitempty"` // an assistant message resent as request input, not a new completion
	Unattributed    bool   `json:"unattributed,omitempty"`     // a logged response, WARN or ERROR whose request is unknown
	SourceLine      int    `json:"source_line,omitempty"`      // a log-level entry's physical start line
	WallTime        string `json:"wall_time,omitempty"`        // a log-level entry's wall time as logged, without a zone
}

// Request is one LM Studio request record. Entries translated from its body
// name it through request_id; a request whose body was not translated has
// none.
type Request struct {
	ID                   string          `json:"id"`
	SessionID            string          `json:"session_id,omitempty"` // listed logical session; the request's own ID for a singleton
	File                 string          `json:"file,omitempty"`       // logical-session views: root-relative file holding the record
	StartLine            int             `json:"start_line"`
	EndLine              int             `json:"end_line"`
	Endpoint             string          `json:"endpoint"`
	ThreadID             string          `json:"thread_id,omitempty"`
	ParentThreadID       string          `json:"parent_thread_id,omitempty"`
	RequestKind          string          `json:"request_kind,omitempty"`
	Model                string          `json:"model,omitempty"`
	WallTime             string          `json:"wall_time"`
	Timestamp            string          `json:"timestamp,omitempty"` // the wall time's one interpretation in the assumed zone, with its offset
	BodyState            string          `json:"body_state,omitempty"`
	Parameters           json.RawMessage `json:"parameters,omitempty"`
	ContextDiscontinuity bool            `json:"context_discontinuity,omitempty"`
	PriorPrefixCount     int             `json:"prior_prefix_count,omitempty"`
	ResponseAttribution  string          `json:"response_attribution"`
}

// UnattributedSummary is a body-free reference to one contributing file's
// unattributed events.
type UnattributedSummary struct {
	File     string      `json:"file"`
	NextFile string      `json:"next_file,omitempty"`
	Counts   EventCounts `json:"counts"`
	Lines    []int       `json:"lines"`
}

// EventCounts counts unattributed events by kind.
type EventCounts struct {
	Response int `json:"response"`
	Warn     int `json:"warn"`
	Error    int `json:"error"`
}

// BlockCommon holds the fields any block may carry.
type BlockCommon struct {
	ItemID     string `json:"item_id,omitempty"`
	Status     string `json:"status,omitempty"`
	Incomplete bool   `json:"incomplete,omitempty"`
	Truncated  bool   `json:"truncated,omitempty"` // the source cut some of this content
}

// TextBlock is a text or thinking block.
type TextBlock struct {
	Type string `json:"type"`
	BlockCommon
	Text string `json:"text"`
}

// ToolUseBlock is a tool call.
type ToolUseBlock struct {
	Type string `json:"type"`
	BlockCommon
	ID    string `json:"id,omitempty"`
	Name  string `json:"name"`
	Input any    `json:"input,omitempty"`
}

// ToolResultBlock is a tool result, linked to its call by ToolUseID.
type ToolResultBlock struct {
	Type string `json:"type"`
	BlockCommon
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error"`
	ExitCode  *int   `json:"exit_code,omitempty"` // emitted for 0; omitted only when unrecorded
}

// PlanBlock is a plan, such as a Codex exec to-do list.
type PlanBlock struct {
	Type string `json:"type"`
	BlockCommon
	Items []PlanItem `json:"items"`
}

// PlanItem is one step of a plan.
type PlanItem struct {
	Text      string `json:"text"`
	Completed bool   `json:"completed"`
}

// ErrorBlock is an error the source recorded.
type ErrorBlock struct {
	Type string `json:"type"`
	BlockCommon
	Message string `json:"message"`
}

// EventBlock keeps an event or item without a specialised translation.
type EventBlock struct {
	Type string `json:"type"`
	BlockCommon
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

// UsageBlock is token usage as the source reports it.
type UsageBlock struct {
	Type string `json:"type"`
	BlockCommon
	Basis    string   `json:"basis"`
	Counters Counters `json:"counters"`
}

// Counters is a usage counter object written in source key order, with each
// value as recorded.
type Counters []transcript.UsageCounter

// MarshalJSON writes the counters as one object, preserving their order.
func (c Counters) MarshalJSON() ([]byte, error) {
	buf := []byte{'{'}
	for i, counter := range c {
		if i > 0 {
			buf = append(buf, ',')
		}
		name, err := json.Marshal(counter.Name)
		if err != nil {
			return nil, err
		}
		buf = append(buf, name...)
		buf = append(buf, ':')
		buf = append(buf, counter.Value...)
	}
	return append(buf, '}'), nil
}

// OtherBlock carries a parsed content kind the schema has no block type for.
type OtherBlock struct {
	Type string `json:"type"`
	BlockCommon
	SourceType string `json:"source_type,omitempty"`
	Text       string `json:"text,omitempty"`
}

// Warning is a non-fatal problem with at most one one-based source position.
type Warning struct {
	Code            string `json:"code"`
	Message         string `json:"message"`
	Line            int    `json:"line,omitempty"`
	ChatMessage     int    `json:"chat_message,omitempty"`
	ExecutionAction int    `json:"execution_action,omitempty"`
}

// Write encodes doc with a two-space indent, no HTML escaping and a trailing newline.
func Write(w io.Writer, doc any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(doc)
}
