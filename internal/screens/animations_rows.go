package screens

// animations_rows.go — the gallery's entries for two pieces of interface art: the duelist card's
// room row and the settings screen's checkbox. Both draw with the game's own functions —
// cards.Spec.Rooms and ui.DrawRoomPulse, systems.DrawCheckbox — so what is shown is what is shipped.

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

// galleryElements is the order the room preview walks the realms in, one per loop.
var galleryElements = []string{"fire", "ice", "lightning", "earth", "arcane"}

// drawRoomRow is the room entry: the duelist card as the game draws it, walking outer, inner and
// portal room in one realm per loop with the current room swelling, and beside it the row in every
// element at once, standing in the inner room.
func drawRoomRow(s *AnimationsScene, gs *state.GlobalState, screen *ebiten.Image, at image.Rectangle, p float64) {
	room := min(int(p*3), 2)
	element := galleryElements[s.loops%len(galleryElements)]

	spec := ui.DuelistSpec(gs, nil, "preview", 12, 7, 42, 60, 3, room)
	spec.Rooms = ui.RoomKeys(element, room)
	ui.BlitCard(gs, screen, at.Min, spec, cards.DuelistStyle)
	ui.DrawRoomPulse(gs, screen, at.Min, element, room)

	// Every element at once: the same card, cropped to the band the row stands in.
	st := cards.DuelistStyle
	left := at.Max.X + 60
	band := image.Rect(0, st.RoomsTop-4, st.Width, st.RoomsTop+st.RoomsSize+4)
	for i, el := range galleryElements {
		top := at.Min.Y + i*(band.Dy()+8)
		spec.Rooms = ui.RoomKeys(el, 1)
		if img := ui.CardImage(gs, spec, cards.DuelistStyle); img != nil {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(left), float64(top))
			screen.DrawImage(img.SubImage(band).(*ebiten.Image), op)
		}
		ui.DrawRoomPulse(gs, screen, image.Pt(left, top-band.Min.Y), el, 1)
		animText(gs, screen, el, left+st.Width+14, top+band.Dy()/2, animNoteSize, animStageInk)
	}
}

// galleryBoxes is the checkbox entry's boxes, one per size per state row, kept between frames so
// each carries its own pop.
var galleryBoxes [4][3]*models.Checkbox

// drawCheckboxes is the checkbox entry: the widget the settings screen uses, one column per size and
// one row per state, the top row ticking on and off so the mark's arrival can be watched.
func drawCheckboxes(s *AnimationsScene, gs *state.GlobalState, screen *ebiten.Image, at image.Rectangle, p float64) {
	ticking := p >= 0.15 && p < 0.65
	rows := []string{"ticking", "on", "disabled", "disabled, on"}
	sizes := []int{68, 44, 32}
	for r, label := range rows {
		cy := at.Min.Y + 40 + r*90
		animText(gs, screen, label, at.Min.X, cy, animNoteSize, animStageInk)
		cx := at.Min.X + 190
		for c, size := range sizes {
			box := galleryBoxes[r][c]
			if box == nil {
				box = &models.Checkbox{Size: size}
				galleryBoxes[r][c] = box
			}
			box.ScreenX, box.ScreenY = cx, cy
			box.Checked = r == 1 || r == 3 || (r == 0 && ticking)
			box.Disabled = r >= 2
			systems.UpdateCheckbox(gs, box)
			systems.DrawCheckbox(screen, box)
			cx += size*2 + 30
		}
		if r == 0 {
			animText(gs, screen, "MUSIC", cx, cy, animNameSize, ui.GroundInk)
		}
	}
}
