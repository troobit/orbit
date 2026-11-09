//go:build !windows

package agents_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/arjenschwarz/orbit/internal/agents"
	"github.com/stretchr/testify/require"
)

func TestExecuteCancellationKillsChildren(t *testing.T) {
	root := t.TempDir()
	result := agents.Execute(context.Background(), agents.ExecuteConfig{
		CLIPath: "sh", Args: []string{"-c", "(sleep 0.5; touch orphan) & wait"},
		WorkDir: root, Timeout: 100 * time.Millisecond, ProcessGroup: true,
	})
	require.ErrorIs(t, result.Err, context.DeadlineExceeded)
	require.Less(t, result.Duration, time.Second)
	time.Sleep(600 * time.Millisecond)
	_, err := os.Stat(filepath.Join(root, "orphan"))
	require.True(t, os.IsNotExist(err), "worker child survived cancellation")
}
