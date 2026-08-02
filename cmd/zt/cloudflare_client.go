package main

import "github.com/casablanque-code/cfzt/internal/cloudflare"

// newCFClient constructs the Cloudflare API client used by every command
// that talks to Cloudflare (doctor, down, init, up). It's a package-level
// var rather than a direct cloudflare.NewClient call so tests can swap in
// cloudflare.NewClientForTesting pointed at an httptest.Server — no
// command needs its own test-only branching to make that possible.
var newCFClient = cloudflare.NewClient
