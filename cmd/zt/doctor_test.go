package main

import (
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/casablanque-code/cfzt/internal/testutil"
)

func TestPluralize(t *testing.T) {
	tests := []struct {
		n    int
		word string
		want string
	}{
		{0, "problem", "0 problems"},
		{1, "problem", "1 problem"},
		{2, "problem", "2 problems"},
	}
	for _, tt := range tests {
		if got := pluralize(tt.n, tt.word); got != tt.want {
			t.Errorf("pluralize(%d, %q) = %q, want %q", tt.n, tt.word, got, tt.want)
		}
	}
}

func TestCheck_Pass(t *testing.T) {
	out := captureStdout(t, func() {
		if ok := check("thing works", nil, "unused hint"); !ok {
			t.Error("check() with nil error = false, want true")
		}
	})
	if !strings.Contains(out, "thing works") {
		t.Errorf("check() output = %q, want it to contain the label", out)
	}
	if strings.Contains(out, "unused hint") {
		t.Errorf("check() output = %q, want no hint printed on success", out)
	}
}

func TestCheck_Fail(t *testing.T) {
	out := captureStdout(t, func() {
		if ok := check("thing broken", fmt.Errorf("boom"), "try turning it off and on"); ok {
			t.Error("check() with non-nil error = true, want false")
		}
	})
	for _, want := range []string{"thing broken", "boom", "try turning it off and on"} {
		if !strings.Contains(out, want) {
			t.Errorf("check() output = %q, want it to contain %q", out, want)
		}
	}
}

func TestCheck_Fail_NoHint(t *testing.T) {
	out := captureStdout(t, func() {
		check("thing broken", fmt.Errorf("boom"), "")
	})
	if strings.Contains(out, "hint:") {
		t.Errorf("check() output = %q, want no hint line when hint is empty", out)
	}
}

func TestCheckWarn_Fail(t *testing.T) {
	out := captureStdout(t, func() {
		checkWarn("optional thing", fmt.Errorf("meh"), "not critical")
	})
	for _, want := range []string{"optional thing", "meh", "not critical"} {
		if !strings.Contains(out, want) {
			t.Errorf("checkWarn() output = %q, want it to contain %q", out, want)
		}
	}
}

func TestCheckPort_Reachable(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	port := ln.Addr().(*net.TCPAddr).Port
	if err := checkPort(port); err != nil {
		t.Errorf("checkPort(%d) error = %v, want nil for a listening port", port, err)
	}
}

func TestCheckPort_Unreachable(t *testing.T) {
	// Bind and immediately close to get a port nothing is listening on,
	// rather than guessing a fixed number that might collide.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	if err := checkPort(port); err == nil {
		t.Errorf("checkPort(%d) = nil error, want an error for a closed port", port)
	}
}

func TestCheckDNS_Resolves(t *testing.T) {
	// "localhost" always resolves without depending on real external DNS
	// being reachable in CI.
	if err := checkDNS("localhost"); err != nil {
		t.Errorf("checkDNS(localhost) error = %v, want nil", err)
	}
}

func TestCheckDNS_Failure(t *testing.T) {
	// .invalid is reserved by RFC 2606 specifically to never resolve.
	if err := checkDNS("definitely-does-not-exist.invalid"); err == nil {
		t.Error("checkDNS() on a .invalid hostname = nil error, want an error")
	}
}

func TestRunDoctor_NoConfig(t *testing.T) {
	testutil.SetHome(t, t.TempDir())

	out := captureStdout(t, func() {
		if err := runDoctor(nil, nil); err != nil {
			t.Fatalf("runDoctor() error = %v", err)
		}
	})
	if !strings.Contains(out, "config file found") {
		t.Errorf("runDoctor() output = %q, want it to report the missing config check", out)
	}
	if !strings.Contains(out, "problem") {
		t.Errorf("runDoctor() output = %q, want a non-zero problem count in the summary", out)
	}
}
