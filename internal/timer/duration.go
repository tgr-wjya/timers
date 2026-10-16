package timer

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func ParseDuration(input string) (time.Duration, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}

	// Remove internal spaces, e.g., "1h 30m" -> "1h30m"
	s = strings.ReplaceAll(s, " ", "")

	// Check for bare numbers lacking units (Review Focus requirement)
	if _, err := strconv.Atoi(s); err == nil {
		return 0, fmt.Errorf("invalid duration: missing time unit (did you mean '%ss' or '%sm'?)", s, s)
	}

	// Handle day formats like "1d" or "2.5d"
	lower := strings.ToLower(s)
	if strings.HasSuffix(lower, "d") {
		numStr := lower[:len(lower)-1]
		if days, err := strconv.ParseFloat(numStr, 64); err == nil {
			hours := days * 24
			return time.Duration(hours * float64(time.Hour)), nil
		}
	}

	// Standard Go time.ParseDuration supports "1h30m", "45s", "1.5h", "10m30s"
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid duration format %q: %w", input, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("duration must be greater than zero")
	}

	return d, nil
}
