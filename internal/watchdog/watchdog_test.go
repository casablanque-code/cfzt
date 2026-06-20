package watchdog

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeLog(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cloudflared.log")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestScanLogTailDetectsFallback(t *testing.T) {
	path := writeLog(t, "2026-06-19T10:00:00Z INF Connection registered connIndex=0\n"+
		"2026-06-19T10:00:05Z INF Switching to fallback protocol http2 connIndex=0\n")

	result, err := scanLogTail(path, 0)
	if err != nil {
		t.Fatalf("scanLogTail error: %v", err)
	}
	if !result.fallbackDetected {
		t.Error("expected fallback to be detected")
	}
	info, _ := os.Stat(path)
	if result.newOffset != info.Size() {
		t.Errorf("newOffset = %d, want %d", result.newOffset, info.Size())
	}
}

func TestScanLogTailNoFallback(t *testing.T) {
	path := writeLog(t, "2026-06-19T10:00:00Z INF Connection registered connIndex=0\n"+
		"2026-06-19T10:00:01Z INF Connection registered connIndex=1\n")

	result, err := scanLogTail(path, 0)
	if err != nil {
		t.Fatalf("scanLogTail error: %v", err)
	}
	if result.fallbackDetected {
		t.Error("did not expect fallback to be detected")
	}
}

func TestScanLogTailIncrementalOnlyScansNewContent(t *testing.T) {
	path := writeLog(t, "2026-06-19T10:00:00Z INF Connection registered\n")

	first, err := scanLogTail(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	if first.fallbackDetected {
		t.Fatal("first scan should not detect fallback")
	}

	// append a fallback line after the first scan
	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	_, _ = f.WriteString("2026-06-19T10:01:00Z INF Switching to fallback protocol http2 connIndex=0\n")
	f.Close()

	second, err := scanLogTail(path, first.newOffset)
	if err != nil {
		t.Fatal(err)
	}
	if !second.fallbackDetected {
		t.Error("second scan should detect the newly appended fallback line")
	}

	// re-scanning from the new offset should find nothing — no double counting
	third, err := scanLogTail(path, second.newOffset)
	if err != nil {
		t.Fatal(err)
	}
	if third.fallbackDetected {
		t.Error("re-scanning already-read content should not detect fallback again")
	}
}

func TestScanLogTailHandlesTruncatedLog(t *testing.T) {
	path := writeLog(t, "short\n")
	// simulate a stale offset beyond current file size (log was rotated)
	result, err := scanLogTail(path, 99999)
	if err != nil {
		t.Fatalf("scanLogTail should handle truncated logs gracefully: %v", err)
	}
	_ = result // should not error, offset resets internally
}

func TestScanLogTailMissingFile(t *testing.T) {
	result, err := scanLogTail("/nonexistent/cloudflared.log", 0)
	if err != nil {
		t.Fatalf("missing log file should not error (tunnel may not have started yet): %v", err)
	}
	if result.fallbackDetected {
		t.Error("missing file should not report a fallback")
	}
}

func TestEvaluateRestartsOnFirstFallback(t *testing.T) {
	path := writeLog(t, "INF Switching to fallback protocol http2 connIndex=0\n")
	ts := &TunnelState{}
	now := time.Now()

	decision, err := Evaluate(ts, path, now)
	if err != nil {
		t.Fatal(err)
	}
	if !decision.ShouldRestart {
		t.Error("expected restart on first fallback detection")
	}
}

func TestEvaluateRespectsBackoffWindow(t *testing.T) {
	path := writeLog(t, "INF Switching to fallback protocol http2 connIndex=0\n")
	now := time.Now()

	ts := &TunnelState{
		LastRestartAt:  now.Add(-1 * time.Minute), // restarted 1 minute ago
		CurrentBackoff: MinBackoff,                // but backoff window is 10 minutes
	}

	decision, err := Evaluate(ts, path, now)
	if err != nil {
		t.Fatal(err)
	}
	if decision.ShouldRestart {
		t.Error("should not restart again within the backoff window")
	}
}

func TestEvaluateAllowsRestartAfterBackoffElapses(t *testing.T) {
	path := writeLog(t, "INF Switching to fallback protocol http2 connIndex=0\n")
	now := time.Now()

	ts := &TunnelState{
		LastRestartAt:  now.Add(-15 * time.Minute), // restarted 15 min ago
		CurrentBackoff: MinBackoff,                 // backoff window was 10 min — elapsed
	}

	decision, err := Evaluate(ts, path, now)
	if err != nil {
		t.Fatal(err)
	}
	if !decision.ShouldRestart {
		t.Error("should restart again once backoff window has elapsed")
	}
}

func TestEvaluateNoRestartWhenHealthy(t *testing.T) {
	path := writeLog(t, "INF Connection registered connIndex=0\n")
	ts := &TunnelState{}

	decision, err := Evaluate(ts, path, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if decision.ShouldRestart {
		t.Error("should not restart a healthy tunnel")
	}
}

func TestEvaluateResetsBackoffWhenHealthy(t *testing.T) {
	path := writeLog(t, "INF Connection registered connIndex=0\n")
	ts := &TunnelState{CurrentBackoff: MaxBackoff}

	_, err := Evaluate(ts, path, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if ts.CurrentBackoff != 0 {
		t.Errorf("backoff should reset to 0 when no fallback detected, got %v", ts.CurrentBackoff)
	}
}

func TestNextBackoffDoublesAndCaps(t *testing.T) {
	cases := []struct {
		current time.Duration
		want    time.Duration
	}{
		{0, MinBackoff},
		{MinBackoff, MinBackoff * 2},
		{MinBackoff * 2, MinBackoff * 4},
		{MaxBackoff, MaxBackoff},         // already at cap
		{MaxBackoff / 2 * 3, MaxBackoff}, // would overshoot — capped
	}
	for _, c := range cases {
		got := nextBackoff(c.current)
		if got != c.want {
			t.Errorf("nextBackoff(%v) = %v, want %v", c.current, got, c.want)
		}
	}
}

func TestRecordRestartUpdatesState(t *testing.T) {
	ts := &TunnelState{}
	now := time.Now()

	RecordRestart(ts, now)

	if !ts.LastRestartAt.Equal(now) {
		t.Errorf("LastRestartAt = %v, want %v", ts.LastRestartAt, now)
	}
	if ts.CurrentBackoff != MinBackoff {
		t.Errorf("CurrentBackoff = %v, want %v", ts.CurrentBackoff, MinBackoff)
	}
}

func TestRuntimeStateRoundtrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	rs, err := LoadRuntimeState()
	if err != nil {
		t.Fatal(err)
	}
	ts := rs.Get("grafana")
	ts.CurrentBackoff = 20 * time.Minute
	ts.LastLogOffset = 1024

	if err := rs.Save(); err != nil {
		t.Fatal(err)
	}

	reloaded, err := LoadRuntimeState()
	if err != nil {
		t.Fatal(err)
	}
	got, ok := reloaded.Tunnels["grafana"]
	if !ok {
		t.Fatal("expected grafana entry after reload")
	}
	if got.CurrentBackoff != 20*time.Minute {
		t.Errorf("CurrentBackoff = %v, want 20m", got.CurrentBackoff)
	}
	if got.LastLogOffset != 1024 {
		t.Errorf("LastLogOffset = %d, want 1024", got.LastLogOffset)
	}
}

func TestRuntimeStateForget(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	rs, _ := LoadRuntimeState()
	rs.Get("vault")
	rs.Forget("vault")

	if _, ok := rs.Tunnels["vault"]; ok {
		t.Error("expected vault entry to be removed after Forget")
	}
}

func TestRuntimeStateCorruptFileDoesNotCrash(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path := filepath.Join(home, ".zt-watchdog-state.json")
	if err := os.WriteFile(path, []byte("{not valid json"), 0600); err != nil {
		t.Fatal(err)
	}

	rs, err := LoadRuntimeState()
	if err != nil {
		t.Fatalf("corrupt state file should not error, got: %v", err)
	}
	if rs.Tunnels == nil {
		t.Error("expected empty initialized map after corrupt state recovery")
	}
}
