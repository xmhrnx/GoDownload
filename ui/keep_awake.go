package ui

// keepAwake stops the computer from idling to sleep while downloads run,
// and lets it sleep again once they stop.
func (r *Root) keepAwake() {
	active := r.s.Totals.Active > 0 || r.s.Totals.Seeding > 0
	switch {
	case active && r.release == nil:
		release, err := r.s.App.KeepAlive("Downloading")
		if err == nil {
			r.release = release
		}
	case !active && r.release != nil:
		r.release()
		r.release = nil
	}
}
