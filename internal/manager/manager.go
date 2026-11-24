// Package manager runs the download list: the queue, the engines for each
// protocol, speed limits, schedules, verification and streaming sources.
// Everything it does is safe to call from any goroutine.
package manager

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/godownload/godownload/internal/bt"
	"github.com/godownload/godownload/internal/httpdl"
	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/internal/ratelimit"
	"github.com/godownload/godownload/internal/store"
)

// Manager owns every download.
type Manager struct {
	store *store.Store

	mu       sync.Mutex
	settings model.Settings
	entries  map[string]*entry
	client   *http.Client
	down     *ratelimit.Limiter
	up       *ratelimit.Limiter
	engine   *bt.Engine
	dirty    bool
	// hadWork tracks whether anything was queued or running, to notice the
	// moment everything is done.
	hadWork      bool
	activeWindow map[string]bool // schedules whose window is open
	windowFired  map[string]bool // open windows whose power action already ran

	onChange func()
	onEvent  func(Event)

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// ErrNotFound means no download has the given ID.
var ErrNotFound = errors.New("download not found")

// New loads the list and settings from st. Call Start to begin working.
func New(st *store.Store, onChange func(), onEvent func(Event)) (*Manager, error) {
	settings, err := st.LoadSettings()
	if err != nil {
		return nil, fmt.Errorf("reading settings: %w", err)
	}
	list, err := st.LoadDownloads()
	if err != nil {
		return nil, fmt.Errorf("reading download list: %w", err)
	}
	client, err := httpdl.NewClient(settings.Proxy)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	m := &Manager{
		store:        st,
		settings:     settings,
		entries:      map[string]*entry{},
		client:       client,
		down:         ratelimit.New(settings.DownloadLimit),
		up:           ratelimit.New(settings.UploadLimit),
		activeWindow: map[string]bool{},
		windowFired:  map[string]bool{},
		onChange:     onChange,
		onEvent:      onEvent,
		ctx:          ctx,
		cancel:       cancel,
	}
	for _, d := range list {
		// A download that was running when the app quit goes back in the queue.
		if d.Status.Active() || d.Status == model.StatusQueued {
			d.Status = model.StatusQueued
		}
		if d.Status == model.StatusSeeding {
			d.Status = model.StatusQueued
		}
		m.entries[d.ID] = newEntry(d)
	}
	return m, nil
}

// Start resumes the queue and begins the once-a-second housekeeping.
func (m *Manager) Start() {
	m.wg.Add(1)
	go m.loop()
	m.mu.Lock()
	m.scheduleLocked()
	m.mu.Unlock()
}

// Close stops every download (they resume on the next start), saves, and
// stops the torrent client.
func (m *Manager) Close() error {
	m.cancel()
	m.wg.Wait()
	m.mu.Lock()
	var running []chan struct{}
	for _, e := range m.entries {
		if e.done != nil {
			running = append(running, e.done)
		}
	}
	m.mu.Unlock()
	for _, done := range running {
		<-done
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	err := m.saveLocked()
	if m.engine != nil {
		err = errors.Join(err, m.engine.Close())
	}
	return err
}

// Settings returns the current settings.
func (m *Manager) Settings() model.Settings {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.settings
}

// UpdateSettings saves new settings and applies what can change while running.
// Torrent network settings apply after a restart.
func (m *Manager) UpdateSettings(s model.Settings) error {
	m.mu.Lock()
	old := m.settings
	m.settings = s
	m.down.Set(s.DownloadLimit)
	m.up.Set(s.UploadLimit)
	if s.Proxy != old.Proxy {
		client, err := httpdl.NewClient(s.Proxy)
		if err != nil {
			m.settings = old
			m.mu.Unlock()
			return err
		}
		m.client = client
	}
	m.scheduleLocked()
	m.mu.Unlock()
	m.changed()
	return m.store.SaveSettings(s)
}

// torrentEngine starts the torrent client the first time a torrent needs it,
// so a list without torrents never pays for one.
func (m *Manager) torrentEngine() (*bt.Engine, error) {
	if m.engine != nil {
		return m.engine, nil
	}
	s := m.settings
	e, err := bt.NewEngine(bt.Config{
		DataDir:       m.store.TorrentDir(),
		Port:          s.TorrentPort,
		DHT:           s.DHT,
		PortForward:   s.PortForward,
		Encryption:    s.Encryption,
		ExtraTrackers: s.ExtraTrackers,
		Upload:        m.up,
		Download:      m.down,
	})
	if err != nil {
		return nil, err
	}
	m.engine = e
	return e, nil
}

// changed tells the UI to refresh.
func (m *Manager) changed() {
	if m.onChange != nil {
		m.onChange()
	}
}

func (m *Manager) emit(ev Event) {
	if m.onEvent != nil {
		m.onEvent(ev)
	}
}

// saveLocked writes the list. Called with m.mu held.
func (m *Manager) saveLocked() error {
	list := make([]model.Download, 0, len(m.entries))
	for _, e := range m.entries {
		list = append(list, e.d)
	}
	slices.SortFunc(list, func(a, b model.Download) int { return a.AddedAt.Compare(b.AddedAt) })
	m.dirty = false
	return m.store.SaveDownloads(list)
}

func (m *Manager) get(id string) (*entry, error) {
	e, ok := m.entries[id]
	if !ok {
		return nil, ErrNotFound
	}
	return e, nil
}

func newID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprint(time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
