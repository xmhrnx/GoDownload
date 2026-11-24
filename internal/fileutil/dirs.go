package fileutil

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// DownloadsDir is the user's Downloads folder: XDG_DOWNLOAD_DIR on Linux,
// ~/Downloads elsewhere.
func DownloadsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	if runtime.GOOS == "linux" {
		if d := xdgDownloadDir(home); d != "" {
			return d
		}
	}
	return filepath.Join(home, "Downloads")
}

func xdgDownloadDir(home string) string {
	if d := os.Getenv("XDG_DOWNLOAD_DIR"); d != "" {
		return d
	}
	cfg, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	f, err := os.Open(filepath.Join(cfg, "user-dirs.dirs"))
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		v, ok := strings.CutPrefix(line, "XDG_DOWNLOAD_DIR=")
		if !ok {
			continue
		}
		v = strings.Trim(v, `"`)
		return strings.Replace(v, "$HOME", home, 1)
	}
	return ""
}

// DataDir is where GoDownload keeps its list, settings and resume data.
func DataDir() (string, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(cfg, "GoDownload")
	return dir, os.MkdirAll(dir, 0o755)
}
