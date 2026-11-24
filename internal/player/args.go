package player

func mpvArgs(url, title string) []string {
	return []string{"--force-media-title=" + title, url}
}

func vlcArgs(url, title string) []string {
	return []string{"--meta-title=" + title, url}
}
