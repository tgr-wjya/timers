package sound

import "testing"

func TestPlay(t *testing.T) {
	// Must not panic on either sound kind
	Play(SoundComplete)
	Play(SoundAlarm)
}
