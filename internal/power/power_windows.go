package power

import "github.com/godownload/godownload/internal/model"

func commands(a model.PowerAction) [][]string {
	switch a {
	case model.PowerShutdown:
		return [][]string{{"shutdown", "/s", "/t", "0"}}
	case model.PowerRestart:
		return [][]string{{"shutdown", "/r", "/t", "0"}}
	case model.PowerSignOut:
		return [][]string{{"shutdown", "/l"}}
	case model.PowerHibernate:
		return [][]string{{"shutdown", "/h"}}
	case model.PowerSleep:
		// SetSuspendState(hibernate=0, force=1, wakeupEventsDisabled=0).
		return [][]string{{"rundll32.exe", "powrprof.dll,SetSuspendState", "0,1,0"}}
	}
	return nil
}
