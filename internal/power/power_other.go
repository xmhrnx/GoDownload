//go:build !windows && !(darwin && !ios) && !((linux && !android) || freebsd || openbsd || netbsd)

package power

import "github.com/godownload/godownload/internal/model"

func commands(model.PowerAction) [][]string { return nil }
