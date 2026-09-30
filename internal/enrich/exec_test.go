package enrich

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Final review, IMPORTANT 2: the timeout must stop the whole command, not just `sh` — a
// grandchild that keeps stdout open used to block Output() until it exited on its own.
func TestExecRunner_TimeoutKillsCompoundCommands(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process groups are unix-only")
	}
	for _, script := range []string{"sleep 3; echo done", "sleep 3 | tr a b"} {
		ctx, cancel := context.WithTimeout(context.Background(), Timeout)
		start := time.Now()
		_, err := ExecRunner(ctx, t.TempDir(), "sh", "-c", script)
		cancel()
		require.Error(t, err, script)
		require.Less(t, time.Since(start), time.Second, script)
		var ee *ExitError
		require.False(t, errors.As(err, &ee), "a timeout is a failure, not an answer: %s", script)
	}
}

func TestExecRunner_NonZeroExitIsExitError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	out, err := ExecRunner(context.Background(), t.TempDir(), "sh", "-c", "echo hi; echo oops >&2; exit 3")
	var ee *ExitError
	require.True(t, errors.As(err, &ee))
	require.Equal(t, 3, ee.Code)
	require.Equal(t, "oops", ee.Stderr)
	require.Equal(t, "hi\n", out)
}
