package main

import (
	"testing"

	"github.com/casablanque-code/cfzt/internal/state"
)

func TestProtocolForExport(t *testing.T) {
	cases := []struct {
		name string
		in   state.Protocol
		want string
	}{
		{"empty is omitted", state.Protocol(""), ""},
		{"auto is omitted", state.ProtocolAuto, ""},
		{"quic is kept", state.ProtocolQUIC, "quic"},
		{"http2 is kept", state.ProtocolHTTP2, "http2"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := protocolForExport(c.in)
			if got != c.want {
				t.Errorf("protocolForExport(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
