package screens

// A hand arriving: the deal, the flip cascade, and the sort that follows.
//
// **Every hand in a fight comes in this way** — the opening one and every refill — which is the
// whole reason this file exists. The opening hand used to appear: `resetDeck` filled the row and
// the first thing the player saw was eight cards already standing there, while a mid-round refill
// flew out of the pile. Two ways of doing one thing, and the one the player meets first was the
// one that said nothing.
//
// # The three stages, and why they are in this order
//
//  1. **Deal.** Cards fly out of the pile left to right, staggered, turning face up as they go —
//     and they land **in pile order**, not in sorted order. What the row says at the end of this
//     stage is "this is what the shuffle gave you".
//  2. **Cascade.** One beat per worn ring that recolors anything in the hand, in worn order. The
//     ring rattles and lights, every card it touches rattles with it and morphs into its new
//     color, all at once. A second ring reading the first one's answer is a second beat, so a
//     lightning card under lightning-to-ice and ice-to-earth is watched going lightning, then
//     ice, then earth — see combat.FlipSteps, which is the walk this is a picture of.
//  3. **Sort.** The row rearranges itself into the player's chosen key, on the slides the sort
//     buttons already use.
//
// **The sort is last, and that is a reversal** *(owner's call, 2026-09-15)*. `spendSelected` used
// to sort *before* anything was animated so a dealt card flew straight to the slot it would end up
// in. What that buys is one journey per card; what it costs is the deal having nothing to say —
// a hand that arrives pre-sorted never shows the player what they were dealt, and the cascade then
// plays over a row that has already been arranged by a key the cascade is about to invalidate.
//
// # It may never change an outcome
//
// **The hand holds the finished cards from the first frame.** `drawHand` applies the whole cascade
// as it deals, exactly as it always did, and what this file owns is the *faces* — the pile face,
// then one per ring. So a card selected while it is still showing its lightning face is the earth
// card the engine will score, and nothing that asks the hand a question gets a different answer
// while a deal is running. That is `shownLife`'s division, one object over: the model moves first
// and the drawing catches up.
//
// The one thing here that does touch the model is the sort at the end, which reorders the row and
// nothing else. It runs on a hand with no round in flight, and `sortHand` resyncs the queue.

import (
	"image"
	"math"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

// dealStage is where a hand's arrival has got to. **Ordered, and the sequence only ever moves
// forward** — there is no path back to an earlier stage, so a deal interrupted by `Init` is
// dropped whole rather than rewound.
type dealStage int

const (
	dealIdle dealStage = iota
	dealDealing
	dealCascading
)

// The cascade's rattle. **Tighter and wider than the relic pane's own** — `relicShakeSwings` is a
// figure being written into a sum, one card at a time, where this is a whole row going at once and
// has to carry over a screen with eight cards changing on it.
const (
	dealShakeSwings = 6.0
	dealShakeWidth  = 9
)

// dealRingTicks is how long one ring's beat lasts: the morph's own wait plus its dissolve, so the
// beat ends on the frame the last card has finished changing. **Read off the morph rather than
// chosen**, or a retune of the dissolve would leave the ring rattling into silence or cut it off
// mid-change.
func dealRingTicks() int { return ui.MorphWaitTicks() + ui.MorphTicks() }

// handDeal is one hand arriving. The zero value is a deal that is not running.
type handDeal struct {
	stage dealStage

	// cards are the cards this deal brought out of the pile, in the order they are dealt.
	// **Survivors are not in it** — a card already in the hand is drawn by the row and slides
	// with everything else at the sort.
	cards []dealtHandCard

	// rings is the cascade: one entry per worn relic that recolors anything in this hand, in worn
	// order, which is the order the flips read each other in.
	rings []dealRing

	// at is the ring currently playing, and len(rings) once the cascade is spent.
	at int

	// hold is the beat one ring's toast runs for.
	hold ui.Travel
}

// dealRing is one relic in the cascade.
//
// **It carries the record key, not the seat it is worn in.** The pane can be re-ordered by a drag
// while a deal runs, and a stored index would then rattle whichever relic had moved into it — the
// same rule the hand's morphs follow by carrying a card's identity rather than its slot.
type dealRing struct {
	key string

	// shake is the ring's own rattle, so the relic pane can ask how far sideways it sits without
	// this file reaching into the pane's state.
	shake ui.Travel
}

// dealtHandCard is one card on its way out of the pile and through the cascade.
type dealtHandCard struct {
	// index and Count locate the slot, exactly as cardFlight's pair does: the hand re-lays out
	// around a resize, so a cached pixel position would be wrong the moment the window moved.
	index, Count int

	flight ui.Travel

	// faces is the card as the pile holds it, then one more for each ring that recolors it.
	// **faces[0] is always the pile's**, which is why a card no ring touches still has one.
	faces []dealFace

	// shown is the face on screen. It steps forward as each ring's morph finishes.
	shown int

	// m is the change currently running on this card, and morphing whether there is one. A morph
	// with no faces is a legal zero value, so the flag is what says it is live.
	m        ui.Morph
	morphing bool

	shake ui.Travel
}

// dealFace is one of a card's faces and the ring that produced it.
type dealFace struct {
	// ring is the index into handDeal.rings this face is the answer to, and -1 for the face the
	// card came out of the pile wearing.
	ring int
	spec cards.Spec
}

// running reports whether a deal is on stage at all.
func (d handDeal) Running() bool { return d.stage != dealIdle }

// startDeal raises the sequence over the cards drawHand has just added.
//
// `from` is the first hand index the deal brought in and `pile` the cards as the draw pile held
// them, parallel to s.hand[from:]. The two come apart because the hand holds the *finished* card
// and the pile holds the one the cascade starts from.
func (s *CombatScene) startDeal(from int, pile []combat.Card) {
	s.Theater.deal = handDeal{}

	// A hand that gained nothing still sorts — a refill that drew no cards is a row that has
	// closed up, and the player's key still applies to what is left.
	if from > len(s.hand) || len(pile) != len(s.hand)-from || len(pile) == 0 {
		s.finishDeal()
		return
	}

	deal := handDeal{stage: dealDealing}

	// Each dealt card's own cascade, which is what the rings are then drawn from.
	steps := make([][]combat.FlipStep, len(pile))
	if s.run != nil {
		for i, raw := range pile {
			steps[i] = s.run.FlipStepsFor(raw)
		}
	}

	// **The rings are walked off the worn row, not off the cards** *(bug, 2026-09-15)*. The first
	// version built this list in the order the first card to meet each ring happened to see it in,
	// on the reasoning that combat.FlipSteps walks the worn row — which is true *per card* and says
	// nothing about the order across a hand. A hand whose first earth card sits before its first
	// lightning card gave the third-worn relic ring 0 and the first-worn relic ring 1, so the row
	// fired right to left.
	//
	// **And it was worse than an ordering.** A card's faces carry the ring that made each one and
	// are shown in sequence, so a face list whose ring indices did not ascend could never be walked
	// to the end — `startDealRing` steps `shown` only when the *next* face belongs to the ring now
	// playing. A lightning card under all three rings stopped on earth and stayed there, wearing a
	// face the hand it is in disagrees with.
	seen := map[string]int{}
	for _, card := range steps {
		for _, st := range card {
			seen[combat.RelicOf(st.Relic).Key] = 0
		}
	}

	// **A ring that touches nothing in this hand is not a beat**: a rattle over a row where nothing
	// changes is the screen saying a relic fired when it did not. That is what `seen` filters.
	ring := map[string]int{}
	if s.run != nil {
		for _, w := range s.run.WornRelics() {
			key := combat.RelicOf(w.Relic).Key
			if _, touches := seen[key]; !touches {
				continue
			}
			if _, had := ring[key]; had {
				continue
			}
			ring[key] = len(deal.rings)
			deal.rings = append(deal.rings, dealRing{key: key})
		}
	}

	count := len(s.hand)
	for i, raw := range pile {
		faces := []dealFace{{ring: -1, spec: s.dealFace(raw)}}

		running := raw
		for _, st := range steps[i] {
			running.Element = st.To
			faces = append(faces, dealFace{
				ring: ring[combat.RelicOf(st.Relic).Key],
				spec: s.dealFace(running),
			})
		}

		deal.cards = append(deal.cards, dealtHandCard{
			index: from + i, Count: count,
			flight: ui.NewTravel(i*flightStaggerPer(), flightTicks()),
			faces:  faces,
		})
	}

	s.Theater.deal = deal
}

// dealFace is one card's face as the deal draws it.
//
// **Always enabled and never selected**, for handFace's reason: those two are the row's mood rather
// than the card's, and a face captured with them baked in would compare the hand's state instead of
// the card's.
// **A scene with no fighter still builds one**, which is the flight tests and `OpeningHand`: the
// face is then the card at its own cost with nothing worn, which is exactly what those callers are
// looking at.
func (s *CombatScene) dealFace(c combat.Card) cards.Spec {
	var d combat.Duelist
	if s.fighter != nil {
		d = s.fighter.Duelist
	}
	return ui.CardSpec(c, ui.HeldBy(d, c), true, false)
}

// tickDeal advances the sequence a frame. Called every tick from Update, whether or not one is
// running, so there is one place the stages hand over.
func (s *CombatScene) tickDeal() {
	d := &s.Theater.deal
	if !d.Running() {
		return
	}

	for i := range d.cards {
		d.cards[i].flight.Tick()
		d.cards[i].shake.Tick()
		if !d.cards[i].morphing {
			continue
		}
		d.cards[i].m.Tick()
		if d.cards[i].m.Done() {
			// The face the morph was heading for is the face the card wears from here.
			d.cards[i].morphing = false
			d.cards[i].shown++
		}
	}
	for i := range d.rings {
		d.rings[i].shake.Tick()
	}

	switch d.stage {
	case dealDealing:
		for _, c := range d.cards {
			if !c.flight.Done() {
				return
			}
		}
		d.stage, d.at = dealCascading, 0
		s.startDealRing()

	case dealCascading:
		d.hold.Tick()
		if !d.hold.Done() {
			return
		}
		d.at++
		s.startDealRing()
	}
}

// startDealRing begins the beat for the ring at d.at, or moves on to the sort when the cascade is
// spent.
//
// **One beat for every card the ring touches** *(the shield break's rule, and the rune's)*. Eight
// cards changing one after another would be eight pauses over a hand the player is waiting to play;
// what the beat says is one thing about the relic rather than eight about cards.
func (s *CombatScene) startDealRing() {
	d := &s.Theater.deal
	if d.at >= len(d.rings) {
		s.finishDeal()
		return
	}

	d.hold = ui.NewTravel(0, dealRingTicks())
	d.rings[d.at].shake = ui.NewTravel(0, dealRingTicks())

	for i := range d.cards {
		c := &d.cards[i]
		next := c.shown + 1
		if next >= len(c.faces) || c.faces[next].ring != d.at {
			continue
		}
		c.m = ui.MorphInto(c.faces[c.shown].spec, c.faces[next].spec, cards.Hand)
		c.morphing = true
		c.shake = ui.NewTravel(0, dealRingTicks())
	}
}

// finishDeal is the last stage: the row rearranges into the player's key and the sequence stands
// down.
//
// **The slides are the sort buttons' own**, which is what stops the deal owning a fourth mover.
func (s *CombatScene) finishDeal() {
	was := len(s.hand)
	order := s.sortHand()

	for to, from := range order {
		if from == to {
			continue
		}
		s.addSlide(ui.CardSlide{
			Travel:    ui.NewTravel(0, ui.SlideTicks()),
			Card:      s.hand[to].Card,
			FromLift:  selectedLift(s.hand[to].selected),
			ToLift:    selectedLift(s.hand[to].selected),
			FromIndex: from, FromCount: was,
			ToIndex: to, ToCount: len(s.hand),
		})
	}

	s.Theater.deal = handDeal{}
	s.syncQueue()
}

// dealtTo reports whether the card in hand slot i is being drawn by the deal, so the row leaves
// that slot alone. Same rule as inboundTo and slidingTo: the card is in the hand, and what is
// suppressed is a drawing.
func (s *CombatScene) dealtTo(i int) bool {
	for _, c := range s.Theater.deal.cards {
		if c.index == i {
			return true
		}
	}
	return false
}

// dealShakeOffset is how far sideways something in the cascade sits this frame.
//
// **A decaying sine, like the relic pane's**, and for the same reason: the card has to come back to
// where it was and be still when it gets there. The figures differ because the job does — see the
// constants above.
func dealShakeOffset(t ui.Travel) int {
	if t.Done() || t.Waiting() {
		return 0
	}
	p := t.Progress()
	return int(math.Sin(p*math.Pi*2*dealShakeSwings) * (1 - p) * float64(dealShakeWidth))
}

// dealRingClock is the beat one ring of the cascade is on, for the relic pane to draw its toast
// from, and whether that ring is firing at all right now.
//
// **It hands back the clock rather than the offsets**, so the shift, the tilt and the light are all
// worked out in the one place that knows what a toast is — see relicToast in combat_relics.go. A
// mark added there reaches the cascade without this file learning about it.
//
// **Keyed on the record rather than the seat**, so a relic dragged along the row during a deal takes
// its rattle with it.
func (s *CombatScene) dealRingClock(key string) (ui.Travel, bool) {
	for _, r := range s.Theater.deal.rings {
		if r.key != key || r.shake.Done() {
			continue
		}
		return r.shake, true
	}
	return ui.Travel{}, false
}

// drawDeal draws the cards the sequence owns: flying out of the pile, standing in the row wearing
// whichever face the cascade has reached, or coming apart into the next one.
func (s *CombatScene) drawDeal(gs *state.GlobalState, screen *ebiten.Image) {
	for _, c := range s.Theater.deal.cards {
		if c.flight.Waiting() {
			continue
		}

		to := slotAt(gs, c.index, c.Count)
		if !c.flight.Done() {
			drawDealtCard(gs, screen, deckStackRect(gs).Min, to, c.flight.Progress(), c.faces[0].spec, s.backSpec())
			continue
		}

		to.X += dealShakeOffset(c.shake)
		if c.morphing {
			ui.DrawMorph(gs, screen, to, c.m)
			continue
		}
		drawSpecAt(gs, screen, to, c.faces[c.shown].spec, cards.Hand)
	}
}

// drawSpecAt blits a finished face at rest.
//
// Separate from drawCard because that one builds its Spec from an combat.Card, and the deal's faces
// are already built — asking it to rebuild one would hand back the card as the *hand* holds it,
// which during a cascade is the face the row has not reached yet.
func drawSpecAt(gs *state.GlobalState, screen *ebiten.Image, at image.Point, spec cards.Spec, st cards.Style) {
	img := ui.CardImage(gs, spec, st)
	if img == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(at.X), float64(at.Y))
	screen.DrawImage(img, op)
}
