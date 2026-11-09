package app

import (
	"time"

	"github.com/egoist/mygo/ui"
)

var white = paper

// lyricsPaneChosen is the pane of lyrics the listener chose last: what
// the app starts on, before the player is made. Now Playing shows an
// episode's in place of either.
func (a *app) lyricsPaneChosen() int {
	if a.settings.LyricsStage {
		return paneStage
	}
	return paneLyrics
}

// npTab is a tab of Now Playing: its pane, and what it says.
type npTab struct {
	pane  int
	label string
}

// npTabs returns the tabs of Now Playing, in order, for what plays: ⌘1
// chooses the first, ⌘2 the second, and on.
func (a *app) npTabs() []npTab {
	if tr := a.pl.Current(); tr != nil && tr.ProgramID != 0 {
		tabs := []npTab{{paneProgram, "简介"}}
		if a.captionsShown() {
			tabs = append(tabs, npTab{paneCaptions, "字幕"})
		}
		return append(tabs, npTab{paneQueue, "待播清单"}, npTab{paneComments, "评论"})
	}
	return []npTab{{paneLyrics, "歌词"}, {paneStage, "律动"}, {paneQueue, "待播清单"}, {paneWall, "选歌"}, {paneComments, "评论"}}
}

func (a *app) paneTabs(c *ui.Context) {
	for _, t := range a.npTabs() {
		label := t.label
		if t.pane == paneComments {
			label = a.commentsTab()
		}
		a.paneTab(c, t.pane, label)
	}
}

// choosePane shows pane i in Now Playing, remembering lyrics or the stage,
// whichever was chosen, for the next song.
func (a *app) choosePane(i int) {
	a.npPane = i
	if i != paneLyrics {
		a.lyricsSizeOpen = false
	}
	if (i == paneLyrics || i == paneStage) && a.settings.LyricsStage != (i == paneStage) {
		a.settings.LyricsStage = i == paneStage
		a.settings.Save()
	}
}

// lyricsSizeChanged resets the lyrics' layout after the size changed: a
// different size changes wrapping and line positions even while paused.
func (a *app) lyricsSizeChanged() {
	a.lyricAt = -2
	a.lyricGlide = lyricGlide{}
	a.settings.Save()
}

// radioChip says which radio plays, over the title: a click on it, or on
// its cross, leaves the radio, playing on the song.
func (a *app) radioChip(c *ui.Context, name string) {
	icon, exit := iconRadio, "退出电台"
	if name == heartRadio {
		icon, exit = iconHeartFill, "退出心动模式"
	}
	chip := ui.ButtonBase(c.Key("radio-chip")).Gap(5).Padding(3, 6, 3, 8).Margin(0, 0, 4, -8).Radius(999).
		AlignItems(ui.Center).AlignSelf(ui.Start).FocusRing(false).Label(exit).Tooltip(exit)
	bg, fg, cross := ui.Transparent, white.Alpha(0.6), float32(0.5)
	if chip.Hovered() || chip.FocusVisible() {
		bg, fg, cross = veil(2), veil(12), 1
	}
	if chip.Pressed() {
		bg = veil(3)
	}
	motion := ui.Animation{Duration: 140 * time.Millisecond}
	chip.Background(chip.AnimateColor("bg", bg, motion)).TextColor(chip.AnimateColor("fg", fg, motion))
	cross = chip.AnimateTo("cross", cross, motion)
	chip.Children(func() {
		ui.Icon(c, icon).Size(12, 12)
		ui.Text(c, radioLabel(name)).FontSize(11.5).FontWeight(700).LetterSpacing(0.3)
		ui.Icon(c, iconX).Size(11, 11).Opacity(cross)
	})
	if chip.Clicked() {
		a.exitRadio()
	}
}

func (a *app) paneTab(c *ui.Context, i int, label string) {
	b := ui.ButtonBase(c.Key(i)).Padding(6, 14).Radius(999).FocusRing(false).Label(label)
	bg, fg := ui.Transparent, white.Alpha(0.55)
	if a.npPane == i {
		bg, fg = veil(3), white
	}
	if b.Hovered() || b.FocusVisible() {
		bg, fg = veil(2), veil(12)
		if a.npPane == i {
			bg = veil(4)
		}
	}
	if b.Pressed() {
		bg, fg = veil(5), white
	}
	motion := ui.Animation{Duration: 140 * time.Millisecond}
	b.Background(b.AnimateColor("bg", bg, motion))
	fg = b.AnimateColor("fg", fg, motion)
	b.Children(func() {
		if i != paneComments {
			ui.Text(c, label).FontSize(13).FontWeight(600).TextColor(fg)
			return
		}
		// The count rolls as it changes, from one song's to the next.
		n, text := a.commentCount(), ""
		if n > 0 {
			text = " " + count(n)
		}
		ui.Text(c, "评论").FontSize(13).FontWeight(600).TextColor(fg)
		rollCount(c, &a.tabCount, n, text, fg)
	})
	if b.Clicked() {
		a.choosePane(i)
	}
}
