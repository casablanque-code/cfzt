package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestHelperProcess isn't a real test — it's spawned as a subprocess by
// the tests below (exec.Command(os.Args[0], ...)), following the standard
// pattern from os/exec's own tests. It only acts when
// GO_WANT_HELPER_PROCESS=1 is set on its environment specifically (never
// globally on the test binary), so it's a no-op under a normal `go test`
// run and can't interfere with any other test in this package.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	args := os.Args
	for len(args) > 0 && args[0] != "--" {
		args = args[1:]
	}
	if len(args) > 0 {
		args = args[1:] // drop the "--" separator itself
	}
	if len(args) == 0 {
		return
	}
	switch args[0] {
	case "print":
		os.Stdout.WriteString("stdout line\n")
		os.Stderr.WriteString("stderr line\n")
	case "fail":
		os.Exit(7)
	}
}

// helperChildArgs builds an argv for exec.Command(os.Args[0], ...) that
// re-invokes this test binary as TestHelperProcess with subCmd passed
// through after "--".
func helperChildArgs(subCmd string) []string {
	return []string{"-test.run=TestHelperProcess", "--", subCmd}
}

func TestRunInternalRun_AppendsChildOutputToLogFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "out.log")
	// Pre-seed the log to confirm we append, not truncate.
	if err := os.WriteFile(logPath, []byte("existing line\n"), 0600); err != nil {
		t.Fatal(err)
	}

	internalRunLogPath = logPath
	t.Cleanup(func() { internalRunLogPath = "" })

	args := append([]string{os.Args[0]}, helperChildArgs("print")...)
	if err := withHelperEnv(func() error {
		return runInternalRun(internalRunCmd, args)
	}); err != nil {
		t.Fatalf("runInternalRun() error = %v", err)
	}

	got, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"existing line", "stdout line", "stderr line"} {
		if !bytes.Contains(got, []byte(want)) {
			t.Errorf("log file = %q, want it to contain %q", got, want)
		}
	}
}

// TestRunInternalRun_PropagatesChildExitCode runs runInternalRun itself in
// a subprocess (not in-process) because runInternalRun calls os.Exit on a
// nonzero child exit code — calling it directly here would kill the `go
// test` process running this very test.
func TestRunInternalRun_PropagatesChildExitCode(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestRunInternalRun_ExitCodeSubprocess")
	cmd.Env = append(os.Environ(), "GO_WANT_EXITCODE_SUBPROCESS=1")

	err := cmd.Run()
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected the subprocess to exit with an error, got %T: %v", err, err)
	}
	if got := exitErr.ExitCode(); got != 7 {
		t.Errorf("exit code = %d, want 7 (propagated from the helper child's own os.Exit(7))", got)
	}
}

// TestRunInternalRun_ExitCodeSubprocess is the subprocess body invoked by
// the test above. It only acts when GO_WANT_EXITCODE_SUBPROCESS=1 is set,
// same guard pattern as TestHelperProcess.
func TestRunInternalRun_ExitCodeSubprocess(t *testing.T) {
	if os.Getenv("GO_WANT_EXITCODE_SUBPROCESS") != "1" {
		return
	}
	dir, err := os.MkdirTemp("", "zt-internal-run-test-")
	if err != nil {
		os.Exit(66) // distinguishable from the exit code under test (7)
	}
	internalRunLogPath = filepath.Join(dir, "out.log")

	args := append([]string{os.Args[0]}, helperChildArgs("fail")...)
	_ = withHelperEnv(func() error {
		return runInternalRun(internalRunCmd, args)
	})
	// runInternalRun should have already called os.Exit(7) via the
	// helper's os.Exit(7). Reaching here means it regressed to
	// swallowing the child's exit code instead of propagating it.
	os.Exit(66)
}

// withHelperEnv sets GO_WANT_HELPER_PROCESS=1 for the duration of fn, so
// runInternalRun's exec.Command(os.Args[0], ...) spawns the no-op-unless-
// flagged TestHelperProcess child rather than re-running the full suite.
// Scoped to fn and restored after, so it never leaks into other tests.
func withHelperEnv(fn func() error) error {
	prev, hadPrev := os.LookupEnv("GO_WANT_HELPER_PROCESS")
	os.Setenv("GO_WANT_HELPER_PROCESS", "1")
	defer func() {
		if hadPrev {
			os.Setenv("GO_WANT_HELPER_PROCESS", prev)
		} else {
			os.Unsetenv("GO_WANT_HELPER_PROCESS")
		}
	}()
	return fn()
}
