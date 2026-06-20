package watchdog

import (
	"bufio"
	"errors"
	"os"
	"strings"
	"time"
)

// scanResult describes what was found in the newly-appended portion of a log.
type scanResult struct {
	fallbackDetected bool
	newOffset        int64
}

// scanLogTail reads a log file starting at fromOffset and reports whether
// the fallback marker appears anywhere in the new content. It never
// re-reads bytes it has already scanned, so this stays cheap even on
// long-running tunnels with large log files.
func scanLogTail(path string, fromOffset int64) (scanResult, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return scanResult{newOffset: fromOffset}, nil
		}
		return scanResult{}, err
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil {
		return scanResult{}, err
	}

	// Log was rotated/truncated (e.g. service restarted and log file was
	// recreated) — restart scanning from the beginning rather than seeking
	// past EOF, which would silently skip content.
	if info.Size() < fromOffset {
		fromOffset = 0
	}

	if _, err := f.Seek(fromOffset, 0); err != nil {
		return scanResult{}, err
	}

	found := false
	scanner := bufio.NewScanner(f)
	// cloudflared lines are short; default scanner buffer is plenty
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), fallbackMarker) {
			found = true
		}
	}
	if err := scanner.Err(); err != nil {
		return scanResult{}, err
	}

	return scanResult{fallbackDetected: found, newOffset: info.Size()}, nil
}

// Decision describes what the watchdog should do for one tunnel on this tick.
type Decision struct {
	ShouldRestart bool
	Reason        string
}

// Evaluate scans a tunnel's log since the last check and decides whether
// a restart is warranted, respecting the per-tunnel backoff window.
func Evaluate(ts *TunnelState, logPath string, now time.Time) (Decision, error) {
	result, err := scanLogTail(logPath, ts.LastLogOffset)
	if err != nil {
		return Decision{}, err
	}
	ts.LastLogOffset = result.newOffset

	if !result.fallbackDetected {
		// Healthy since last check — relax backoff so a single old
		// fallback doesn't keep this tunnel on an inflated wait forever.
		resetBackoff(ts)
		return Decision{}, nil
	}

	backoff := ts.CurrentBackoff
	if backoff <= 0 {
		backoff = MinBackoff
	}

	if !ts.LastRestartAt.IsZero() && now.Sub(ts.LastRestartAt) < backoff {
		return Decision{
			Reason: "fallback detected but within backoff window",
		}, nil
	}

	return Decision{
		ShouldRestart: true,
		Reason:        "QUIC fallback detected — restarting to retry QUIC",
	}, nil
}

// RecordRestart updates bookkeeping after the watchdog actually restarts a tunnel.
func RecordRestart(ts *TunnelState, now time.Time) {
	ts.LastRestartAt = now
	ts.CurrentBackoff = nextBackoff(ts.CurrentBackoff)
}
