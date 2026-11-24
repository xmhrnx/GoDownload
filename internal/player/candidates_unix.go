//go:build !windows && !(darwin && !ios)

package player

func candidates() []known {
	return []known{
		{GoPlay, "GoPlay", []string{"goplay", "${HOME}/.local/bin/goplay"}, plain},
		{"mpv", "mpv", []string{"mpv"}, mpvArgs},
		{"vlc", "VLC", []string{"vlc"}, vlcArgs},
		{"celluloid", "Celluloid", []string{"celluloid"}, plain},
		{"smplayer", "SMPlayer", []string{"smplayer"}, plain},
		{"haruna", "Haruna", []string{"haruna"}, plain},
	}
}
