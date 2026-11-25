package dialogs

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/ui/state"
	"github.com/godownload/godownload/ui/theme"
)

// browserSettings: the extension's connection and pairing token.
type browserSettings struct {
	s        *state.State
	capture  *giode.SwitchElement
	token    *giode.InputElement
	generate *giode.ButtonElement
	copy     *giode.ButtonElement
}

func newBrowserSettings(s *state.State) *browserSettings {
	st := theme.Field
	st.Padding = giode.SymmetricInset(10, 7)
	st.Width = 260
	b := &browserSettings{
		s:       s,
		capture: newSwitch("Capture downloads from the browser"),
		token:   giode.Input.Text("No token: any extension on this computer may connect", st),
	}
	b.generate = giode.Button("Generate", theme.QuietButton).OnClick(func() {
		buf := make([]byte, 16)
		if _, err := rand.Read(buf); err != nil {
			s.Notify(err)
			return
		}
		b.token.SetText(hex.EncodeToString(buf))
	})
	b.copy = giode.Button("Copy", theme.QuietButton).Icon(theme.Icon("copy", 14, "")).OnClick(func() {
		s.App.SetClipboard(b.token.Text())
		s.Toasts.Show(giode.ToastInfo, "Token copied. Paste it in the extension's options.")
	})
	return b
}

func (b *browserSettings) Title() string { return "Browser" }
func (b *browserSettings) Icon() string  { return "globe" }

func (b *browserSettings) Load(st model.Settings) {
	b.capture.SetChecked(st.CaptureBrowser)
	b.token.SetText(st.APIToken)
}

func (b *browserSettings) Apply(st *model.Settings) error {
	st.CaptureBrowser = b.capture.Checked()
	st.APIToken = b.token.Text()
	b.s.Server.SetToken(st.APIToken)
	return nil
}

func (b *browserSettings) View() giode.Element {
	return giode.Stack(giode.Styles{Gap: 6, Width: -1},
		giode.Text("Install the GoDownload extension (Chrome, Edge, Brave, Opera or Firefox) from the extension folder that comes with GoDownload; its README has the steps. It sends downloads, links you right-click and videos it finds on pages.",
			giode.Styles{FontSize: theme.SizeSmall, Color: "theme.muted", LineHeight: 17}),
		switchRow(b.capture, "Off: the browser keeps its downloads; right-click › Download with GoDownload still works."),
		settingRow("Pairing token", "Optional. When set, only an extension with the same token can add downloads.",
			giode.Row(giode.Styles{Gap: 6, Align: giode.AlignCenter, Wrap: giode.WrapNoWrap}, b.token, b.generate, b.copy)),
	)
}
