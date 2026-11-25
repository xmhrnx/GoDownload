// GoDownload is a download manager: multi-connection HTTP downloads with
// resume and repair, torrents with per-torrent seeding control, video
// streams, playing while downloading, schedules and a browser extension.
package main

import (
	"errors"
	"fmt"
	"image"
	"log"
	"net/url"
	"os"
	"runtime/debug"
	"time"

	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/assets"
	"github.com/godownload/godownload/internal/fileutil"
	"github.com/godownload/godownload/internal/manager"
	"github.com/godownload/godownload/internal/server"
	"github.com/godownload/godownload/internal/store"
	"github.com/godownload/godownload/ui"
	"github.com/godownload/godownload/ui/state"
	"github.com/godownload/godownload/ui/theme"
)

func main() {
	// A download manager mostly moves bytes from sockets to disk; a small
	// heap target keeps it light without costing throughput.
	debug.SetGCPercent(50)
	debug.SetMemoryLimit(128 << 20)
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// goDownload is the running app's parts. They are created in startup, which
// runs only in the first copy of the app: a second launch just hands its
// arguments (links, .torrent files) to the first and exits.
type goDownload struct {
	dataDir string
	app     *giode.App
	m       *manager.Manager
	srv     *server.Server
	st      *state.State
	root    *ui.Root
}

func run() error {
	dataDir, err := fileutil.DataDir()
	if err != nil {
		return fmt.Errorf("data folder: %w", err)
	}
	if err := theme.Setup(); err != nil {
		return err
	}
	g := &goDownload{dataDir: dataDir}
	g.app = giode.New(giode.Config{
		Title:       "GoDownload",
		Size:        image.Pt(1240, 780),
		MinSize:     image.Pt(900, 560),
		Resizable:   true,
		Frameless:   true,
		Center:      true,
		Font:        theme.Font,
		Icon:        assets.Icon,
		Background:  "theme.background",
		CloseToTray: true,
		MaxFPS:      60,
		SingleInstanceLock: &giode.SingleInstanceLock{
			UniqueID:     "5d0c2f2e-godownload-0b6a7f8e",
			OnLaunchArgs: func(args []string) { g.root.OpenArgs(args) },
		},
		OnStartup:  g.startup,
		OnShutdown: g.shutdown,
		OnOpenURL:  func(u *url.URL) { g.root.OpenArgs([]string{u.String()}) },
		OnFileDrop: func(paths []string) { g.root.OpenArgs(paths) },
	})
	return g.app.Run(func() giode.Element { return g.root.View() })
}

func (g *goDownload) startup() {
	if err := g.start(); err != nil {
		log.Fatal(err)
	}
}

// start loads the list, starts the local server and builds the window.
func (g *goDownload) start() error {
	var err error
	g.m, err = manager.New(store.Open(g.dataDir),
		func() { g.st.Changed() },
		func(ev manager.Event) { g.app.Post(func() { g.root.OnEvent(ev) }) },
	)
	if err != nil {
		return err
	}
	settings := g.m.Settings()
	theme.Apply(settings.Theme)
	g.app.SetCloseToTray(settings.CloseToTray)

	g.srv, err = server.Start(server.Config{
		Port:      settings.APIPort,
		Token:     settings.APIToken,
		Intercept: func() bool { return g.m.Settings().CaptureBrowser },
		OnAdd: func(links []server.Link, batch bool) {
			g.app.Post(func() { g.root.FromBrowser(links, batch) })
		},
	}, g.m)
	if err != nil {
		return err
	}

	toasts := giode.Toaster().Placement(giode.ToastBottomEnd).Duration(5 * time.Second).Max(4).
		Inset(giode.Inset{Bottom: 36, Right: 8})
	g.st = state.New(g.app, g.m, g.srv, toasts)
	g.root = ui.NewRoot(g.st)
	g.m.Start()
	g.root.Start()
	if p := g.srv.Port(); p != settings.APIPort {
		// The extension finds the app on the fallback ports too, but say why
		// the configured one wasn't used.
		toasts.Show(giode.ToastWarning, fmt.Sprintf(
			"Browser extension port %d is unavailable (in use, or reserved by the system); using %d instead.", settings.APIPort, p))
	}
	g.root.OpenArgs(os.Args[1:])
	return nil
}

// shutdown saves the list and stops the server when the app quits.
func (g *goDownload) shutdown() {
	if g.m == nil {
		return
	}
	if err := errors.Join(g.srv.Close(), g.m.Close()); err != nil {
		log.Printf("shutting down: %v", err)
	}
}
