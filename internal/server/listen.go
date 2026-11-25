package server

import (
	"errors"
	"fmt"
	"net"
)

// FallbackPorts are tried, in order, when the configured port can't be used.
// The browser extension looks for the app on the same list, so it still
// connects. Windows often reserves whole port ranges for Hyper-V, WSL or
// Docker (netsh int ipv4 show excludedportrange protocol=tcp), so the list
// is spread far apart. Keep it in sync with FALLBACK_PORTS in
// extension/src/lib/api.js.
var FallbackPorts = []int{47821, 18821, 28821, 38821, 58821, 8821}

// listen opens the first port that works: the configured one, then the
// fallbacks. It reports every port that failed, for the user.
func listen(port int) (net.Listener, error) {
	ports := append([]int{port}, FallbackPorts...)
	var errs []error
	seen := map[int]bool{}
	for _, p := range ports {
		if p <= 0 || seen[p] {
			continue
		}
		seen[p] = true
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", p))
		if err == nil {
			return ln, nil
		}
		errs = append(errs, fmt.Errorf("port %d: %w", p, err))
	}
	return nil, errors.Join(errs...)
}
