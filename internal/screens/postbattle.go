package screens

// The post-battle screen: **pick a worm, then pick the card it eats.**
//
// It is the first of the between-fight scenes — a shop and a room choice come after it — and all
// of them are ordinary scenes in the registry rather than modes of the combat screen. That is
// what keeps `CombatScene` from growing a fourth phase, and it is why the chain is wired as
// screen changes: win → post-battle → combat.
//
// **Two stages, worm first** *(2026-08-17)*. Two worms are drawn from the catalogue and offered
// as cards; choosing one deals a hand off the run deck to apply it to. It ran the other way round
// first — pick a card, then say what to do to it — and the reason it turned is that the *worm* is
// the reward. What you were given for winning has to be the thing on the screen when you arrive,
// and a menu of verbs under a chosen card made the reward look like a property of the card.
//
// **A worm varies a card the game already defines.** It recolours it, removes it, or copies it;
// the concept is never touched, so nothing here can produce a card `internal/combat` cannot
// resolve. See `internal/session/worm.go` for the target vocabulary and why it is short.
//
// **The offer is indices into the run deck, not cards.** Alteration happens between fights, when
// no pile is live and nothing else is touching the list, so a position is unambiguous for exactly
// as long as the screen is up. That is what makes card identity unnecessary — see MECHANICS.md
// for what mid-fight alteration would cost instead.

import (
	"fmt"
	"image"
	"math/rand"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/trace"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"image/color"
)

// wormsOffered is how many alterations a win puts up, and vitaeReward is what the third card is
// worth.
//
// **Three cards, and one of them is always the vitae** *(2026-08-17)*. Declining used to be a
// button off to one side, which made it the odd thing out: the reward for a fight was two cards
// and an escape hatch. It is a *card* now, so every path through the screen is the same path —
// **you pick a card and it flies to the middle** — and choosing money over a change to the deck is
// a decision with a price rather than a refusal.
const (
	wormsOffered = 2
	vitaeReward  = 5
)

// prize is one of the three cards on the table: a worm, or the vitae.
//
// **A kind rather than a worm with a `vitae` target**, deliberately. A worm targets a *card* and
// this one does not touch the deck at all — putting it in that vocabulary would mean every worm
// carrying a target it might not have, which is the shape `Category` was refused for on the action
// cards.
type prize struct {
	vitae bool

	// taken is set once this prize has been picked and paid out. **It stays in the row rather than
	// being removed from it** — a card leaving would move the two beside it, and the vitae's seat
	// never moving is the whole reason it is always last. Only a ring that adds a pick can produce a
	// row with a taken card still in it.
	taken bool
	worm  session.Worm
}

func (p prize) name() string {
	if p.vitae {
		return "Vitae"
	}
	return p.worm.Name
}

// Where the two rows sit and where the controls sit under them. Percentages anchor the groups;
// offsets inside a group stay in pixels, per CLAUDE.md.
const (
	// The worm row sits where the eye lands, and the card row sits lower, so the second stage
	// reads as "this worm, applied down there" rather than as a new screen.
	wormRowPct  = 26
	offerRowPct = 56

	offerTitleTop = 52
	offerHintTop  = 96

	offerButtonsPct   = 88
	offerButtonWidth  = 220
	offerButtonHeight = 56
)

// stage is how far through the choice the player has got.
type stage int

const (
	// pickWorm: the two worms are up and nothing else is.
	pickWorm stage = iota

	// pickCard: a worm is chosen, and the hand it applies to is dealt.
	pickCard

	// morph: a card is chosen and the screen shows **what it would become**, before and after,
	// with nothing committed. Nothing on this screen may surprise the player: a worm reads as a
	// gift and the whole reason to show the result first is that some of them are trades.
	morph

	// settled: the alteration is taken and the new card is shown alone, so the thing that was won
	// is looked at before it disappears into a deck of forty-eight.
	settled
)

// This screen's two clocks, **both proportions of the game's one speed** *(2026-08-21)*.
//
// They were raw tick counts — 26 and 100 — written before `beat` existed, which meant this screen
// was outside the setting the duel is paced by: turning the game's speed down would have sped up a
// round and left the reward screen exactly as slow as it was. See clock.go. The one behaviour
// change is that the flight is 25 ticks rather than 26, which is a frame and a half.
//
// `var` rather than `const` because `beat` is a function, exactly like `victoryHoldTicks`.
var (
	// settleFlightTicks is how long the won card takes to cross to the middle.
	//
	// **Cards fly to where they are going, everywhere in this game.** A card that appears in the
	// middle is a card that was never anywhere else, and the whole point of this screen is that a
	// thing was *won* and has come to you.
	settleFlightTicks = beat(1, 1)

	// morphHoldTicks is how long the finished card is held before the screen leaves. **Long enough
	// to read, short enough not to need a button** — the click that took the worm is the last
	// input the player has to make.
	morphHoldTicks = beat(4, 1)
)

// PostBattleScene offers one alteration to the run deck.
type PostBattleScene struct {
	// prizes is the three cards on the table — two worms and the vitae — and chosen is which one,
	// an index into it, or -1.
	prizes []prize
	chosen int

	// offer is the hand, as indices into the run deck. **Dealt fresh off the whole deck**,
	// ignoring whatever the fight left in the piles: a reward is about what you own, not about
	// what you happened to draw.
	offer []int

	stage stage

	// **There is no skip button** *(2026-08-17)*. Declining used to be one, off to the side, which
	// made it the odd thing out on a screen otherwise made of cards. The vitae card is what
	// replaced it: taking money instead of altering the deck is a choice among three, not an exit.
	backButton *models.Button
	takeButton *models.Button

	// aimed is which offered card the worm is pointed at while the morph is shown, and before/after
	// are what it looks like on each side of the change. **Computed once, when the card is picked**
	// rather than every frame: `preview` runs the worm against a throwaway copy of the run, and
	// doing that in Draw would be a screen that alters the deck sixty times a second.
	aimed  int
	before combat.Card
	after  combat.Card

	// removes is a preview that has no "after" card, because the card is gone. Drawn as an empty
	// seat rather than as a second card.
	removes bool

	// held counts the settled stage down, and it does not start until the flight has landed.
	held int

	// taking is the Take button's request, consumed by Update — a button's OnClick reaches no
	// global state, and the flight it starts needs the layout.
	taking bool

	// arrival is the won card's journey to the middle, and arrivedFrom is the seat it set off
	// from — a prize's place in the row, or the morph's after-slot.
	arrival     travel
	arrivedFrom image.Rectangle

	// applyNow is the confirmed alteration, run against the real deck once the settled stage is
	// over. **The deck is not touched while the result is on screen**, which is what lets the
	// after-card be drawn from a preview rather than from a deck already altered — and it means
	// leaving the screen any other way changes nothing.
	applyNow func(*session.Session)

	// pendingWhat is what the trace line will say once the alteration lands.
	pendingWhat string

	// picksLeft is how many prizes this visit still owes the player, from `session.Picks` — the
	// `prizes-dealt` moment, which the Hungry ring is what moves off 1.
	//
	// **A second pick is another card out of the same row**, not a fresh row: the offer was dealt
	// once, and re-dealing it would make the second pick a different draw of the same fight. The card
	// offer *is* re-dealt, because a worm may have removed a card and every index after it has moved.
	picksLeft int
}

// Init deals both offers. **Re-entered on every visit**, because each fight earns its own.
func (s *PostBattleScene) Init(gs *state.GlobalState) {
	if s.backButton == nil {
		s.backButton = models.NewButton(offerButtonWidth, offerButtonHeight, "Back", s.back)
		s.backButton.BaseColor = color.RGBA{R: 120, G: 132, B: 150, A: 255}

		s.takeButton = models.NewButton(offerButtonWidth, offerButtonHeight, "Take it",
			func() { s.taking = true })
		s.takeButton.BaseColor = color.RGBA{R: 90, G: 170, B: 100, A: 255}
	}

	s.chosen, s.aimed = -1, -1
	s.stage = pickWorm
	s.removes, s.held = false, 0
	s.arrival, s.arrivedFrom, s.taking = travel{}, image.Rectangle{}, false
	s.pendingWhat, s.applyNow = "", nil
	s.prizes = dealPrizes(gs)
	s.offer = dealOffer(gs)
	s.picksLeft = gs.Run.Picks()

	s.place(gs)

	trace.Logf("postbattle", "after fight %d: prizes %v, %d cards of %d, %d to take",
		gs.Run.Fight(), prizeNames(s.prizes), len(s.offer), gs.Run.Size(), s.picksLeft)
}

func prizeNames(ps []prize) []string {
	out := make([]string, 0, len(ps))
	for _, p := range ps {
		out = append(out, p.name())
	}
	return out
}

// dealPrizes is the three cards: two worms drawn from the catalogue, then the vitae.
//
// **The vitae is last and is not drawn**, so its seat never moves. A player who has learned that
// the money is on the right can take it without reading, which is the point of a card that is
// always there.
func dealPrizes(gs *state.GlobalState) []prize {
	out := make([]prize, 0, wormsOffered+1)
	for _, w := range dealWorms(gs) {
		out = append(out, prize{worm: w})
	}
	return append(out, prize{vitae: true})
}

// dealWorms picks which alterations are offered: a shuffle of the catalogue, cut to two.
//
// **Its own stream** (`seeds.WormOffer`), separate from the cards. They are drawn from different
// lists and change on different schedules — adding a worm to the catalogue would otherwise reroll
// which *cards* every fight of every run offered.
//
// **Distinct by construction**, since it shuffles the catalogue rather than drawing twice: being
// offered the same worm as both options would be a choice that is not one.
func dealWorms(gs *state.GlobalState) []session.Worm {
	all := session.Worms()

	rng := rand.New(rand.NewSource(seeds.ForFight(gs.RunSeed, seeds.WormOffer, gs.Run.Fight())))
	rng.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })

	if len(all) > wormsOffered {
		all = all[:wormsOffered]
	}
	return all
}

// dealOffer picks which cards the worm may be applied to: a shuffle of every index in the run
// deck, cut to the hand size.
//
// **Its own stream** (`seeds.RewardHand`), and per fight. Sharing the player's shuffle would make
// the offer a function of how many cards were drawn in the fight just won, so playing a longer
// duel would change what winning it was worth.
//
// **The offer is the hand size**, which is `handSize` today and becomes a value on the duelist
// the day a brand can widen it. Both readers take the same number rather than each declaring one.
func dealOffer(gs *state.GlobalState) []int {
	if gs.Run == nil || gs.Run.Size() == 0 {
		return nil
	}

	idx := make([]int, gs.Run.Size())
	for i := range idx {
		idx[i] = i
	}

	rng := rand.New(rand.NewSource(seeds.ForFight(gs.RunSeed, seeds.RewardHand, gs.Run.Fight())))
	rng.Shuffle(len(idx), func(i, j int) { idx[i], idx[j] = idx[j], idx[i] })

	if len(idx) > handSize {
		idx = idx[:handSize]
	}

	// Sorted so the row reads as positions in the deck rather than as the order the shuffle
	// happened to produce. *Which* cards are offered is the random part; where each one sits is
	// not, and it costs nothing to make the row stable.
	sortInts(idx)
	return idx
}

// sortInts is an insertion sort over at most a hand's worth of indices. Written out rather than
// reaching for sort.Slice: this runs once per fight over eight items, and a comparator closure is
// more machinery than the job needs.
func sortInts(v []int) {
	for i := 1; i < len(v); i++ {
		for j := i; j > 0 && v[j] < v[j-1]; j-- {
			v[j], v[j-1] = v[j-1], v[j]
		}
	}
}

func (s *PostBattleScene) place(gs *state.GlobalState) {
	y := gs.PctY(offerButtonsPct)
	s.backButton.ScreenX, s.backButton.ScreenY = gs.PctX(50), y
	// The morph's two buttons are a pair, so Back moves in beside Take rather than staying where
	// it sits under the card row. Set at draw time in morphButtons.
}

// morphButtons puts Back and Take side by side, centred, for the stage where they are the only
// two controls. Skip is not offered there: the choice at that point is take it or step back, and a
// third exit reading "take nothing" beside a card you are looking at is a way to lose a reward by
// misreading a button.
func (s *PostBattleScene) morphButtons(gs *state.GlobalState) {
	step := offerButtonWidth + 24
	y := gs.PctY(offerButtonsPct)
	s.backButton.ScreenX, s.backButton.ScreenY = gs.PctX(50)-step/2, y
	s.takeButton.ScreenX, s.takeButton.ScreenY = gs.PctX(50)+step/2, y
}

func (s *PostBattleScene) Update(gs *state.GlobalState) error {
	// The settled stage is a held picture rather than a choice: the card that was won is on
	// screen, and when the hold runs out the screen leaves by itself.
	if s.stage == settled {
		if !s.arrival.done() {
			s.arrival.tick()
			return nil
		}
		s.held--
		if s.held <= 0 {
			if s.applyNow != nil {
				s.applyNow(gs.Run)
				trace.Logf("postbattle", "%s, deck now %d", s.pendingWhat, gs.Run.Size())
				s.applyNow = nil
			}
			if s.rearm(gs) {
				return nil
			}
			advanceRun(gs)
		}
		return nil
	}

	if s.taking {
		s.taking = false
		s.take(gs)
		return nil
	}

	s.click(gs)

	switch s.stage {
	case pickCard:
		s.place(gs)
		systems.UpdateButton(gs, s.backButton)
	case morph:
		s.morphButtons(gs)
		systems.UpdateButton(gs, s.takeButton)
		systems.UpdateButton(gs, s.backButton)
	}

	return nil
}

// click is the press on whichever row is live. **Only one row is clickable at a time**, which is
// what keeps the two stages honest: a card cannot be chosen before a worm names what would happen
// to it.
func (s *PostBattleScene) click(gs *state.GlobalState) {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}
	at := image.Pt(gs.MouseX, gs.MouseY)

	if s.stage != pickWorm && s.stage != pickCard {
		return
	}

	if s.stage == pickWorm {
		for i := range s.prizes {
			if at.In(s.wormSlot(gs, i)) {
				s.takePrize(gs, i)
				return
			}
		}
		return
	}

	for i := range s.offer {
		if at.In(s.offerSlot(gs, i)) {
			s.aimAt(gs, i)
			return
		}
	}
}

// takePrize is the click on the prize row.
//
// **The two kinds part company here and nowhere else.** A worm goes on to choose a card; the vitae
// has nothing to choose, so it flies straight to the middle and the screen settles on it. Both
// paths end the same way — a card in the centre — which is the whole reason the money is a card.
func (s *PostBattleScene) takePrize(gs *state.GlobalState, i int) {
	if i < 0 || i >= len(s.prizes) || s.prizes[i].taken {
		return
	}
	s.chosen = i

	if !s.prizes[i].vitae {
		s.stage = pickCard
		return
	}

	// **What the vitae card pays is the run's business, not this screen's** *(2026-08-17)*. The
	// `prizes-dealt` moment is where the Soul Taker ring turns 5 into 10, and it is a flat addition
	// rather than a scaling — see MECHANICS.md, where the two vitae rings are deliberately different
	// objects.
	paid := gs.Run.PrizeVitae(vitaeReward)
	s.applyNow = func(run *session.Session) { run.AddVitae(paid) }
	s.pendingWhat = fmt.Sprintf("took %d vitae", paid)
	s.settle(gs, s.wormSlot(gs, i))
}

// rearm is what a second pick is: the taken prize is struck off, the row stays where it is, and the
// screen goes back to the top. It reports whether another pick is owed.
//
// **The card offer is re-dealt and the prize row is not.** A worm may have removed a card, so every
// index the old offer held has moved — where the prizes are the same three cards they always were,
// one of them now spent. The seed is the same, which is deliberate: it is the same fight's offer,
// re-resolved against a deck that changed.
func (s *PostBattleScene) rearm(gs *state.GlobalState) bool {
	s.picksLeft--
	if s.picksLeft <= 0 {
		return false
	}

	if s.chosen >= 0 && s.chosen < len(s.prizes) {
		s.prizes[s.chosen].taken = true
	}

	s.chosen, s.aimed = -1, -1
	s.stage = pickWorm
	s.removes, s.held = false, 0
	s.arrival, s.arrivedFrom, s.taking = travel{}, image.Rectangle{}, false
	s.pendingWhat, s.applyNow = "", nil
	s.offer = dealOffer(gs)
	s.place(gs)

	trace.Logf("postbattle", "another pick: %d left, %d cards", s.picksLeft, gs.Run.Size())
	return true
}

// settle starts the last stage: the won card flies from wherever it was to the middle, is held
// there long enough to read, and then the screen leaves.
//
// **The flight is what the stage is**, not decoration on top of it: the hold does not begin until
// the card lands, so a slower flight is a longer look rather than a card arriving late to a
// countdown already running.
func (s *PostBattleScene) settle(gs *state.GlobalState, from image.Rectangle) {
	s.stage, s.held = settled, morphHoldTicks
	s.arrival = newTravel(0, settleFlightTicks)
	s.arrivedFrom = from
}

// aimAt points the chosen worm at one offered card and works out what it would become.
//
// **The preview runs the real worm against a throwaway copy of the run**, rather than a second
// implementation of what each target does. A preview computed by its own arithmetic is a preview
// that can disagree with the thing it is previewing, which is the one failure this screen exists
// to prevent.
func (s *PostBattleScene) aimAt(gs *state.GlobalState, slot int) {
	worm, ok := s.chosenWorm()
	if !ok || slot < 0 || slot >= len(s.offer) {
		return
	}
	deckIndex := s.offer[slot]

	before, ok := gs.Run.Card(deckIndex)
	if !ok || !gs.Run.CanApply(worm, deckIndex) {
		// **A worm that would change nothing is refused rather than shown** — a Smash cannot be
		// promoted and a plan card has no ladder — so the click does nothing and the card stays
		// pickable. Saying no here is why CanApply exists.
		return
	}

	trial := session.New(gs.Run.Deck())
	if !trial.Apply(worm, deckIndex) {
		return
	}

	s.aimed, s.before, s.stage = slot, before, morph
	s.removes = worm.Target == session.TargetRemove

	if worm.Target == session.TargetDuplicate {
		// The copy is appended, so the card that arrived is the last one — and it is the *new*
		// card that is the reward, even though it is identical to the one that was picked.
		s.after, _ = trial.Card(trial.Size() - 1)
		return
	}
	if !s.removes {
		s.after, _ = trial.Card(deckIndex)
	}
}

// wormSlot is where one offered worm is drawn, and the rectangle it is clicked in.
//
// **One function for both**, the same rule the hand follows: a card hit-tested against a
// rectangle it is not drawn in is exactly the bug this shape prevents.
//
// In the second stage the chosen worm stays on screen, alone and centred, so what is about to
// happen is still stated while the card is picked.
func (s *PostBattleScene) wormSlot(gs *state.GlobalState, i int) image.Rectangle {
	top := gs.PctY(wormRowPct)

	if s.stage == pickCard {
		left := gs.PctX(50) - cardWidth/2
		return image.Rect(left, top, left+cardWidth, top+cardHeight)
	}

	n := len(s.prizes)
	pitch := cardWidth + 40
	width := (n-1)*pitch + cardWidth
	left := gs.PctX(50) - width/2
	return image.Rect(left+i*pitch, top, left+i*pitch+cardWidth, top+cardHeight)
}

// offerSlot is where one offered card is drawn, and the rectangle it is clicked in.
func (s *PostBattleScene) offerSlot(gs *state.GlobalState, i int) image.Rectangle {
	n := len(s.offer)
	pitch := handPitch(gs, n)
	width := (n-1)*pitch + cardWidth
	left := gs.PctX(50) - width/2
	top := gs.PctY(offerRowPct)
	return image.Rect(left+i*pitch, top, left+i*pitch+cardWidth, top+cardHeight)
}

// take commits the previewed alteration. **It does not leave the screen** — the result is held on
// the settled stage first, because the card that was won disappearing straight into a deck of
// forty-eight is a reward the player never sees.
func (s *PostBattleScene) take(gs *state.GlobalState) {
	worm, ok := s.chosenWorm()
	if !ok || s.aimed < 0 || s.aimed >= len(s.offer) {
		return
	}
	deckIndex := s.offer[s.aimed]

	s.pendingWhat = fmt.Sprintf("%s on card %d", worm.Record, deckIndex)
	s.applyNow = func(run *session.Session) { run.Apply(worm, deckIndex) }

	// It sets off from the after-slot it has been sitting in, so the card the player has been
	// looking at is the card that travels.
	_, afterAt := morphSlots(gs)
	s.settle(gs, afterAt)
}

// back steps one stage out: the morph returns to the cards, and the cards return to the worms.
// **Nothing is committed until the take**, so a player who picked up the wrong worm or aimed it at
// the wrong card is never stuck with either.
func (s *PostBattleScene) back() {
	if s.stage == morph {
		s.aimed, s.stage = -1, pickCard
		return
	}
	s.chosen, s.stage = -1, pickWorm
}

func (s *PostBattleScene) chosenPrize() (prize, bool) {
	if s.chosen < 0 || s.chosen >= len(s.prizes) {
		return prize{}, false
	}
	return s.prizes[s.chosen], true
}

// chosenWorm is the chosen prize when it is a worm. The vitae card reaches none of the code that
// asks this, because taking it goes straight to the flight.
func (s *PostBattleScene) chosenWorm() (session.Worm, bool) {
	p, ok := s.chosenPrize()
	if !ok || p.vitae {
		return session.Worm{}, false
	}
	return p.worm, true
}

func (s *PostBattleScene) Draw(gs *state.GlobalState, screen *ebiten.Image) {
	screen.Fill(screenGround)

	heading := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 34}
	small := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 18}

	line := func(y int, face *text.GoTextFace, msg string) {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(gs.PctX(50)), float64(y))
		op.PrimaryAlign = text.AlignCenter
		op.ColorScale.ScaleWithColor(groundInk)
		text.Draw(screen, msg, face, op)
	}

	line(offerTitleTop, heading, s.title())
	line(offerHintTop, small, s.hint(gs))

	switch s.stage {
	case morph:
		s.drawMorph(gs, screen, small)
		systems.DrawButton(gs, screen, s.backButton)
		systems.DrawButton(gs, screen, s.takeButton)
		return

	case settled:
		s.drawSettled(gs, screen)
		return
	}

	s.drawWorms(gs, screen)

	if s.stage == pickCard {
		for i, deckIndex := range s.offer {
			card, ok := gs.Run.Card(deckIndex)
			if !ok {
				continue
			}
			// **A card the worm cannot change is dimmed rather than hidden**, so the row still
			// says what was offered and the reason one of them is unavailable is visible instead
			// of a click that silently does nothing.
			worm, _ := s.chosenWorm()
			usable := gs.Run.CanApply(worm, deckIndex)
			drawCard(gs, screen, s.offerSlot(gs, i).Min, cards.Hand, card, deckCardCost(gs, card), usable, false)
		}
		systems.DrawButton(gs, screen, s.backButton)
	}
}

// morphSlots is where the before and after cards sit: side by side, centred, with room between
// them for the arrow. **The same footprint as a hand card**, so the thing being changed is the
// size it was when it was picked.
func morphSlots(gs *state.GlobalState) (before, after image.Rectangle) {
	const gap = 120
	top := gs.PctY(36)
	left := gs.PctX(50) - cardWidth - gap/2
	right := gs.PctX(50) + gap/2

	return image.Rect(left, top, left+cardWidth, top+cardHeight),
		image.Rect(right, top, right+cardWidth, top+cardHeight)
}

// drawMorph is the preview: what the card is, what it becomes, and nothing committed.
//
// **A removal has no after card**, so the right-hand seat is left as an outline. Drawing the same
// card twice would say the worm did nothing, and drawing nothing at all would leave the row
// looking half-finished rather than deliberately empty.
func (s *PostBattleScene) drawMorph(gs *state.GlobalState, screen *ebiten.Image,
	face *text.GoTextFace) {

	beforeAt, afterAt := morphSlots(gs)

	drawCard(gs, screen, beforeAt.Min, cards.Hand, s.before, deckCardCost(gs, s.before), true, false)

	arrow := &text.DrawOptions{}
	arrow.GeoM.Translate(float64(gs.PctX(50)), float64(beforeAt.Min.Y+cardHeight/2))
	arrow.PrimaryAlign = text.AlignCenter
	arrow.SecondaryAlign = text.AlignCenter
	arrow.ColorScale.ScaleWithColor(groundInk)
	text.Draw(screen, gone, &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 40}, arrow)

	if s.removes {
		drawEmptySeat(screen, afterAt)
		return
	}
	drawCard(gs, screen, afterAt.Min, cards.Hand, s.after, deckCardCost(gs, s.after), true, true)
}

// gone is the mark between the two cards. A hyphen and a chevron rather than an em dash: the
// kubasta font has no U+2014 and draws a missing-glyph box for it.
const gone = "->"

// settledSeat is where the won card comes to rest.
func settledSeat(gs *state.GlobalState) image.Rectangle {
	top := gs.PctY(36)
	return image.Rect(gs.PctX(50)-cardWidth/2, top, gs.PctX(50)+cardWidth/2, top+cardHeight)
}

// drawSettled is the card the player won, **flying to the middle and then held there**.
//
// It travels from the seat it was already in — a prize's place in the row, or the morph's
// after-slot — so the card the player has been looking at is the card that moves. A card that
// simply appeared in the centre would be a card that was never anywhere else, which is the
// opposite of what a prize is.
//
// **A removal has nothing to fly.** What was won is an absence, so the empty seat is drawn where
// the card would have landed and nothing crosses the screen.
func (s *PostBattleScene) drawSettled(gs *state.GlobalState, screen *ebiten.Image) {
	seat := settledSeat(gs)

	if s.removes {
		drawEmptySeat(screen, seat)
		return
	}

	at := flyingTo(s.arrivedFrom, seat, s.arrival)

	card := s.after
	if p, ok := s.chosenPrize(); ok && p.vitae {
		drawSpecCard(gs, screen, at, vitaeSpec(gs.Run.PrizeVitae(vitaeReward), true))
		return
	}
	drawCard(gs, screen, at, cards.Hand, card, deckCardCost(gs, card), true, false)
}

func (s *PostBattleScene) title() string {
	switch s.stage {
	case morph:
		return "The worm turns"
	case settled:
		if p, ok := s.chosenPrize(); ok && p.vitae {
			return "Taken"
		}
		if s.removes {
			return "Eaten"
		}
		return "Changed"
	default:
		return "A worm turns up"
	}
}

// drawPrizeCard draws whichever kind of prize this is. One function so a caller never has to know,
// which is what keeps the vitae from being a special case anywhere but here.
func drawPrizeCard(gs *state.GlobalState, screen *ebiten.Image, at image.Point,
	p prize, enabled bool) {

	if p.vitae {
		drawSpecCard(gs, screen, at, vitaeSpec(prizeVitae(gs), enabled))
		return
	}
	drawWormCard(gs, screen, at, p.worm, enabled)
}

// prizeVitae is what the money card is worth to this run — the base plus whatever the rings add.
// **The card says the true figure**, because a card offering 5 that pays 10 would make the ring
// invisible at exactly the moment it is doing its work.
func prizeVitae(gs *state.GlobalState) int {
	if gs.Run == nil {
		return vitaeReward
	}
	return gs.Run.PrizeVitae(vitaeReward)
}

// drawWorms puts the offer up as cards. **A worm is a card because it is a thing you are given**,
// and the game already has one visual language for that — a name, a border colour and a line
// saying what it does.
//
// It borrows `cards.Hand` rather than taking a style of its own: a worm has no cost and no form,
// which that style draws as nothing at all, so what is left is exactly the name and the text. A
// dedicated style is what this wants once a worm has art.
func (s *PostBattleScene) drawWorms(gs *state.GlobalState, screen *ebiten.Image) {
	if s.stage == pickCard {
		if p, ok := s.chosenPrize(); ok {
			drawPrizeCard(gs, screen, s.wormSlot(gs, s.chosen).Min, p, true)
		}
		return
	}
	for i, p := range s.prizes {
		drawPrizeCard(gs, screen, s.wormSlot(gs, i).Min, p, !p.taken)
	}
}

func (s *PostBattleScene) hint(gs *state.GlobalState) string {
	switch s.stage {
	case morph:
		w, _ := s.chosenWorm()
		if s.removes {
			return w.Name + " eats this card"
		}
		return w.Name + " makes it this"
	case settled:
		if p, ok := s.chosenPrize(); ok && p.vitae {
			return "spend it in the shop"
		}
		if s.removes {
			return "one fewer card to draw"
		}
		return "into the deck it goes"
	case pickCard:
		w, _ := s.chosenWorm()
		return fmt.Sprintf("%s - pick the card it takes", w.Name)
	default:
		if s.picksLeft > 1 {
			return fmt.Sprintf("take %d - %d cards in your deck, %d vitae in hand",
				s.picksLeft, gs.Run.Size(), gs.Run.Vitae())
		}
		return fmt.Sprintf("take one - %d cards in your deck, %d vitae in hand",
			gs.Run.Size(), gs.Run.Vitae())
	}
}
