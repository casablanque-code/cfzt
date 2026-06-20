package watchdog

import (
	"fmt"
	"time"

	"github.com/casablanque-code/cfzt/internal/cloudflared"
	"github.com/casablanque-code/cfzt/internal/state"
)

// Restarter abstracts the actual restart mechanism so this package stays
// testable without depending on systemd/launchd directly.
type Restarter func(tunnelName string) error

// EventLogger receives a human-readable line for every action taken,
// so the caller can print or log it as it sees fit.
type EventLogger func(line string)

// TickResult summarizes what happened during one RunOnce pass.
type TickResult struct {
	Checked   int
	Restarted []string
	Skipped   []string // protocol pinned, not "auto" — not our concern
	Errors    map[string]error
}

// RunOnce evaluates every tunnel in local state once: tunnels pinned to a
// specific protocol (not "auto") are left alone, since the user made an
// explicit choice. For "auto" tunnels, it tails the log since the last
// check and restarts the tunnel's service if a QUIC→HTTP/2 fallback was
// detected and the per-tunnel backoff window has elapsed.
func RunOnce(restart Restarter, log EventLogger) (TickResult, error) {
	result := TickResult{Errors: make(map[string]error)}

	store, err := state.LoadStore()
	if err != nil {
		return result, fmt.Errorf("loading tunnel state: %w", err)
	}

	rs, err := LoadRuntimeState()
	if err != nil {
		return result, fmt.Errorf("loading watchdog state: %w", err)
	}

	now := time.Now()
	tunnels := store.All()

	// Drop bookkeeping for tunnels that no longer exist, so the watchdog
	// state file doesn't grow forever across teardown/recreate cycles.
	live := make(map[string]bool, len(tunnels))
	for _, t := range tunnels {
		live[t.Name] = true
	}
	for name := range rs.Tunnels {
		if !live[name] {
			rs.Forget(name)
		}
	}

	for _, t := range tunnels {
		if t.Protocol != "" && t.Protocol != state.ProtocolAuto {
			result.Skipped = append(result.Skipped, t.Name)
			continue
		}
		result.Checked++

		logPath, err := cloudflared.LogPath(t.Name)
		if err != nil {
			result.Errors[t.Name] = err
			continue
		}

		ts := rs.Get(t.Name)
		decision, err := Evaluate(ts, logPath, now)
		if err != nil {
			result.Errors[t.Name] = err
			continue
		}

		if decision.ShouldRestart {
			if log != nil {
				log(fmt.Sprintf("%s: %s", t.Name, decision.Reason))
			}
			if err := restart(t.Name); err != nil {
				result.Errors[t.Name] = err
				continue
			}
			RecordRestart(ts, now)
			result.Restarted = append(result.Restarted, t.Name)
		}
	}

	if err := rs.Save(); err != nil {
		return result, fmt.Errorf("saving watchdog state: %w", err)
	}

	return result, nil
}
