package transcript

import (
	"encoding/json"
	"time"
)

// Body states of an included LM Studio request record. An empty BodyState
// means the body decoded and its content was translated.
const (
	BodyStateNotLogged  = "not_logged" // the request header has no ` with body` suffix
	BodyStateMalformed  = "malformed"  // the body is not valid JSON
	BodyStateIncomplete = "incomplete" // the body is unterminated at end of input
	BodyStateOversized  = "oversized"  // the body exceeds the parser byte limit
)

// LMStudioRequest is one included request record's boundary metadata:
// endpoint, thread identity, parameters, body state and the extension result
// against its thread predecessor. It is parallel to ParseResult.Entries
// rather than embedded in them, so it is written once per request instead of
// on every entry the request produced. Entries belong to a request through
// Entry.RequestID.
type LMStudioRequest struct {
	ID                   string          // the parser's temporary StartLine key until the sessions layer assigns the final ID
	StartLine, EndLine   int             // one-based physical line range of the record
	Endpoint             string          // the logged POST path
	ThreadID             string          // client_metadata.thread_id as recorded; empty = singleton session
	ParentThreadID       string          // client_metadata parent-thread identifier, when recorded
	RequestKind          string          // client_metadata recorded role (dialogue turn, compaction, memory, side request)
	Model                string          // recorded request model
	Timestamp            string          // wall time as logged
	ResolvedTime         time.Time       // zero when the wall time has no unique interpretation
	BodyState            string          // "" or a BodyState* value
	Parameters           json.RawMessage // remaining top-level body fields, including client_metadata and server-held-context references
	ContextDiscontinuity bool            // the logged input does not extend the thread predecessor's
	PriorPrefixCount     int             // leading items identical to the predecessor's; 0 = none
}
