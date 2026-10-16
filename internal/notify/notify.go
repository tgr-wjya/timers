package notify

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

type Notifier interface {
	Notify(title, message string, critical bool) error
}

type DefaultNotifier struct{}

func NewNotifier() Notifier {
	return &DefaultNotifier{}
}

func (n *DefaultNotifier) Notify(title, message string, critical bool) error {
	// Emit terminal OSC 777 / OSC 99 desktop notification sequence for Kitty and modern terminals
	fmt.Fprintf(os.Stdout, "\033]777;notify;%s;%s\007", title, message)

	// Dispatch OS native desktop notification asynchronously
	go func() {
		_ = n.notifySync(title, message, critical)
	}()

	return nil
}

func (n *DefaultNotifier) notifySync(title, message string, critical bool) error {
	switch runtime.GOOS {
	case "darwin":
		script := fmt.Sprintf(`display notification %q with title %q`, message, title)
		cmd := exec.Command("osascript", "-e", script)
		return cmd.Run()

	case "windows":
		psScript := fmt.Sprintf(`
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] > $null
$template = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02)
$nodes = $template.GetElementsByTagName("text")
$nodes.Item(0).AppendChild($template.CreateTextNode(%q)) > $null
$nodes.Item(1).AppendChild($template.CreateTextNode(%q)) > $null
$toast = [Windows.UI.Notifications.ToastNotification]::new($template)
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("timer").Show($toast)
`, title, message)
		cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)
		return cmd.Run()

	default: // Linux / Unix
		if path, err := exec.LookPath("notify-send"); err == nil {
			args := []string{
				"--app-name=timer",
				"--icon=alarm-symbolic",
			}
			if critical {
				args = append(args, "--urgency=critical")
			}
			args = append(args, title, message)
			cmd := exec.Command(path, args...)
			return cmd.Run()
		}
		return nil
	}
}
