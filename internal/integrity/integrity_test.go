package integrity

import "testing"

func TestHash_StableAndSensitiveToChange(t *testing.T) {
	a := Hash([]byte("services:\n  grafana:\n    port: 3000\n"))
	b := Hash([]byte("services:\n  grafana:\n    port: 3000\n"))
	if a != b {
		t.Error("Hash() not stable for identical content")
	}

	c := Hash([]byte("services:\n  grafana:\n    port: 4000\n"))
	if a == c {
		t.Error("Hash() didn't change when content changed")
	}
}

func TestDiff_NoChange(t *testing.T) {
	content := "a\nb\nc"
	if diff := Diff(content, content); len(diff) != 0 {
		t.Errorf("Diff() = %v, want empty for identical content", diff)
	}
}

func TestDiff_AddedAndRemovedLines(t *testing.T) {
	old := "a\nb\nc"
	new := "a\nx\nc\nd"

	diff := Diff(old, new)
	if !contains(diff, "- b") {
		t.Errorf("Diff() = %v, want removed line \"- b\"", diff)
	}
	if !contains(diff, "+ x") {
		t.Errorf("Diff() = %v, want added line \"+ x\"", diff)
	}
	if !contains(diff, "+ d") {
		t.Errorf("Diff() = %v, want added line \"+ d\"", diff)
	}
}

func TestDiff_EmptyOld(t *testing.T) {
	diff := Diff("", "a\nb")
	if len(diff) != 2 {
		t.Fatalf("Diff() = %v, want 2 added lines", diff)
	}
}

func contains(lines []string, want string) bool {
	for _, l := range lines {
		if l == want {
			return true
		}
	}
	return false
}
