package ui

// orbs.go — an orb drawn on the screen rather than into a card: the swell a fighter card's orb
// takes when its figure grows, and a loose orb — a payment on the reward screen, on its row and on
// its way to the purse. The card's own orbs are drawn by internal/cards (cards.Style.StatOrbs);
// what is here is the motion, which a cached card face cannot carry.

import (
	"image"
	"image/color"
	"math"
	"strconv"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The figure inside a loose orb, matching the card's: its size against the orb's, and the widest
// share of the orb it may take.
const (
	orbFigureShare = 0.64
	orbFigureFit   = 0.80
)

// orbTextureStep is the grain the orb's texture is reduced to: a loose orb is drawn from the next
// size up on this step and scaled down, so a swelling or flying orb does not ask ArtMarkImage for a
// new size every frame.
const orbTextureStep = 32

// DrawOrb draws the orb `art` centered on (cx, cy), `size` across, with `figure` written across its
// middle in white, in the interface's own lettering set from `font`.
func DrawOrb(font *text.GoTextFaceSource, screen *ebiten.Image, art, figure string, cx, cy, size float64) {
	if size <= 0 {
		return
	}
	n := int(math.Ceil(size/orbTextureStep)) * orbTextureStep
	if img := systems.ArtMarkImage(art, n, n); img != nil {
		k := size / float64(img.Bounds().Dx())
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM.Scale(k, k)
		op.GeoM.Translate(cx-size/2, cy-size/2)
		screen.DrawImage(img, op)
	}
	if figure == "" {
		return
	}

	pt := size * orbFigureShare
	if w := systems.MeasureFigure(figure, systems.UIHeightOf(pt)); w > size*orbFigureFit {
		pt *= size * orbFigureFit / w
	}
	op := &text.DrawOptions{}
	op.GeoM.Translate(cx, cy)
	op.PrimaryAlign = text.AlignCenter
	op.SecondaryAlign = text.AlignCenter
	op.ColorScale.ScaleWithColor(color.White)
	systems.DrawUI(screen, figure, &text.GoTextFace{Source: font, Size: pt}, op)
}

// OrbSwellPeak is how far an orb grows when its figure does: up fast, back slowly.
const OrbSwellPeak = 1.2

func orbSwellUp() int   { return Beat(1, 8) }
func orbSwellDown() int { return Beat(1, 2) }

// OrbSwell is an orb's scale `age` ticks after its figure grew: 1 before and after, OrbSwellPeak at
// the top.
func OrbSwell(age int) float64 {
	up, down := orbSwellUp(), orbSwellDown()
	switch {
	case age < 0 || age >= up+down:
		return 1
	case age < up:
		return 1 + (OrbSwellPeak-1)*EaseOut(float64(age)/float64(up))
	default:
		return 1 + (OrbSwellPeak-1)*(1-EaseOut(float64(age-up)/float64(down)))
	}
}

// OrbSwells watches one fighter card's figures and swells an orb whenever its figure grows — a
// surge banked, a grant to DMG landing, a payment reaching the purse.
//
// **It watches the figures the card is drawn with**, never the events that moved them, so whatever
// makes a figure grow swells its orb with nothing to wire, and the swell lands on the frame the card
// shows the new figure. A figure that falls does not swell. The first look at a card only learns its
// figures, and so does the first look after a frame it was not drawn on, so entering a screen
// swells nothing.
type OrbSwells struct {
	seen  bool
	drawn int

	last  [cards.MaxStatLines]int
	start [cards.MaxStatLines]int
	on    [cards.MaxStatLines]bool
}

// Draw notes the figures `spec` carries and draws, over a card in style `st` whose top-left is
// `card`, every orb still swelling. Call it once a frame, straight after the card itself.
func (o *OrbSwells) Draw(gs *state.GlobalState, screen *ebiten.Image, card image.Point, spec cards.Spec, st cards.Style) {
	if gs.Count-o.drawn > 1 {
		*o = OrbSwells{}
	}
	o.drawn = gs.Count
	for i, line := range spec.Stats {
		n, err := strconv.Atoi(line.Value)
		if err != nil {
			continue
		}
		if o.seen && n > o.last[i] {
			o.start[i], o.on[i] = gs.Count, true
		}
		o.last[i] = n
	}
	o.seen = true

	for i, line := range spec.Stats {
		if !o.on[i] {
			continue
		}
		age := gs.Count - o.start[i]
		if age >= orbSwellUp()+orbSwellDown() {
			o.on[i] = false
			continue
		}
		box, ok := cards.StatOrbBox(st, i)
		if !ok {
			continue
		}
		cx := float64(card.X) + float64(box.Min.X+box.Max.X)/2
		cy := float64(card.Y) + float64(box.Min.Y+box.Max.Y)/2
		DrawOrb(gs.Fonts["kubasta"], screen, st.StatOrbs[i].Art, line.Value, cx, cy, float64(box.Dx())*OrbSwell(age))
	}
}

// StatOrbCenter is the middle of stat i's orb on a card in style `st` whose rectangle is `card`, and
// whether the style draws one there.
func StatOrbCenter(card image.Rectangle, st cards.Style, i int) (image.Point, bool) {
	box, ok := cards.StatOrbBox(st, i)
	if !ok {
		return image.Point{}, false
	}
	return card.Min.Add(image.Pt((box.Min.X+box.Max.X)/2, (box.Min.Y+box.Max.Y)/2)), true
}
