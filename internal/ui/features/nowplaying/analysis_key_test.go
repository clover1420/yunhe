package nowplaying

import "testing"

// The key of pitch classes heard as a key's profile is that key, major or
// minor; silence has none.
func TestKeyOfProfiles(t *testing.T) {
	var g, a [12]float32
	for i := range 12 {
		g[(7+i)%12] = float32(keyMajor[i])
		a[(9+i)%12] = float32(keyMinor[i])
	}
	if name, sure := keyOf(g); name != "G 大调" || sure <= 0 {
		t.Errorf("G major's profile: %q, sure %.2f", name, sure)
	}
	if name, _ := keyOf(a); name != "A 小调" {
		t.Errorf("A minor's profile: %q", name)
	}
	if name, _ := keyOf([12]float32{}); name != "" {
		t.Errorf("silence: %q", name)
	}
}
