//go:build darwin && !ios

package player

func candidates() []known {
	return []known{
		{GoPlay, "GoPlay", []string{"/Applications/GoPlay.app/Contents/MacOS/GoPlay", "${HOME}/Applications/GoPlay.app/Contents/MacOS/GoPlay", "goplay"}, plain},
		{"iina", "IINA", []string{"/Applications/IINA.app/Contents/MacOS/iina-cli"}, plain},
		{"mpv", "mpv", []string{"mpv", "/opt/homebrew/bin/mpv", "/usr/local/bin/mpv", "/Applications/mpv.app/Contents/MacOS/mpv"}, mpvArgs},
		{"vlc", "VLC", []string{"/Applications/VLC.app/Contents/MacOS/VLC"}, vlcArgs},
	}
}
