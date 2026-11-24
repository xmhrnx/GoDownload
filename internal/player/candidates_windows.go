package player

func candidates() []known {
	return []known{
		{GoPlay, "GoPlay", []string{"goplay", `${LOCALAPPDATA}\Programs\GoPlay\GoPlay.exe`, `${ProgramFiles}\GoPlay\GoPlay.exe`}, plain},
		{"mpv", "mpv", []string{"mpv", `${ProgramFiles}\mpv\mpv.exe`}, mpvArgs},
		{"vlc", "VLC", []string{"vlc", `${ProgramFiles}\VideoLAN\VLC\vlc.exe`, `${ProgramFiles(x86)}\VideoLAN\VLC\vlc.exe`}, vlcArgs},
		{"mpc-hc", "MPC-HC", []string{`${ProgramFiles}\MPC-HC\mpc-hc64.exe`, `${ProgramFiles(x86)}\K-Lite Codec Pack\MPC-HC64\mpc-hc64.exe`}, plain},
		{"potplayer", "PotPlayer", []string{`${ProgramFiles}\DAUM\PotPlayer\PotPlayerMini64.exe`}, plain},
	}
}
