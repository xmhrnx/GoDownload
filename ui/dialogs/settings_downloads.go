package dialogs

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/ui/components"
	"github.com/godownload/godownload/ui/theme"
)

// downloadSettings: the queue, connections, speed, retries and proxy.
type downloadSettings struct {
	maxActive *giode.NumberInputElement
	conns     *giode.NumberInputElement
	retries   *giode.NumberInputElement
	limit     *components.RateField
	limitVal  int64
	ask       *giode.SwitchElement
	proxy     *giode.InputElement
}

func newDownloadSettings() *downloadSettings {
	d := &downloadSettings{
		maxActive: newNumber(1, 20, 1),
		conns:     newNumber(1, 32, 1),
		retries:   newNumber(0, 100, 1),
		ask:       newSwitch("Ask where to save downloads from the browser"),
	}
	d.limit = components.NewRateField(140, func(v int64) { d.limitVal = v })
	st := theme.Field
	st.Padding = giode.SymmetricInset(10, 7)
	st.Width = 260
	d.proxy = giode.Input.Text("http://host:port or socks5://host:port", st)
	return d
}

func (d *downloadSettings) Title() string { return "Downloads" }
func (d *downloadSettings) Icon() string  { return "download" }

func (d *downloadSettings) Load(st model.Settings) {
	d.maxActive.SetValue(float64(st.MaxActive))
	d.conns.SetValue(float64(st.Connections))
	d.retries.SetValue(float64(st.Retries))
	d.limit.Set(st.DownloadLimit)
	d.limitVal = st.DownloadLimit
	d.ask.SetChecked(st.AskWhereToSave)
	d.proxy.SetText(st.Proxy)
}

func (d *downloadSettings) Apply(st *model.Settings) error {
	st.MaxActive = int(d.maxActive.Value())
	st.Connections = int(d.conns.Value())
	st.Retries = int(d.retries.Value())
	st.DownloadLimit = d.limitVal
	st.AskWhereToSave = d.ask.Checked()
	st.Proxy = d.proxy.Text()
	return nil
}

func (d *downloadSettings) View() giode.Element {
	return giode.Stack(giode.Styles{Gap: 4, Width: -1},
		settingRow("Downloads at the same time", "More wait in the queue and start as others finish.", d.maxActive),
		settingRow("Connections per download", "Each file is split into parts fetched in parallel. 8 suits most servers; some allow fewer.", d.conns),
		settingRow("Speed limit (KB/s)", "For all downloads together. Empty for no limit; 2M is 2 MB/s.", d.limit),
		settingRow("Retries", "How often a broken connection is retried before the download fails.", d.retries),
		settingRow("Proxy", "Leave empty to use the system's.", d.proxy),
		switchRow(d.ask, "Off: downloads the browser sends start right away in their category folder."),
	)
}
