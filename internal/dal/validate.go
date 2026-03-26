package dal

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
)

var resourceNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

// ValidateResourceName returns an error if name is empty or contains characters
// outside [a-zA-Z0-9_].
func ValidateResourceName(name string) error {
	if name == "" {
		return errors.New("resource name must not be empty")
	}
	if !resourceNameRegex.MatchString(name) {
		return fmt.Errorf("resource name %q contains invalid characters: only alphanumeric characters and underscores are allowed", name)
	}
	return nil
}

// ParseLimitOffset parses limitStr and offsetStr into integers.
// Defaults: limit=100 when limitStr is empty, offset=0 when offsetStr is empty.
// Returns an error if either value is non-integer, if limit <= 0, offset < 0, or limit > 100.
func ParseLimitOffset(limitStr, offsetStr string) (int, int, error) {
	limit := 100
	if limitStr != "" {
		v, err := strconv.Atoi(limitStr)
		if err != nil {
			return 0, 0, fmt.Errorf("limit must be an integer, got %q", limitStr)
		}
		limit = v
	}

	offset := 0
	if offsetStr != "" {
		v, err := strconv.Atoi(offsetStr)
		if err != nil {
			return 0, 0, fmt.Errorf("offset must be an integer, got %q", offsetStr)
		}
		offset = v
	}

	if limit <= 0 {
		return 0, 0, fmt.Errorf("limit must be greater than 0, got %d", limit)
	}
	if limit > 100 {
		return 0, 0, fmt.Errorf("limit must not exceed 100, got %d", limit)
	}
	if offset < 0 {
		return 0, 0, fmt.Errorf("offset must be non-negative, got %d", offset)
	}

	return limit, offset, nil
}
