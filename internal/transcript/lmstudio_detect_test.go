package transcript

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readLMFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "lmstudio", name))
	require.NoError(t, err)
	return string(data)
}

func lmRequestHeader(path string) string {
	return "[2026-03-02 09:15:01][DEBUG] Received request: POST to " + path
}

func TestProbeLMStudio(t *testing.T) {
	tests := map[string]struct {
		input string
		want  bool
	}{
		"responses":                    {lmRequestHeader("/v1/responses") + " with body {\n}\n", true},
		"chat completions":             {lmRequestHeader("/v1/chat/completions") + " with body {\n}\n", true},
		"completions":                  {lmRequestHeader("/v1/completions") + " with body {\n}\n", true},
		"messages":                     {lmRequestHeader("/v1/messages") + " with body {\n}\n", true},
		"v0 chat completions":          {lmRequestHeader("/api/v0/chat/completions") + " with body {\n}\n", true},
		"v0 completions":               {lmRequestHeader("/api/v0/completions") + " with body {\n}\n", true},
		"native chat":                  {lmRequestHeader("/api/v1/chat") + " with body {\n}\n", true},
		"request without a body":       {lmRequestHeader("/v1/responses") + "\n", true},
		"tags before the message":      {"[2026-03-02 09:15:01][INFO][tag-a][tag-b] Received request: POST to /v1/responses\n", true},
		"CRLF line endings":            {"[2026-03-02 09:15:00][INFO] Ready\r\n" + lmRequestHeader("/v1/messages") + "\r\n", true},
		"later header in the chunk":    {"[2026-03-02 09:15:00][INFO] Ready\n" + lmRequestHeader("/api/v1/chat") + " with body {}\n", true},
		"embeddings":                   {lmRequestHeader("/v1/embeddings") + " with body {}\n", false},
		"token counting":               {lmRequestHeader("/v1/responses/input_tokens") + " with body {}\n", false},
		"message token counting":       {lmRequestHeader("/v1/messages/count_tokens") + " with body {}\n", false},
		"model management":             {lmRequestHeader("/api/v1/models/unload") + " with body {}\n", false},
		"decisions":                    {lmRequestHeader("/v1/decisions") + " with body {}\n", false},
		"included path as prefix":      {lmRequestHeader("/v1/responses/extra") + "\n", false},
		"query string":                 {lmRequestHeader("/v1/responses?x=1") + "\n", false},
		"GET request":                  {"[2026-03-02 09:15:01][DEBUG] Received request: GET to /v1/responses\n", false},
		"signature inside a body line": {`  "content": "[2026-03-02 09:15:01][DEBUG] Received request: POST to /v1/responses"` + "\n", false},
		"header without a level":       {"[2026-03-02 09:15:01] Received request: POST to /v1/responses\n", false},
		"notices only":                 {"[2026-03-02 09:15:00][INFO] Ready\n", false},
		"empty":                        {"", false},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, ProbeLMStudio([]byte(tc.input)))
		})
	}
}

func TestDetectFormat_LMStudio(t *testing.T) {
	for _, name := range []string{"endpoints.log", "classification.log", "threads.log", "tools.log", "recovery.log", "kiro_marker.log"} {
		t.Run(name, func(t *testing.T) {
			input := readLMFixture(t, name)
			got, replay, err := DetectFormat(strings.NewReader(input))
			require.NoError(t, err)
			assert.Equal(t, FormatLMStudio, got)
			assert.Equal(t, input, string(replay))
		})
	}

	t.Run("leading BOM", func(t *testing.T) {
		got, _, err := DetectFormat(strings.NewReader(bom + readLMFixture(t, "endpoints.log")))
		require.NoError(t, err)
		assert.Equal(t, FormatLMStudio, got)
	})

	t.Run("detection reads no further than the chunk", func(t *testing.T) {
		input := readLMFixture(t, "endpoints.log") + strings.Repeat("[2026-03-02 10:00:00][INFO] Notice\n", 4000)
		require.Greater(t, len(input), detectChunkBytes)
		got, replay, err := DetectFormat(strings.NewReader(input))
		require.NoError(t, err)
		assert.Equal(t, FormatLMStudio, got)
		assert.Len(t, replay, detectChunkBytes, "the exec probe never ran past the chunk")
	})
}

// A Kiro key-name heuristic matches the body of kiro_marker.log on its own;
// the LM Studio signature still wins (Q13, requirement 1.1).
func TestDetectFormat_LMStudioSignatureBeatsKiroMarkers(t *testing.T) {
	input := readLMFixture(t, "kiro_marker.log")
	kiro, validated := detectKiroFormat([]byte(input))
	require.NotEqual(t, FormatUnknown, kiro, "the fixture must trip a Kiro heuristic")
	require.False(t, validated)

	res, err := Parse(strings.NewReader(input))
	require.NoError(t, err)
	assert.Equal(t, FormatLMStudio, res.Format)
	require.Len(t, res.LMStudioRequests, 1)
	assert.JSONEq(t, `{"metadata":{"conversation_id":"conversation-example","history":[],"executionId":"execution-example","chat":[]}}`,
		string(res.LMStudioRequests[0].Parameters))
}

func TestDetectFormat_LMStudioHeadersWithoutRequest(t *testing.T) {
	padding := strings.Repeat("[2026-03-02 09:00:00][INFO] Notice\n", 2000)
	tests := map[string]string{
		"notice-only rotated file":       readLMFixture(t, "notice_only.log"),
		"request beyond the bound":       padding + lmRequestHeader("/v1/responses") + " with body {}\n",
		"excluded and unsupported paths": readLMFixture(t, "notice_only.log") + lmRequestHeader("/v1/embeddings") + "\n" + lmRequestHeader("/v1/decisions") + "\n",
		"Kiro key names in a notice body": "[2026-03-02 09:00:00][INFO] Returning {\n" +
			`  "conversation_id": "c", "history": [], "executionId": "e", "chat": [], "metadata": {}` + "\n}\n",
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			_, _, err := DetectFormat(strings.NewReader(input))
			require.ErrorIs(t, err, errLMStudioWithoutRequest)

			_, err = Parse(strings.NewReader(input))
			require.Error(t, err, "no Claude fallback for LM Studio headers")
			assert.Contains(t, err.Error(), "-a lmstudio")
		})
	}

	t.Run("forced parsing keeps the records", func(t *testing.T) {
		res, err := ParseJSONLWithFormat(strings.NewReader(readLMFixture(t, "notice_only.log")), FormatLMStudio)
		require.NoError(t, err)
		assert.Equal(t, FormatLMStudio, res.Format)
		assert.Empty(t, res.LMStudioRequests)
		assert.Empty(t, res.Warnings)
	})
}

// A full chunk can end inside the request header: the path counts only when
// text after it shows it is complete.
func TestDetectFormat_LMStudioChunkBoundary(t *testing.T) {
	header := lmRequestHeader("/v1/responses")
	build := func(headerEnd int) string {
		prefix := "[2026-03-02 09:00:00][INFO] "
		filler := headerEnd - len(header) - len(prefix) - 1
		require.Positive(t, filler)
		return prefix + strings.Repeat("x", filler) + "\n" + header + " with body {}\n"
	}

	tests := map[string]struct {
		headerEnd int // byte offset where the request path ends
		want      Format
	}{
		"path ends at the bound":         {detectChunkBytes, FormatUnknown},
		"text after the path is visible": {detectChunkBytes - 1, FormatLMStudio},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			input := build(tc.headerEnd)
			require.Equal(t, header, input[tc.headerEnd-len(header):tc.headerEnd])
			got, _, err := DetectFormat(strings.NewReader(input))
			if tc.want == FormatUnknown {
				require.True(t, errors.Is(err, errLMStudioWithoutRequest), "got %v", err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
