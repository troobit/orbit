package transcript

import (
	"bytes"
	"cmp"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Translation of LM Studio request and response bodies into shared entries.
// Request shapes follow the public endpoint documentation cited in
// testdata/lmstudio/PROVENANCE.md.

// lmConversationFields names the body fields each endpoint's conversation is
// read from, separate system text first (requirement 2.2). Every other
// top-level field except a string model stays in Parameters (requirement 2.9).
var lmConversationFields = map[string][]string{
	"/v1/chat/completions":     {"messages"},
	"/api/v0/chat/completions": {"messages"},
	"/v1/completions":          {"prompt"},
	"/api/v0/completions":      {"prompt"},
	"/v1/messages":             {"system", "messages"},
	"/v1/responses":            {"instructions", "input"},
	"/api/v1/chat":             {"system_prompt", "input"},
}

// lmItem is one logged conversation item: an element of messages, input or
// prompt, a string input or prompt, or a separate instructions, system or
// system_prompt field. Each becomes one entry, so a request's entries line up
// with its logged items for history-prefix comparison.
type lmItem struct {
	entryType string // "user", "assistant" or "system"
	role      string // as recorded; "tool" for tool outputs
	content   []ContentItem
	digest    [32]byte
}

type lmWarning struct {
	code, message string
}

// lmRequestTranslation is what a decoded request body yields.
type lmRequestTranslation struct {
	model                                 string
	threadID, parentThreadID, requestKind string
	parameters                            json.RawMessage
	items                                 []lmItem
	warnings                              []lmWarning
}

// lmTranslator builds content for one request occurrence or one logged
// response. Every tool identifier it writes is prefixed with scope, so tool
// links never leave the occurrence even when clients reuse identifiers; the
// recorded identifier stays on ItemID (requirement 2.8).
type lmTranslator struct {
	scope    string
	warnings []lmWarning
}

func (t *lmTranslator) warn(code, format string, args ...any) {
	t.warnings = append(t.warnings, lmWarning{code: code, message: fmt.Sprintf(format, args...)})
}

func (t *lmTranslator) toolUse(id, name string, input any) ContentItem {
	scoped := ""
	if id != "" {
		scoped = t.scope + id
	}
	return ContentItem{Type: "tool_use", ID: scoped, ItemID: id, Name: name, Input: input}
}

func (t *lmTranslator) toolResult(id, content string, isError bool) ContentItem {
	scoped := ""
	if id != "" {
		scoped = t.scope + id
	}
	return ContentItem{Type: "tool_result", ToolUseID: scoped, ItemID: id, Content: content, IsError: isError}
}

// unrecognized keeps a value apsis has no translation for as an event block
// with its payload, and warns (requirement 2.3).
func (t *lmTranslator) unrecognized(what, kind string, raw json.RawMessage) ContentItem {
	t.warn(WarningUnrecognizedRecord, "unrecognized %s: %s", what, kind)
	return ContentItem{Type: "event", EventType: kind, Payload: compactJSON(raw)}
}

func textItem(s string) ContentItem { return ContentItem{Type: "text", Text: s} }

// newItem starts the item for one logged value of a conversation field.
func newItem(field string, raw json.RawMessage, role string) lmItem {
	entryType := "user"
	switch role {
	case "assistant":
		entryType = "assistant"
	case "system", "developer":
		entryType = "system"
	}
	return lmItem{entryType: entryType, role: role, digest: lmDigest(field, raw)}
}

// invalidItem keeps a logged item whose shape does not decode as one event.
func (t *lmTranslator) invalidItem(field string, raw json.RawMessage, problem string) lmItem {
	t.warn(WarningInvalidRecord, "%s %s", field, problem)
	item := newItem(field, raw, "user")
	item.content = []ContentItem{{Type: "event", EventType: field, Payload: compactJSON(raw)}}
	return item
}

// lmDigest identifies a logged item by its field and its value with object
// keys sorted, so key order is irrelevant while array order and string
// contents stay significant (requirement 3.5).
func lmDigest(field string, raw json.RawMessage) [32]byte {
	canonical := []byte(compactJSON(raw))
	if v := decodeJSONValue(raw); v != nil {
		if b, err := json.Marshal(v); err == nil {
			canonical = b
		}
	}
	return sha256.Sum256(append([]byte(field+"\x00"), canonical...))
}

// translateLMRequest translates a decoded request body: the recorded model,
// thread metadata, parameters and one item per logged conversation value. A
// body that is not a JSON object becomes one generic item.
func translateLMRequest(endpoint string, body json.RawMessage, scope string) lmRequestTranslation {
	t := &lmTranslator{scope: scope}
	fields, ok := orderedFields(body)
	if !ok {
		t.warn(WarningUnrecognizedRecord, "%s request body is not a JSON object", endpoint)
		item := newItem("body", body, "user")
		item.content = []ContentItem{{Type: "event", EventType: "request_body", Payload: compactJSON(body)}}
		return lmRequestTranslation{items: []lmItem{item}, warnings: t.warnings}
	}

	out := lmRequestTranslation{}
	consumed := map[string]bool{}
	if raw, ok := fieldValue(fields, "model"); ok {
		if model, ok := jsonString(raw); ok {
			out.model = model
			consumed["model"] = true
		}
	}
	if raw, ok := fieldValue(fields, "client_metadata"); ok {
		out.threadID, out.parentThreadID, out.requestKind = lmClientMetadata(raw)
	}

	conversation := lmConversationFields[endpoint]
	if required := conversation[len(conversation)-1]; !hasField(fields, required) {
		t.warn(WarningInvalidRecord, "%s request body has no %s", endpoint, required)
	}
	for _, field := range conversation {
		raw, ok := fieldValue(fields, field)
		if !ok {
			continue
		}
		consumed[field] = true
		switch {
		case field == "messages" && endpoint == "/v1/messages":
			out.items = append(out.items, t.array(field, raw, t.anthropicMessage)...)
		case field == "messages":
			out.items = append(out.items, t.array(field, raw, t.chatMessage)...)
		case field == "prompt":
			out.items = append(out.items, t.prompt(field, raw)...)
		case field == "system":
			out.items = append(out.items, t.anthropicSystem(field, raw))
		case field == "input" && endpoint == "/api/v1/chat":
			out.items = append(out.items, t.input(field, raw, t.nativeInputItem)...)
		case field == "input":
			out.items = append(out.items, t.input(field, raw, t.responsesItem)...)
		default: // instructions, system_prompt
			out.items = append(out.items, t.systemString(field, raw))
		}
	}

	out.parameters = lmParameters(fields, consumed)
	out.warnings = t.warnings
	return out
}

func hasField(fields []jsonField, key string) bool {
	_, ok := fieldValue(fields, key)
	return ok
}

// lmParameters keeps every unconsumed top-level field in source order, with
// values as logged (requirement 2.9). It returns nil when none remain.
func lmParameters(fields []jsonField, consumed map[string]bool) json.RawMessage {
	var buf bytes.Buffer
	for _, f := range fields {
		if consumed[f.key] {
			continue
		}
		if buf.Len() == 0 {
			buf.WriteByte('{')
		} else {
			buf.WriteByte(',')
		}
		key, _ := json.Marshal(f.key)
		buf.Write(key)
		buf.WriteByte(':')
		buf.Write(f.value)
	}
	if buf.Len() == 0 {
		return nil
	}
	buf.WriteByte('}')
	return buf.Bytes()
}

// lmClientMetadata reads request identity from client_metadata (requirements
// 3.1, 3.7). The flat thread_id groups requests; the parent thread comes from
// Codex's x-codex-parent-thread-id key and the request kind from request_kind
// inside its x-codex-turn-metadata JSON string (pinned Codex
// responses_metadata.rs). A missing, truncated or unparseable value yields
// "", and every value stays in Parameters as logged.
func lmClientMetadata(raw json.RawMessage) (threadID, parentThreadID, requestKind string) {
	var metadata map[string]json.RawMessage
	if json.Unmarshal(raw, &metadata) != nil {
		return "", "", ""
	}
	threadID = lmIdentifier(metadata["thread_id"])
	parentThreadID = lmIdentifier(metadata["x-codex-parent-thread-id"])
	if blob := lmIdentifier(metadata["x-codex-turn-metadata"]); blob != "" {
		var turn map[string]json.RawMessage
		if json.Unmarshal([]byte(blob), &turn) == nil {
			requestKind = lmIdentifier(turn["request_kind"])
			parentThreadID = cmp.Or(parentThreadID, lmIdentifier(turn["parent_thread_id"]))
		}
	}
	return threadID, parentThreadID, requestKind
}

// lmIdentifier returns a nonempty, untruncated JSON string value, else "".
// Equal truncated representations do not show the original values matched.
func lmIdentifier(raw json.RawMessage) string {
	s, ok := jsonString(raw)
	if !ok || strings.Contains(s, lmTruncationMarker) {
		return ""
	}
	return s
}

// array translates each element of a JSON array field.
func (t *lmTranslator) array(field string, raw json.RawMessage, element func(string, json.RawMessage) lmItem) []lmItem {
	var elements []json.RawMessage
	if json.Unmarshal(raw, &elements) != nil {
		return []lmItem{t.invalidItem(field, raw, "is not an array")}
	}
	items := make([]lmItem, 0, len(elements))
	for _, e := range elements {
		items = append(items, element(field, e))
	}
	return items
}

// input translates a string input as user text and an array input item by item.
func (t *lmTranslator) input(field string, raw json.RawMessage, element func(string, json.RawMessage) lmItem) []lmItem {
	if s, ok := jsonString(raw); ok {
		item := newItem(field, raw, "user")
		item.content = []ContentItem{textItem(s)}
		return []lmItem{item}
	}
	return t.array(field, raw, element)
}

// systemString translates instructions or system_prompt.
func (t *lmTranslator) systemString(field string, raw json.RawMessage) lmItem {
	s, ok := jsonString(raw)
	if !ok {
		return t.invalidItem(field, raw, "is not a string")
	}
	item := newItem(field, raw, "system")
	item.content = []ContentItem{textItem(s)}
	return item
}

// prompt translates a completions prompt: a string, or one item per string
// of an array.
func (t *lmTranslator) prompt(field string, raw json.RawMessage) []lmItem {
	if _, ok := jsonString(raw); ok {
		return []lmItem{t.promptString(field, raw)}
	}
	return t.array(field, raw, func(field string, e json.RawMessage) lmItem { return t.promptString(field, e) })
}

func (t *lmTranslator) promptString(field string, raw json.RawMessage) lmItem {
	s, ok := jsonString(raw)
	if !ok {
		return t.invalidItem(field, raw, "holds a value that is not a string")
	}
	item := newItem(field, raw, "user")
	item.content = []ContentItem{textItem(s)}
	return item
}

// lmKnownRoles are the message roles apsis translates.
var lmKnownRoles = map[string]bool{"user": true, "assistant": true, "system": true, "developer": true, "tool": true}

// messageRole decodes a message's role, warning about roles apsis does not know.
func (t *lmTranslator) messageRole(field string, raw json.RawMessage) (string, bool) {
	var m struct {
		Role *string `json:"role"`
	}
	if json.Unmarshal(raw, &m) != nil || m.Role == nil || *m.Role == "" {
		return "", false
	}
	if !lmKnownRoles[*m.Role] {
		t.warn(WarningUnrecognizedRecord, "unrecognized %s role: %s", field, *m.Role)
	}
	return *m.Role, true
}

// chatMessage translates an OpenAI chat message: reasoning, content, tool
// calls, or a role "tool" output.
func (t *lmTranslator) chatMessage(field string, raw json.RawMessage) lmItem {
	role, ok := t.messageRole(field, raw)
	if !ok {
		return t.invalidItem(field, raw, "element has no role")
	}
	var m struct {
		Content          json.RawMessage   `json:"content"`
		ReasoningContent string            `json:"reasoning_content"`
		Reasoning        string            `json:"reasoning"`
		ToolCalls        []json.RawMessage `json:"tool_calls"`
		ToolCallID       string            `json:"tool_call_id"`
	}
	if json.Unmarshal(raw, &m) != nil {
		return t.invalidItem(field, raw, "element does not decode")
	}
	item := newItem(field, raw, role)
	item.content = t.chatContent(role, m.ReasoningContent, m.Reasoning, m.Content, m.ToolCallID, m.ToolCalls)
	return item
}

// chatContent builds an OpenAI chat message's content. It also translates a
// chat completion response choice's message.
func (t *lmTranslator) chatContent(role, reasoningContent, reasoning string, content json.RawMessage, toolCallID string, toolCalls []json.RawMessage) []ContentItem {
	var items []ContentItem
	if thinking := cmp.Or(reasoningContent, reasoning); thinking != "" {
		items = append(items, ContentItem{Type: "thinking", Thinking: thinking})
	}
	if role == "tool" {
		items = append(items, t.toolResult(toolCallID, resultText(content), false))
	} else {
		items = append(items, t.parts(content, "chat content part type", "text")...)
	}
	for _, call := range toolCalls {
		items = append(items, t.chatToolCall(call))
	}
	return items
}

// parts translates string content or an array of typed parts, where the
// textTypes carry a text field.
func (t *lmTranslator) parts(raw json.RawMessage, what string, textTypes ...string) []ContentItem {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	if s, ok := jsonString(raw); ok {
		if s == "" {
			return nil
		}
		return []ContentItem{textItem(s)}
	}
	var parts []json.RawMessage
	if json.Unmarshal(raw, &parts) != nil {
		return []ContentItem{t.unrecognized("content shape", "content", raw)}
	}
	items := make([]ContentItem, 0, len(parts))
	for _, part := range parts {
		var p struct {
			Type string  `json:"type"`
			Text *string `json:"text"`
		}
		_ = json.Unmarshal(part, &p)
		if p.Text != nil && slices.Contains(textTypes, p.Type) {
			items = append(items, textItem(*p.Text))
			continue
		}
		items = append(items, t.unrecognized(what, cmp.Or(p.Type, "untyped"), part))
	}
	return items
}

// chatToolCall translates an OpenAI function tool call.
func (t *lmTranslator) chatToolCall(raw json.RawMessage) ContentItem {
	var call struct {
		ID       string `json:"id"`
		Type     string `json:"type"`
		Function *struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"function"`
	}
	if json.Unmarshal(raw, &call) != nil || call.Function == nil || (call.Type != "" && call.Type != "function") {
		return t.unrecognized("tool call type", cmp.Or(call.Type, "untyped"), raw)
	}
	return t.toolUse(call.ID, call.Function.Name, toolArguments(call.Function.Arguments))
}

// toolArguments decodes a JSON-encoded arguments string; other strings stay as recorded.
func toolArguments(arguments string) any {
	if v, ok := decodeJSONValue(json.RawMessage(arguments)).(map[string]any); ok {
		return v
	}
	return arguments
}

// resultText is a tool output as text: a string as recorded, other JSON compact.
func resultText(raw json.RawMessage) string {
	if s, ok := jsonString(raw); ok {
		return s
	}
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	return string(compactJSON(raw))
}

// anthropicSystem translates a Messages system field: a string or text blocks.
func (t *lmTranslator) anthropicSystem(field string, raw json.RawMessage) lmItem {
	item := newItem(field, raw, "system")
	item.content = t.anthropicBlocks(raw)
	return item
}

// anthropicMessage translates a Messages message: string content or blocks.
func (t *lmTranslator) anthropicMessage(field string, raw json.RawMessage) lmItem {
	role, ok := t.messageRole(field, raw)
	if !ok {
		return t.invalidItem(field, raw, "element has no role")
	}
	var m struct {
		Content json.RawMessage `json:"content"`
	}
	_ = json.Unmarshal(raw, &m)
	item := newItem(field, raw, role)
	item.content = t.anthropicBlocks(m.Content)
	return item
}

// anthropicBlocks translates Messages content: a string, or text, thinking,
// tool_use and tool_result blocks.
func (t *lmTranslator) anthropicBlocks(raw json.RawMessage) []ContentItem {
	if s, ok := jsonString(raw); ok {
		return []ContentItem{textItem(s)}
	}
	var blocks []json.RawMessage
	if json.Unmarshal(raw, &blocks) != nil {
		if len(raw) == 0 || string(raw) == "null" {
			return nil
		}
		return []ContentItem{t.unrecognized("content shape", "content", raw)}
	}
	items := make([]ContentItem, 0, len(blocks))
	for _, block := range blocks {
		var b struct {
			Type      string          `json:"type"`
			Text      *string         `json:"text"`
			Thinking  *string         `json:"thinking"`
			ID        string          `json:"id"`
			Name      string          `json:"name"`
			Input     json.RawMessage `json:"input"`
			ToolUseID string          `json:"tool_use_id"`
			Content   json.RawMessage `json:"content"`
			IsError   bool            `json:"is_error"`
		}
		_ = json.Unmarshal(block, &b)
		switch {
		case b.Type == "text" && b.Text != nil:
			items = append(items, textItem(*b.Text))
		case b.Type == "thinking" && b.Thinking != nil:
			items = append(items, ContentItem{Type: "thinking", Thinking: *b.Thinking})
		case b.Type == "tool_use":
			items = append(items, t.toolUse(b.ID, b.Name, decodeJSONValue(b.Input)))
		case b.Type == "tool_result":
			items = append(items, t.toolResult(b.ToolUseID, resultText(b.Content), b.IsError))
		default:
			items = append(items, t.unrecognized("content block type", cmp.Or(b.Type, "untyped"), block))
		}
	}
	return items
}

// lmResponsesItem holds the fields of a Responses API item apsis translates.
type lmResponsesItem struct {
	Type      string          `json:"type"`
	Role      string          `json:"role"`
	Content   json.RawMessage `json:"content"`
	CallID    string          `json:"call_id"`
	Name      string          `json:"name"`
	Arguments *string         `json:"arguments"`
	Input     *string         `json:"input"`
	Output    json.RawMessage `json:"output"`
	Summary   json.RawMessage `json:"summary"`
}

// responsesItem translates a /v1/responses input item.
func (t *lmTranslator) responsesItem(field string, raw json.RawMessage) lmItem {
	var it lmResponsesItem
	if json.Unmarshal(raw, &it) != nil {
		return t.invalidItem(field, raw, "item does not decode")
	}
	switch it.Type {
	case "", "message":
		role, ok := t.messageRole(field, raw)
		if !ok {
			return t.invalidItem(field, raw, "message has no role")
		}
		item := newItem(field, raw, role)
		item.content = t.parts(it.Content, "input content part type", "input_text", "output_text", "text")
		return item
	case "function_call", "custom_tool_call", "reasoning":
		item := newItem(field, raw, "assistant")
		item.content = t.responsesOutput(it, raw, 0)
		return item
	case "function_call_output", "custom_tool_call_output":
		item := newItem(field, raw, "tool")
		item.content = []ContentItem{t.toolResult(it.CallID, resultText(it.Output), false)}
		return item
	}
	item := newItem(field, raw, cmp.Or(it.Role, "user"))
	item.content = []ContentItem{t.unrecognized("input item type", it.Type, raw)}
	return item
}

// responsesOutput translates an item a model produced: Responses
// function_call, custom_tool_call, reasoning and message items, and native
// chat tool_call, reasoning, message and invalid_tool_call items. index
// identifies a native tool call, which records no identifier of its own.
func (t *lmTranslator) responsesOutput(it lmResponsesItem, raw json.RawMessage, index int) []ContentItem {
	switch it.Type {
	case "function_call":
		return []ContentItem{t.toolUse(it.CallID, it.Name, toolArguments(deref(it.Arguments)))}
	case "custom_tool_call":
		return []ContentItem{t.toolUse(it.CallID, it.Name, deref(it.Input))}
	case "reasoning":
		return []ContentItem{{Type: "thinking", Thinking: reasoningText(it.Summary, it.Content)}}
	case "message":
		return t.parts(it.Content, "output content part type", "output_text", "text")
	case "tool_call":
		var call struct {
			Tool      string          `json:"tool"`
			Arguments json.RawMessage `json:"arguments"`
			Output    json.RawMessage `json:"output"`
		}
		_ = json.Unmarshal(raw, &call)
		id := t.scope + "output-" + strconv.Itoa(index)
		return []ContentItem{
			{Type: "tool_use", ID: id, Name: call.Tool, Input: decodeJSONValue(call.Arguments)},
			{Type: "tool_result", ToolUseID: id, Content: resultText(call.Output)},
		}
	case "invalid_tool_call":
		// Documented native output with no conversation translation.
		return []ContentItem{{Type: "event", EventType: it.Type, Payload: compactJSON(raw)}}
	}
	return []ContentItem{t.unrecognized("output item type", cmp.Or(it.Type, "untyped"), raw)}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// reasoningText joins a reasoning item's summary and content text. Native chat
// records content as a string; Responses records typed parts.
func reasoningText(summary, content json.RawMessage) string {
	var texts []string
	for _, raw := range []json.RawMessage{summary, content} {
		if s, ok := jsonString(raw); ok {
			texts = append(texts, s)
			continue
		}
		var parts []struct {
			Text string `json:"text"`
		}
		_ = json.Unmarshal(raw, &parts)
		for _, p := range parts {
			if p.Text != "" {
				texts = append(texts, p.Text)
			}
		}
	}
	return strings.Join(texts, "\n")
}

// nativeInputItem translates a native chat typed input item as user context.
func (t *lmTranslator) nativeInputItem(field string, raw json.RawMessage) lmItem {
	var it struct {
		Type    string  `json:"type"`
		Content *string `json:"content"`
	}
	_ = json.Unmarshal(raw, &it)
	item := newItem(field, raw, "user")
	if it.Type == "text" && it.Content != nil {
		item.content = []ContentItem{textItem(*it.Content)}
	} else {
		item.content = []ContentItem{t.unrecognized("input item type", cmp.Or(it.Type, "untyped"), raw)}
	}
	return item
}

// translateLMResponse translates a logged response body into assistant
// content: chat and text completion choices, Responses or native chat output
// items, or Messages content blocks, followed by usage and stats. It also
// returns the body's recorded model.
func translateLMResponse(body json.RawMessage, scope string) ([]ContentItem, string, []lmWarning) {
	t := &lmTranslator{scope: scope}
	unknownShape := func() []ContentItem {
		t.warn(WarningUnrecognizedRecord, "unrecognized response body shape")
		return []ContentItem{{Type: "event", EventType: "response", Payload: compactJSON(body)}}
	}
	fields, ok := orderedFields(body)
	if !ok {
		return unknownShape(), "", t.warnings
	}
	model, _ := jsonString(fieldValueOrNil(fields, "model"))

	var content []ContentItem
	choices, hasChoices := fieldValue(fields, "choices")
	output, hasOutput := fieldValue(fields, "output")
	blocks, hasBlocks := fieldValue(fields, "content")
	switch {
	case hasChoices:
		content = t.choices(choices)
	case hasOutput:
		var items []json.RawMessage
		if json.Unmarshal(output, &items) != nil {
			content = unknownShape()
			break
		}
		for i, raw := range items {
			var it lmResponsesItem
			_ = json.Unmarshal(raw, &it)
			content = append(content, t.responsesOutput(it, raw, i)...)
		}
	case hasBlocks:
		content = t.anthropicBlocks(blocks)
	default:
		content = unknownShape()
	}

	if counters, ok := orderedCounters(fieldValueOrNil(fields, "usage")); ok {
		content = append(content, ContentItem{Type: "usage", Usage: &Usage{Counters: counters}})
	}
	if counters, ok := orderedCounters(fieldValueOrNil(fields, "stats")); ok {
		content = append(content, ContentItem{Type: "usage", Usage: &Usage{Basis: UsageBasisStats, Counters: counters}})
	}
	return content, model, t.warnings
}

func fieldValueOrNil(fields []jsonField, key string) json.RawMessage {
	raw, _ := fieldValue(fields, key)
	return raw
}

// choices translates chat completion messages and text completion text.
func (t *lmTranslator) choices(raw json.RawMessage) []ContentItem {
	var choices []struct {
		Message *struct {
			Role             string            `json:"role"`
			Content          json.RawMessage   `json:"content"`
			ReasoningContent string            `json:"reasoning_content"`
			Reasoning        string            `json:"reasoning"`
			ToolCalls        []json.RawMessage `json:"tool_calls"`
		} `json:"message"`
		Text *string `json:"text"`
	}
	if json.Unmarshal(raw, &choices) != nil {
		return []ContentItem{t.unrecognized("response choices shape", "choices", raw)}
	}
	var content []ContentItem
	for _, c := range choices {
		switch {
		case c.Message != nil:
			m := c.Message
			content = append(content, t.chatContent(cmp.Or(m.Role, "assistant"), m.ReasoningContent, m.Reasoning, m.Content, "", m.ToolCalls)...)
		case c.Text != nil:
			content = append(content, textItem(*c.Text))
		}
	}
	return content
}
