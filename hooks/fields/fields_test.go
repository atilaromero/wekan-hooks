package fields

import "testing"

func TestNormalizeString(t *testing.T) {
	table := []struct {
		input  string
		expect string
	}{
		{"a", "a"},
		{"a/b", "a"},
		{"a/b/c", "a"},
		{"/b/c", ""},
		{" a /b/c", "a"},
		{" a!@#$ /b/c", "a"},
		{"a_1$", "a_1"},
		{"a-1!", "a-1"},
	}
	for _, tt := range table {
		got := normalizeString(tt.input)
		if got != tt.expect {
			t.Errorf("expect: '%s', got '%s'", tt.expect, got)
		}
	}
}
