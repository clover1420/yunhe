package lyricsui

import "testing"

// Occupied buttons and their safety margins never overlap the selected lyrics gap.
func TestTaskbarLyricsFreeSpace(t *testing.T) {
	tests := []struct {
		name         string
		occupied     []Interval
		start, width int
	}{
		{"centered buttons", []Interval{{400, 600}, {900, 1000}}, 8, 280},
		{"left before wider right gap", []Interval{{180, 500}}, 8, 164},
		{"left widgets", []Interval{{0, 150}, {500, 800}}, 158, 280},
		{"overlapping intervals", []Interval{{400, 800}, {0, 450}, {900, 1000}}, 0, 0},
		{"narrow but usable", []Interval{{0, 600}, {800, 1000}}, 608, 184},
		{"full", []Interval{{0, 1000}}, 0, 0},
		{"unknown", nil, 0, 0},
		{"invalid", []Interval{{20, 10}}, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, width := TaskbarFreeSpace(1000, 280, 100, 8, tt.occupied)
			if start != tt.start || width != tt.width {
				t.Fatalf("got (%d,%d), want (%d,%d)", start, width, tt.start, tt.width)
			}
		})
	}
}

// Long sentences remain inside their content range while both short rows begin beside the cover.
func TestTaskbarLyricScrollBounds(t *testing.T) {
	for _, tc := range []struct{ width, available, progress, want float32 }{
		{80, 100, 0, 0}, {80, 100, 80, 0}, {300, 100, 0, 0},
		{300, 100, 100, -45}, {300, 100, 300, -200}, {300, 100, 500, -200},
	} {
		if got := taskbarLyricOffset(tc.width, tc.available, tc.progress); got != tc.want {
			t.Errorf("offset(%v,%v,%v)=%v want=%v", tc.width, tc.available, tc.progress, got, tc.want)
		}
	}
}

// A left-aligned taskbar must hide the lyrics rather than let them cross its middle.
func TestTaskbarLyricsPlacementSpaceAcrossCenter(t *testing.T) {
	if x, width := TaskbarPlacementSpace(1000, 280, 100, 8, []Interval{{0, 120}, {480, 1000}}, true); width != 0 || x != 0 {
		t.Fatalf("left taskbar let lyrics cross its middle: %d+%d", x, width)
	}
	if x, width := TaskbarPlacementSpace(1000, 280, 100, 8, []Interval{{0, 480}}, true); width != 280 || x+width <= 500 {
		t.Fatalf("left taskbar missed real right-side space: %d+%d", x, width)
	}
}
