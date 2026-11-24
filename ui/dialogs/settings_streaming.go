package dialogs

import (
	"context"

	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/internal/player"
	"github.com/godownload/godownload/ui/state"
	"github.com/godownload/godownload/ui/theme"
)

// streamingSettings: which player opens streams, and the read-ahead buffer.
type streamingSettings struct {
	s       *state.State
	ids     []string
	player  *giode.SelectElement
	command *giode.InputElement
	buffer  *giode.NumberInputElement
	goplay  *giode.ButtonElement
}

func newStreamingSettings(s *state.State) *streamingSettings {
	st := theme.Field
	st.Padding = giode.SymmetricInset(10, 7)
	g := &streamingSettings{
		s:       s,
		command: giode.Input.Text(`e.g. /usr/bin/myplayer --title "{title}" {url}`, st),
		buffer:  newNumber(4, 512, 4),
	}
	g.goplay = giode.Button("Get GoPlay", theme.PrimaryButton).Icon(theme.Icon("tv", 15, "#FFFFFF")).OnClick(func() {
		go func() {
			err := s.App.OpenURL(context.Background(), player.GoPlayURL)
			s.App.Post(func() { s.Notify(err) })
		}()
	})
	g.player = giode.Input.Select(theme.Field, "Automatic")
	return g
}

func (g *streamingSettings) Title() string { return "Streaming" }
func (g *streamingSettings) Icon() string  { return "tv" }

func (g *streamingSettings) Load(st model.Settings) {
	g.ids = []string{player.Auto}
	names := []string{"Automatic (GoPlay first)"}
	for _, p := range player.Detect() {
		g.ids = append(g.ids, p.ID)
		names = append(names, p.Name)
	}
	g.ids = append(g.ids, player.Custom, player.System)
	names = append(names, "Custom command", "Web browser")
	g.player.SetOptions(names...)
	for i, id := range g.ids {
		if id == st.Player {
			g.player.SetSelected(i)
		}
	}
	g.command.SetText(st.PlayerCommand)
	g.buffer.SetValue(float64(st.StreamBufferMB))
}

func (g *streamingSettings) Apply(st *model.Settings) error {
	st.Player = g.ids[g.player.Selected()]
	st.PlayerCommand = g.command.Text()
	st.StreamBufferMB = int(g.buffer.Value())
	return nil
}

func (g *streamingSettings) View() giode.Element {
	promo := giode.Row(giode.Styles{Gap: 14, Align: giode.AlignCenter, Width: -1, Wrap: giode.WrapNoWrap,
		Padding: giode.UniformInset(14), BorderRadius: 12, BackgroundGradient: giode.Linear(120, "#2A2250", "#14324A")},
		theme.Icon("tv", 28, "#FFFFFF"),
		giode.Stack(giode.Styles{Gap: 3, FlexGrow: 1},
			giode.Text("GoPlay", giode.Styles{FontSize: theme.SizeTitle, FontWeight: giode.FontWeightBold, Color: "#FFFFFF"}),
			giode.Text("Made for streams from GoDownload: shows what's downloaded on the seek bar and starts instantly.",
				giode.Styles{FontSize: theme.SizeSmall, Color: "#D6D9E8", LineHeight: 17}),
		),
		g.goplay,
	)
	kids := []giode.Element{
		giode.Text("Play while downloading streams the file over a local link, so seeking anywhere works: the part you jump to is downloaded first and the player waits for it instead of failing.",
			giode.Styles{FontSize: theme.SizeSmall, Color: "theme.muted", LineHeight: 17}),
		promo,
		settingRow("Player", "Any player that opens network streams works.", giode.Box(giode.Styles{Width: 240}, g.player)),
	}
	if g.ids != nil && g.ids[g.player.Selected()] == player.Custom {
		kids = append(kids, settingRow("Command", "{url} is the stream link, {title} the file name.", giode.Box(giode.Styles{Width: 320}, g.command)))
	}
	kids = append(kids, settingRow("Read-ahead (MB)", "How far past the play position is fetched first.", g.buffer))
	return giode.Stack(giode.Styles{Gap: 10, Width: -1}, kids...)
}
