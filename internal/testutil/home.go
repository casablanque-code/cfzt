// Package testutil holds small helpers shared across this module's test
// files. It is test-only: nothing outside a _test.go file should import it.
package testutil

import "testing"

// SetHome overrides the directory os.UserHomeDir() resolves to for the
// duration of the test. os.UserHomeDir() reads $HOME on Linux/macOS but
// %USERPROFILE% (falling back to %HOMEDRIVE%%HOMEPATH%) on Windows — HOME
// is not consulted there at all. Setting only HOME in a test is a no-op on
// Windows: file operations silently fall through to the real CI runner's
// profile directory instead of the test's temp dir, corrupting state left
// over between test runs. Setting both env vars here keeps one call site
// portable across all three platforms.
func SetHome(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
}
