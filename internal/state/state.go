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
	Name         string       `json:"name"`
	TunnelID     string       `json:"tunnel_id"`
	Port         int          `json:"port"`
	Hostname     string       `json:"hostname"`
	Protocol     Protocol     `json:"protocol,omitempty"`
	PID          int          `json:"pid,omitempty"`
	Status       TunnelStatus `json:"status"`
	Public       bool         `json:"public,omitempty"`        // true if created with --public (no ZT Access app)
	AllowEmails  []string     `json:"allow_emails,omitempty"`  // emails passed via --allow; empty + !Public means bypass policy
	DockerDetect bool         `json:"docker_detect,omitempty"` // true if port was (or should be) auto-detected via --docker
	ContainerPort string      `json:"container_port,omitempty"` // container-side port pinned via --container-port; empty = lowest published TCP port
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// ManifestSnapshot records the content of a zt.yaml as of its last
// successful `zt apply`, so a later apply can detect the file having
// changed on disk since then (see Store.ManifestSnapshot).
type ManifestSnapshot struct {
	Hash      string    `json:"hash"`
	Content   string    `json:"content"`
	UpdatedAt time.Time `json:"updated_at"`
}
