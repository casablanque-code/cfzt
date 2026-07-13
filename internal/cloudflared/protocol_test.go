package cloudflared

import (
	"os"
	"path/filepath"
	"testing"
)

func writeLog(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "cloudflared.log")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("writing test log: %v", err)
	}
	return path
}

func TestDetectEffectiveProtocol_MissingLog(t *testing.T) {
	got := DetectEffectiveProtocol(filepath.Join(t.TempDir(), "does-not-exist.log"))
	if got != EffectiveUnknown {
		t.Fatalf("expected EffectiveUnknown for missing log, got %q", got)
	}
}

func TestDetectEffectiveProtocol_NoSignal(t *testing.T) {
	path := writeLog(t, "2026-07-13T20:24:16Z INF Updated to new configuration version=0\n")
	got := DetectEffectiveProtocol(path)
	if got != EffectiveUnknown {
		t.Fatalf("expected EffectiveUnknown, got %q", got)
	}
}

func TestDetectEffectiveProtocol_QUICRegistered(t *testing.T) {
	path := writeLog(t, `2026-07-13T20:24:17Z INF Registered tunnel connection connIndex=1 connection=50a5223a event=0 ip=198.41.192.77 location=ams07 protocol=quic
2026-07-13T20:24:18Z INF Registered tunnel connection connIndex=2 connection=b533e1ca event=0 ip=198.41.200.63 location=ams18 protocol=quic
`)
	got := DetectEffectiveProtocol(path)
	if got != EffectiveQUIC {
		t.Fatalf("expected quic, got %q", got)
	}
}

// This mirrors a real-world log: cloudflared's startup connectivity
// pre-check finds UDP dead and every connection registers on http2 -
// the exact shape reported against the auto-restart-loop bug.
func TestDetectEffectiveProtocol_PrecheckDegradedToHTTP2(t *testing.T) {
	path := writeLog(t, `2026-07-13T20:24:17Z INF Registered tunnel connection connIndex=1 connection=50a5223a-f9c5-4211-a5bb-0a6f9ad159b6 event=0 ip=198.41.192.77 location=ams07 protocol=http2
2026-07-13T20:24:18Z INF Registered tunnel connection connIndex=2 connection=b533e1ca-d617-4af7-8629-095a4a6d0fee event=0 ip=198.41.200.63 location=ams18 protocol=http2
2026-07-13T20:24:19Z INF Registered tunnel connection connIndex=3 connection=fe35ba8a-888b-440d-b0be-a5b21ab722e0 event=0 ip=198.41.192.57 location=ams21 protocol=http2
2026-07-13T20:24:25Z INF precheck component="UDP Connectivity" details="QUIC connection failed" run_id=d39bc56f status=fail target=region1.v2.argotunnel.com
2026-07-13T20:24:25Z INF precheck complete hard_fail=false run_id=d39bc56f-a7ce-494c-b64f-d74fb85ded63 suggested_protocol=http2
`)
	got := DetectEffectiveProtocol(path)
	if got != EffectiveHTTP2 {
		t.Fatalf("expected http2, got %q", got)
	}
}

// A mid-session fallback: tunnel started on quic, then dropped to http2
// later in the log. The last signal should win, not the first.
func TestDetectEffectiveProtocol_MidSessionFallback(t *testing.T) {
	path := writeLog(t, `2026-07-13T20:24:17Z INF Registered tunnel connection connIndex=1 connection=aaa event=0 protocol=quic
2026-07-13T20:40:02Z INF Switching to fallback protocol http2
2026-07-13T20:40:03Z INF Registered tunnel connection connIndex=1 connection=bbb event=0 protocol=http2
`)
	got := DetectEffectiveProtocol(path)
	if got != EffectiveHTTP2 {
		t.Fatalf("expected http2 after mid-session fallback, got %q", got)
	}
}
