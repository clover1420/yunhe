package app

import "lilt/internal/ui/base/appui"

// The app reads the shared controls through these names while its views
// still live in this package; feature packages import lilt/internal/ui/base/appui.
type buttonSurface = appui.ButtonSurface

const (
	mediaSurface          = appui.MediaSurface
	playerSurface         = appui.PlayerSurface
	playerSelectedSurface = appui.PlayerSelectedSurface
)

var (
	iconButton     = appui.IconButton
	playButton     = appui.PlayButton
	playPause      = appui.PlayPause
	roundButton    = appui.RoundButton
	pillButton     = appui.PillButton
	mediaFeedback  = appui.MediaFeedback
	playerFeedback = appui.PlayerFeedback
	segmented      = appui.Segmented

	artIn    = appui.ArtIn
	artImage = appui.ArtImage
	artFade  = appui.ArtFade
	bitmap   = appui.Bitmap

	edgeShown = appui.EdgeShown

	clock       = appui.Clock
	plays       = appui.Plays
	count       = appui.Count
	greeting    = appui.Greeting
	megabytes   = appui.Megabytes
	commentTime = appui.RelativeTime
)

var (
	strokeIcon = appui.StrokeIcon
	fillIcon   = appui.FillIcon
)

var (
	iconPlay      = appui.IconPlay
	iconPause     = appui.IconPause
	iconNext      = appui.IconNext
	iconPrev      = appui.IconPrev
	iconShuffle   = appui.IconShuffle
	iconRepeat    = appui.IconRepeat
	iconRepeat1   = appui.IconRepeat1
	iconVolume    = appui.IconVolume
	iconVolumeLow = appui.IconVolumeLow
	iconMute      = appui.IconMute
	iconSearch    = appui.IconSearch
	iconLocate    = appui.IconLocate
	iconHome      = appui.IconHome
	iconDaily     = appui.IconDaily
	iconCharts    = appui.IconCharts
	iconQueue     = appui.IconQueue
	iconLyrics    = appui.IconLyrics
	iconDown      = appui.IconDown
	iconArrowDown = appui.IconArrowDown
	iconBack      = appui.IconBack
	iconForward   = appui.IconForward
	iconNote      = appui.IconNote
	iconPurchased = appui.IconPurchased
	iconHeart     = appui.IconHeart
	iconHeartFill = appui.IconHeartFill
	iconUser      = appui.IconUser
	iconPodcast   = appui.IconPodcast
	iconHeadset   = appui.IconHeadset
	iconRadio     = appui.IconRadio
	iconTrash     = appui.IconTrash
	iconThumb     = appui.IconThumb
	iconComment   = appui.IconComment
	iconPlus      = appui.IconPlus
	iconCheck     = appui.IconCheck
	iconX         = appui.IconX
	iconMore      = appui.IconMore
	iconThumbFill = appui.IconThumbFill
	iconReload    = appui.IconReload
	iconUpDown    = appui.IconUpDown
	iconSettings  = appui.IconSettings
	iconSun       = appui.IconSun
	iconInfo      = appui.IconInfo
	iconSignOut   = appui.IconSignOut
	iconGrid      = appui.IconGrid
)
