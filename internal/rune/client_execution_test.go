package rune

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func fakeRuneCommand(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("test launcher uses a Unix shell")
	}
	path := t.TempDir()
	executable, err := os.Executable()
	require.NoError(t, err)
	t.Setenv("ORBIT_RUNE_TEST_BINARY", executable)
	t.Setenv("ORBIT_RUNE_TEST_PROCESS", "1")
	t.Setenv("PATH", path+string(os.PathListSeparator)+os.Getenv("PATH"))
	script := "#!/bin/sh\nexec \"$ORBIT_RUNE_TEST_BINARY\" -test.run=^TestRuneCommandProcess$ -- \"$@\"\n"
	require.NoError(t, os.WriteFile(filepath.Join(path, "rune"), []byte(script), 0755))
}

func TestRuneQueriesHonorCancellation(t *testing.T) {
	fakeRuneCommand(t)
	t.Setenv("ORBIT_RUNE_TEST_DELAY", "500ms")
	queries := map[string]func(*Client) error{
		"list all":       func(c *Client) error { _, err := c.ListAll(); return err },
		"list pending":   func(c *Client) error { _, err := c.ListPending(); return err },
		"next phase":     func(c *Client) error { _, err := c.GetNextPhase(); return err },
		"select task":    func(c *Client) error { _, err := c.Select(false); return err },
		"select phase":   func(c *Client) error { _, err := c.Select(true); return err },
		"phase overview": func(c *Client) error { _, err := c.GetPhaseSummaries(); return err },
	}
	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
			defer cancel()
			start := time.Now()
			err := query(NewClientContext(ctx, "tasks.md"))
			require.ErrorIs(t, err, context.DeadlineExceeded)
			require.Less(t, time.Since(start), 2*time.Second)
		})
	}
}

func TestRuneQueriesRejectFailureEnvelope(t *testing.T) {
	fakeRuneCommand(t)
	t.Setenv("ORBIT_RUNE_TEST_OUTPUT", `{"success":false,"message":"cannot read ledger"}`)
	client := NewClient("tasks.md")
	_, err := client.ListAll()
	require.ErrorContains(t, err, "cannot read ledger")
	_, err = client.ListPending()
	require.ErrorContains(t, err, "cannot read ledger")
	_, err = client.GetNextPhase()
	require.ErrorContains(t, err, "cannot read ledger")
	_, err = client.Select(false)
	require.ErrorContains(t, err, "cannot read ledger")
}

func TestListAllRejectsMissingOrMalformedTaskPayload(t *testing.T) {
	fakeRuneCommand(t)
	for _, output := range []string{
		"", "{}", "null", "[]", `{"success":true}`, `{"success":true,"count":0}`,
		`{"success":true,"data":[]}`, `{"success":true,"count":1,"data":[]}`,
		`{"Tasks":{}}`, `{"Tasks":"unexpected"}`,
	} {
		t.Run(output, func(t *testing.T) {
			t.Setenv("ORBIT_RUNE_TEST_OUTPUT", output)
			_, err := NewClient("tasks.md").ListAll()
			require.Error(t, err, "malformed task payload must not imply completion")
		})
	}
}

func TestListAllAcceptsExplicitEmptyTaskPayload(t *testing.T) {
	fakeRuneCommand(t)
	for _, output := range []string{
		`{"Tasks":null}`, `{"Tasks":[]}`, `{"success":true,"count":0,"data":[]}`,
	} {
		t.Run(output, func(t *testing.T) {
			t.Setenv("ORBIT_RUNE_TEST_OUTPUT", output)
			tasks, err := NewClient("tasks.md").ListAll()
			require.NoError(t, err)
			require.Empty(t, tasks)
		})
	}
}

func TestRuneCancellationKillsChildren(t *testing.T) {
	fakeRuneCommand(t)
	marker := filepath.Join(t.TempDir(), "child")
	t.Setenv("ORBIT_RUNE_TEST_CHILD", marker)
	t.Setenv("ORBIT_RUNE_TEST_DELAY", "2s")
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	finished := make(chan error, 1)
	go func() {
		_, err := NewClientContext(ctx, "tasks.md").ListAll()
		finished <- err
	}()
	require.Eventually(t, func() bool {
		_, err := os.Stat(marker + ".ready")
		return err == nil
	}, 2*time.Second, 10*time.Millisecond)
	cancel()
	select {
	case err := <-finished:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("rune cancellation did not return")
	}
	time.Sleep(400 * time.Millisecond)
	_, err := os.Stat(marker)
	require.True(t, os.IsNotExist(err), "rune child survived cancellation")
}

func TestRuneClientNilContext(t *testing.T) {
	fakeRuneCommand(t)
	_, err := NewClientContext(nil, "tasks.md").ListAll() //nolint:staticcheck // Verify the constructor's explicit nil-context fallback.
	require.NoError(t, err)
}

// TestRuneCommandProcess is the disposable executable behind the fake rune CLI.
func TestRuneCommandProcess(t *testing.T) {
	if os.Getenv("ORBIT_RUNE_TEST_PROCESS") != "1" {
		return
	}
	if marker := os.Getenv("ORBIT_RUNE_TEST_CHILD"); marker != "" {
		child := exec.Command("sh", "-c", `sleep 0.3; printf survived > "$ORBIT_RUNE_TEST_CHILD"`)
		child.Stdout, child.Stderr = os.Stdout, os.Stderr
		require.NoError(t, child.Start())
		require.NoError(t, os.WriteFile(marker+".ready", nil, 0600))
	}
	if delay := os.Getenv("ORBIT_RUNE_TEST_DELAY"); delay != "" {
		duration, err := time.ParseDuration(delay)
		require.NoError(t, err)
		time.Sleep(duration)
	}
	output, present := os.LookupEnv("ORBIT_RUNE_TEST_OUTPUT")
	if !present {
		output = `{"success":true,"count":0,"data":[]}`
	}
	fmt.Println(output)
	os.Exit(0)
}
