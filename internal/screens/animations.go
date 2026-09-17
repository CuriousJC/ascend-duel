package screens

// The animation gallery: every gesture the game can make, by name, on a loop.
//
// **It exists to give the gestures names.** There are a dozen distinct movements on the combat
// screen and they were previously reachable only by producing the situation each one belongs to —
// a break needs a shield eating an attack, a toast needs a relic firing into a sum, a cascade needs
// two flip rings and a hand with the right colors in it. So "make the toast louder" was a sentence
// with no shared referent, and the first minute of every conversation about one went on
// establishing which movement was meant. This page is the vocabulary.
//
// **It is a view and it is a debug view.** Nothing here touches a run, a duel or the profile;
// every entry draws with the game's own functions rather than reproducing them, which is the rule
// that matters — `docs/sheets/` is a picture of the catalogs and is worthless the moment it draws
// something the game does not. A gesture that had to be re-implemented here to be shown is a
// gesture this page would eventually lie about, so the shared ones were split out of their callers
// instead; see `outboundGeoM` and `drawDealtCard`.
//
// # Why a screen and not a tool
//
// The sheets under `docs/sheets/` work because `internal/cards` renders without a graphics
// context. **Motion needs a window and a clock**, so no tool can show one — a still frame of a
// dissolve is a picture of a card with holes in it. The scripted demo's PNGs have the same problem.
//
// # Why chrome and not the title menu
//
// It is reached from a square in the frame's bottom strip, drawn only while
// `state.DebugAnimations` is on — the placement grid's arrangement, and the same argument:
// instrumentation that must be deliberately turned on rather than found. A row on the title menu
// would be a debug page a player can reach. **Off by default**, set once in `main.go`, no runtime
// toggle, because the input vocabulary has no keyboard.
//
// # What it is not
//
// It is not a station of a run, so it has no `session.Phase` and no entry in `flow.go` — same
// shape as Settings, Achievements and Credits. It records `gs.ReturnScreen`, so Back works from
// wherever it was opened.

import (
	"image"
	"image/color"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// The page's shape.
const (
	animTitle     = "ANIMATIONS"
	animTitleSize = 34

	// animNameSize is a row in the list, animNoteSize the symbol under the stage and
	// animWhatSize the sentence beside it.
	animNameSize = 20
	animNoteSize = 16
	animWhatSize = 17

	animRowHeight = 34
)

// animHoldTicks is the still beat between one loop and the next. **A gesture read back to back with
// no gap reads as one longer gesture**, which is exactly the confusion the page exists to clear up.
//
// On the game's own clock, like every other duration here, so the gallery slows down and speeds up
// with the speed slider exactly as the thing it is showing does.
func animHoldTicks() int { return ui.Beat(2, 3) }

// animListLeft and the rest are percentages, per the placement rule: a group is anchored by a
// percentage and laid out in pixels from there.
const (
	animListLeftPct  = 5
	animListTopPct   = 20
	animStageLeftPct = 34
	animStageTopPct  = 24
	animNotesTopPct  = 76
)

// animStageInk is the caption under the stage: the symbol that implements the gesture, which is
// what makes the name usable in a sentence about the code as well as about the screen.
var animStageInk = color.RGBA{R: 120, G: 108, B: 90, A: 255}

// gesture is one named movement.
//
// **The name is the point of the struct.** `where` names the symbol so a conversation can move from
// the picture to the code in one step, and `what` is the one line that says when the game makes
// this movement — which is the half a picture cannot show.
type animGesture struct {
	name  string
	where string
	what  string

	// ticks is how long one run of it takes, read off the game's own clock so the gallery slows
	// down and speeds up with the speed slider exactly as the real thing does.
	ticks func() int

	// draw puts the gesture on the stage. `at` is the card-sized rectangle the gesture is centered
	// on and `p` is 0..1 through it. **Raw progress, not eased** — the easing belongs to the
	// gesture and several of them are deliberately not eased at all.
	draw func(s *AnimationsScene, gs *state.GlobalState, screen *ebiten.Image, at image.Rectangle, p float64)
}

// AnimationsScene is the gallery.
type AnimationsScene struct {
	back *models.Button
	rows []*models.Button

	// at is the selected gesture and t its clock: one run, then a still hold, then round again.
	at int
	t  ui.Travel

	// m is the morph the three morph entries drive. **Rebuilt when the loop restarts** rather than
	// driven by a progress figure, because a morph owns its own clock — asking it to be a pure
	// function of p would be a second implementation of the thing being reviewed.
	m ui.Morph
}

// animGestures is the vocabulary, in the order a card meets them: arriving, changing, acting,
// leaving.
//
// **Every entry calls the game's own drawing.** An entry that had to reproduce one would be the
// stale-sheet failure — a picture of a gesture the game does not make — so a shared gesture gets
// split out of its caller rather than copied here.
//
// **`shake` and `toast` are one clock and three marks** *(2026-09-15)*, which is a question the page
// was built to answer and then answered badly by listing them as peers. `shakeOffset` is the
// sideways rattle; the toast is that rattle **plus a tilt and a lit border**, all three off the one
// `travel` — see `relicToast`, which exists so a mark cannot reach one caller and not the other.
// What differs is the wearer: a **card** paying into the sum rattles and nothing else
// (`drawPlayedCards`), a **relic** firing rattles, rocks and lights. They stay two entries because
// the difference is exactly those two extra marks and the pair wants to be looked at side by side;
// what they must not do is read as two movements.
var animGestures = []animGesture{
	{
		name:  "deal",
		where: "drawDealtCard",
		what:  "A card out of the draw pile: it grows to hand size and turns face up on the way.",
		ticks: flightTicks,
		draw: func(s *AnimationsScene, gs *state.GlobalState, screen *ebiten.Image, at image.Rectangle, p float64) {
			from := image.Pt(at.Min.X-360, at.Min.Y+120)
			drawDealtCard(gs, screen, from, at.Min, p, animFace(gs, animCardA), animBack(gs))
		},
	},
	{
		name:  "sort slide",
		where: "cardSlide, cardslide.go",
		what:  "A card shuffling from one seat in the row to another. Flat, full size, no turn.",
		ticks: ui.SlideTicks,
		draw: func(s *AnimationsScene, gs *state.GlobalState, screen *ebiten.Image, at image.Rectangle, p float64) {
			from := image.Pt(at.Min.X-260, at.Min.Y)
			ui.BlitCard(gs, screen, ui.LerpPoint(from, at.Min, ui.EaseOut(p)), animFace(gs, animCardA), cards.Hand)
		},
	},
	{
		name:  "lift",
		where: "selectedLift, tableFireLift",
		what:  "Vertical, and it means picked: selected in the hand, or acting on the table.",
		ticks: ui.SlideTicks,
		draw: func(s *AnimationsScene, gs *state.GlobalState, screen *ebiten.Image, at image.Rectangle, p float64) {
			up := at
			up.Min.Y -= int(float64(tableFireLift) * ui.EaseOut(animThereAndBack(p)))
			ui.BlitCard(gs, screen, up.Min, animFace(gs, animCardA), cards.Hand)
		},
	},
	{
		name:  "shake",
		where: "shakeOffset, combat_relics.go",
		what:  "Sideways, and it means paying now. On a card it is the whole gesture: no light.",
		ticks: relicShakeTicks,
		draw: func(s *AnimationsScene, gs *state.GlobalState, screen *ebiten.Image, at image.Rectangle, p float64) {
			at.Min.X += shakeOffset(animTravelAt(relicShakeTicks(), p))
			ui.BlitCard(gs, screen, at.Min, animFace(gs, animCardA), cards.Hand)
		},
	},
	{
		name:  "toast",
		where: "relicToast, combat_relics.go",
		what:  "A relic firing: it rattles, rocks left then right, and its border lights.",
		ticks: relicShakeTicks,
		draw: func(s *AnimationsScene, gs *state.GlobalState, screen *ebiten.Image, at image.Rectangle, p float64) {
			r, ok := animRelic(gs)
			if !ok {
				return
			}

			// **Drawn exactly as the relic pane draws it**, including the branch: a relic at rest is
			// blitted and a toasting one is flown. That is what makes the light visible at all here
			// — the page shows the same card lit during the beat and unlit through the hold after
			// it, where a permanently-lit card had nothing to be brighter *than*.
			toast := sumToast(animTravelAt(relicShakeTicks(), p))
			if !toast.lit {
				ui.DrawRelicCard(gs, screen, at.Min, r, "", true, false)
				return
			}
			ui.DrawFlyingCard(gs, screen, ui.RelicSpec(gs, r, "", true, true),
				cards.RelicStyle, toast.geoAt(at.Min))
		},
	},
	{
		name:  "cascade shake",
		where: "dealShakeOffset, combat_deal.go",
		what:  "The deal's own rattle — tighter and wider, because a whole row goes at once.",
		ticks: dealRingTicks,
		draw: func(s *AnimationsScene, gs *state.GlobalState, screen *ebiten.Image, at image.Rectangle, p float64) {
			at.Min.X += dealShakeOffset(animTravelAt(dealRingTicks(), p))
			ui.BlitCard(gs, screen, at.Min, animFace(gs, animCardA), cards.Hand)
		},
	},
	{
		name:  "morph into",
		where: "morphInto, cardmorph.go",
		what:  "A card becoming a different card. The face comes apart on a lattice, never at random.",
		ticks: animMorphTicks,
		draw: func(s *AnimationsScene, gs *state.GlobalState, screen *ebiten.Image, at image.Rectangle, p float64) {
			ui.DrawMorph(gs, screen, at.Min, s.m)
		},
	},
	{
		name:  "morph in",
		where: "morphIn, cardmorph.go",
		what:  "A card arriving out of nothing — a copy the run did not own a moment ago.",
		ticks: animMorphTicks,
		draw: func(s *AnimationsScene, gs *state.GlobalState, screen *ebiten.Image, at image.Rectangle, p float64) {
			ui.DrawMorph(gs, screen, at.Min, s.m)
		},
	},
	{
		name:  "morph away",
		where: "morphAway, cardmorph.go",
		what:  "A card eaten, with nothing behind it.",
		ticks: animMorphTicks,
		draw: func(s *AnimationsScene, gs *state.GlobalState, screen *ebiten.Image, at image.Rectangle, p float64) {
			ui.DrawMorph(gs, screen, at.Min, s.m)
		},
	},
	{
		name:  "shatter",
		where: "drawSpreadingCracks, MarkShattered",
		what:  "A shield eating an attack whole. The cracks travel outward, then the mark settles.",
		ticks: shatterSpreadTicks,
		draw: func(s *AnimationsScene, gs *state.GlobalState, screen *ebiten.Image, at image.Rectangle, p float64) {
			ui.BlitCard(gs, screen, at.Min, animFace(gs, animCardB), cards.Hand)
			drawSpreadingCracks(screen, at, animCardB, p)
		},
	},
	{
		name:  "discard",
		where: "outboundGeoM, combat_flight.go",
		what:  "A spent card thrown off the left of the table: it lifts, turns and shrinks as it goes.",
		ticks: flightTicks,
		draw: func(s *AnimationsScene, gs *state.GlobalState, screen *ebiten.Image, at image.Rectangle, p float64) {
			ui.DrawFlyingCard(gs, screen, animFace(gs, animCardA), cards.Hand,
				outboundGeoM(at.Min, ui.EaseIn(p)))
		},
	},
}

// The two cards the page is drawn with. **Two rather than one**, because a morph needs a card to
// change *into* and a single card morphing into itself would show the lattice and nothing else.
var (
	animCardA = combat.Card{Concept: combat.Bash, Element: combat.Fire}
	animCardB = combat.Card{Concept: combat.Slice, Element: combat.Ice}
)

// animMorphTicks is the whole of a morph including its wait, which is what the loop has to outlast.
func animMorphTicks() int { return ui.MorphWaitTicks() + ui.MorphTicks() }

// animFace is one of the page's cards as a finished face. **heldByRun, so a page opened with no run
// still draws** — it prices the card at its own cost with nothing worn, which is what this page
// wants anyway: a gesture, not a build.
func animFace(gs *state.GlobalState, c combat.Card) cards.Spec {
	return ui.CardSpec(c, ui.HeldByRun(gs, c), true, false)
}

// animBack is the card back the deal turns over from.
func animBack(gs *state.GlobalState) cards.Spec {
	return cards.Spec{FaceDown: true}
}

// animRelic is a relic to toast. **Whichever the catalog answers first by sorted key**, so the page
// needs no record named in it — a relic key written down here is a key that stops existing.
func animRelic(gs *state.GlobalState) (data.RelicData, bool) {
	for _, key := range combat.RelicKeys() {
		if r, ok := gs.Relics[key]; ok {
			return r, true
		}
	}
	return data.RelicData{}, false
}

// animTravelAt is a travel wound forward to a fraction of its own length, which is how the two
// shakes are shown: both are functions of a `travel` rather than of a figure, and the page drives
// the real one rather than reproducing the decay.
func animTravelAt(ticks int, p float64) ui.Travel {
	t := ui.NewTravel(0, ticks)
	t.Age = int(p * float64(ticks))
	return t
}

// animThereAndBack is a 0..1 that goes up and comes back, for the gestures that hold and release
// rather than travelling somewhere.
func animThereAndBack(p float64) float64 {
	if p < 0.5 {
		return p * 2
	}
	return (1 - p) * 2
}

// Init builds the list on first entry and positions everything every time.
func (s *AnimationsScene) Init(gs *state.GlobalState) {
	if s.back == nil {
		s.back = models.NewButton(220, 60, "BACK", func() { s.leave(gs) })
		for i := range animGestures {
			i := i
			b := models.NewButton(280, animRowHeight-6, animGestures[i].name, func() { s.pick(gs, i) })
			b.TextSize = animNameSize
			s.rows = append(s.rows, b)
		}
	}

	left := gs.PctX(animListLeftPct)
	top := gs.PctY(animListTopPct)
	for i, b := range s.rows {
		b.ScreenX = left + 140
		b.ScreenY = top + i*animRowHeight + animRowHeight/2
	}

	s.back.ScreenX, s.back.ScreenY = left+140, gs.ScreenHeight-70
	s.pick(gs, s.at)
}

// pick selects a gesture and starts it from the top. **Restarting rather than letting it run on**
// is what makes the list a way of comparing two gestures: both are watched from their first frame.
func (s *AnimationsScene) pick(gs *state.GlobalState, i int) {
	s.at = i
	s.t = ui.NewTravel(0, animGestures[i].ticks()+animHoldTicks())
	s.raiseMorph(gs)
}

// raiseMorph builds the morph the three morph entries draw, if this gesture is one of them.
func (s *AnimationsScene) raiseMorph(gs *state.GlobalState) {
	a, b := animFace(gs, animCardA), animFace(gs, animCardB)
	switch animGestures[s.at].name {
	case "morph into":
		s.m = ui.MorphInto(a, b, cards.Hand)
	case "morph in":
		s.m = ui.MorphIn(b, cards.Hand)
	case "morph away":
		s.m = ui.MorphAway(a, cards.Hand)
	default:
		s.m = ui.Morph{}
	}
}

func (s *AnimationsScene) Update(gs *state.GlobalState) error {
	for i, b := range s.rows {
		b.Latched = i == s.at
		systems.UpdateButton(gs, b)
	}
	systems.UpdateButton(gs, s.back)

	s.t.Tick()
	s.m.Tick()
	if s.t.Done() {
		s.pick(gs, s.at)
	}
	return nil
}

func (s *AnimationsScene) Draw(gs *state.GlobalState, screen *ebiten.Image) {
	ui.FillGround(screen)

	animText(gs, screen, animTitle, gs.PctX(animListLeftPct), gs.PctY(11),
		animTitleSize, ui.GroundInk)

	for _, b := range s.rows {
		systems.DrawButton(gs, screen, b)
	}
	systems.DrawButton(gs, screen, s.back)

	g := animGestures[s.at]
	stage := image.Rect(
		gs.PctX(animStageLeftPct), gs.PctY(animStageTopPct),
		gs.PctX(animStageLeftPct)+cardWidth, gs.PctY(animStageTopPct)+cardHeight)

	// The hold at the end of the loop is a still frame of the finished gesture, so `p` is clamped
	// rather than allowed to run past 1.
	p := float64(s.t.Age) / float64(g.ticks())
	if p > 1 {
		p = 1
	}
	g.draw(s, gs, screen, stage, p)

	animText(gs, screen, g.where, gs.PctX(animStageLeftPct), gs.PctY(animNotesTopPct),
		animNoteSize, animStageInk)
	animText(gs, screen, g.what, gs.PctX(animStageLeftPct), gs.PctY(animNotesTopPct)+26,
		animWhatSize, ui.GroundInk)
}

// animText is one left-aligned line, which is every piece of type on this page.
func animText(gs *state.GlobalState, screen *ebiten.Image, line string, x, y int, size float64, ink color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.SecondaryAlign = text.AlignCenter
	op.ColorScale.ScaleWithColor(ink)
	text.Draw(screen, line, &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: size}, op)
}

// leave goes back to whichever screen opened this one, on the same terms Settings and Credits are
// under: the run's phase is untouched, because this is not a station of one.
func (s *AnimationsScene) leave(gs *state.GlobalState) {
	gs.ActiveScreen = gs.ReturnScreen
	gs.NewScreen = true
}
