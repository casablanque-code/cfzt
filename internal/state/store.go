package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const stateFileName = ".zt-state.json"

type Store struct {
	Tunnels map[string]*Tunnel `json:"tunnels"`
}

func statePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, stateFileName), nil
}

func LoadStore() (*Store, error) {
	path, err := statePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Store{Tunnels: make(map[string]*Tunnel)}, nil
		}
		return nil, err
	}
	var s Store
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("malformed state file: %w", err)
	}
	if s.Tunnels == nil {
		s.Tunnels = make(map[string]*Tunnel)
	}
	return &s, nil
}

func (s *Store) Save() error {
	path, err := statePath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func (s *Store) Set(t *Tunnel) {
	s.Tunnels[t.Name] = t
}

func (s *Store) Get(name string) (*Tunnel, bool) {
	t, ok := s.Tunnels[name]
	return t, ok
}

func (s *Store) Delete(name string) {
	delete(s.Tunnels, name)
}

func (s *Store) All() []*Tunnel {
	out := make([]*Tunnel, 0, len(s.Tunnels))
	for _, t := range s.Tunnels {
		out = append(out, t)
	}
	return out
}
