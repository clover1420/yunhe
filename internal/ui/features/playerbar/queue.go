package playerbar

import (
	"github.com/egoist/mygo/ui"

	"lilt/internal/ui/base/appui"
)

// queueButton opens the queue above the player without leaving the current page.
func queueButton(c *ui.Context, d Deps) {
	p := d.Palette
	surface := appui.PlayerSurface
	if *d.QueueOpen {
		surface = appui.PlayerSelectedSurface
	}
	b := appui.IconButton(c.Key("queue"), appui.IconQueue, 16, p.Muted, "待播清单", surface).
		Disabled(d.Player.Current() == nil)
	if b.Clicked() {
		*d.QueueOpen = !*d.QueueOpen
		if *d.QueueOpen {
			*d.QueuePopupAt = -1
		}
	}
	ink := d.Ink(false)
	ui.PopoverBase(c.Key("queue-popover"), b, d.QueueOpen, func(panel ui.Element) {
		const inset = 16
		w, h := c.Size()
		// Include the list padding and panel border so short queues fit without scrolling.
		height := min(float32(len(d.Player.Queue())*60+68), 460, h-d.TopBar(c)-Height-24)
		panel.AttachTo(b, ui.AnchorTopEnd, ui.AnchorBottomEnd).
			Margin(0, 0, (Height-32)/2+8, 0).Size(min(320, w-32), height).
			Padding(8, 0).TextColor(ink.Text).Label("待播清单")
		d.Pane(panel, 14, false)
		// Overlay content sits outside the page's Space shortcut scope.
		panel.OnShortcut(0, ui.KeySpace, d.Player.TogglePlay)
		// Align the heading with the covers inside the rows' 10-DIP padding.
		ui.Row(c).Height(40).PaddingX(inset + 10).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "待播清单").FontSize(15).FontWeight(700).TextColor(ink.Text)
			ui.Textf(c, "%d", len(d.Player.Queue())).FontSize(12).FontFeatures("tnum").TextColor(ink.Muted)
		})
		// Equal gutters keep highlights centered and leave the scrollbar outside song hit targets.
		d.QueueTracks(c, d.QueuePopup, d.QueuePopupAt, p).
			Padding(4, inset, 8, inset).ScrollbarInsets(4, 0, 8, 0)
	})
}
