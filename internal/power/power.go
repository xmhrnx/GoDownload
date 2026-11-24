// Package power puts the computer to sleep, shuts it down, restarts it or
// signs the user out, for "when downloads finish" actions.
package power

import (
	"fmt"
	"os/exec"

	"github.com/godownload/godownload/internal/model"
)

// Perform carries out a system action. PowerNone and PowerQuitApp are the
// app's own business and do nothing here.
func Perform(a model.PowerAction) error {
	if a == model.PowerNone || a == model.PowerQuitApp {
		return nil
	}
	cmds := commands(a)
	if len(cmds) == 0 {
		return fmt.Errorf("%s isn't supported on this system", a.Title())
	}
	var last error
	for _, c := range cmds {
		out, err := exec.Command(c[0], c[1:]...).CombinedOutput()
		if err == nil {
			return nil
		}
		last = fmt.Errorf("%s: %v %s", c[0], err, out)
	}
	return last
}
