package state

import "time"

type TunnelStatus string

const (
	StatusRunning TunnelStatus = "running"
	StatusStopped TunnelStatus = "stopped"
	StatusError   TunnelStatus = "error"
)

type Protocol string

const (
	ProtocolAuto  Protocol = "auto"
	ProtocolQUIC  Protocol = "quic"
	ProtocolHTTP2 Protocol = "http2"
)

type Tunnel struct {
	Name      string       `json:"name"`
	TunnelID  string       `json:"tunnel_id"`
	Port      int          `json:"port"`
	Hostname  string       `json:"hostname"`
	Protocol  Protocol     `json:"protocol,omitempty"`
	PID       int          `json:"pid,omitempty"`
	Status    TunnelStatus `json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}
