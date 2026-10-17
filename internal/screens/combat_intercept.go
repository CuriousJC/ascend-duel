package screens

// combat_intercept.go — a hit and the shield that stops it, meeting on the hit's way in.
//
// **A stopped hit flies exactly as a landed one does, and is stopped on its way** *(owner's call)*.
// Its figure leaves wherever it was worked out — a creature's card, or the total at the end of a
// duelist's line — on the path and at the pace a landing figure takes into the target's card. A
// shield sets off at the same moment to meet it — the player's out of its heap, a creature's out of
// its own card — and reaches the path at the moment the figure gets there. The two break apart
// there, the crumble a stone uses. A matched block writes SURGE over the break; a hit of the
// creature's own element writes FIZZLE, because that is the creature shielding itself in it.
//
// **Where on the path they meet is one rule**, `interceptAt` — today the point the path crosses
// the screen's horizontal middle, with a place a share short of the target card built beside it.
//
// **Nothing lands on a card.** The attack card that was stopped is left as it is.
//
// **It is a view over the resolved log and cannot change an outcome.** The shield leaving its heap is
// the screen's pick, as every spent shield is — the engine counts shields per element and never says
// whose. The cursor waits for an intercept the way it waits for a hit: pacing, never a result.

import (
	"image"
	"image/color"
	"math"
	"strconv"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// interceptPlace is where along a hit's path its shield meets it.
type interceptPlace int

const (
	// meetScreenMiddle is where the path crosses the screen's horizontal middle.
	meetScreenMiddle interceptPlace = iota

	// meetNearTarget is interceptNearShare of the path short of the target's card.
	meetNearTarget
)

// interceptAt is the rule the game uses, and interceptNearShare how far short of the target
// meetNearTarget stops a hit.
const (
	interceptAt        = meetScreenMiddle
	interceptNearShare = 0.15
)

// interceptShare is how far along the path from `from` to `to` the meeting is, 0 to 1. A path that
// never crosses the middle — both ends on one side of it — is met halfway.
func interceptShare(gs *state.GlobalState, place interceptPlace, from, to image.Point) float64 {
	switch place {
	case meetNearTarget:
		return 1 - interceptNearShare
	}
	mid := float64(gs.ScreenWidth) / 2
	dx := float64(to.X - from.X)
	if dx == 0 {
		return 0.5
	}
	share := (mid - float64(from.X)) / dx
	if share <= 0 || share >= 1 {
		return 0.5
	}
	return share
}

// intercept is one hit stopped on its way into its target.
type intercept struct {
	// figure is the hit as it would have landed, written the way a landing figure is, and ink the
	// element of the card that threw it.
	figure string
	ink    cards.Element

	// side is who swung and target whose card the hit was flying into. seat is the swinging card's
	// seat in its row when the figure leaves a card, and -1 when it leaves a line's total at `from`
	// — a line is laid out once and never moves.
	side, target combat.Side
	seat         int
	from         image.Point

	// share is how far along its path the hit is met, and meetTicks the tick the figure gets there
	// at a landing figure's pace. Both are fixed when it is raised.
	share     float64
	meetTicks int

	// shield is the stopping shield's element and shieldFrom where it was when it set off.
	shield     cards.Element
	shieldFrom ui.ShieldPose

	// label is SURGE, FIZZLE or nothing, written over the break.
	label string

	// stopped is the seat of the opponent's attack a player's shield stopped, for the tutorial, and
	// -1 for any other intercept.
	stopped int

	t ui.Travel

	// figureImg is the figure drawn once onto its own image at the moment of impact, so it can be
	// broken into pieces like a card.
	figureImg *ebiten.Image
}

func (ic *intercept) Tick()     { ic.t.Tick() }
func (ic intercept) Done() bool { return ic.t.Done() }

// The clocks, in the game's own beats. The flight is a landing figure's, so a stopped hit travels
// at the pace a landed one does and is simply cut short.
func interceptFlyTicks() int   { return hitFlyTicks() }
func interceptBreakTicks() int { return ui.Beat(6, 5) }
func interceptHoldTicks() int  { return ui.Beat(2, 5) }

// aim fixes where along its path the hit is met and starts its clock: the flight to that point, the
// break, and the word's hold.
//
// **The figure keeps a landing figure's easing**, so the tick it reaches the meeting is where that
// easing passes `share` — the hit is the one it would have been, stopped, rather than a slower one
// aimed at a nearer point.
func (ic *intercept) aim(share float64) {
	ic.share = ui.Clamp01(share)
	ic.meetTicks = max(1, int(math.Ceil(float64(interceptFlyTicks())*(1-math.Sqrt(1-ic.share)))))
	ic.t = ui.NewTravel(0, ic.meetTicks+interceptBreakTicks()+interceptHoldTicks())
}

// met reports that the figure and the shield have reached each other.
func (ic intercept) met() bool { return ic.t.Age >= ic.meetTicks }

// sinceMeeting is how many ticks the two have been breaking apart.
func (ic intercept) sinceMeeting() int { return ic.t.Age - ic.meetTicks }

// The labels' inks. SURGE is the action-point bar's amber, because what a surge banked is budget;
// FIZZLE the attack red a line's verdict was written in.
func surgeInk() color.RGBA  { return apSpentColor }
func fizzleInk() color.RGBA { return ui.VerbInkFor(combat.CategoryAttack) }

// interceptScale is the size a shield is drawn at where it meets the hit.
const interceptScale = 1.0

// ---- raising them ----

// noteBlock sets a stopped hit and the shield that stops it on their way to each other. `Target` is
// whose shield was spent; the attack is the other side's.
//
// **A hand-forming attacker's block was raised by its line**, which walks the event — see
// throwColumn — so this is reached only for a solo attacker's swing.
func (s *CombatScene) noteBlock(gs *state.GlobalState, e combat.Event) {
	el := ui.ArtFor(e.Element)
	attacker := combat.SideA
	if e.Target == combat.SideA {
		attacker = combat.SideB
	}
	seat := e.Slot
	if seats := s.firingSeatsOf(attacker); len(seats) > 0 {
		seat = seats[0]
	}
	ic := intercept{
		figure: "-" + strconv.Itoa(e.Would), side: attacker, target: e.Target, seat: seat,
		ink:    s.handCardElement(attacker, seat),
		shield: el, shieldFrom: s.spendShield(gs, e.Target, el),
		stopped: -1,
	}
	if e.Surged {
		ic.label = "SURGE"
	}
	if e.Target == combat.SideA {
		ic.stopped = seat
	}
	s.raiseIntercept(gs, ic)
}

// interceptLine stops a duelist's line on its way out: a block by the opponent's shield, or a fizzle
// on the opponent's own element, which is the opponent shielding itself in it.
func (s *CombatScene) interceptLine(gs *state.GlobalState, col *mathColumn, e combat.Event) {
	side := s.Theater.mathBox.side
	target := combat.SideB
	if side == combat.SideB {
		target = combat.SideA
	}
	ic := intercept{
		figure: "-" + col.total().text, side: side, target: target, seat: -1, from: col.total().at,
		ink:     s.handCardElement(side, col.seat),
		stopped: -1,
	}
	switch e.Kind {
	case combat.KindFizzled:
		ic.shield = ui.ArtFor(e.Element)
		ic.shieldFrom = s.fighterShieldPose(gs, target)
		ic.label = "FIZZLE"
	case combat.KindBlocked:
		ic.shield = ui.ArtFor(e.Element)
		ic.shieldFrom = s.spendShield(gs, e.Target, ic.shield)
		if e.Surged {
			ic.label = "SURGE"
		}
	}
	col.spent = true
	s.raiseIntercept(gs, ic)
}

// raiseIntercept aims one along its path and puts it on stage, with the trail behind its figure.
func (s *CombatScene) raiseIntercept(gs *state.GlobalState, ic intercept) {
	ic.aim(interceptShare(gs, interceptAt, s.interceptOrigin(gs, ic), fighterCenter(gs, ic.target)))
	s.Theater.intercepts = append(s.Theater.intercepts, ic)
	s.raiseInterceptTrail(ic)
}

// spendShield takes the shield that stops a hit off its owner's heaps, or out of the owner's own
// card when nothing on screen was holding one.
func (s *CombatScene) spendShield(gs *state.GlobalState, side combat.Side, el cards.Element) ui.ShieldPose {
	if pose, ok := s.takeShield(gs, side, el); ok {
		return pose
	}
	return s.fighterShieldPose(gs, side)
}

// fighterShieldPose is a shield standing on one fighter's own corner card.
func (s *CombatScene) fighterShieldPose(gs *state.GlobalState, side combat.Side) ui.ShieldPose {
	r := fighterRect(gs, side)
	scale := float64(r.Dy()) / cardHeight
	return ui.HeapPose(float64(r.Min.X+r.Max.X)/2, float64(r.Min.Y+r.Max.Y)/2, 0, 1, scale)
}

// fighterRect and fighterCenter are one fighter's corner card: what a hit at them flies into.
func fighterRect(gs *state.GlobalState, side combat.Side) image.Rectangle {
	if side == combat.SideA {
		return ui.DuelistCardRect(gs)
	}
	return ui.EnemyCardRect(gs)
}

func fighterCenter(gs *state.GlobalState, side combat.Side) image.Point {
	r := fighterRect(gs, side)
	return image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
}

// firingSeatsOf is the marked seats on one side's row.
func (s *CombatScene) firingSeatsOf(side combat.Side) []int {
	if side == combat.SideB {
		return s.Theater.enemyFiringSeats
	}
	return s.Theater.firingSeats
}

// stoppedSeats is which of the opponent's attacks a player's shield has met this round.
func (s *CombatScene) stoppedSeats() map[int]bool {
	out := map[int]bool{}
	for _, ic := range s.Theater.intercepts {
		if ic.stopped >= 0 && ic.met() {
			out[ic.stopped] = true
		}
	}
	return out
}

// ---- drawing them ----

// interceptOrigin is where an intercept's figure set off, recomputed for a card's seat as every
// mover on this screen is.
func (s *CombatScene) interceptOrigin(gs *state.GlobalState, ic intercept) image.Point {
	if ic.seat < 0 {
		return ic.from
	}
	return s.handCardCenter(gs, ic.side, ic.seat)
}

// drawIntercepts draws every hit and shield on their way to each other, breaking apart where they
// met, and the word over the break.
func (s *CombatScene) drawIntercepts(gs *state.GlobalState, screen *ebiten.Image) {
	for i := range s.Theater.intercepts {
		ic := &s.Theater.intercepts[i]
		if !ic.Done() {
			drawIntercept(gs, screen, ic, s.interceptOrigin(gs, *ic), fighterCenter(gs, ic.target))
		}
	}
}

// interceptMeet is where on the path from origin to target the two meet.
func interceptMeet(ic intercept, origin, target image.Point) image.Point {
	return ui.LerpPoint(origin, target, ic.share)
}

// drawIntercept draws one: the figure on its way into `target` and the shield closing on the point
// of its path where they meet, then both crumbling there with the word over them. **The gallery
// calls this too**, so the page and the game are one drawing.
func drawIntercept(gs *state.GlobalState, screen *ebiten.Image, ic *intercept, origin, target image.Point) {
	meet := interceptMeet(*ic, origin, target)
	if !ic.met() {
		// The figure exactly as a landing one flies, so it is the same hit until it is stopped.
		p := ui.EaseOut(ui.Clamp01(float64(ic.t.Age) / float64(interceptFlyTicks())))
		at := ui.LerpPoint(origin, target, p)
		drawMathText(gs, screen, ic.figure, hitFigureSize, hitInkFor(ic.ink), at, hitFromScale, 1, false)
		q := ui.EaseOut(ui.Clamp01(float64(ic.t.Age) / float64(ic.meetTicks)))
		to := ui.ShieldPose{X: float64(meet.X), Y: float64(meet.Y), Scale: interceptScale, Alpha: 1}
		ui.DrawShield(screen, ic.shield, ui.LerpPose(ic.shieldFrom, to, q))
		return
	}

	age := ic.sinceMeeting()
	if age < interceptBreakTicks() {
		if ic.figureImg == nil {
			ic.figureImg = figureImage(gs, ic.figure, hitInkFor(ic.ink))
		}
		// **The seeds are fixed per element**, not per hit: the crumble's lattice is cached by seed
		// and size, so a seed per hit would grow that cache for the length of a session.
		fw, fh := ic.figureImg.Bounds().Dx(), ic.figureImg.Bounds().Dy()
		ui.DrawCrumble(screen, ic.figureImg, image.Pt(meet.X-fw/2, meet.Y-fh/2), 7, age, interceptBreakTicks())
		if art := systems.ArtMarkImage(ui.ShieldKey(ic.shield), ui.ShieldSize, ui.ShieldSize); art != nil {
			half := ui.ShieldSize / 2
			ui.DrawCrumble(screen, art, image.Pt(meet.X-half, meet.Y-half), uint32(ic.shield)*977+1, age,
				interceptBreakTicks())
		}
	}
	if ic.label != "" {
		ink := surgeInk()
		if ic.label == "FIZZLE" {
			ink = fizzleInk()
		}
		drawInterceptLabel(gs, screen, ic.label, ink, meet, age, interceptBreakTicks()+interceptHoldTicks())
	}
}

// figureImage is a hit's figure drawn once onto an image of its own, centred, so the break can take
// it apart in pieces.
func figureImage(gs *state.GlobalState, figure string, ink color.RGBA) *ebiten.Image {
	h := int(hitFigureSize * 1.6)
	w := int(mathWidth(gs, figure, hitFigureSize)) + h/2
	img := ebiten.NewImage(w, h)
	drawMathText(gs, img, figure, hitFigureSize, ink, image.Pt(w/2, h/2), hitFromScale, 1, false)
	return img
}

// interceptLabelSize is SURGE's and FIZZLE's type size.
const interceptLabelSize = 72

// drawInterceptLabel writes a word over a break, popping in as the two meet and fading over the end
// of its clock.
func drawInterceptLabel(gs *state.GlobalState, screen *ebiten.Image, word string, ink color.RGBA,
	at image.Point, age, ticks int) {

	pop := ui.Clamp01(float64(age) / float64(ui.Beat(1, 4)))
	scale := 1.35 - 0.35*ui.EaseOut(pop)
	alpha := float32(pop)
	if fade := ticks - ui.Beat(1, 3); age > fade {
		alpha *= float32(1 - ui.Clamp01(float64(age-fade)/float64(ticks-fade)))
	}

	op := &text.DrawOptions{}
	op.PrimaryAlign, op.SecondaryAlign = text.AlignCenter, text.AlignCenter
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(at.X), float64(at.Y))
	op.ColorScale.ScaleWithColor(ink)
	op.ColorScale.ScaleAlpha(alpha)
	systems.DrawUI(screen, word, &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: interceptLabelSize}, op)
}
