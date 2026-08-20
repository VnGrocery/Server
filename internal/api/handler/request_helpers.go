package handler

import (
	"fmt"
	"strconv"
	"strings"
)

func parsePositiveIntQuery(raw, field string) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, fmt.Errorf("%s is required", field)
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", field)
	}
	return parsed, nil
}

// parseOptionalPositiveIntQuery reads a query parameter that may be absent.
//
// Absent means "use the default", which is not the same as a caller sending
// nonsense: an unparseable value is still an error rather than being quietly
// replaced by the default.
func parseOptionalPositiveIntQuery(raw, field string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, nil
	}
	return parsePositiveIntQuery(raw, field)
}
