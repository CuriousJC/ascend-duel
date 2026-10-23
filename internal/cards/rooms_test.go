package cards

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/systems"
)

// Every key the room row can name is a file: each room reached in each element, and each room ahead.
// A key naming nothing draws nothing, which on this row reads as a realm with a room missing.
func TestEveryRoomGlyphIsAFile(t *testing.T) {
	elements := []string{"fire", "ice", "lightning", "earth", "arcane", ""}
	for _, el := range elements {
		for room := 0; room < RoomSlots; room++ {
			for _, key := range RoomArtKeys(el, room) {
				if systems.ArtMark(key, DuelistStyle.RoomsSize, DuelistStyle.RoomsSize) == nil {
					t.Errorf("room %d in %q names %q, which is not in the assets", room, el, key)
				}
			}
		}
	}
}

// The row colors every room up to and including the current one, and leaves the rest neutral.
func TestTheRoomRowColorsWhatIsReached(t *testing.T) {
	got := RoomArtKeys("ice", 1)
	want := [RoomSlots]string{"room-door-ice", "room-double-door-ice", "room-portal-neutral"}
	if got != want {
		t.Errorf("standing in the inner room of an ice realm draws %v, want %v", got, want)
	}
}

// The glyphs stand clear of the health bar under them.
func TestTheRoomRowClearsTheHealthBar(t *testing.T) {
	st := DuelistStyle
	if bottom := st.RoomsTop + st.RoomsSize; bottom > st.HealthBarTop {
		t.Errorf("the room row reaches y=%d, into the health bar at y=%d", bottom, st.HealthBarTop)
	}
}
