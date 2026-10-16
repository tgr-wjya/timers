package sound

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"timers/assets/sounds"
)

type SoundKind string

const (
	SoundComplete SoundKind = "complete"
	SoundAlarm    SoundKind = "alarm"
)

type AudioPlayer interface {
	Play(kind SoundKind) error
}

type DefaultAudioPlayer struct {
	tempChimePath string
	initOnce      sync.Once
}

func NewAudioPlayer() AudioPlayer {
	return &DefaultAudioPlayer{}
}

func (p *DefaultAudioPlayer) ensureTempChime() string {
	p.initOnce.Do(func() {
		tmpDir := os.TempDir()
		target := filepath.Join(tmpDir, "timer_chime.wav")
		if _, err := os.Stat(target); os.IsNotExist(err) && len(sounds.ChimeWAV) > 0 {
			_ = os.WriteFile(target, sounds.ChimeWAV, 0644)
		}
		p.tempChimePath = target
	})
	return p.tempChimePath
}

func (p *DefaultAudioPlayer) Play(kind SoundKind) error {
	// Ring terminal ASCII bell immediately as universal fallback
	fmt.Fprint(os.Stderr, "\a")

	// Run audio playback asynchronously in background
	go func() {
		_ = p.playSync(kind)
	}()

	return nil
}

func (p *DefaultAudioPlayer) playSync(kind SoundKind) error {
	switch runtime.GOOS {
	case "darwin":
		soundFile := "/System/Library/Sounds/Glass.aiff"
		if kind == SoundAlarm {
			soundFile = "/System/Library/Sounds/Ping.aiff"
		}
		cmd := exec.Command("afplay", soundFile)
		return cmd.Run()

	case "windows":
		cmd := exec.Command("powershell", "-c", "[System.Media.SystemSounds]::Exclamation.Play()")
		return cmd.Run()

	default: // Linux / Unix
		soundFile := "/usr/share/sounds/freedesktop/stereo/complete.oga"
		if kind == SoundAlarm {
			soundFile = "/usr/share/sounds/freedesktop/stereo/alarm-clock-elapsed.oga"
		}

		// If system sound does not exist, use embedded chime
		if _, err := os.Stat(soundFile); os.IsNotExist(err) {
			soundFile = p.ensureTempChime()
		}

		// Try pw-play (PipeWire on Nobara/Fedora)
		if pw, err := exec.LookPath("pw-play"); err == nil {
			if err := exec.Command(pw, soundFile).Run(); err == nil {
				return nil
			}
		}

		// Try paplay (PulseAudio)
		if pa, err := exec.LookPath("paplay"); err == nil {
			if err := exec.Command(pa, soundFile).Run(); err == nil {
				return nil
			}
		}

		// Try aplay (ALSA)
		if alsa, err := exec.LookPath("aplay"); err == nil {
			chime := p.ensureTempChime()
			if err := exec.Command(alsa, "-q", chime).Run(); err == nil {
				return nil
			}
		}

		return nil
	}
}
