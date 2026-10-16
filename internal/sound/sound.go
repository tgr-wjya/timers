package sound

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"timers/assets/sounds"
)

type SoundKind string

const (
	SoundComplete SoundKind = "complete"
	SoundAlarm    SoundKind = "alarm"
)

func Play(kind SoundKind) {
	fmt.Fprint(os.Stderr, "\a") // universal terminal bell

	go func() {
		switch runtime.GOOS {
		case "darwin":
			file := "/System/Library/Sounds/Glass.aiff"
			if kind == SoundAlarm {
				file = "/System/Library/Sounds/Ping.aiff"
			}
			_ = exec.Command("afplay", file).Run()
		case "windows":
			_ = exec.Command("powershell", "-c", "[System.Media.SystemSounds]::Exclamation.Play()").Run()
		default: // Linux / Unix
			file := "/usr/share/sounds/freedesktop/stereo/complete.oga"
			if kind == SoundAlarm {
				file = "/usr/share/sounds/freedesktop/stereo/alarm-clock-elapsed.oga"
			}
			if _, err := os.Stat(file); os.IsNotExist(err) && len(sounds.ChimeWAV) > 0 {
				file = filepath.Join(os.TempDir(), "timer_chime.wav")
				_ = os.WriteFile(file, sounds.ChimeWAV, 0644)
			}
			for _, bin := range []string{"pw-play", "paplay", "aplay"} {
				if path, err := exec.LookPath(bin); err == nil {
					if err := exec.Command(path, file).Run(); err == nil {
						return
					}
				}
			}
		}
	}()
}
