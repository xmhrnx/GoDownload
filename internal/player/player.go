// Package player finds media players on the computer and starts them on a
// stream URL, so a file can be watched while it downloads.
package player

import (
	"errors"
	"os"
	"os/exec"
	"strings"
)

// Player is a media player that can open an http:// stream.
type Player struct {
	ID   string
	Name string
	Path string
	// args builds the command line for a URL and a title.
	args func(url, title string) []string
}

// IDs for the settings.
const (
	Auto   = "auto"
	Custom = "custom"
	System = "system"
	GoPlay = "goplay"
)

// ErrNotFound means no player could be found; the caller falls back to the
// system's handler for the URL.
var ErrNotFound = errors.New("no media player found")

type known struct {
	id, name string
	paths    []string
	args     func(url, title string) []string
}

func plain(url, _ string) []string { return []string{url} }

// Detect lists the installed players, GoPlay first.
func Detect() []Player {
	var out []Player
	for _, k := range candidates() {
		if p, ok := locate(k); ok {
			out = append(out, p)
		}
	}
	return out
}

func locate(k known) (Player, bool) {
	for _, p := range k.paths {
		if strings.ContainsAny(p, `/\`) {
			if _, err := os.Stat(expand(p)); err == nil {
				return Player{ID: k.id, Name: k.name, Path: expand(p), args: k.args}, true
			}
			continue
		}
		if found, err := exec.LookPath(p); err == nil {
			return Player{ID: k.id, Name: k.name, Path: found, args: k.args}, true
		}
	}
	return Player{}, false
}

func expand(p string) string {
	return os.Expand(p, func(v string) string {
		if v == "HOME" {
			h, _ := os.UserHomeDir()
			return h
		}
		return os.Getenv(v)
	})
}

// Resolve finds the player for a setting: a player ID, Auto (the first
// installed, GoPlay preferred) or Custom with a command template.
func Resolve(id, command string) (Player, error) {
	switch id {
	case Custom:
		parts := splitCommand(command)
		if len(parts) == 0 {
			return Player{}, errors.New("no custom player command set")
		}
		return Player{ID: Custom, Name: "Custom", Path: parts[0], args: func(url, title string) []string {
			return fillTemplate(parts[1:], url, title)
		}}, nil
	case System:
		return Player{}, ErrNotFound
	}
	players := Detect()
	for _, p := range players {
		if p.ID == id {
			return p, nil
		}
	}
	if id == Auto && len(players) > 0 {
		return players[0], nil
	}
	return Player{}, ErrNotFound
}

// Launch starts the player on url without waiting for it.
func Launch(p Player, url, title string) error {
	cmd := exec.Command(p.Path, p.args(url, title)...)
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}

func fillTemplate(args []string, url, title string) []string {
	out := make([]string, 0, len(args)+1)
	hasURL := false
	for _, a := range args {
		if strings.Contains(a, "{url}") {
			hasURL = true
		}
		a = strings.ReplaceAll(a, "{url}", url)
		out = append(out, strings.ReplaceAll(a, "{title}", title))
	}
	if !hasURL {
		out = append(out, url)
	}
	return out
}

// splitCommand splits a command line on spaces, keeping quoted parts together.
func splitCommand(s string) []string {
	var out []string
	var cur strings.Builder
	var quote rune
	inWord := false
	for _, r := range s {
		switch {
		case quote != 0 && r == quote:
			quote = 0
		case quote == 0 && (r == '"' || r == '\''):
			quote, inWord = r, true
		case quote == 0 && (r == ' ' || r == '\t'):
			if inWord {
				out = append(out, cur.String())
				cur.Reset()
				inWord = false
			}
		default:
			cur.WriteRune(r)
			inWord = true
		}
	}
	if inWord {
		out = append(out, cur.String())
	}
	return out
}

// GoPlayURL is where to get GoPlay, offered when no player is installed.
const GoPlayURL = "https://github.com/mehran9675/goplay"
