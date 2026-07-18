package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_NotFound(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	_, err := Load()
	if err == nil {
		t.Fatal("Load() = nil error, want error when no config file exists")
	}
	if !strings.Contains(err.Error(), "zt init") {
		t.Errorf("error = %q, want it to point at `zt init`", err.Error())
	}
}

func TestLoad_Malformed(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := os.WriteFile(filepath.Join(home, configFileName), []byte("{not json"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("Load() = nil error, want error for malformed JSON")
	}
	if !strings.Contains(err.Error(), "malformed") {
		t.Errorf("error = %q, want it to say the config is malformed", err.Error())
	}
}

func TestLoad_Incomplete(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Valid JSON, but missing account_id and domain.
	if err := os.WriteFile(filepath.Join(home, configFileName), []byte(`{"api_token":"tok"}`), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("Load() = nil error, want error for incomplete config")
	}
	if !strings.Contains(err.Error(), "zt init") {
		t.Errorf("error = %q, want it to point at `zt init`", err.Error())
	}
}

func TestSaveThenLoad_RoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	want := &Config{APIToken: "tok-123", AccountID: "acct-456", Domain: "example.com"}
	if err := Save(want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if *got != *want {
		t.Errorf("Load() = %+v, want %+v", got, want)
	}
}

func TestSave_FilePermissions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := &Config{APIToken: "tok", AccountID: "acct", Domain: "example.com"}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	info, err := os.Stat(filepath.Join(home, configFileName))
	if err != nil {
		t.Fatalf("stat config file: %v", err)
	}
	// The config file holds an API token - it must not be group/world readable.
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("config file permissions = %o, want 0600", perm)
	}
}

func TestConfigFilePath_EndsWithFileName(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got := ConfigFilePath()
	want := filepath.Join(home, configFileName)
	if got != want {
		t.Errorf("ConfigFilePath() = %q, want %q", got, want)
	}
}
