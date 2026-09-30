package render

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCountdown(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	require.Equal(t, "~1h29m", Countdown(now.Unix()+5340, now))
	require.Equal(t, "~12m", Countdown(now.Unix()+720, now))
	require.Equal(t, "~2d3h", Countdown(now.Unix()+2*86400+3*3600+59, now))
	require.Equal(t, "<1m", Countdown(now.Unix()+30, now))
}

func TestCountdown_Past(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	require.Equal(t, "", Countdown(now.Unix()-1, now))
	require.Equal(t, "", Countdown(now.Unix(), now))
}

func TestWidthAndTruncate(t *testing.T) {
	s := "\x1b[32mhello\x1b[0m 世界"
	require.Equal(t, 10, Width(s))
	require.Equal(t, "\x1b[32mhel…\x1b[0m", Truncate(s, 4))
	require.Equal(t, s, Truncate(s, 10))
	require.Equal(t, "…", Truncate("abc", 1))
}

func TestDuration(t *testing.T) {
	require.Equal(t, "45s", Duration(45_000))
	require.Equal(t, "12m5s", Duration(725_000))
	require.Equal(t, "1h0m", Duration(3_600_000))
	require.Equal(t, "0s", Duration(0))
}
