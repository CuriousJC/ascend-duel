package ui

// rooms.go — the duelist card's room row, as the screen sees it: which glyph each room draws, and
// the current room swelling gently over the card. The row itself is drawn into the card by
// internal/cards (cards.Spec.Rooms); what is here is the journey's half and the motion, which a
// cached card face cannot carry.

import (
	"image"
	"math"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
)

// RoomKeys is the duelist card's room row for a run on `fight` in a realm of `element`.
func RoomKeys(element string, fight int) [cards.RoomSlots]string {
	return cards.RoomArtKeys(element, fight%FightsPerRealm)
}

// RealmElement is the element of the realm the run is on, or "" with no run.
func RealmElement(gs *state.GlobalState) string {
	if gs == nil || gs.Run == nil {
		return ""
	}
	return gs.Run.Element()
}

// roomPulseSwell is how far the current room grows at the top of its swell, and
// roomPulseNum/roomPulseDen the length of one swell in beats.
const (
	roomPulseSwell = 0.07
	roomPulseNum   = 3
	roomPulseDen   = 2
)

// DrawRoomPulse draws the current room's glyph over a duelist card whose top-left is `card`, for a
// run on `fight` in a realm of `element`, swelling and settling on the free clock. **It never
// shrinks below the glyph the card already carries**, so the card's own copy is always covered and
// never shows as a second outline.
func DrawRoomPulse(gs *state.GlobalState, screen *ebiten.Image, card image.Point, element string, fight int) {
	st := cards.DuelistStyle
	if st.RoomsSize <= 0 {
		return
	}
	room := fight % FightsPerRealm
	key := RoomKeys(element, fight)[room]
	img := systems.ArtMarkImage(key, st.RoomsSize, st.RoomsSize)
	if img == nil {
		return
	}
	phase := float64(gs.Count) * 2 * math.Pi / float64(Beat(roomPulseNum, roomPulseDen))
	swell := 1 + roomPulseSwell*(0.5+0.5*math.Sin(phase))

	half := float64(st.RoomsSize) / 2
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-half, -half)
	op.GeoM.Scale(swell, swell)
	op.GeoM.Translate(float64(card.X+cards.RoomX(st, room)), float64(card.Y+st.RoomsTop)+half)
	screen.DrawImage(img, op)
}
