package lyricsui

import "sort"

const (
	TaskbarEdgeGap    = 8
	TaskbarCoverInset = 6
	// Match the visible margin from the taskbar edge to the album cover.
	TaskbarCoverGap = TaskbarEdgeGap + TaskbarCoverInset
)

// Native placement uses physical pixels in the Explorer parent's client space.
type Placement struct {
	Parent, Taskbar     uintptr
	X, Y, Width, Height int
	DPI                 uint32
	Dark                bool
	// RightAligned anchors the lyrics beside the notification area on
	// left-aligned taskbars and mirrors the cover to the right edge.
	RightAligned bool
}

type Interval struct{ start, end int }

// Find a real gap without moving Explorer's buttons. No occupancy means unknown.
func TaskbarFreeSpace(length, requested, minimum, gap int, occupied []Interval) (int, int) {
	if length <= 0 || requested < minimum || minimum <= 0 || gap < 0 || len(occupied) == 0 {
		return 0, 0
	}
	blocked := make([]Interval, 0, len(occupied))
	for _, item := range occupied {
		if item.end <= item.start || item.end <= 0 || item.start >= length {
			continue
		}
		blocked = append(blocked, Interval{max(0, item.start-gap), min(length, item.end+gap)})
	}
	if len(blocked) == 0 {
		return 0, 0
	}
	sort.Slice(blocked, func(i, j int) bool { return blocked[i].start < blocked[j].start })
	// Prefer the leftmost usable space, even when another gap would be wider.
	// Keep the same margin at the outer taskbar edges as around its controls.
	cursor := gap
	for _, item := range blocked {
		if width := min(requested, item.start-cursor); width >= minimum {
			return cursor, width
		}
		cursor = max(cursor, item.end)
	}
	if width := min(requested, length-gap-cursor); width >= minimum {
		return cursor, width
	}
	return 0, 0
}

// Right-anchored placement mirrors the taskbar so the same scan picks the
// rightmost usable gap beside the notification area, then mirrors it back.
func TaskbarFreeSpaceRight(length, requested, minimum, gap int, occupied []Interval) (int, int) {
	mirrored := make([]Interval, len(occupied))
	for i, item := range occupied {
		mirrored[i] = Interval{length - item.end, length - item.start}
	}
	x, width := TaskbarFreeSpace(length, requested, minimum, gap, mirrored)
	if width == 0 {
		return 0, 0
	}
	return length - x - width, width
}

// Centered taskbars strictly stay on the left side and hide when it fills up;
// left-aligned taskbars anchor beside the right notification area so opening
// and closing apps does not bounce the lyrics across the taskbar.
func TaskbarPlacementSpace(length, requested, minimum, gap int, occupied []Interval, leftTaskbar bool) (int, int) {
	if leftTaskbar {
		x, width := TaskbarFreeSpaceRight(length, requested, minimum, gap, occupied)
		if width == 0 || x+width <= length/2 {
			return 0, 0
		}
		return x, width
	}
	x, width := TaskbarFreeSpace(length, requested, minimum, gap, occupied)
	if width == 0 || x >= length/2 {
		return 0, 0
	}
	return x, width
}
