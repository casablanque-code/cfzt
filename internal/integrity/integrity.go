// Package integrity provides content hashing and a minimal line diff, used
// by `zt apply` to detect that zt.yaml changed since it was last applied.
package integrity

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// Hash returns the hex-encoded SHA-256 digest of content.
func Hash(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// Diff returns a line-level diff between oldContent and newContent, each
// line prefixed with "-" (removed) or "+" (added). It's LCS-based, adequate
// for showing what changed in a small manifest — not a general diff engine.
func Diff(oldContent, newContent string) []string {
	oldLines := splitLines(oldContent)
	newLines := splitLines(newContent)
	return lcsDiff(oldLines, newLines)
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimRight(s, "\n"), "\n")
}

func lcsDiff(a, b []string) []string {
	n, m := len(a), len(b)
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			switch {
			case a[i] == b[j]:
				lcs[i][j] = lcs[i+1][j+1] + 1
			case lcs[i+1][j] >= lcs[i][j+1]:
				lcs[i][j] = lcs[i+1][j]
			default:
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}

	var out []string
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case a[i] == b[j]:
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			out = append(out, fmt.Sprintf("- %s", a[i]))
			i++
		default:
			out = append(out, fmt.Sprintf("+ %s", b[j]))
			j++
		}
	}
	for ; i < n; i++ {
		out = append(out, fmt.Sprintf("- %s", a[i]))
	}
	for ; j < m; j++ {
		out = append(out, fmt.Sprintf("+ %s", b[j]))
	}
	return out
}
