package screens

import (
	"image"
	"image/color"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
)

// The combo dialog: the blow's arithmetic acted out at the size of the screen, on the beat the
// hand fires.
//
// **It exists because the sum was the one number on this screen nobody could source**
// *(2026-08-18)*. The Resolution feed has always printed it — `(40 x 1.5 = 60)` — in sixteen-point
// text on the third line of a three-line box, beside the sentence it belongs to. That is a
// record, and a record is the right thing for the feed to be; what it is not is an *explanation*.
// A player watching a blow land could see the total and could not see which card paid for which
// part of it, so the multiplier read as a number the game had decided rather than one they had
// built. The fix is not a better sentence: it is showing the figures leaving the cards.
//
// **The dialog says nothing the feed does not.** Every figure in it comes off the same
// `KindCombo` event the feed's line does — `ComboAmounts`, `Multiplier`, `Amount` — and nothing
// here multiplies, adds or rounds. That is the rule to hold: this is a second *drawing* of one
// event, never a second arithmetic. If it ever needs a figure the event does not carry, the field
// goes on the event.
//
// **It may not change an outcome**, the same constraint as playback speed, the debug flags,
// `internal/trace` and every card in flight. What it does change is *pacing* — playback stops
// while it runs, and `advancePlayback` holds the cursor rather than the box racing a dwell it
// cannot fit inside. A round therefore takes longer with a combo in it, which is the point: the
// blow is the moment of the round and it used to go past in one and a quarter seconds.
//
// **It draws over the Resolution feed**, in the feed's own collapsed band, and clears itself
// before the damage lands. The band was chosen because it is where the player is already looking
// for what happened, and because it is the only place on the screen with room for forty-point
// numerals. What it costs is three lines of the record for about three seconds — which the feed
// still holds and redraws the moment the box is gone.

const (
	// The script's beats, in ticks at 60 a second.
	//
	// **They are separate constants rather than one scaled number**, because they answer
	// different questions: how long a shout needs to be read, how long a figure takes to cross a
	// quarter of the screen, and how long an operator needs to register. Tuning one should not
	// move the others.
	mathShoutTicks  = 34 // the hand's name popping in
	mathTermTicks   = 22 // one card's figure flying down into the row
	mathSymbolTicks = 11 // a +, an x or an = appearing in place
	mathTotalTicks  = 26 // the answer landing
	mathHoldTicks   = 40 // the finished sum held before the box clears

	// Type sizes. The figures are the point of the box, so they are the biggest thing on the
	// screen that is not a card; the operators are smaller because they are punctuation.
	mathTermSize   = 38
	mathSymbolSize = 30
	mathTotalSize  = 50
	mathShoutSize  = 62

	// mathItemGap is the air between one item of the sum and the next.
	mathItemGap = 16

	// mathPreviewSize is the hand's name while the player is still choosing, and it is smaller
	// than the shout on purpose. **A preview proposes and an announcement records** — the same
	// split the caption and the Resolution feed already keep — so the planned hand must not arrive
	// at the size the fired one does, or pressing DUEL! would change nothing about the loudest
	// thing on screen and the player would learn to stop reading it.
	mathPreviewSize = 40

	// mathPreviewAlpha is how solid it is. Far enough back to read as a proposal rather than an
	// event, and it is the second mark saying so: the size alone was ambiguous against a long
	// hand name, which fills more of the band than a short one at any size.
	mathPreviewAlpha = 0.62

	// mathShoutGap is how far clear of the bracketed cards the shout sits. The bracket already
	// stands `comboRingInset` off the cards, so this is measured from the ring rather than from
	// the card edge.
	mathShoutGap = 28

	// The scales an item is drawn at as it arrives. A flown figure grows into place, which reads
	// as coming toward the reader; an operator and the total drop *onto* the line from bigger,
	// which reads as being stamped there.
	mathFlyFromScale  = 0.45
	mathPopFromScale  = 1.7
	mathTotalPopScale = 2.4
	mathShoutPopScale = 2.1
)

// mathItem is one thing written on the line: a card's figure, an operator, the multiplier, or
// the answer.
//
// **Every item's resting place is computed once, before any of them is shown.** The alternative —
// laying the line out again as each item appears — would recentre the whole sum on every beat, so
// the figures already on screen would crawl sideways while the player was reading them. The line
// is measured in full at the start and revealed left to right into space it has already claimed.
type mathItem struct {
	text string
	size float64
	tint color.RGBA

	// from is where a flying item sets off: the centre of the card that paid the figure. Items
	// that are not flown leave this zero and pop in place.
	from image.Point
	fly  bool

	// at is the item's resting centre, filled by layOutMath.
	at image.Point

	// t is this item's own clock, started when the script reaches it.
	t travel
}

// comboMathBox is the whole dialog: a shout, a line of items, and a hold at the end.
//
// **It holds no arithmetic and no rules.** Everything in `items` is a string formatted once from
// the event; nothing here can disagree with the resolver, because nothing here computes.
type comboMathBox struct {
	active bool

	// shout is the hand's name — `PAIR!` — and shoutAt its centre. Empty when no hand was built,
	// which is the High Card: a lone attack shouting its own name is the same emptying of the
	// word that keeps `COMBO!` off a single Strike in the feed.
	shout   string
	shoutAt image.Point
	shoutT  travel

	items []mathItem

	// at is the item the script is currently running, and equals len(items) once they are all up.
	at int

	// hold is the pause on the finished sum, before the box clears and playback resumes.
	hold travel
}

// startComboMath builds the dialog for one KindCombo event and starts it running.
//
// **The layout needs the screen and `applyEvent` does not have it** — playback runs from
// `Update`, which does — so this takes `gs` and is called from there.
func (s *CombatScene) startComboMath(gs *state.GlobalState, e combat.Event) {
	if e.Kind != combat.KindCombo || e.ComboCardCount < 1 {
		return
	}

	box := comboMathBox{
		active: true,
		shout:  shoutFor(e),
		shoutT: newTravel(0, mathShoutTicks),
		hold:   newTravel(0, mathHoldTicks),
		items:  mathScript(e),
	}

	// **Where each figure sets off from is the screen's business, not the script's.** The script
	// says what the sum reads; this says which card on the table paid which term, which is the one
	// part of the box that has to know how a row is laid out.
	term := 0
	for i := range box.items {
		if !box.items[i].fly {
			continue
		}
		if term < e.ComboCardCount {
			box.items[i].from = s.comboCardCentre(gs, e.Side, e.ComboCards[term])
			term++
			continue
		}
		// The only flying item past the hand's own cards is the multiplier.
		box.items[i].from = s.comboMultiplierOrigin(gs, e)
	}

	s.layOutMath(gs, &box)
	box.shoutAt = s.comboShoutAt(gs, e)

	s.mathBox = box
}

// mathScript is the sum as a list of things to write, in the order they appear: a figure per card
// of the hand, a plus between each pair, then the multiplier, then the answer.
//
// **It takes no screen and computes no arithmetic**, which is what makes it the testable half of
// the box. Every string in it is formatted from a field the resolver already filled, so a test can
// pin what the player is shown without needing a window to draw it in — the same property that
// keeps `internal/combat` testable, applied to the one part of this file that is not geometry.
func mathScript(e combat.Event) []mathItem {
	var items []mathItem

	for i := 0; i < e.ComboCardCount; i++ {
		if i > 0 {
			items = append(items, mathOperator("+"))
		}
		items = append(items, mathItem{
			text: strconv.Itoa(e.ComboAmounts[i]),
			size: mathTermSize,
			tint: groundInk,
			fly:  true,
			t:    newTravel(0, mathTermTicks),
		})
	}

	// **The multiplier is dropped when it is the identity**, exactly as the feed's line drops it.
	// `20 x 1 = 20` is a sum with nothing in it, and the High Card is the commonest turn in the
	// game — so it would be the arithmetic shown most often and saying least.
	if e.Multiplier != comboIdentity {
		items = append(items, mathOperator("x"), mathItem{
			text: comboMultiplierText(e.Multiplier),
			size: mathTermSize,
			tint: attentionYellow,
			fly:  true,
			t:    newTravel(0, mathTermTicks),
		})
	}

	return append(items, mathOperator("="), mathItem{
		text: strconv.Itoa(e.Amount),
		size: mathTotalSize,
		tint: verbInkFor(combat.CategoryAttack),
		t:    newTravel(0, mathTotalTicks),
	})
}

// mathOperator is a `+`, an `x` or an `=`: punctuation, so it pops in place rather than flying.
func mathOperator(str string) mathItem {
	return mathItem{
		text: str,
		size: mathSymbolSize,
		tint: mathOperatorInk(),
		t:    newTravel(0, mathSymbolTicks),
	}
}

// handWasBuilt is the one predicate for "the player made something", and it is the engine's own:
// a hand of two or more cards, as opposed to the High Card falling through.
//
// **It is asked of the catalogue rather than of the card count on the event**, so a hand added to
// `data/combos.json` is covered without this file learning about it.
func handWasBuilt(e combat.Event) bool {
	hand, ok := combat.HandByID(e.Hand)
	return ok && hand.Cards() > 1
}

// shoutFor is the hand's name made into an exclamation: `PAIR!`, `FOUR OF A KIND!`. It is empty
// for a hand nobody built, which is the High Card — a lone attack shouting its own name is the
// same emptying of the word that keeps `COMBO!` off a single Strike in the feed.
//
// **The name comes from the catalogue**, like every other place the screen names a hand, so a
// hand renamed in `data/combos.json` is renamed once. Caps are safe at this size — the kubasta
// note in CLAUDE.md is about small text, where `VITAE` renders as `VITRE`; at sixty points the
// diagonal on an uppercase A is unmistakable.
func shoutFor(e combat.Event) string {
	if !handWasBuilt(e) {
		return ""
	}
	hand, _ := combat.HandByID(e.Hand)
	return handShout(hand.Name)
}

// handShout is a hand's name made into the exclamation, and it is one function because the planned
// hand and the fired one have to be the same words. `drawPlannedHand` writes the name during
// selection and `shoutFor` writes it on the beat it fires; two spellings of PAIR would read as two
// different things happening.
func handShout(name string) string { return upper(name) + "!" }

// mathOperatorInk is the ground ink faded toward the screen: an operator is punctuation between
// two figures, and drawing it at the figures' weight would make `+` compete with `20`.
//
// **Faded toward the ground, not scaled toward black.** The combat screen's ground is cream, so
// `ColorAtStrength` would make this *louder* than the figures rather than quieter — the trap
// CLAUDE.md's colour section describes, and one this screen has fallen into before.
func mathOperatorInk() color.RGBA { return systems.ColorToward(groundInk, screenGround, 45) }

// comboMathRect is the band the sum is written in: the Resolution feed's rows, at the table's
// width.
//
// **It takes its height from the feed and its width from the table, and neither is an accident.**
//
// The vertical band is the feed's *collapsed* box, computed here rather than taken from
// `feedRect`. A player holding the feed open grows that box upward, and a dialog that moved with
// it would re-lay a line of figures out from under a reader mid-flight — so the box claims the
// band the feed occupies at rest, whatever the feed is currently doing.
//
// The width is deliberately **not** the feed's. `feedRect` spans `handBand`, which is a function
// of how many cards are in the hand — and that is fine for the feed, whose rows are left-aligned
// sentences that simply have less room, but wrong for a centred line of large figures that does
// not wrap and cannot shrink. A two-card hand gives a band around 330px against a widest sum of
// roughly 640, so the arithmetic would have run off both ends of it in exactly the rounds a duel
// is decided in. `TestTheWidestSumFitsItsBand` is what found that and what holds it.
//
// The table's insets are the right width because the box is drawn on the bare ground rather than
// inside a pane — it is an overlay at the same width as the two rows of cards it is about, not a
// pane whose edges have to line up with another pane's.
func (s *CombatScene) comboMathRect(gs *state.GlobalState) image.Rectangle {
	bottom := gs.PctY(handTopPct) - feedGapAboveCards
	return image.Rect(tableInset, bottom-feedHeight(), gs.ScreenWidth-tableInset, bottom)
}

// layOutMath measures every item and centres the finished line in the band.
func (s *CombatScene) layOutMath(gs *state.GlobalState, box *comboMathBox) {
	r := s.comboMathRect(gs)

	widths := make([]float64, len(box.items))
	total := 0.0
	for i := range box.items {
		widths[i], _ = text.Measure(box.items[i].text, mathFace(gs, box.items[i].size), 0)
		total += widths[i]
	}
	if len(box.items) > 1 {
		total += float64(mathItemGap * (len(box.items) - 1))
	}

	x := float64(r.Min.X+r.Max.X)/2 - total/2
	cy := (r.Min.Y + r.Max.Y) / 2
	for i := range box.items {
		box.items[i].at = image.Pt(int(x+widths[i]/2), cy)
		x += widths[i] + mathItemGap
	}
}

// mathFace is the box's type. One font at four sizes; the screen has exactly one face.
func mathFace(gs *state.GlobalState, size float64) *text.GoTextFace {
	return &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: size}
}

// comboCardCentre is where a figure sets off from: the middle of the card that paid it, lifted,
// because every card of the hand is raised by the time the combo event arrives.
//
// **It recomputes the seat rather than storing a point**, exactly as every card in flight on this
// screen does — the row re-lays out under a moving thing, and a cached coordinate goes stale.
func (s *CombatScene) comboCardCentre(gs *state.GlobalState, side combat.Side, seat int) image.Point {
	var at image.Point
	if side == combat.SideB {
		at = enemySeatAt(gs, seat, len(s.enemyDealt), s.enemySplit())
	} else {
		at = playedSeatAt(gs, seat, len(s.resolved), s.playedSplit())
	}
	at = lift(at, true)
	return image.Pt(at.X+cardWidth/2, at.Y+cardHeight/2)
}

// comboBracketBox is the rectangle the ring is drawn round: the union of the cards that formed
// the hand, standing off by the same inset `drawComboBracket` uses.
//
// **The two agree by sharing a number, not by being the same code.** `drawComboBracket` takes a
// list of positions and knows nothing about events; this takes an event and knows nothing about
// drawing. What they share is `comboRingInset`, which is the thing that would have to move.
func (s *CombatScene) comboBracketBox(gs *state.GlobalState, e combat.Event) image.Rectangle {
	var box image.Rectangle
	for _, seat := range e.ComboCards[:e.ComboCardCount] {
		c := s.comboCardCentre(gs, e.Side, seat)
		card := image.Rect(c.X-cardWidth/2, c.Y-cardHeight/2, c.X+cardWidth/2, c.Y+cardHeight/2)
		if box.Empty() {
			box = card
			continue
		}
		box = box.Union(card)
	}
	if box.Empty() {
		return box
	}
	return box.Inset(-comboRingInset)
}

// comboShoutAt is where the hand's name is written: **clear of the cards it names, on whichever
// side of them the row is not**.
//
// The player's row is left-aligned, so its shout takes the space to the right; the opponent's is
// right-aligned, so its shout goes left. That is what "beside the hand" has to mean on a screen
// where the two hands are anchored to opposite edges — a fixed side would put one of them on top
// of the cards it is about.
func (s *CombatScene) comboShoutAt(gs *state.GlobalState, e combat.Event) image.Point {
	box := s.comboBracketBox(gs, e)
	if box.Empty() {
		return image.Pt(gs.ScreenWidth/2, tableRowTop(gs)+cardHeight/2)
	}

	cy := (box.Min.Y + box.Max.Y) / 2
	if e.Side == combat.SideB {
		return image.Pt((tableInset+box.Min.X-mathShoutGap)/2, cy)
	}
	return image.Pt((box.Max.X+mathShoutGap+gs.ScreenWidth-tableInset)/2, cy)
}

// comboMultiplierOrigin is where the multiplier flies from: the shout, when there is one, because
// the shout is what the multiplier *is* — `PAIR!` and `1.5` are one fact said twice, and having
// the figure leave the word is what joins them. With no shout it drops in from above the band.
func (s *CombatScene) comboMultiplierOrigin(gs *state.GlobalState, e combat.Event) image.Point {
	if handWasBuilt(e) {
		return s.comboShoutAt(gs, e)
	}
	r := s.comboMathRect(gs)
	return image.Pt((r.Min.X+r.Max.X)/2, r.Min.Y-cardHeight/2)
}

// --- the clock ---------------------------------------------------------------------------

// running reports whether the box is holding the round. **Playback does not advance while this is
// true**, which is what makes the dialog a beat of the round rather than something drawn over
// one — and it is the only thing on this screen that can stop the cursor.
func (b *comboMathBox) running() bool {
	if !b.active {
		return false
	}
	return b.at < len(b.items) || !b.hold.done()
}

// tick runs one frame of the script: the shout, then each item in turn, then the hold.
//
// **One item at a time and never two at once.** The whole point of the box is that a figure
// arrives, is read, and is then joined by an operator; overlapping the beats would put the sum on
// screen at the speed the feed already manages.
func (b *comboMathBox) tick() {
	if !b.active {
		return
	}
	if b.shout != "" && !b.shoutT.done() {
		b.shoutT.tick()
		return
	}
	if b.at < len(b.items) {
		b.items[b.at].t.tick()
		if b.items[b.at].t.done() {
			b.at++
		}
		return
	}
	b.hold.tick()
}

// clear takes the box down. Called when the script finishes and whenever a round or a fight
// starts, so a box left up by a screen change cannot outlive the round it describes.
func (b *comboMathBox) clear() { *b = comboMathBox{} }

// --- drawing -----------------------------------------------------------------------------

// drawPlannedHand writes the name of the hand the current selection has already formed, in the
// band the sum will be written across the moment DUEL! is pressed.
//
// **The name goes where its arithmetic is going to go** *(2026-08-18)*. That continuity is the
// whole reason it is here rather than over the hand row: the player reads THREE OF A KIND, presses
// DUEL!, and the figures land in the space the words were occupying. Putting it above the cards was
// impossible anyway — see drawComboPreview, whose comment records that the feed sits five pixels
// over the resting row and a selected card lifts into it.
//
// **It carries no number, and that is a rule rather than an omission.** `Blow.Base` is the
// resolver working against a strength and a shock roll that have not happened, so a figure shown
// here could be contradicted by the round a second later — worse than no figure. The name is the
// part that is already true.
//
// **Only a built hand is named**, and the preview cannot disagree with the shout about which hands
// those are: `Blow.Formed` is `Hand.Cards() > 1` and `handWasBuilt` asks an event the same
// question, so HIGH CARD never appears in either place. That is the same emptying of the word that
// keeps COMBO! off a single Strike in the feed.
//
// It draws nothing once playback starts, because `previewBlow` is gated on `planning()` — so this
// and the real shout can never be on screen together.
func (s *CombatScene) drawPlannedHand(gs *state.GlobalState, screen *ebiten.Image) {
	blow, ok := s.previewAttack()
	if !ok {
		return
	}

	r := s.comboMathRect(gs)
	at := image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
	drawMathText(gs, screen, handShout(blow.Hand.Name), mathPreviewSize, attentionYellow,
		at, 1, mathPreviewAlpha)
}

// drawComboMath draws the shout and however much of the sum has been revealed.
//
// **Items past `at` are not drawn at all**, rather than drawn transparent. A figure fading up
// from nothing over the eleven ticks before its turn would make the line look pre-written, which
// is exactly the impression the box exists to break.
func (s *CombatScene) drawComboMath(gs *state.GlobalState, screen *ebiten.Image) {
	b := &s.mathBox
	if !b.active {
		return
	}

	if b.shout != "" {
		drawMathText(gs, screen, b.shout, mathShoutSize, attentionYellow, b.shoutAt,
			popScale(mathShoutPopScale, b.shoutT), alphaOf(b.shoutT))
	}

	for i := range b.items {
		it := &b.items[i]
		switch {
		case i < b.at:
			drawMathText(gs, screen, it.text, it.size, it.tint, it.at, 1, 1)
		case i == b.at:
			drawArrivingMathItem(gs, screen, it)
		}
	}
}

// drawArrivingMathItem is the one item currently in motion.
//
// **A flown figure travels and grows; a stamped one only shrinks onto the line.** The difference
// is the whole grammar of the box: something that flies came from a card, and something that pops
// is punctuation the game supplied.
func drawArrivingMathItem(gs *state.GlobalState, screen *ebiten.Image, it *mathItem) {
	if !it.fly {
		pop := mathPopFromScale
		if it.size == mathTotalSize {
			pop = mathTotalPopScale
		}
		drawMathText(gs, screen, it.text, it.size, it.tint, it.at, popScale(pop, it.t), alphaOf(it.t))
		return
	}

	t := easeOut(it.t.progress())
	drawMathText(gs, screen, it.text, it.size, it.tint,
		lerpPoint(it.from, it.at, t), mathFlyFromScale+(1-mathFlyFromScale)*t, 1)
}

// popScale eases a scale down to 1 from `from`. Used by everything that appears in place.
func popScale(from float64, t travel) float64 {
	return from + (1-from)*easeOut(t.progress())
}

// alphaOf fades an appearing item up over the first half of its beat, so a stamped item is not
// simply absent on one frame and present on the next.
func alphaOf(t travel) float32 {
	p := t.progress() * 2
	if p > 1 {
		p = 1
	}
	return float32(p)
}

// drawMathText writes one string centred on a point, at a scale and an alpha.
//
// **Centred by its measured box rather than by an alignment flag**, because the item also scales:
// an aligned draw scales about the text's own origin, so the figure would slide sideways as it
// grew. Measuring and translating by half puts the middle of the glyphs on the point at every
// scale, which is what lets a figure grow without drifting.
func drawMathText(gs *state.GlobalState, screen *ebiten.Image, str string, size float64,
	tint color.RGBA, at image.Point, scale float64, alpha float32) {

	if str == "" || alpha <= 0 {
		return
	}
	face := mathFace(gs, size)
	w, h := text.Measure(str, face, 0)

	op := &text.DrawOptions{}
	op.GeoM.Translate(-w/2, -h/2)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(at.X), float64(at.Y))
	op.ColorScale.ScaleWithColor(tint)
	op.ColorScale.ScaleAlpha(alpha)
	text.Draw(screen, str, face, op)
}

// upper is the counterpart of `lower` in combat_panes.go, and it is written out rather than
// taken from `strings` for no reason beyond that `lower` is: the two sit next to each other in
// the same sentence-building code and reading one should not send you to a different idiom.
func upper(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			r -= 'a' - 'A'
		}
		out = append(out, r)
	}
	return string(out)
}
