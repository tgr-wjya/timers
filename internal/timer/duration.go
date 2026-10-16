package timer

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	dayRegex   = regexp.MustCompile(`(?i)^(\d+(?:\.\d+)?)\s*d(?:ays?)?$`)
	bareNumber = regexp.MustCompile(`^\d+$`)
)

func ParseDuration(input string) (time.Duration, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}

	// Remove internal spaces, e.g., "1h 30m" -> "1h30m"
	s = strings.ReplaceAll(s, " ", "")

	// Check for bare numbers lacking units (Review Focus requirement)
	if bareNumber.MatchString(s) {
		return 0, fmt.Errorf("invalid duration: missing time unit (did you mean '%ss' or '%sm'?)", s, s)
	}

	// Handle day formats like "1d" or "2.5d"
	if matches := dayRegex.FindStringSubmatch(s); matches != nil {
		days, err := strconv.ParseFloat(matches[1], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid day duration: %w", err)
		}
		hours := days * 24
		return time.Duration(hours * float64(time.Hour)), nil
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
