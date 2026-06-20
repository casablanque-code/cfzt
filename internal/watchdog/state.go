// Package watchdog implements a background watcher that mitigates a
// long-standing cloudflared behaviour: once cloudflared falls back from
// QUIC to HTTP/2 (e.g. due to a transient UDP blip), it never retries
// QUIC again on its own — see https://github.com/cloudflare/cloudflared/issues/1534.
// This only matters for tunnels running with protocol "auto" (the default);
// tunnels explicitly pinned to --protocol http2 or --protocol quic are
// left untouched, since that's a deliberate choice.
//
// The watchdog tails each tunnel's cloudflared.log for the line
// "Switching to fallback protocol http2", and if found, restarts the
// tunnel's service after a backoff delay so cloudflared gets a fresh
// chance to negotiate QUIC again. Restarts back off exponentially per
// tunnel to avoid flapping a tunnel whose UDP path is persistently broken.
package watchdog

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// fallbackMarker is the exact, version-stable cloudflared log line that
// indicates a QUIC → HTTP/2 fallback occurred. Confirmed unchanged across
// cloudflared releases from 2022.4 through 2025.9.
const fallbackMarker = "Switching to fallback protocol http2"

const (
	// DefaultPollInterval is how often the watchdog checks each tunnel's log.
	DefaultPollInterval = 30 * time.Second
	// MinBackoff is the shortest wait between restart attempts for a tunnel
	// that keeps falling back to http2.
	MinBackoff = 10 * time.Minute
	// MaxBackoff caps the exponential backoff so a persistently broken
	// tunnel still gets retried roughly once an hour, not abandoned forever.
	MaxBackoff = 60 * time.Minute
)

// TunnelState tracks per-tunnel restart bookkeeping. Stored separately
// from the main ~/.zt-state.json so the watchdog never contends with
// interactive commands (up/down/list) for that file.
type TunnelState struct {
	// LastLogOffset is the byte offset up to which the log has been scanned.
	LastLogOffset int64 `json:"last_log_offset"`
	// LastRestartAt is when the watchdog last restarted this tunnel due to fallback.
	LastRestartAt time.Time `json:"last_restart_at,omitempty"`
	// CurrentBackoff is the current wait duration before the next allowed restart.
	CurrentBackoff time.Duration `json:"current_backoff,omitempty"`
}

// RuntimeState is the on-disk watchdog state file: ~/.zt-watchdog-state.json
type RuntimeState struct {
	Tunnels map[string]*TunnelState `json:"tunnels"`
}

func runtimeStatePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".zt-watchdog-state.json"), nil
}

// LoadRuntimeState reads the watchdog's own state file, creating an empty
// one in memory if it doesn't exist yet.
func LoadRuntimeState() (*RuntimeState, error) {
	path, err := runtimeStatePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &RuntimeState{Tunnels: make(map[string]*TunnelState)}, nil
		}
		return nil, err
	}
	var rs RuntimeState
	if err := json.Unmarshal(data, &rs); err != nil {
		// corrupt state file — don't crash the watchdog, start fresh
		return &RuntimeState{Tunnels: make(map[string]*TunnelState)}, nil
	}
	if rs.Tunnels == nil {
		rs.Tunnels = make(map[string]*TunnelState)
	}
	return &rs, nil
}

// Save persists the watchdog's runtime state.
func (rs *RuntimeState) Save() error {
	path, err := runtimeStatePath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(rs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// Get returns (creating if absent) the per-tunnel state.
func (rs *RuntimeState) Get(name string) *TunnelState {
	ts, ok := rs.Tunnels[name]
	if !ok {
		ts = &TunnelState{}
		rs.Tunnels[name] = ts
	}
	return ts
}

// Forget removes bookkeeping for a tunnel — call when a tunnel is torn down.
func (rs *RuntimeState) Forget(name string) {
	delete(rs.Tunnels, name)
}

// nextBackoff doubles the current backoff (or starts at MinBackoff),
// capped at MaxBackoff.
func nextBackoff(current time.Duration) time.Duration {
	if current <= 0 {
		return MinBackoff
	}
	next := current * 2
	if next > MaxBackoff {
		return MaxBackoff
	}
	return next
}

// resetBackoff is called whenever a tunnel is found healthy (no fallback
// detected since the last successful restart), so a single old fallback
// doesn't keep inflating the backoff indefinitely.
func resetBackoff(ts *TunnelState) {
	ts.CurrentBackoff = 0
}
