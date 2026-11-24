package dialogs

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/ui/components"
	"github.com/godownload/godownload/ui/theme"
)

// torrentSettings: network, seeding defaults and upload speed.
type torrentSettings struct {
	port     *giode.NumberInputElement
	dht      *giode.SwitchElement
	upnp     *giode.SwitchElement
	encrypt  *giode.SwitchElement
	trackers *giode.SwitchElement
	seed     *giode.SwitchElement
	ratio    *giode.NumberInputElement
	upLimit  *components.RateField
	upVal    int64
}

func newTorrentSettings() *torrentSettings {
	t := &torrentSettings{
		port:     newNumber(1024, 65535, 1),
		dht:      newSwitch("Find peers without trackers (DHT)"),
		upnp:     newSwitch("Open the port on the router (UPnP / NAT-PMP)"),
		encrypt:  newSwitch("Prefer encrypted connections"),
		trackers: newSwitch("Add well-known public trackers to public torrents"),
		seed:     newSwitch("Seed after downloading by default"),
		ratio:    giode.Input.Number(2, giode.Styles{Width: 140}).Range(0, 100).Step(0.5).Decimals(1),
	}
	t.upLimit = components.NewRateField(140, func(v int64) { t.upVal = v })
	return t
}

func (t *torrentSettings) Title() string { return "Torrents" }
func (t *torrentSettings) Icon() string  { return "magnet" }

func (t *torrentSettings) Load(st model.Settings) {
	t.port.SetValue(float64(st.TorrentPort))
	t.dht.SetChecked(st.DHT)
	t.upnp.SetChecked(st.PortForward)
	t.encrypt.SetChecked(st.Encryption)
	t.trackers.SetChecked(st.ExtraTrackers)
	t.seed.SetChecked(st.SeedByDefault)
	t.ratio.SetValue(st.SeedRatio)
	t.upLimit.Set(st.UploadLimit)
	t.upVal = st.UploadLimit
}

func (t *torrentSettings) Apply(st *model.Settings) error {
	st.TorrentPort = int(t.port.Value())
	st.DHT = t.dht.Checked()
	st.PortForward = t.upnp.Checked()
	st.Encryption = t.encrypt.Checked()
	st.ExtraTrackers = t.trackers.Checked()
	st.SeedByDefault = t.seed.Checked()
	st.SeedRatio = t.ratio.Value()
	st.UploadLimit = t.upVal
	return nil
}

func (t *torrentSettings) View() giode.Element {
	return giode.Stack(giode.Styles{Gap: 4, Width: -1},
		settingRow("Upload limit (KB/s)", "For all torrents together. Each torrent can have its own limit and seeding switch in its details.", t.upLimit),
		switchRow(t.seed, "Turn seeding on or off for each torrent in the add dialog or its details."),
		settingRow("Stop seeding at ratio", "Uploaded ÷ size. 0 seeds until you stop it.", t.ratio),
		settingRow("Listening port", "Peers connect here. Changes apply after restarting GoDownload.", t.port),
		switchRow(t.dht, ""),
		switchRow(t.upnp, "Lets peers reach you, which makes torrents much faster."),
		switchRow(t.encrypt, ""),
		switchRow(t.trackers, "Helps torrents whose own trackers are dead find peers. Never added to private torrents."),
		giode.Text("Network settings apply after restarting GoDownload.", theme.Muted(theme.SizeSmall)),
	)
}
