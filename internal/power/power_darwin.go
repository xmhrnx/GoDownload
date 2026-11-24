//go:build darwin && !ios

package power

import "github.com/godownload/godownload/internal/model"

func commands(a model.PowerAction) [][]string {
	systemEvents := func(verb string) []string {
		return []string{"osascript", "-e", `tell application "System Events" to ` + verb}
	}
	switch a {
	case model.PowerShutdown:
		return [][]string{systemEvents("shut down")}
	case model.PowerRestart:
		return [][]string{systemEvents("restart")}
	case model.PowerSignOut:
		return [][]string{systemEvents("log out")}
	case model.PowerSleep, model.PowerHibernate:
		return [][]string{{"pmset", "sleepnow"}}
	}
	return nil
}
