package agents_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/arjenschwarz/orbit/internal/agents"
	"github.com/stretchr/testify/require"
)

func TestExecuteLiveOutput(t *testing.T) {
	root := t.TempDir()
	stdout, err := os.Create(filepath.Join(root, "stdout"))
	require.NoError(t, err)
	defer func() { _ = stdout.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	finished := make(chan *agents.ExecuteResult, 1)
	go func() {
		finished <- agents.Execute(ctx, agents.ExecuteConfig{
			CLIPath: "sh", Args: []string{"-c", "echo working; while [ ! -e release ]; do sleep 0.01; done; echo done"},
			WorkDir: root, Stdout: stdout,
		})
	}()
	require.Eventually(t, func() bool {
		data, err := os.ReadFile(stdout.Name())
		return err == nil && strings.Contains(string(data), "working")
	}, 2*time.Second, 10*time.Millisecond)
	select {
	case <-finished:
		t.Fatal("execution finished before the live output was observed")
	default:
	}
	require.NoError(t, os.WriteFile(filepath.Join(root, "release"), nil, 0600))
	result := <-finished
	require.NoError(t, result.Err)
	require.Equal(t, "working\ndone\n", string(result.Stdout))
	data, err := os.ReadFile(stdout.Name())
	require.NoError(t, err)
	require.Equal(t, string(result.Stdout), string(data))
}
