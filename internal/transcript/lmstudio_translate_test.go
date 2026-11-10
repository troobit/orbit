package transcript

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// describeLMItem renders the parts of a content item translation decides.
func describeLMItem(item ContentItem) string {
	var s string
	switch item.Type {
	case "text":
		s = "text:" + item.Text
	case "thinking":
		s = "thinking:" + item.Thinking
	case "tool_use":
		input, _ := json.Marshal(item.Input)
		s = fmt.Sprintf("tool_use:%s:%s:%s", item.ItemID, item.Name, input)
	case "tool_result":
		s = fmt.Sprintf("tool_result:%s:%s", item.ItemID, item.Content)
		if item.IsError {
			s += ":error"
		}
	case "event":
		s = "event:" + item.EventType
	case "usage":
		names := make([]string, 0, len(item.Usage.Counters))
		for _, c := range item.Usage.Counters {
			names = append(names, c.Name+"="+string(c.Value))
		}
		s = fmt.Sprintf("usage:%s:%s", item.Usage.Basis, strings.Join(names, ","))
	default:
		s = item.Type + ":" + item.Text
	}
	if item.Truncated {
		s += " [truncated]"
	}
	return s
}

// describeLMEntries renders request entries as "type/role: item | item".
func describeLMEntries(entries []Entry) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		items := make([]string, 0, len(e.Message.Content))
		for _, item := range e.Message.Content {
			items = append(items, describeLMItem(item))
		}
		out = append(out, e.Type+"/"+e.Message.Role+": "+strings.Join(items, " | "))
	}
	return out
}

func parseLMFixture(t *testing.T, name string) *ParseResult {
	t.Helper()
	t.Setenv("TZ", "")
	res, err := ParseLMStudio(strings.NewReader(readLMFixture(t, name)))
	require.NoError(t, err)
	return res
}

// Every included endpoint, and both native chat input forms, translate with
// their supplied roles; everything else in the body stays a parameter
// (requirements 2.1, 2.2, 2.9).
func TestParseLMStudio_EndpointTranslation(t *testing.T) {
	res := parseLMFixture(t, "endpoints.log")
	byRequest := lmEntriesByRequest(res.Entries)

	type want struct {
		endpoint   string
		model      string
		parameters string
		entries    []string
	}
	tests := []want{
		{"/v1/chat/completions", "example-model-a", `{"temperature":0.2,"stream":false}`, []string{
			"system/system: text:You answer briefly.",
			"user/user: text:Name a prime number.",
		}},
		{"/api/v0/chat/completions", "example-model-a", `{"max_tokens":-1,"stream":false}`, []string{
			"user/user: text:Introduce yourself.",
		}},
		{"/v1/completions", "example-model-b", `{"max_tokens":8,"stop":"\n"}`, []string{
			"user/user: text:The capital of the example country is",
		}},
		{"/api/v0/completions", "example-model-b", `{"temperature":0.7}`, []string{
			"user/user: text:First prompt.",
			"user/user: text:Second prompt.",
		}},
		{"/v1/messages", "example-model-a", `{"max_tokens":256}`, []string{
			"system/system: text:You are a careful assistant.",
			"user/user: text:Summarise the example.",
		}},
		{"/v1/responses", "example-model-a", `{"store":false,"stream":false}`, []string{
			"system/system: text:Reply in one sentence.",
			"user/user: text:What is an example?",
		}},
		{"/api/v1/chat", "example-model-b", `{"store":true}`, []string{
			"system/system: text:Be concise.",
			"user/user: text:Tell me about the example.",
		}},
		{"/api/v1/chat", "example-model-b", `{"previous_response_id":"resp_example0"}`, []string{
			"user/user: text:Describe this picture.",
			"user/user: event:image",
		}},
	}
	require.Len(t, res.LMStudioRequests, len(tests))
	for i, tc := range tests {
		req := res.LMStudioRequests[i]
		t.Run(fmt.Sprintf("%d %s", i, tc.endpoint), func(t *testing.T) {
			assert.Equal(t, tc.endpoint, req.Endpoint)
			assert.Equal(t, tc.model, req.Model)
			assert.Equal(t, tc.parameters, string(req.Parameters), "source order and logged values")
			assert.Equal(t, tc.entries, describeLMEntries(byRequest[req.ID]))
			for _, e := range byRequest[req.ID] {
				assert.Equal(t, req.Model, e.Model)
				assert.Equal(t, lmProvider, e.Provider)
				assert.False(t, e.Unattributed)
			}
		})
	}

	assert.Equal(t, []ParseWarning{
		{Line: 125, Code: WarningUnrecognizedRecord, Message: "unrecognized input item type: image"},
	}, res.Warnings, "a native image item is kept as an event, with a warning")
}

func TestParseLMStudio_LoggedResponses(t *testing.T) {
	res := parseLMFixture(t, "endpoints.log")
	var responses []string
	for _, e := range res.Entries {
		if e.Unattributed {
			responses = append(responses, describeLMEntries([]Entry{e})...)
			assert.Empty(t, e.RequestID, "attribution unknown (requirement 4.1)")
			assert.False(t, e.SuppliedContext)
		}
	}
	assert.Equal(t, []string{
		"assistant/assistant: text:Seven. | usage::prompt_tokens=20,completion_tokens=2,total_tokens=22",
		"assistant/assistant: text:An example illustrates a rule. | usage::input_tokens=18,output_tokens=7,total_tokens=25",
	}, responses)
}

// Tool links stay inside one request occurrence or logged-response event,
// whatever identifiers clients reuse (requirement 2.8).
func TestParseLMStudio_ToolScoping(t *testing.T) {
	res := parseLMFixture(t, "tools.log")
	byRequest := lmEntriesByRequest(res.Entries)
	require.Len(t, res.LMStudioRequests, 4)

	assert.Equal(t, []string{
		"user/user: text:List the files.",
		`assistant/assistant: thinking:I should list the files. | tool_use:call_1:list_files:{"path":"."}`,
		"user/tool: tool_result:call_1:a.txt\nb.txt",
		"user/user: text:Now read the first file and report every line of it from the ... <Truncated in logs> ... to the end of the file. [truncated]",
	}, describeLMEntries(byRequest["1"]))
	assert.Equal(t, []string{
		"user/user: text:A different client.",
		`assistant/assistant: tool_use:call_1:read_file:"not json"`,
		"user/tool: tool_result:call_1:file contents",
	}, describeLMEntries(byRequest["49"]), "arguments that are not JSON stay as recorded")
	assert.Equal(t, []string{
		"system/system: text:System block.",
		"user/user: text:Check the build.",
		`assistant/assistant: thinking:Plan the check. | text:Running it. | tool_use:toolu_1:bash:{"command":"make"}`,
		`user/user: tool_result:toolu_1:ok | tool_result:toolu_missing:[{"type":"text","text":"orphan"}]:error`,
		"user/user: event:image",
	}, describeLMEntries(byRequest["101"]), "an unmatched result is kept")
	assert.Equal(t, []string{
		"system/developer: text:Developer note.",
		"assistant/assistant: thinking:Thinking about it.",
		`assistant/assistant: tool_use:call_patch:apply_patch:"*** Begin Patch"`,
		"user/tool: tool_result:call_patch:Done",
		"user/user: event:web_search_call",
		"user/user: event:input_image",
	}, describeLMEntries(byRequest["173"]))

	// Scoped identifiers: the same recorded call_1 links within each occurrence only.
	ids := map[string][]string{}
	for _, e := range res.Entries {
		for _, item := range e.Message.Content {
			if item.Type == "tool_use" || item.Type == "tool_result" {
				ids[item.ItemID] = append(ids[item.ItemID], item.ID+item.ToolUseID)
			}
		}
	}
	assert.Equal(t, []string{"req:1:call_1", "req:1:call_1", "req:49:call_1", "req:49:call_1", "evt:76:call_1"}, ids["call_1"])
	assert.Equal(t, []string{"req:101:toolu_1", "req:101:toolu_1", "evt:260:toolu_1"}, ids["toolu_1"])
	assert.Equal(t, []string{"req:101:toolu_missing"}, ids["toolu_missing"])
	assert.Equal(t, []string{"evt:219:output-1", "evt:219:output-1"}, ids[""], "a native tool call links its own output")

	for _, e := range res.Entries {
		assert.Equal(t, e.Type == "assistant" && !e.Unattributed, e.SuppliedContext,
			"assistant items resent as request input are supplied context (requirement 2.4)")
	}

	assert.Equal(t, []ParseWarning{
		{Line: 101, Code: WarningUnrecognizedRecord, Message: "unrecognized content block type: image"},
		{Line: 173, Code: WarningUnrecognizedRecord, Message: "unrecognized input item type: web_search_call"},
		{Line: 173, Code: WarningUnrecognizedRecord, Message: "unrecognized input content part type: input_image"},
	}, res.Warnings)
}

func TestParseLMStudio_UnrecognizedContentKeepsPayload(t *testing.T) {
	res := parseLMFixture(t, "tools.log")
	byRequest := lmEntriesByRequest(res.Entries)
	search := byRequest["173"][4].Message.Content[0]
	assert.Equal(t, `{"type":"web_search_call","id":"ws_1","status":"completed"}`, string(search.Payload))
	image := byRequest["101"][4].Message.Content[0]
	assert.JSONEq(t, `{"type":"image","source":{"type":"base64","media_type":"image/png","data":"AAAA"}}`, string(image.Payload))
}

func TestTranslateLMResponse(t *testing.T) {
	tests := map[string]struct {
		body     string
		content  []string
		model    string
		warnings []lmWarning
	}{
		"chat completion with reasoning and tool calls": {
			body: `{"model":"m","choices":[{"message":{"role":"assistant","content":"Hi","reasoning_content":"why",` +
				`"tool_calls":[{"id":"c1","type":"function","function":{"name":"f","arguments":"{\"a\":1}"}}]}}]}`,
			content: []string{"thinking:why", "text:Hi", `tool_use:c1:f:{"a":1}`},
			model:   "m",
		},
		"text completion choices": {
			body:    `{"choices":[{"text":"one"},{"text":"two"}]}`,
			content: []string{"text:one", "text:two"},
		},
		"v0 usage and stats": {
			body:    `{"choices":[{"text":"x"}],"usage":{"total_tokens":3},"stats":{"tokens_per_second":51.4}}`,
			content: []string{"text:x", "usage::total_tokens=3", "usage:stats:tokens_per_second=51.4"},
		},
		"responses output": {
			body: `{"model":"m","output":[{"type":"reasoning","summary":[{"type":"summary_text","text":"s"}],"content":[{"type":"reasoning_text","text":"r"}]},` +
				`{"type":"message","role":"assistant","content":[{"type":"output_text","text":"done"}]},` +
				`{"type":"function_call","call_id":"c2","name":"g","arguments":"{}"}]}`,
			content: []string{"thinking:s\nr", "text:done", "tool_use:c2:g:{}"},
			model:   "m",
		},
		"messages content": {
			body:     `{"type":"message","content":[{"type":"text","text":"ok"},{"type":"redacted_thinking","data":"x"}]}`,
			content:  []string{"text:ok", "event:redacted_thinking"},
			warnings: []lmWarning{{WarningUnrecognizedRecord, "unrecognized content block type: redacted_thinking"}},
		},
		"unknown output item": {
			body:     `{"output":[{"type":"future_item"}]}`,
			content:  []string{"event:future_item"},
			warnings: []lmWarning{{WarningUnrecognizedRecord, "unrecognized output item type: future_item"}},
		},
		"unknown shape": {
			body:     `{"result":"x"}`,
			content:  []string{"event:response"},
			warnings: []lmWarning{{WarningUnrecognizedRecord, "unrecognized response body shape"}},
		},
		"not an object": {
			body:     `"plain"`,
			content:  []string{"event:response"},
			warnings: []lmWarning{{WarningUnrecognizedRecord, "unrecognized response body shape"}},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			content, model, warnings := translateLMResponse(json.RawMessage(tc.body), "evt:1:")
			got := make([]string, 0, len(content))
			for _, item := range content {
				got = append(got, describeLMItem(item))
			}
			assert.Equal(t, tc.content, got)
			assert.Equal(t, tc.model, model)
			assert.Equal(t, tc.warnings, warnings)
		})
	}
}

func TestTranslateLMRequest_Shapes(t *testing.T) {
	tests := map[string]struct {
		endpoint   string
		body       string
		entries    []string
		parameters string
		warnings   []lmWarning
	}{
		"parameters keep order, precision and markers": {
			endpoint: "/v1/responses",
			body: `{"temperature":0.10,"input":"q","max_output_tokens":1e3,"model":"m",` +
				`"integration_headers":{"authorization":"Bearer abc ... <Truncated in logs> ... xyz"},"reasoning":{"effort":"high"}}`,
			entries: []string{"user/user: text:q"},
			parameters: `{"temperature":0.10,"max_output_tokens":1e3,` +
				`"integration_headers":{"authorization":"Bearer abc ... <Truncated in logs> ... xyz"},"reasoning":{"effort":"high"}}`,
		},
		"non-string model stays a parameter": {
			endpoint:   "/v1/completions",
			body:       `{"model":7,"prompt":"p"}`,
			entries:    []string{"user/user: text:p"},
			parameters: `{"model":7}`,
		},
		"missing conversation field": {
			endpoint:   "/v1/chat/completions",
			body:       `{"model":"m","stream":true}`,
			parameters: `{"stream":true}`,
			warnings:   []lmWarning{{WarningInvalidRecord, "/v1/chat/completions request body has no messages"}},
		},
		"conversation field of the wrong type": {
			endpoint: "/v1/messages",
			body:     `{"messages":"text"}`,
			entries:  []string{"user/user: event:messages"},
			warnings: []lmWarning{{WarningInvalidRecord, "messages is not an array"}},
		},
		"message without a role": {
			endpoint: "/v1/chat/completions",
			body:     `{"messages":[{"content":"x"}]}`,
			entries:  []string{"user/user: event:messages"},
			warnings: []lmWarning{{WarningInvalidRecord, "messages element has no role"}},
		},
		"unknown role keeps its content": {
			endpoint: "/v1/chat/completions",
			body:     `{"messages":[{"role":"function","content":"x"}]}`,
			entries:  []string{"user/function: text:x"},
			warnings: []lmWarning{{WarningUnrecognizedRecord, "unrecognized messages role: function"}},
		},
		"chat content parts and an unknown tool call type": {
			endpoint: "/v1/chat/completions",
			body: `{"messages":[{"role":"user","content":[{"type":"text","text":"a"},{"type":"image_url","image_url":{"url":"u"}}]},` +
				`{"role":"assistant","tool_calls":[{"id":"x","type":"custom"}]}]}`,
			entries: []string{"user/user: text:a | event:image_url", "assistant/assistant: event:custom"},
			warnings: []lmWarning{
				{WarningUnrecognizedRecord, "unrecognized chat content part type: image_url"},
				{WarningUnrecognizedRecord, "unrecognized tool call type: custom"},
			},
		},
		"token-array prompt": {
			endpoint: "/api/v0/completions",
			body:     `{"prompt":["a",[1,2]]}`,
			entries:  []string{"user/user: text:a", "user/user: event:prompt"},
			warnings: []lmWarning{{WarningInvalidRecord, "prompt holds a value that is not a string"}},
		},
		"non-string instructions": {
			endpoint: "/v1/responses",
			body:     `{"instructions":["x"],"input":"q"}`,
			entries:  []string{"user/user: event:instructions", "user/user: text:q"},
			warnings: []lmWarning{{WarningInvalidRecord, "instructions is not a string"}},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tr := translateLMRequest(tc.endpoint, json.RawMessage(tc.body), "req:1:")
			var entries []string
			for _, item := range tr.items {
				entries = append(entries, describeLMEntries([]Entry{{Type: item.entryType, Message: &Message{Role: item.role, Content: item.content}}})...)
			}
			assert.Equal(t, tc.entries, entries)
			assert.Equal(t, tc.parameters, string(tr.parameters))
			assert.Equal(t, tc.warnings, tr.warnings)
		})
	}
}

func TestLMClientMetadata(t *testing.T) {
	tests := map[string]struct {
		raw                         string
		thread, parent, requestKind string
	}{
		"thread only":                {raw: `{"thread_id":"t"}`, thread: "t"},
		"empty thread":               {raw: `{"thread_id":""}`},
		"truncated thread":           {raw: `{"thread_id":"t ... <Truncated in logs> ... t"}`},
		"non-string thread":          {raw: `{"thread_id":7}`},
		"no metadata object":         {raw: `"t ... <Truncated in logs> ... t"`},
		"parent header key":          {raw: `{"thread_id":"t","x-codex-parent-thread-id":"p"}`, thread: "t", parent: "p"},
		"kind from turn blob":        {raw: `{"thread_id":"t","x-codex-turn-metadata":"{\"request_kind\":\"compaction\"}"}`, thread: "t", requestKind: "compaction"},
		"parent from turn blob":      {raw: `{"x-codex-turn-metadata":"{\"parent_thread_id\":\"p\",\"request_kind\":\"turn\"}"}`, parent: "p", requestKind: "turn"},
		"header key wins":            {raw: `{"x-codex-parent-thread-id":"p1","x-codex-turn-metadata":"{\"parent_thread_id\":\"p2\"}"}`, parent: "p1"},
		"truncated turn blob":        {raw: `{"thread_id":"t","x-codex-turn-metadata":"{\"request_kind\":\"turn\",\"x ... <Truncated in logs> ... \"}"}`, thread: "t"},
		"unparseable turn blob":      {raw: `{"thread_id":"t","x-codex-turn-metadata":"not json"}`, thread: "t"},
		"session id is not a thread": {raw: `{"session_id":"s","prompt_cache_key":"k"}`},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			thread, parent, kind := lmClientMetadata(json.RawMessage(tc.raw))
			assert.Equal(t, tc.thread, thread)
			assert.Equal(t, tc.parent, parent)
			assert.Equal(t, tc.requestKind, kind)
		})
	}
}

// Client metadata stays a parameter as logged even when it sets thread
// identity (requirements 2.9, 3.7).
func TestParseLMStudio_ThreadMetadata(t *testing.T) {
	res := parseLMFixture(t, "threads.log")
	type identity struct{ thread, parent, kind string }
	var got []identity
	for _, req := range res.LMStudioRequests {
		got = append(got, identity{req.ThreadID, req.ParentThreadID, req.RequestKind})
		var params map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(req.Parameters, &params))
		if req.StartLine != 44 {
			assert.Contains(t, params, "client_metadata")
		}
	}
	assert.Equal(t, []identity{
		{"thread-alpha", "", "turn"},
		{"", "", ""}, // no client_metadata
		{"thread-alpha", "", "turn"},
		{"thread-alpha", "", "compaction"},
		{"thread-alpha", "", "memory"},
		{"thread-beta", "thread-alpha", ""}, // truncated turn blob: no request kind
		{"", "", ""},                        // truncated thread_id: singleton
		{"", "", ""},                        // empty thread_id: singleton
	}, got)
	assert.Contains(t, string(res.LMStudioRequests[5].Parameters), `\"sandbox\":\"wor ... <Truncated in logs> ... \"}`,
		"the truncated blob stays raw")
}
