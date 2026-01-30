package version

import "testing"

func TestCompare(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected int // -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
	}{
		{"same version", "v0.0.1-rc8", "v0.0.1-rc8", 0},
		{"rc8 < rc9", "v0.0.1-rc8", "v0.0.1-rc9", -1},
		{"rc9 > rc8", "v0.0.1-rc9", "v0.0.1-rc8", 1},
		{"rc9 < rc10", "v0.0.1-rc9", "v0.0.1-rc10", -1},
		{"rc10 > rc9", "v0.0.1-rc10", "v0.0.1-rc9", 1},
		{"rc10 < rc11", "v0.0.1-rc10", "v0.0.1-rc11", -1},
		{"rc11 > rc10", "v0.0.1-rc11", "v0.0.1-rc10", 1},
		{"rc2 < rc10", "v0.0.1-rc2", "v0.0.1-rc10", -1},
		{"rc10 > rc2", "v0.0.1-rc10", "v0.0.1-rc2", 1},
		{"release > rc", "v0.0.1", "v0.0.1-rc10", 1},
		{"rc < release", "v0.0.1-rc10", "v0.0.1", -1},
		{"different base", "v0.0.1-rc8", "v0.0.2-rc1", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Compare(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("Compare(%q, %q) = %d, want %d", tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}

func TestCompareSuffixes(t *testing.T) {
	tests := []struct {
		name     string
		s1       string
		s2       string
		expected int
	}{
		{"rc10 < rc11", "rc10", "rc11", -1},
		{"rc11 > rc10", "rc11", "rc10", 1},
		{"rc2 < rc10", "rc2", "rc10", -1},
		{"rc10 > rc2", "rc10", "rc2", 1},
		{"rc9 < rc10", "rc9", "rc10", -1},
		{"rc10 > rc9", "rc10", "rc9", 1},
		{"same suffix", "rc10", "rc10", 0},
		{"beta1 < beta10", "beta1", "beta10", -1},
		{"beta10 > beta1", "beta10", "beta1", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompareSuffixes(tt.s1, tt.s2)
			if result != tt.expected {
				t.Errorf("CompareSuffixes(%q, %q) = %d, want %d", tt.s1, tt.s2, result, tt.expected)
			}
		})
	}
}
