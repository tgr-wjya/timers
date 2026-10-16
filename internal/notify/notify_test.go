package notify

import (
	"testing"
)

func TestSend(t *testing.T) {
	// Must not panic or hang
	Send("Test Timer", "Focus session completed", true)
}
