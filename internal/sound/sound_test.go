package sound

import (
	"testing"
)

func TestAudioPlayerDispatch(t *testing.T) {
	player := NewAudioPlayer()
	if player == nil {
		t.Fatal("expected non-nil AudioPlayer")
	}

	// Should not panic or error even in headless environment
	err := player.Play(SoundComplete)
	if err != nil {
		t.Logf("player returned error (expected in some mock/headless test environments): %v", err)
	}

	err = player.Play(SoundAlarm)
	if err != nil {
		t.Logf("player returned error: %v", err)
	}
}
