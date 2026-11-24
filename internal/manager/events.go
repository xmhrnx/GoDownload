package manager

import "github.com/godownload/godownload/internal/model"

// EventKind says what happened.
type EventKind int

const (
	// EventCompleted: a download finished.
	EventCompleted EventKind = iota
	// EventFailed: a download stopped with an error.
	EventFailed
	// EventInfo: something worth telling the user, such as repair results.
	EventInfo
	// EventAllDone: every download finished and a power action is due.
	EventAllDone
)

// Event is something the UI should tell the user about.
type Event struct {
	Kind    EventKind
	ID      string
	Title   string
	Message string
	// Action is the power action to run, for EventAllDone.
	Action model.PowerAction
	// Expired marks a failure from an expired link, which a fresh link can fix.
	Expired bool
}
