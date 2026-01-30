package version

import (
	"regexp"
	"strconv"
	"strings"
)

var suffixRegexp = regexp.MustCompile(`^([a-zA-Z]+)(\d+)$`)

// Compare compares two version strings
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
// Handles semantic versions like v0.0.1-rc4
func Compare(v1, v2 string) int {
	// Remove 'v' prefix if present
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	// If versions are identical, return 0
	if v1 == v2 {
		return 0
	}

	// Split versions into base and suffix (e.g., "0.0.1-rc4" -> ["0.0.1", "rc4"])
	v1Parts := strings.SplitN(v1, "-", 2)
	v2Parts := strings.SplitN(v2, "-", 2)

	v1Base := v1Parts[0]
	v2Base := v2Parts[0]

	// Compare base versions (e.g., "0.0.1")
	if v1Base != v2Base {
		// Simple string comparison for base versions
		// This works for most cases, but could be improved with proper semver parsing
		if v1Base < v2Base {
			return -1
		}
		return 1
	}

	// If base versions are the same, compare suffixes (e.g., "rc4" vs "rc5")
	v1Suffix := ""
	v2Suffix := ""
	if len(v1Parts) > 1 {
		v1Suffix = v1Parts[1]
	}
	if len(v2Parts) > 1 {
		v2Suffix = v2Parts[1]
	}

	// If one has a suffix and the other doesn't, the one without suffix is newer (e.g., "0.0.1" > "0.0.1-rc5")
	if v1Suffix == "" && v2Suffix != "" {
		return 1
	}
	if v1Suffix != "" && v2Suffix == "" {
		return -1
	}

	// Compare suffixes - handle numeric suffixes properly (e.g., rc10 vs rc11)
	return CompareSuffixes(v1Suffix, v2Suffix)
}

// CompareSuffixes compares version suffixes like "rc10", "rc11", "beta1", etc.
// It handles numeric suffixes properly so rc10 < rc11 (not rc10 < rc2)
func CompareSuffixes(s1, s2 string) int {
	match1 := suffixRegexp.FindStringSubmatch(s1)
	match2 := suffixRegexp.FindStringSubmatch(s2)

	// If both match the pattern (e.g., "rc10"), compare numerically
	if len(match1) == 3 && len(match2) == 3 {
		prefix1, numStr1 := match1[1], match1[2]
		prefix2, numStr2 := match2[1], match2[2]

		// If prefixes match (e.g., both "rc"), compare numbers
		if prefix1 == prefix2 {
			num1, err1 := strconv.Atoi(numStr1)
			num2, err2 := strconv.Atoi(numStr2)

			if err1 == nil && err2 == nil {
				if num1 < num2 {
					return -1
				}
				if num1 > num2 {
					return 1
				}
				return 0
			}
		}

		// If prefixes don't match, compare prefixes first
		if prefix1 != prefix2 {
			if prefix1 < prefix2 {
				return -1
			}
			if prefix1 > prefix2 {
				return 1
			}
		}

		// If prefixes match but we couldn't parse numbers, fall back to numeric comparison
		num1, err1 := strconv.Atoi(numStr1)
		num2, err2 := strconv.Atoi(numStr2)

		if err1 == nil && err2 == nil {
			if num1 < num2 {
				return -1
			}
			if num1 > num2 {
				return 1
			}
			return 0
		}
	}

	// Fall back to string comparison for non-standard formats
	if s1 < s2 {
		return -1
	}
	if s1 > s2 {
		return 1
	}
	return 0
}
