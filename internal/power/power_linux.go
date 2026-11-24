//go:build (linux && !android) || freebsd || openbsd || netbsd

package power

import (
	"os"

	"github.com/godownload/godownload/internal/model"
)

// commands tries systemd first, then desktop-specific tools, so the action
// works without root on the common desktops.
func commands(a model.PowerAction) [][]string {
	switch a {
	case model.PowerShutdown:
		return [][]string{{"systemctl", "poweroff"}, {"loginctl", "poweroff"}}
	case model.PowerRestart:
		return [][]string{{"systemctl", "reboot"}, {"loginctl", "reboot"}}
	case model.PowerSleep:
		return [][]string{{"systemctl", "suspend"}, {"loginctl", "suspend"}}
	case model.PowerHibernate:
		return [][]string{{"systemctl", "hibernate"}, {"loginctl", "hibernate"}}
	case model.PowerSignOut:
		var cmds [][]string
		if id := os.Getenv("XDG_SESSION_ID"); id != "" {
			cmds = append(cmds, []string{"loginctl", "terminate-session", id})
		}
		return append(cmds,
			[]string{"gnome-session-quit", "--logout", "--no-prompt"},
			[]string{"qdbus", "org.kde.ksmserver", "/KSMServer", "logout", "0", "0", "0"},
			[]string{"xfce4-session-logout", "--logout"},
			[]string{"loginctl", "terminate-user", os.Getenv("USER")},
		)
	}
	return nil
}
