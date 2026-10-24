package screens

// **A found room, opened up: its picture large, with what the tooltip would have said under it.**
//
// Resting on a found entry in the realm book draws this in place of a tooltip, the way the bestiary
// opens a creature. A thumbnail four across is enough to know a room by and too small to look at,
// and the fight only ever shows the room behind the cards — so this is the one place in the game the
// whole picture is seen.
//
// **One room is kept decoded at a time.** A room is a full-size JPEG, so the large reduction is made
// the first frame the cursor rests on it and dropped when another room replaces it; a cache keyed by
// every room rested on would hold a page of large pictures nobody is looking at.

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/ui"
)

// The panel's shape. **The room is drawn at 800x450**, the room's own 16:9 at the widest that leaves
// a panel inside half the screen, so it can stand on the half away from the entry it is about.
const (
	roomPeekW = 800
	roomPeekH = 450
)

// roomPeek is the large room kept for the panel, and the asset key it was reduced from.
var roomPeek struct {
	key string
	img *ebiten.Image
}

// roomPeekImage is a room reduced to the panel's size, reusing the one already made when it is the
// same room.
func roomPeekImage(gs *state.GlobalState, key string) *ebiten.Image {
	if roomPeek.key != key {
		roomPeek.key, roomPeek.img = key, reduceRoom(gs.ImageData[key], roomPeekW, roomPeekH)
	}
	return roomPeek.img
}

// roomPeekLines are the words under the room — the lines its tooltip carries, the count last —
// wrapped to the picture's width.
func roomPeekLines(e *compendiumEntry, face *text.GoTextFace) []models.TipLine {
	var out []models.TipLine
	for _, line := range append(append([]string{}, e.lines...), e.extra...) {
		if line == "" {
			continue
		}
		out = append(out, systems.WrapLine(ui.TipLine(line), face, roomPeekW)...)
	}
	return out
}

// drawRoomPeek draws the open room, if the cursor is on one.
func (s *CompendiumScene) drawRoomPeek(gs *state.GlobalState, screen *ebiten.Image) {
	e := s.roomPeek
	if e == nil {
		return
	}
	kubasta := gs.Fonts["kubasta"]
	titleFace := &text.GoTextFace{Source: kubasta, Size: peekTitleSize}
	lineFace := &text.GoTextFace{Source: kubasta, Size: peekLineSize}

	wrapped := roomPeekLines(e, lineFace)

	w := peekPad*2 + roomPeekW
	h := peekPad*2 + int(peekTitleSize) + peekGap/2 + roomPeekH + peekGap/2 +
		len(wrapped)*(int(peekLineSize)+peekLineGap)

	// **On the other half of the screen from the entry it is about**, as the creature's panel is.
	cx := gs.ScreenWidth * 3 / 4
	if s.peekSeat.Min.X+s.peekSeat.Dx()/2 > gs.ScreenWidth/2 {
		cx = gs.ScreenWidth / 4
	}
	x := min(max(cx-w/2, 0), gs.ScreenWidth-w)
	y := min(max(gs.ScreenHeight/2-h/2, 0), gs.ScreenHeight-h)

	vector.FillRect(screen, float32(x), float32(y), float32(w), float32(h), systems.PanelSurface, false)
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h), 2, systems.PanelEdgeInk, false)

	top := y + peekPad
	systems.DrawLine(screen, ui.TipLine(e.title), titleFace, x+peekPad, top, systems.PanelSpeech)
	top += int(peekTitleSize) + peekGap/2

	// The room, or the default room darkened while it has no picture of its own — the thumbnail's rule.
	key := e.backdrop
	painted := len(gs.ImageData[key]) > 0
	if !painted {
		key = data.DefaultBackgroundArt
	}
	if img := roomPeekImage(gs, key); img != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(x+peekPad), float64(top))
		if !painted {
			op.ColorScale.Scale(catalogUndrawnScale, catalogUndrawnScale, catalogUndrawnScale, 1)
		}
		screen.DrawImage(img, op)
	}
	vector.StrokeRect(screen, float32(x+peekPad), float32(top), roomPeekW, roomPeekH, 2, systems.PanelEdgeInk, false)
	top += roomPeekH + peekGap/2

	for _, line := range wrapped {
		systems.DrawLine(screen, line, lineFace, x+peekPad, top, systems.PanelInk)
		top += int(peekLineSize) + peekLineGap
	}
}
