package notify

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

func Send(title, message string, critical bool) {
	fmt.Fprintf(os.Stdout, "\033]777;notify;%s;%s\007", title, message)

	go func() {
		switch runtime.GOOS {
		case "darwin":
			script := fmt.Sprintf(`display notification %q with title %q`, message, title)
			_ = exec.Command("osascript", "-e", script).Run()
		case "windows":
			psScript := fmt.Sprintf(`[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] > $null; $t = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent(0); $t.GetElementsByTagName('text')[0].AppendChild($t.CreateTextNode(%q)) > $null; [Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('timer').Show([Windows.UI.Notifications.ToastNotification]::new($t))`, title+" - "+message)
			_ = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psScript).Run()
		default:
			if path, err := exec.LookPath("notify-send"); err == nil {
				args := []string{"--app-name=timer", "--icon=alarm-symbolic"}
				if critical {
					args = append(args, "--urgency=critical")
				}
				args = append(args, title, message)
				_ = exec.Command(path, args...).Run()
			}
		}
	}()
}
