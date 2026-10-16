package notify

import (
	"testing"
)

func TestNotifierDispatch(t *testing.T) {
	n := NewNotifier()
	if n == nil {
		t.Fatal("expected non-nil Notifier")
	}

	// Non-blocking notification dispatch should not panic or hang
	err := n.Notify("Test Timer", "Focus session completed", true)
	if err != nil {
		t.Logf("notify returned error (expected in mock/headless test environments): %v", err)
	}
}
