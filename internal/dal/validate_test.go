package dal

import (
	"strconv"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// Feature: mariadb-dal-api, Property 15: Resource name validation rejects invalid characters

// validCharGen generates a single character string from [a-zA-Z0-9_].
func validCharGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"
		idx := rapid.IntRange(0, len(charset)-1).Draw(t, "idx")
		return string(charset[idx])
	})
}

// invalidCharGen generates a single rune that is NOT in [a-zA-Z0-9_].
func invalidCharGen() *rapid.Generator[rune] {
	return rapid.Custom(func(t *rapid.T) rune {
		// Use printable ASCII range 32-126, excluding [a-zA-Z0-9_]
		// Valid ranges to exclude: 48-57 (0-9), 65-90 (A-Z), 95 (_), 97-122 (a-z)
		// Printable ASCII outside those: 32-47, 58-64, 91-94, 96, 123-126
		ranges := [][2]int{
			{32, 47},
			{58, 64},
			{91, 94},
			{96, 96},
			{123, 126},
		}
		// Count total options
		total := 0
		for _, r := range ranges {
			total += r[1] - r[0] + 1
		}
		pick := rapid.IntRange(0, total-1).Draw(t, "pick")
		for _, r := range ranges {
			size := r[1] - r[0] + 1
			if pick < size {
				return rune(r[0] + pick)
			}
			pick -= size
		}
		return '!'
	})
}

// TestPropertyValidResourceNameAccepted verifies that any non-empty name
// consisting solely of [a-zA-Z0-9_] characters is accepted by ValidateResourceName.
// Validates: Requirements 9.2
func TestPropertyValidResourceNameAccepted(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate 1-20 valid characters and join them
		length := rapid.IntRange(1, 20).Draw(t, "length")
		var sb strings.Builder
		for i := 0; i < length; i++ {
			sb.WriteString(validCharGen().Draw(t, "char"))
		}
		name := sb.String()

		if err := ValidateResourceName(name); err != nil {
			t.Fatalf("expected nil error for valid name %q, got: %v", name, err)
		}
	})
}

// TestPropertyInvalidResourceNameRejected verifies that any name containing at
// least one character outside [a-zA-Z0-9_] is rejected by ValidateResourceName.
// Validates: Requirements 9.2
func TestPropertyInvalidResourceNameRejected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Build optional valid prefix and suffix around one invalid character
		prefixLen := rapid.IntRange(0, 10).Draw(t, "prefixLen")
		suffixLen := rapid.IntRange(0, 10).Draw(t, "suffixLen")

		var sb strings.Builder
		for i := 0; i < prefixLen; i++ {
			sb.WriteString(validCharGen().Draw(t, "prefixChar"))
		}
		sb.WriteRune(invalidCharGen().Draw(t, "invalidChar"))
		for i := 0; i < suffixLen; i++ {
			sb.WriteString(validCharGen().Draw(t, "suffixChar"))
		}
		name := sb.String()

		if err := ValidateResourceName(name); err == nil {
			t.Fatalf("expected error for invalid name %q, got nil", name)
		}
	})
}

// TestEmptyResourceNameRejected verifies that an empty string is rejected.
// Validates: Requirements 9.2
func TestEmptyResourceNameRejected(t *testing.T) {
	if err := ValidateResourceName(""); err == nil {
		t.Fatal("expected error for empty resource name, got nil")
	}
}

// Feature: mariadb-dal-api, Property 16: Limit and offset validation rejects invalid values

// nonNegativeIntStringGen generates a string representation of a non-negative integer (0 to 100).
func nonNegativeIntStringGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		v := rapid.IntRange(1, 100).Draw(t, "nonNegInt")
		return strconv.Itoa(v)
	})
}

// negativeIntStringGen generates a string representation of a negative integer.
func negativeIntStringGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		v := rapid.IntRange(-10000, -1).Draw(t, "negInt")
		return strconv.Itoa(v)
	})
}

// nonIntegerStringGen generates strings that are not valid integers (e.g. "abc", "1.5", "!").
func nonIntegerStringGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		options := []string{"abc", "1.5", "!", "foo", "1e3", "1_000", "--1", "1 2", "null", "true"}
		idx := rapid.IntRange(0, len(options)-1).Draw(t, "idx")
		return options[idx]
	})
}

// TestPropertyValidLimitOffsetAccepted verifies that non-negative integer strings
// for both limit and offset are accepted and parsed correctly.
// Validates: Requirements 9.3
func TestPropertyValidLimitOffsetAccepted(t *testing.T) {
	// Feature: mariadb-dal-api, Property 16: Limit and offset validation rejects invalid values
	rapid.Check(t, func(t *rapid.T) {
		limitStr := nonNegativeIntStringGen().Draw(t, "limitStr")
		offsetStr := nonNegativeIntStringGen().Draw(t, "offsetStr")

		limit, offset, err := ParseLimitOffset(limitStr, offsetStr)
		if err != nil {
			t.Fatalf("expected no error for limit=%q offset=%q, got: %v", limitStr, offsetStr, err)
		}
		expectedLimit, _ := strconv.Atoi(limitStr)
		expectedOffset, _ := strconv.Atoi(offsetStr)
		if limit != expectedLimit {
			t.Fatalf("expected limit=%d, got %d", expectedLimit, limit)
		}
		if offset != expectedOffset {
			t.Fatalf("expected offset=%d, got %d", expectedOffset, offset)
		}
	})
}

// TestPropertyNegativeLimitRejected verifies that a negative integer string for limit returns an error.
// Validates: Requirements 9.3
func TestPropertyNegativeLimitRejected(t *testing.T) {
	// Feature: mariadb-dal-api, Property 16: Limit and offset validation rejects invalid values
	rapid.Check(t, func(t *rapid.T) {
		limitStr := negativeIntStringGen().Draw(t, "limitStr")
		offsetStr := nonNegativeIntStringGen().Draw(t, "offsetStr")

		_, _, err := ParseLimitOffset(limitStr, offsetStr)
		if err == nil {
			t.Fatalf("expected error for negative limit=%q, got nil", limitStr)
		}
	})
}

// TestPropertyNegativeOffsetRejected verifies that a negative integer string for offset returns an error.
// Validates: Requirements 9.3
func TestPropertyNegativeOffsetRejected(t *testing.T) {
	// Feature: mariadb-dal-api, Property 16: Limit and offset validation rejects invalid values
	rapid.Check(t, func(t *rapid.T) {
		limitStr := nonNegativeIntStringGen().Draw(t, "limitStr")
		offsetStr := negativeIntStringGen().Draw(t, "offsetStr")

		_, _, err := ParseLimitOffset(limitStr, offsetStr)
		if err == nil {
			t.Fatalf("expected error for negative offset=%q, got nil", offsetStr)
		}
	})
}

// TestPropertyNonIntegerLimitRejected verifies that a non-integer string for limit returns an error.
// Validates: Requirements 9.3
func TestPropertyNonIntegerLimitRejected(t *testing.T) {
	// Feature: mariadb-dal-api, Property 16: Limit and offset validation rejects invalid values
	rapid.Check(t, func(t *rapid.T) {
		limitStr := nonIntegerStringGen().Draw(t, "limitStr")
		offsetStr := nonNegativeIntStringGen().Draw(t, "offsetStr")

		_, _, err := ParseLimitOffset(limitStr, offsetStr)
		if err == nil {
			t.Fatalf("expected error for non-integer limit=%q, got nil", limitStr)
		}
	})
}

// TestPropertyNonIntegerOffsetRejected verifies that a non-integer string for offset returns an error.
// Validates: Requirements 9.3
func TestPropertyNonIntegerOffsetRejected(t *testing.T) {
	// Feature: mariadb-dal-api, Property 16: Limit and offset validation rejects invalid values
	rapid.Check(t, func(t *rapid.T) {
		limitStr := nonNegativeIntStringGen().Draw(t, "limitStr")
		offsetStr := nonIntegerStringGen().Draw(t, "offsetStr")

		_, _, err := ParseLimitOffset(limitStr, offsetStr)
		if err == nil {
			t.Fatalf("expected error for non-integer offset=%q, got nil", offsetStr)
		}
	})
}
