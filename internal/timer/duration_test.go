package timer

import (
	"strings"
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		wantErr  bool
		errSub   string
	}{
		{input: "45s", expected: 45 * time.Second},
		{input: "10m", expected: 10 * time.Minute},
		{input: "1h30m", expected: 90 * time.Minute},
		{input: "1h 30m", expected: 90 * time.Minute},
		{input: "2h", expected: 2 * time.Hour},
		{input: "10m30s", expected: 10*time.Minute + 30*time.Second},
		{input: "1.5h", expected: 90 * time.Minute},
		{input: "1d", expected: 24 * time.Hour},
		{input: "2d", expected: 48 * time.Hour},
		{input: "90m", expected: 90 * time.Minute},
		{input: "", wantErr: true, errSub: "empty duration"},
		{input: "abc", wantErr: true, errSub: "invalid duration format"},
		{input: "90", wantErr: true, errSub: "missing time unit"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, err := ParseDuration(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for input %q, got nil", tc.input)
				}
				if tc.errSub != "" && !strings.Contains(err.Error(), tc.errSub) {
					t.Fatalf("expected error containing %q, got %q", tc.errSub, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for input %q: %v", tc.input, err)
			}
			if got != tc.expected {
				t.Fatalf("expected %v, got %v for input %q", tc.expected, got, tc.input)
			}
		})
	}
}
