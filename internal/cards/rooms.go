package cards

// rooms.go — the duelist card's room row: three authored glyphs, a door, a double door and a
// portal, standing on one line across the card and saying how far through its realm the run is.
// RoomArtKeys names the glyphs from an element and a room index, so this package learns nothing
// about a journey beyond the fact that a realm has RoomSlots rooms.

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/systems"
)

// RoomSlots is how many rooms a realm holds and so how many glyphs the row draws.
const RoomSlots = 3

// roomStems is the row, left to right: the outer room's door, the inner room's double door, the
// portal room's portal.
var roomStems = [RoomSlots]string{"room-door", "room-double-door", "room-portal"}

// roomNeutral is the coloring of a room still ahead.
const roomNeutral = "neutral"

// RoomArtKeys is the row's art keys for a player standing in room `room` (0..RoomSlots-1) of a
// realm of `element`: the rooms reached so far, the current one included, in the element, and the
// rest neutral. **An empty element draws every room neutral** rather than naming a file that does
// not exist — the same totality MarkArtKey keeps.
func RoomArtKeys(element string, room int) [RoomSlots]string {
	var keys [RoomSlots]string
	for k, stem := range roomStems {
		coloring := element
		if k > room || element == "" {
			coloring = roomNeutral
		}
		keys[k] = stem + "-" + coloring
	}
	return keys
}

// RoomX is glyph k's center across a card of style st: the width inside the text margins, shared
// in thirds. Exported so a screen drawing over one glyph — the current room swelling — finds it
// where this package put it.
func RoomX(st Style, k int) int {
	inner := st.Width - 2*st.TextLeft
	return st.TextLeft + inner*(2*k+1)/(2*RoomSlots)
}

// drawRooms draws Spec.Rooms on a style that has a row. **Each glyph is placed by its canvas, not by
// its ink**, so the three stand on the floor line the art was drawn to rather than each being
// centered on its own bounds.
func drawRooms(dst *image.RGBA, s Spec, st Style) {
	if st.RoomsSize <= 0 {
		return
	}
	for k, key := range s.Rooms {
		if key == "" {
			continue
		}
		glyph := systems.ArtMark(key, st.RoomsSize, st.RoomsSize)
		if glyph == nil {
			continue
		}
		x := RoomX(st, k) - st.RoomsSize/2
		at := image.Rect(x, st.RoomsTop, x+st.RoomsSize, st.RoomsTop+st.RoomsSize)
		blitGlyph(dst, at, glyph, 1, glyphFade(s), st)
	}
}
