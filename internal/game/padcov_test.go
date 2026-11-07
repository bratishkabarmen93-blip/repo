package game

import (
	"fmt"
	"testing"
)

// TestPadCoverage prints the map with pad numbers and road coverage at a few ranges.
func TestPadCoverage(t *testing.T) {
	for _, spec := range AllLevels() {
		l, err := ParseLevel(spec)
		if err != nil {
			t.Fatal(err)
		}
		out := ""
		for i, s := range l.Slots {
			c := s.Center()
			out += fmt.Sprintf(" #%d(%d,%d) %.1f/%.1f/%.1f", i, s.X, s.Y, l.Path.Coverage(c, 2.3), l.Path.Coverage(c, 2.8), l.Path.Coverage(c, 4))
		}
		t.Logf("%s:%s", spec.ID, out)
	}
}
