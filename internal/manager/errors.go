package manager

import (
	"context"
	"errors"
	"net"
	"os"
	"strings"
	"time"

	"github.com/godownload/godownload/internal/hls"
	"github.com/godownload/godownload/internal/httpdl"
)

// now is time.Now, kept in one place so tests can see what the manager sees.
var now = time.Now

// friendlyError turns an engine error into a sentence for the list.
func friendlyError(err error) string {
	var se *httpdl.StatusError
	var dnsErr *net.DNSError
	switch {
	case errors.As(err, &se) && se.Expired():
		return "The link has expired or needs a login. Get a fresh link from the page and use Refresh link."
	case errors.As(err, &se):
		return se.Error()
	case errors.As(err, &dnsErr):
		return "Can't find the server " + dnsErr.Name + ". Check the address or your connection."
	case errors.Is(err, hls.ErrLive):
		return err.Error()
	case errors.Is(err, os.ErrPermission):
		return "No permission to write to the download folder."
	case errors.Is(err, context.DeadlineExceeded):
		return "The server stopped responding."
	case strings.Contains(err.Error(), "no space left"):
		return "The disk is full."
	}
	return err.Error()
}
