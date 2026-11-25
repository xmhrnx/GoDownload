package server

import (
	"errors"
	"fmt"
	"net"
)

// Ports are the ports the local server may use, tried in order; the
// browser extension looks for the app on the same list, so there is nothing
// to configure. Windows often reserves whole port ranges for Hyper-V, WSL
// or Docker (netsh int ipv4 show excludedportrange protocol=tcp), so the
// list is spread far apart. Keep it in sync with APP_PORTS in
// extension/src/lib/api.js.
var Ports = []int{47821, 18821, 28821, 38821, 58821, 8821}

// listen opens the first of ports that is free.
func listen(ports []int) (net.Listener, error) {
	var errs []error
	for _, p := range ports {
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", p))
		if err == nil {
			return ln, nil
		}
		errs = append(errs, fmt.Errorf("port %d: %w", p, err))
	}
	return nil, errors.Join(errs...)
}
