package codex

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/arjenschwarz/orbit/internal/agents"
	"github.com/stretchr/testify/require"
)

func TestParseEvents(t *testing.T) {
	for name, data := range map[string]struct {
		output  string
		failed  bool
		message string
	}{
		"failed turn":      {`{"type":"turn.failed","error":{"message":"model unavailable"}}`, true, "model unavailable"},
		"top level error":  {`{"type":"error","message":"connection refused"}`, true, "connection refused"},
		"recoverable item": {`{"type":"item.completed","item":{"type":"error","message":"tool retry"}}`, false, ""},
		"metadata warning": {"warning: model metadata missing\n{\"type\":\"turn.completed\"}", false, ""},
	} {
		t.Run(name, func(t *testing.T) {
			result := &agents.RunResult{Output: data.output}
			parseEvents(result)
			require.Equal(t, data.failed, result.IsError)
			require.Equal(t, data.output, result.Output)
			if data.failed {
				require.Contains(t, result.Errors, data.message)
			}
		})
	}
}

func TestParseEventsLargeAssistantMessage(t *testing.T) {
	message := strings.Repeat("x", 128*1024) + "\nORBIT-HALT: review needed"
	item, err := json.Marshal(map[string]any{"type": "item.completed", "item": map[string]string{"type": "agent_message", "text": message}})
	require.NoError(t, err)
	result := &agents.RunResult{Output: string(item) + "\n" + `{"type":"turn.completed"}`}
	parseEvents(result)
	require.Equal(t, message, result.FinalOutput)
	require.Equal(t, 1, result.NumTurns)
	require.False(t, result.IsError)
}
