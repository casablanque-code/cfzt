package main

import (
	"reflect"
	"testing"

	"github.com/spf13/cobra"
)

func testRoot() *cobra.Command {
	root := &cobra.Command{Use: "zt"}
	root.AddCommand(&cobra.Command{Use: "status <name>"})
	root.AddCommand(&cobra.Command{Use: "logs <name>"})
	root.AddCommand(&cobra.Command{Use: "restart <name>"})
	root.AddCommand(&cobra.Command{Use: "down <name>"})
	root.AddCommand(&cobra.Command{Use: "up <name> [port]"})
	root.AddCommand(&cobra.Command{Use: "list", Aliases: []string{"ls"}})
	return root
}

func TestReorderArgs(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{"name-first swaps to canonical order", []string{"zt", "grafana", "status"}, []string{"zt", "status", "grafana"}},
		{"canonical order untouched", []string{"zt", "status", "grafana"}, []string{"zt", "status", "grafana"}},
		{"name-first with trailing flags preserved", []string{"zt", "grafana", "logs", "-f"}, []string{"zt", "logs", "grafana", "-f"}},
		{"unrelated second word not swapped", []string{"zt", "grafana", "somethingelse"}, []string{"zt", "grafana", "somethingelse"}},
		{"up is never reordered (two real positionals)", []string{"zt", "up", "grafana"}, []string{"zt", "up", "grafana"}},
		{"leading flag not touched", []string{"zt", "--help", "status"}, []string{"zt", "--help", "status"}},
		{"alias recognized as known command", []string{"zt", "ls", "status"}, []string{"zt", "ls", "status"}},
		{"too short is a no-op", []string{"zt", "status"}, []string{"zt", "status"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := append([]string(nil), c.in...)
			reorderArgs(got, testRoot())
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("reorderArgs(%v) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}
