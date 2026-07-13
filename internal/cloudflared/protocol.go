package cloudflared

import (
	"bufio"
	"os"
	"regexp"
)

// EffectiveProtocol is what cloudflared actually negotiated, as opposed to
// what the tunnel is configured/pinned to. cloudflared decides this at
// startup (connectivity pre-check, "suggested_protocol=...") and can also
// change it mid-session (fallback marker below).
type EffectiveProtocol string

const (
	EffectiveUnknown EffectiveProtocol = ""
	EffectiveQUIC    EffectiveProtocol = "quic"
	EffectiveHTTP2   EffectiveProtocol = "http2"
)

// registeredConnRe matches lines like:
//
//	INF Registered tunnel connection connIndex=1 ... protocol=http2
//
// which cloudflared emits once per edge connection, on every start and
// whenever a connection is re-established (including after an in-session
// fallback). This is the most reliable live signal, since it reflects
// what actually got negotiated rather than what was merely suggested.
var registeredConnRe = regexp.MustCompile(`Registered tunnel connection.*protocol=(quic|http2)`)

// precheckRe matches the summary line cloudflared prints after its startup
// connectivity pre-check, e.g.:
//
//	precheck complete hard_fail=false ... suggested_protocol=http2
//
// Used as a fallback when no "Registered tunnel connection" line has been
// seen yet (e.g. tunnel is still coming up).
var precheckRe = regexp.MustCompile(`precheck complete.*suggested_protocol=(quic|http2)`)

// DetectEffectiveProtocol tails a tunnel's cloudflared.log and returns the
// protocol currently in effect, based on the most recent signal found.
// It scans the whole file each call; logs are rotated per-service-restart
// via systemd/launchd so this stays cheap in practice. Returns
// EffectiveUnknown if the log doesn't exist yet or contains no signal.
func DetectEffectiveProtocol(logPath string) EffectiveProtocol {
	f, err := os.Open(logPath)
	if err != nil {
		return EffectiveUnknown
	}
	defer func() { _ = f.Close() }()

	var last EffectiveProtocol

	scanner := bufio.NewScanner(f)
	// cloudflared lines can be long (the precheck table), give some headroom.
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		// Chronological order in the log wins: whichever signal appears
		// last - a registered connection or a precheck summary - is the
		// most current statement of what cloudflared is actually using.
		if m := registeredConnRe.FindStringSubmatch(line); m != nil {
			last = EffectiveProtocol(m[1])
			continue
		}
		if m := precheckRe.FindStringSubmatch(line); m != nil {
			last = EffectiveProtocol(m[1])
		}
	}

	return last
}
