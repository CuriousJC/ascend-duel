package screens

// **A found creature, opened up: its card at full size beside the deck it is dealt.**
//
// Resting on a found entry in the bestiary draws this in place of a tooltip. A creature's whole
// personality is which hits come round how often, and the fight only ever shows the played half of
// that — so the page is the one place in the game a player can read the deck itself.
//
// **It is the game's own drawing, not a picture of it.** The card is `ui.EnemySpec` and every deck
// card is `ui.CardSpec`, held by the creature as it is built for a fight, so the faces here are the
// faces on the table. The figures are its record's bases — `entities.NewEnemyFrom` at fight 0 —
// because a creature met later in the journey is grown to where it is met, and the page is about
// the creature rather than about one meeting with it.
//
// **It is hidden until the colour is found**, like every catalog's tooltip: an unfound entry keeps
// the placeholder and says nothing.

import (
	"image/color"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/decks"
	"github.com/curiousjc/ascend-duel/internal/entities"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/ui"
)

// The panel's shape. **The creature's card is half as big again as it is in a fight** and the deck
// cards four fifths of a hand card's, so the creature reads as the subject and its deck as what it
// holds.
const (
	peekPad       = 28
	peekGap       = 32
	peekCardNum   = 3
	peekCardDen   = 2
	peekDeckNum   = 4
	peekDeckDen   = 5
	peekDeckCols  = 3
	peekDeckGapX  = 16
	peekDeckGapY  = 10
	peekCaptionH  = 32
	peekTitleSize = 36
	peekLineSize  = systems.TextLarge
	peekLineGap   = 8
)

// creaturePeek is one bestiary entry's creature: the record, the colour it was dealt as, and what the
// player has had of it in that colour.
type creaturePeek struct {
	record  data.MotifRecord
	element string
	motif   string
	beaten  int
	fell    int
}

// peekDeckRow is one concept of the deck and how many copies of it the creature holds.
type peekDeckRow struct {
	card   combat.Card
	copies int
}

// deckRows is the creature's deck in this colour, one row per concept in the record's own order.
func (p *creaturePeek) deckRows() []peekDeckRow {
	var out []peekDeckRow
	for _, c := range decks.EnemyCards(p.record.Record, p.element) {
		if n := len(out); n > 0 && out[n-1].card.Concept == c.Concept {
			out[n-1].copies++
			continue
		}
		out = append(out, peekDeckRow{card: c, copies: 1})
	}
	return out
}

// lines are the words under the card: what the creature is, its figures, and what the player has
// had of it.
func (p *creaturePeek) lines(c *entities.Combatant) []string {
	out := []string{
		p.motif + ", " + strings.ToLower(p.record.Tier),
		"HP " + strconv.Itoa(c.MaxLife) + "   DMG " + strconv.Itoa(c.DMG) + "   AP " + strconv.Itoa(c.Actions),
		"Beaten " + times(p.beaten),
	}
	if p.beaten == 0 {
		out[2] = "Never beaten"
	}
	if p.fell > 0 {
		out = append(out, "Fell to it "+times(p.fell))
	}
	return out
}

// drawPeek draws the open creature, if the cursor is on one.
func (s *CompendiumScene) drawPeek(gs *state.GlobalState, screen *ebiten.Image) {
	p := s.peek
	if p == nil {
		return
	}
	creature := entities.NewEnemyFrom(p.record, p.element, 0, gs.Journey)
	rows := p.deckRows()
	lines := p.lines(creature)

	big := cards.EnemyStyle.Scaled(peekCardNum, peekCardDen)
	small := cards.Hand.Scaled(peekDeckNum, peekDeckDen)
	cols := min(peekDeckCols, max(len(rows), 1))
	deckRowsDown := (len(rows) + cols - 1) / cols
	cellH := small.Height + peekCaptionH

	titleH := int(peekTitleSize) + peekLineGap + int(peekLineSize) + peekGap
	leftH := big.Height + peekGap/2 + len(lines)*(int(peekLineSize)+peekLineGap)
	deckW := cols*small.Width + (cols-1)*peekDeckGapX
	deckH := deckRowsDown*cellH + max(deckRowsDown-1, 0)*peekDeckGapY
	w := peekPad*2 + big.Width + peekGap + deckW
	h := peekPad*2 + titleH + max(leftH, deckH)

	// **On the other half of the screen from the card it is about**, so the panel never covers the
	// entry the cursor is resting on.
	cx := gs.ScreenWidth * 3 / 4
	if s.peekSeat.Min.X+s.peekSeat.Dx()/2 > gs.ScreenWidth/2 {
		cx = gs.ScreenWidth / 4
	}
	x := min(max(cx-w/2, 0), gs.ScreenWidth-w)
	y := min(max(gs.ScreenHeight/2-h/2, 0), gs.ScreenHeight-h)

	vector.FillRect(screen, float32(x), float32(y), float32(w), float32(h), systems.PanelSurface, false)
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h), 2, systems.PanelEdgeInk, false)

	kubasta := gs.Fonts["kubasta"]
	titleFace := &text.GoTextFace{Source: kubasta, Size: peekTitleSize}
	lineFace := &text.GoTextFace{Source: kubasta, Size: peekLineSize}

	// The name, then the colour it is dealt as in that colour's own ink.
	top := y + peekPad
	s.peekText(screen, p.record.FullName(), titleFace, x+peekPad, top, systems.PanelSpeech)
	top += int(peekTitleSize) + peekLineGap
	s.peekText(screen, elementTitle(p.element), lineFace, x+peekPad, top, elementWordInk(p.element))
	top += int(peekLineSize) + peekGap

	// The creature's own card, at fight size and a half.
	if img := ui.CardImage(gs, ui.EnemySpec(gs, creature, creature.Name, creature.MaxLife), big); img != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(x+peekPad), float64(top))
		screen.DrawImage(img, op)
	}
	ly := top + big.Height + peekGap/2
	for _, line := range lines {
		s.peekText(screen, line, lineFace, x+peekPad, ly, systems.PanelInk)
		ly += int(peekLineSize) + peekLineGap
	}

	// Its deck, a card per concept with the copies under it.
	left := x + peekPad + big.Width + peekGap
	for i, r := range rows {
		cx := left + (i%cols)*(small.Width+peekDeckGapX)
		cy := top + (i/cols)*(cellH+peekDeckGapY)
		spec := ui.CardSpec(r.card, ui.HeldBy(creature.Duelist, r.card), true, false)
		if img := ui.CardImage(gs, spec, small); img != nil {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(cx), float64(cy))
			screen.DrawImage(img, op)
		}
		cap := &text.DrawOptions{}
		cap.GeoM.Translate(float64(cx+small.Width/2), float64(cy+small.Height+peekCaptionH/2))
		cap.PrimaryAlign = text.AlignCenter
		cap.SecondaryAlign = text.AlignCenter
		cap.ColorScale.ScaleWithColor(systems.PanelInk)
		systems.DrawUI(screen, "x"+strconv.Itoa(r.copies), lineFace, cap)
	}
}

// peekText writes one line of the panel's prose, its top-left at (x, y).
func (s *CompendiumScene) peekText(screen *ebiten.Image, str string, face *text.GoTextFace, x, y int, ink color.RGBA) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(ink)
	systems.DrawText(screen, str, face, op)
}

// elementTitle is an element's name with a capital: "Fire".
func elementTitle(element string) string {
	if element == "" {
		return ""
	}
	return strings.ToUpper(element[:1]) + element[1:]
}

// elementWordInk is the colour an element's name is written in — the element vocabulary's, through
// elementInk — or the panel's ink for a name the rules cannot read.
func elementWordInk(element string) color.RGBA {
	el, ok := combat.ParseElement(element)
	if !ok {
		return systems.PanelInk
	}
	return elementInk(ui.ArtFor(el))
}
