package screens

// The post-battle screen: **pick a worm, then pick the card it eats.**
//
// It is the first of the between-fight scenes — a shop and a room choice come after it — and all
// of them are ordinary scenes in the registry rather than modes of the combat screen. That is
// what keeps `CombatScene` from growing a fourth phase, and it is why the chain is wired as
// screen changes: win → post-battle → combat.
//
// **It opens by reading the win out** *(2026-08-22)*. Before anything is offered, the screen types
// what the fight paid — interest, a tenth of the life you kept, what the room is worth — and each
// figure flies to the duelist card as its sentence lands. The build is on screen the whole time: the
// player's card in the corner and their relics beside it, so a worm is chosen against the thing it
// would be changing. See postbattle_prose.go and buildband.go.
//
// **Two stages after that, worm first** *(2026-08-17)*. Two worms are drawn from the catalogue and offered
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
	"github.com/curiousjc/ascend-duel/internal/achieve"
	"image"
	"math/rand"
	"sort"

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

// wormsOffered is how many alterations a win puts up.
//
// **Two worms and nothing else** *(owner's call, 2026-08-22)*. A third card paying vitae used to
// stand beside them, so that declining a change to the deck was a choice among three with a price.
// The win pays vitae by itself now — read out at the top of this screen — so the money card was
// charging for something the player had already been given, and the offer is the two creatures the
// prose says are fleeing. **Taking neither is a button again**, deliberately: the offer is free, so
// walking away costs nothing and does not need to look like a card.
const wormsOffered = 2

// prize is one of the cards on the table. **Every one of them is a worm** since the vitae card went
// — the struct survives because a prize is a worm *plus what this visit has done with it*, which
// the catalogue record has no business carrying.
type prize struct {
	// taken is set once this prize has been picked. **It stays in the row rather than being removed
	// from it** — a card leaving would move the one beside it. Only a relic that adds a pick can
	// produce a row with a taken card still in it.
	taken bool
	worm  session.Worm
}

func (p prize) name() string { return p.worm.Name }

// Where the two rows sit and where the controls sit under them. Percentages anchor the groups;
// offsets inside a group stay in pixels, per CLAUDE.md.
const (
	// **The prose has done its job once the offer is up, and the two rows take the screen** — the
	// worms where the eye lands, the cards they may eat below them.
	//
	// **The worms used to sit at 58% under the payout** *(owner's call, 2026-08-22)*, on the
	// argument that what a win paid and what it is offering are one picture. That could not
	// survive both rows being on screen at once: a worm row at 58% is 280 tall against a button
	// strip at 88%, so there is nowhere for the cards to go. **The narration therefore clears when
	// the offer arrives rather than one stage later**, which is the cost of the reversed gesture
	// and was taken deliberately.
	wormChosenRowPct = 34

	// **64 rather than 62 since 2026-09-06**, to buy the headroom a selected card lifts into. See
	// offerSelectedNudge: the row is a card tall and rises by 26 when one is picked, and at 62 the
	// lifted card's top edge landed inside the worm row above it.
	offerRowPct = 64

	// The title, the hint and the narration all hang off the bottom of the build band, each by its
	// own drop. **They were absolute pixels until 2026-09-05** — 262, 300 and 296, written when the
	// band ended around y=253 — and the band has moved twice since: the screen went to 1920x1080
	// and the cards grew a quarter with it. The band now ends at 311, so all three were being drawn
	// *inside* it, which is the worn relic struck through the first line of the payout. These are
	// the same three gaps that arithmetic produced, measured from the band rather than from the top
	// of the screen, so the next time either moves the text follows.
	offerTitleDrop = 9
	offerProseDrop = 43
	offerHintDrop  = 47

	proseLineGap = 42

	offerButtonsPct   = 88
	offerButtonWidth  = 400
	offerButtonHeight = 76

	// wormRowGap is the air on each side of the skip button, which stands **in** the worm row
	// rather than under it *(owner's call, 2026-09-06)*. It was centred at offerButtonsPct — 88%,
	// the seat the shop's Leave button takes — and with both rows of cards on screen the offer row
	// reached y=949 against a button centred at 950, so the cards were drawn straight over it.
	//
	// **Between the two worms rather than beside them**, which is what makes it read as one row:
	// taking neither is the third answer to the question the two cards are asking, so it stands
	// where a third card would.
	wormRowGap = 40

	// offerSelectedNudge is how far a picked card lifts out of the offer row.
	//
	// **The hand's own figure** — see selectedNudge in combat_actionbox.go — because this is the
	// same gesture on the same kind of row, and a selection that rose by a different amount on the
	// two screens would be two gestures wearing one name.
	offerSelectedNudge = selectedNudge
)

// stage is how far through the choice the player has got.
type stage int

const (
	// narrate: the win is being read out and nothing is offered yet. **Every visit starts here**,
	// because the payout is the first thing that happened and the worms are what it leads to.
	narrate stage = iota

	// choosing: both rows are up — the worms, and the cards they may eat.
	//
	// **It was two stages until 2026-09-06** *(owner's call)*, worm first and then the card. The
	// order reversed with the parasite's: **select the card, then click the worm**, so a consumable
	// is pointed at a card the same way everywhere in the game. One gesture in one order meant one
	// stage — the two rows are on screen together, because the player is choosing between them
	// rather than passing through them.
	choosing

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

	// settledHoldTicks is how long the finished card is held before the screen leaves. **Long
	// enough to read, short enough not to need a button** — the click that picked the card is the
	// last input the player has to make.
	settledHoldTicks = beat(4, 1)
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

	// prose is the payout, typed out a sentence at a time, and the figures it flies to the duelist
	// card. See postbattle_prose.go.
	prose typewriter

	// entry is each offered worm's flight in from the side of the screen — one per prize, indexed
	// alike. **Cards fly; they never appear**, and a worm arriving from off-screen is the picture
	// the prose has just described: two creatures fleeing the enemy you beat.
	entry []travel

	// relicDrag is the press in progress over the worn relic row in the build band. **The row is
	// reorderable here like everywhere else** — worn order is a rule, and between fights is when a
	// player is thinking about their build.
	relicDrag cardDrag

	// **Skipping is a button again** *(2026-08-22)*, after the vitae card that replaced it was
	// removed. It takes neither worm and pays nothing extra — the win has already paid — so it is
	// an exit rather than a third choice, which is exactly why it is not a card.
	skipButton *models.Button

	// tut is Bob, when a run is being taught. See tutorial.go, and combat.go for the same field.
	tut tutorialOverlay

	// selected is which offered card is picked out, or -1. **A worm takes exactly one target**, so
	// this is one index rather than a set — see consumableTarget, which is what asks whether it is
	// enough for the worm being clicked.
	//
	// **It is the offer row's counterpart of the hand's `selected` flag**, and it is deliberately a
	// different shape: the hand's selection is also the round's queue and may hold five, where this
	// selects the one card a worm is about to eat.
	selected int

	// aimed is which offered card the worm was pointed at, and after is what it became.
	// **Computed once, when the card is picked** rather than every frame: the worm is run against a
	// throwaway copy of the run, and doing that in Draw would be a screen that alters the deck
	// sixty times a second.
	aimed int
	after combat.Card

	// before is the card as it was when the player clicked it, and it is what flies to the middle.
	//
	// **The card that moves is the card the player was looking at** *(owner's call, 2026-09-08)*.
	// It used to be `after` that flew, which meant the alteration had already happened by the time
	// anything moved: the card left the row as one thing and arrived as another, with the change
	// itself never on screen. The change is now its own beat once the flight has landed — see
	// `change`, and cardmorph.go for the machinery.
	before combat.Card

	// change is the alteration happening: the old face coming apart and the new one coming through
	// it. Which of the three shapes it takes is decided in aimAt, by what the worm did.
	change morph

	// removes says the alteration has no "after" card, because the card is gone. What is left when
	// the dissolve finishes is an empty seat.
	removes bool

	// copied says the alteration added a card rather than changing one, so two cards are on screen
	// at the end: the original, untouched, and the copy arriving out of nothing beside it.
	copied bool

	// held counts the settled stage down, and it does not start until the flight has landed.
	held int

	// skipping is the Skip button's request, consumed by Update — a button's OnClick reaches no
	// global state, and leaving the screen needs it.
	skipping bool

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

	// tip explains whichever card the cursor is on: what a worm will do and how long it lasts, or
	// what one of the offered deck cards is worth.
	tip models.Tooltip

	// How the offer row is arranged, and the block of tabs that chooses it — the same widget the
	// combat screen's hand carries *(owner's call, 2026-09-05)*. **Eight overlapping cards are
	// eight overlapping cards wherever they are dealt**, so the row a worm is pointed at is read
	// the same way a hand is.
	//
	// **sortMode is the working copy of `gs.HandSort`**, exactly as CombatScene's is: the button
	// callback moves this, and Update writes it back. So a player who arranges by element in a
	// duel meets an offer already arranged by element.
	sortMode handSort
	sortTabs *sortTabs

	// slides is the offer row rearranging itself, on the shared mover — see cardslide.go. **The
	// same widget behaves the same way on both screens** *(owner's call, 2026-09-05)*: a card that
	// changes where it is on screen travels there, and a sort that re-laid this row out instantly
	// while sliding the hand would be two controls wearing one set of labels.
	slides []cardSlide

	// picksLeft is how many prizes this visit still owes the player, from `session.Picks` — the
	// `prizes-dealt` moment, which the Hungry relic is what moves off 1.
	//
	// **A second pick is another card out of the same row**, not a fresh row: the offer was dealt
	// once, and re-dealing it would make the second pick a different draw of the same fight. The card
	// offer *is* re-dealt, because a worm may have removed a card and every index after it has moved.
	picksLeft int
}

// Init deals both offers. **Re-entered on every visit**, because each fight earns its own.
func (s *PostBattleScene) Init(gs *state.GlobalState) {
	if s.skipButton == nil {
		s.skipButton = models.NewButton(offerButtonWidth, offerButtonHeight, "LET THEM ESCAPE",
			func() { s.skipping = true })
		s.skipButton.BaseColor = color.RGBA{R: 120, G: 132, B: 150, A: 255}
	}

	s.chosen, s.aimed, s.selected = -1, -1, -1
	s.stage = narrate
	s.removes, s.copied, s.held = false, false, 0
	s.change = morph{}
	s.arrival, s.arrivedFrom = travel{}, image.Rectangle{}
	s.pendingWhat, s.applyNow = "", nil
	s.prizes = dealPrizes(gs)
	s.entry = make([]travel, len(s.prizes))
	s.prose.setLines(payoutLines(gs))
	s.skipping = false
	s.offer = dealOffer(gs)
	s.picksLeft = gs.Run.Picks()
	s.tip = models.Tooltip{DwellTicks: tipDwell}

	s.sortMode = handSortOf(gs)
	if s.sortTabs == nil {
		s.sortTabs = newSortTabs(s.sortTabRect, s.setSort)
	}
	s.slides = nil
	s.sortOffer(gs)

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

// dealPrizes is the offer: two worms drawn from the catalogue.
func dealPrizes(gs *state.GlobalState) []prize {
	out := make([]prize, 0, wormsOffered)
	for _, w := range dealWorms(gs) {
		out = append(out, prize{worm: w})
	}
	return out
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

// place puts the skip button in its seat in the worm row. **Read off wormRowSeats**, so the button
// and the cards beside it cannot disagree about where the row is.
func (s *PostBattleScene) place(gs *state.GlobalState) {
	_, button := wormRowSeats(gs, len(s.prizes))
	s.skipButton.ScreenX = button.Min.X + button.Dx()/2
	s.skipButton.ScreenY = button.Min.Y + button.Dy()/2
}

func (s *PostBattleScene) Update(gs *state.GlobalState) error {
	// **Before this screen's own input**, for the reason combat.go runs it first: the gate it
	// sets is what every widget below reads.
	s.tut.update(gs, s)

	// **The narration is the whole screen while it runs.** Nothing is clickable but the click that
	// reads it, which is what keeps a payout from being half-read while a worm is already being
	// chosen.
	//
	// **Two clicks, not one** *(owner's call, 2026-09-08)*: the first fills the block, the second
	// hands the screen to the worms. One gesture doing both meant the only way to see the whole
	// payout at once was also the way past it.
	//
	// **The click is not gated** — deliberately, and it is the one place on this screen that
	// ignores `CursorAllowed`. The tutorial's step here anchors the duelist card, so a gated
	// narration is one a taught player has no way to hurry, and hurrying it changes nothing: the
	// claims are the same either way, per the rule at the top of postbattle_prose.go. **Bob's own
	// bubble is the one exception**, since a press on NEXT landing on the screen underneath it
	// would answer two questions with one click.
	if s.stage == narrate {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && !s.tut.coversCursor(gs) {
			if s.prose.filled() {
				s.prose.release()
			} else {
				s.prose.skip(gs)
			}
		}
		s.prose.tick(gs, func(i int) image.Point { return proseLineAt(gs, i) })
		if s.prose.finished() {
			s.beginOffer(gs)
		}
		return nil
	}

	// **The relic row is live from the moment the narration ends**, under the panels rather than
	// over them: a drag started behind an open deck panel would be a card moving where the player
	// cannot see it. It runs before the stage branches below, because the settled stage returns
	// early and the row is still on screen through it.
	s.updateRelicRow(gs)

	// The settled stage is a held picture rather than a choice: the card that was won is on
	// screen, and when the hold runs out the screen leaves by itself.
	if s.stage == settled {
		if !s.arrival.done() {
			s.arrival.tick()
			return nil
		}
		// **The change is its own beat, and it does not start until the card has landed** — the
		// same rule the hold below follows, and for the same reason: a dissolve running over a
		// moving card would put the one thing worth watching on a target the eye is still chasing.
		if !s.change.done() {
			s.change.tick()
			return nil
		}
		s.held--
		if s.held <= 0 {
			if s.applyNow != nil {
				s.applyNow(gs.Run)
				trace.Logf("postbattle", "%s, deck now %d", s.pendingWhat, gs.Run.Size())
				s.applyNow = nil

				// **The alteration is a moment, raised where the deck actually changes.** A worm
				// that removed a card leaves nothing behind, so there is nothing to name and the
				// moment is not raised — see achieve.MomentCardAltered, which carries the resulting
				// card's label rather than the worm's, because several worms can arrive at one card.
				if !s.removes {
					earnMoment(gs, achieve.CardAltered(s.after.Label()))
				}
			}
			if s.rearm(gs) {
				return nil
			}
			advanceRun(gs)
		}
		return nil
	}

	if s.skipping {
		s.skipping = false
		trace.Logf("postbattle", "caught neither worm")
		advanceRun(gs)
		return nil
	}

	// The worms' arrival. **They are clickable while they fly** — a card is where its layout
	// function says it is, and the flight is a ghost over that seat, the same rule the combat
	// screen's hand follows.
	for i := range s.entry {
		s.entry[i].tick()
	}

	s.click(gs)

	switch s.stage {
	case choosing:
		s.place(gs)
		systems.UpdateButton(gs, s.skipButton)

		// **Placed every tick rather than at Init**, unlike every other widget on this screen: the
		// block hangs off the offer row's right edge, and a second pick re-deals that row against a
		// deck a worm may have shortened. A block placed once would then stand beside a row that
		// had moved out from under it.
		s.sortTabs.place(gs)
		s.sortTabs.update(gs, true)
		setHandSort(gs, s.sortMode)
		s.sortOffer(gs)
		s.slides = advance(s.slides)
	}

	s.hover(gs)
	systems.UpdateTooltip(gs, &s.tip)
	return nil
}

// hover points the tooltip at whichever card the cursor is on, in whichever row is live.
//
// **Only the row that can be clicked is explained**, which is the same rule the click itself
// follows: a tooltip on a row that is not the current stage's would be describing a choice the
// player cannot make yet.
func (s *PostBattleScene) hover(gs *state.GlobalState) {
	at := image.Pt(gs.MouseX, gs.MouseY)

	// A gated step takes the tooltips with the clicks; see the combat screen's hover.
	if !gs.CursorAllowed() {
		return
	}

	// **The band is live at every stage, so its relics are explained at every stage.** They are not
	// a choice this screen offers — which is exactly why the rule above does not cover them: a
	// worn relic is what the choice is being *judged against*, and "what does the one I am wearing
	// actually do" is the question a worm is picked by.
	if hoverBuildRelics(gs, at, &s.tip) {
		return
	}

	switch s.stage {
	case choosing:
		// **The prizes are deliberately not tooltipped**, as they were not before the two stages
		// merged: a worm's whole rule is printed on its face, where a deck card's is not. What a
		// dim prize means — "not for the card you have selected" — is the one thing that is new
		// here and is left to the row rather than to a tooltip.
		for i, deckIndex := range s.offer {
			seat := s.offerSlot(gs, i)
			card, ok := gs.Run.Card(deckIndex)
			if !ok || !at.In(seat) {
				continue
			}
			title, lines := cardTip(card, heldByRun(gs, card))
			s.tip.Point(seat, tipLine(title), tipLines(lines))
			return
		}
	}
}

// click is the press on whichever row is live. **Only one row is clickable at a time**, which is
// what keeps the two stages honest: a card cannot be chosen before a worm names what would happen
// to it.
func (s *PostBattleScene) click(gs *state.GlobalState) {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || !gs.CursorAllowed() {
		return
	}
	at := image.Pt(gs.MouseX, gs.MouseY)

	if s.stage != choosing {
		return
	}

	// **The card row is asked first**, because it is the row a click is most often meant for and
	// the two do not overlap. Selecting is free and reversible; clicking a worm spends the pick.
	for i := range s.offer {
		if at.In(s.offerSlot(gs, i)) {
			s.selectOffered(i)
			return
		}
	}

	for i := range s.prizes {
		if at.In(s.wormSlot(gs, i)) {
			s.takePrize(gs, i)
			return
		}
	}
}

// selectOffered picks a card out of the offer row, or puts it back.
//
// **Clicking the selected card deselects it**, which is the hand row's own gesture — a card clicked
// into the queue is clicked out of it — so the one thing a player already knows how to undo works
// here too.
//
// **One card at a time.** A worm eats exactly one, so a second click moves the selection rather than
// adding to it; there is no set for it to be wrong about.
func (s *PostBattleScene) selectOffered(i int) {
	if s.selected == i {
		s.selected = -1
		return
	}
	s.selected = i
	s.tip.Forget()
}

// selectedDeckIndex is the offer's current pick as an index into the run deck, and whether there is
// one.
func (s *PostBattleScene) selectedDeckIndex() (int, bool) {
	if s.selected < 0 || s.selected >= len(s.offer) {
		return 0, false
	}
	return s.offer[s.selected], true
}

// wormSpendable is whether clicking this prize now would take it: a card is selected, and this worm
// can actually change that card.
//
// **It is the same question the click asks and the same one the card's lit state reads**, which is
// what stops a prize looking available and doing nothing. See consumableTarget.
func (s *PostBattleScene) wormSpendable(gs *state.GlobalState, p prize) bool {
	if p.taken {
		return false
	}
	target := consumableTarget{
		needs: 1,
		legal: func(ids []int) bool { return gs.Run.CanApply(p.worm, ids[0]) },
	}
	if idx, ok := s.selectedDeckIndex(); ok {
		return target.satisfiedBy([]int{idx})
	}
	return target.satisfiedBy(nil)
}

// takePrize is the click on the prize row: this worm is spent on the card that is selected.
//
// **It refuses rather than falling back**, on the predicate the card's lit state already read — a
// prize drawn dim cannot be taken, and a prize drawn lit always works.
func (s *PostBattleScene) takePrize(gs *state.GlobalState, i int) {
	if i < 0 || i >= len(s.prizes) || !s.wormSpendable(gs, s.prizes[i]) {
		return
	}
	slot, ok := s.selected, true
	if _, ok = s.selectedDeckIndex(); !ok {
		return
	}
	s.chosen = i
	s.tip.Forget()
	s.aimAt(gs, slot)
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

	s.chosen, s.aimed, s.selected = -1, -1, -1
	s.stage = choosing
	s.removes, s.copied, s.held = false, false, 0
	s.change = morph{}
	s.arrival, s.arrivedFrom = travel{}, image.Rectangle{}
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
	s.stage, s.held = settled, settledHoldTicks
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
		// promoted and a defend card has no ladder — so the click does nothing and the card stays
		// pickable. Saying no here is why CanApply exists.
		return
	}

	trial := session.New(gs.Run.Deck())
	if !trial.Apply(worm, deckIndex) {
		return
	}

	s.aimed = slot
	s.before = before
	s.removes = worm.Target == session.TargetRemove
	s.copied = worm.Target == session.TargetDuplicate

	switch {
	case s.copied:
		// The copy is appended, so the card that arrived is the last one — and it is the *new*
		// card that is the reward, even though it is identical to the one that was picked.
		s.after, _ = trial.Card(trial.Size() - 1)
	case !s.removes:
		s.after, _ = trial.Card(deckIndex)
	}

	// **What the worm did decides which shape the change takes**, and the three cases are the
	// three things a worm can be: it recoloured the card, it ate it, or it made a second one. See
	// cardmorph.go — the morph is handed two finished faces and works out the rest.
	beforeSpec := cardSpec(before, heldByRun(gs, before), true, false)
	switch {
	case s.removes:
		s.change = morphAway(beforeSpec, cards.Hand)
	case s.copied:
		// **The original is not changed, so it does not morph.** It stands where it landed and the
		// copy arrives out of nothing beside it, which is the only honest picture of a duplicate:
		// there is no old face to come apart.
		s.change = morphIn(cardSpec(s.after, heldByRun(gs, s.after), true, false), cards.Hand)
	default:
		s.change = morphInto(beforeSpec,
			cardSpec(s.after, heldByRun(gs, s.after), true, false), cards.Hand)
	}

	// **The click is the commitment** *(owner's call, 2026-09-05)*. It was a preview with Take and
	// Back under it; picking the card is now the whole decision, and what follows is the result
	// being shown rather than a question about it. The deck is still not touched until the settled
	// stage is over — see applyNow — so the card on screen is drawn from the trial run above.
	s.pendingWhat = fmt.Sprintf("%s on card %d", worm.Record, deckIndex)
	s.applyNow = func(run *session.Session) { run.Apply(worm, deckIndex) }
	s.tip.Forget()
	s.settle(gs, s.offerSlot(gs, slot))
}

// wormSlot is where one offered worm is drawn, and the rectangle it is clicked in.
//
// **One function for both**, the same rule the hand follows: a card hit-tested against a
// rectangle it is not drawn in is exactly the bug this shape prevents.
//
// In the second stage the chosen worm stays on screen, alone and centred, so what is about to
// happen is still stated while the card is picked.
func (s *PostBattleScene) wormSlot(gs *state.GlobalState, i int) image.Rectangle {
	// **The row sits where the chosen worm used to** *(2026-09-06)*. Both rows are up at once now,
	// so the worms take the seat the prose vacates and the cards they may eat go below them —
	// which is the layout the second stage already had, with every prize in it rather than one.
	seats, _ := wormRowSeats(gs, len(s.prizes))
	if i < 0 || i >= len(seats) {
		return image.Rectangle{}
	}
	return seats[i]
}

// wormRowSeats lays the whole worm row out: a seat per prize, and the skip button standing between
// them.
//
// **One function for the cards and the button**, which is the rule every row in this game follows —
// a control placed by its own arithmetic beside cards placed by theirs is how the two come to
// overlap, which is exactly what happened when the button was left at 88%.
//
// **The button takes the middle slot.** With the usual two prizes that is literally between them;
// with an odd number it sits left of centre, which is arbitrary and harmless — nothing deals an odd
// number today, and a row that refused to lay one out would be worse than one that leans.
func wormRowSeats(gs *state.GlobalState, n int) (prizes []image.Rectangle, button image.Rectangle) {
	top := gs.PctY(wormChosenRowPct)

	// The row is n cards and one button, with a gap between every pair.
	width := n*cardWidth + offerButtonWidth + (n)*wormRowGap
	x := gs.PctX(50) - width/2

	mid := n / 2
	prizes = make([]image.Rectangle, 0, n)
	for i := 0; i < n; i++ {
		if i == mid {
			button = image.Rect(x, top+(cardHeight-offerButtonHeight)/2,
				x+offerButtonWidth, top+(cardHeight+offerButtonHeight)/2)
			x += offerButtonWidth + wormRowGap
		}
		prizes = append(prizes, image.Rect(x, top, x+cardWidth, top+cardHeight))
		x += cardWidth + wormRowGap
	}
	if mid >= n {
		button = image.Rect(x, top+(cardHeight-offerButtonHeight)/2,
			x+offerButtonWidth, top+(cardHeight+offerButtonHeight)/2)
	}
	return prizes, button
}

// offerRow is the whole row of offered cards: what the seats are cut out of, and what the sort
// block is hung off. **One function rather than the arithmetic twice**, the reason cardBandWidth
// is one on the combat screen — a row and the tabs beside it disagreeing about where the row ends
// is a block standing in the middle of the cards.
func (s *PostBattleScene) offerRow(gs *state.GlobalState) image.Rectangle {
	return offerRowOf(gs, len(s.offer))
}

// offerRowOf is that row **for a stated number of cards**, which is what a slide needs: a row of
// eight is not centred where a row of seven is, so a card leaving one and landing in the other has
// to be able to ask about both. Same reason a cardSlide carries a count at each end.
func offerRowOf(gs *state.GlobalState, n int) image.Rectangle {
	width := (n-1)*handPitch(gs, n) + cardWidth
	left := gs.PctX(50) - width/2
	top := gs.PctY(offerRowPct)
	return image.Rect(left, top, left+width, top+cardHeight)
}

// offerSlot is where one offered card is drawn, and the rectangle it is clicked in.
//
// **A selected card lifts out of the row**, the hand's own gesture — see cardSlot, which does the
// same thing for the same reason. It was the card's `Selected` border alone until 2026-09-06, and
// that was not enough to see: with the worms lit by the selection, the screen said a card had been
// picked and did not say which one.
//
// **The lift is in this function rather than in the drawing**, so the protruding part of a card is
// clickable rather than merely visible — the drawn-here-clicked-there bug every row in this game is
// shaped to avoid.
func (s *PostBattleScene) offerSlot(gs *state.GlobalState, i int) image.Rectangle {
	at := s.offerSeat(gs, i, len(s.offer))
	if i == s.selected {
		at.Y -= offerSelectedNudge
	}
	return image.Rect(at.X, at.Y, at.X+cardWidth, at.Y+cardHeight)
}

// offerSeat is that seat as a point, for a stated row size — the counterpart of the combat
// screen's slotAt, and what the shared mover is handed.
func (s *PostBattleScene) offerSeat(gs *state.GlobalState, i, count int) image.Point {
	row := offerRowOf(gs, count)
	return image.Pt(row.Min.X+i*handPitch(gs, count), row.Min.Y)
}

// sortTabRect is the i'th tab of the sort block: one block, no air in it, its top edge on the
// offer row's top edge.
//
// **It hangs off the cards, not off a column of its own**, which is the rule the combat screen's
// block follows — see sortTabRect there. This screen has no control column, and the row is centred
// rather than banded, so the anchor is the row's own right edge and sortColumnGap is the same air
// the hand leaves.
func (s *PostBattleScene) sortTabRect(gs *state.GlobalState, i int) image.Rectangle {
	row := s.offerRow(gs)
	left := row.Max.X + sortColumnGap
	top := row.Min.Y + i*ControlButtonHeight
	return image.Rect(left, top, left+ControlColumnWidth(), top+ControlButtonHeight)
}

// setSort is the press on a tab. **It records the mode and nothing else** — the row is rearranged
// by sortOffer on the same tick, which is where the global state a sort needs is available; a
// button's OnClick reaches none on any screen in this package.
func (s *PostBattleScene) setSort(mode handSort) { s.sortMode = mode }

// sortOffer arranges the offer row and sends every card that moved sliding to its new place.
//
// **The row is a list of deck indices rather than of cards**, so it sorts a permutation and
// rebuilds — which is also what makes the slides possible at all, for the reason sortHand does it:
// two identical cards cannot be told apart after the fact by looking at them, and a card sliding
// has to know where it set off from.
//
// **Stable, over the deck order dealOffer left it in**, so two identical cards keep their
// positions in the deck as the tie-break and pressing the same tab twice cannot shuffle them.
//
// **The cards have already moved by the time the slides exist**, exactly as on the combat screen:
// s.offer is in its new order the instant this returns, and every slide is a ghost of a card that
// is already where it is going.
//
// **It runs every tick while the row is up**, not only on a press, which is what makes it right
// after a re-deal for a second pick — the same reason the combat screen re-sorts on every refill.
// A stable sort over an already-sorted list is a walk of eight items that raises nothing.
func (s *PostBattleScene) sortOffer(gs *state.GlobalState) {
	if gs.Run == nil {
		return
	}

	card := func(deckIndex int) (combat.Card, bool) { return gs.Run.Card(deckIndex) }

	order := make([]int, len(s.offer))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(i, j int) bool {
		a, aok := card(s.offer[order[i]])
		b, bok := card(s.offer[order[j]])
		if !aok || !bok {
			// A deck index the run cannot resolve should not exist here — the offer is dealt off
			// the deck between fights — but ordering by index rather than panicking keeps a row
			// the player can still read if one ever does.
			return s.offer[order[i]] < s.offer[order[j]]
		}
		return handLess(s.sortMode, a, b)
	})

	sorted := make([]int, len(s.offer))
	for to, from := range order {
		sorted[to] = s.offer[from]
	}
	s.offer = sorted

	// Nothing in this row stands proud of it: there is no selection on this screen, so the lift
	// every slide carries is zero.
	s.slides = slidesFor(s.slides, order, func(i int) actionCard {
		c, _ := card(s.offer[i])
		return c
	}, func(int) int { return 0 })
}

// drawSlides draws the offered cards moving within their row, on the shared mover.
//
// **A sliding card is drawn usable**, whatever the worm could do to it. The dimming says "this one
// cannot be picked", which is a fact about a card sitting in a seat waiting to be clicked; a card
// in flight is not being offered yet, and re-deriving it mid-slide would make the row flicker as
// cards crossed each other.
func (s *PostBattleScene) drawSlides(gs *state.GlobalState, screen *ebiten.Image) {
	drawCardSlides(gs, screen, s.slides,
		func(gs *state.GlobalState, i, count int) image.Point { return s.offerSeat(gs, i, count) },
		func(sl cardSlide) cards.Spec {
			return cardSpec(sl.card, heldByRun(gs, sl.card), true, false)
		})
}

func (s *PostBattleScene) chosenPrize() (prize, bool) {
	if s.chosen < 0 || s.chosen >= len(s.prizes) {
		return prize{}, false
	}
	return s.prizes[s.chosen], true
}

// chosenWorm is the worm the player picked, if they have.
func (s *PostBattleScene) chosenWorm() (session.Worm, bool) {
	p, ok := s.chosenPrize()
	if !ok {
		return session.Worm{}, false
	}
	return p.worm, true
}

func (s *PostBattleScene) Draw(gs *state.GlobalState, screen *ebiten.Image) {
	fillGround(screen)

	heading := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 34}
	small := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 18}
	prose := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 26}

	// **The build is on screen for the whole visit**, every stage of it: what the payout landed on,
	// and what a worm is about to change.
	drawBuildBand(gs, screen, gs.Run.Vitae(), &s.relicDrag)

	// **The narration stays up while the offer is made**, and only clears once a worm is chosen.
	if s.stage == narrate {
		s.drawProse(gs, screen, prose)
	}

	if s.stage == narrate {
		return
	}

	line := func(y int, face *text.GoTextFace, msg string) {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(gs.PctX(50)), float64(y))
		op.PrimaryAlign = text.AlignCenter
		op.ColorScale.ScaleWithColor(groundInk)
		text.Draw(screen, msg, face, op)
	}

	// **Neither stage that offers a choice is titled** *(owner's call, 2026-09-05)*. The worms
	// arrive under the sentence saying they are fleeing, and the card row under the worm that is
	// going to eat one — in both cases the thing on screen says what the screen is for, and a
	// heading over it was a caption on a picture nobody had trouble reading.
	if s.stage != choosing {
		line(offerTitleTop(gs), heading, s.title())
		line(offerHintTop(gs), small, s.hint(gs))
	}

	defer systems.DrawTooltip(gs, screen, &s.tip)

	switch s.stage {
	case settled:
		s.drawSettled(gs, screen)
		return
	}

	s.drawWorms(gs, screen)

	if s.stage == choosing {
		systems.DrawButton(gs, screen, s.skipButton)
	}

	if s.stage == choosing {
		for i, deckIndex := range s.offer {
			card, ok := gs.Run.Card(deckIndex)
			if !ok {
				continue
			}
			// A seat a card is still sliding into is left empty until it lands — the same rule
			// the hand follows, and for the same reason: the list is already in its new order,
			// so what is suppressed is a second drawing of a card that is on screen elsewhere.
			if slideInto(s.slides, i) {
				continue
			}
			// **Every card is selectable and the worms are what go dim** *(2026-09-06)*. The row
			// used to dim a card the chosen worm could not change, which was the same rule read in
			// the other direction — with the card picked first there is no worm yet to ask, so the
			// legality lands on the prize row instead. See wormSpendable.
			drawCard(gs, screen, s.offerSlot(gs, i).Min, cards.Hand, card, heldByRun(gs, card),
				true, i == s.selected)
		}
		s.drawSlides(gs, screen)
		s.sortTabs.draw(gs, screen)
	}

	// **Bob over everything, and the spotlight with him.** See combat.go's Draw, whose last line
	// this is the counterpart of: the scrim dims what is already drawn, so nothing may follow it.
	s.tut.draw(gs, screen, s)
}

// settledSeat is where the won card comes to rest.
//
// **A copy needs two seats and everything else needs one**, so the row is laid out for the number
// of cards that will be standing in it at the end — which is what stops the original having to
// slide aside when the copy turns up. See settledSeats.
func settledSeat(gs *state.GlobalState) image.Rectangle {
	return settledSeats(gs, 1)[0]
}

// settledSeats lays the settled row out for n cards, centred.
func settledSeats(gs *state.GlobalState, n int) []image.Rectangle {
	if n < 1 {
		n = 1
	}
	top := gs.PctY(36)
	width := n*cardWidth + (n-1)*wormRowGap
	x := gs.PctX(50) - width/2

	out := make([]image.Rectangle, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, image.Rect(x, top, x+cardWidth, top+cardHeight))
		x += cardWidth + wormRowGap
	}
	return out
}

// drawSettled is the card the player picked, **flying to the middle, changing there, and then held
// while they read what it became**.
//
// It travels from the seat it was already in, as the card it was — so the card the player has been
// looking at is the card that moves, and the alteration happens in front of them rather than in the
// gap between two frames. A card that arrived already changed would be a card the player has to
// re-read to find out what happened. See cardmorph.go.
//
// **The flight carries the old face and the morph carries the new one**, which is why nothing here
// asks what the worm did: `change` was handed the two faces in aimAt and is the only thing that
// knows which of the three shapes this is. A removal ends on an empty seat, a duplicate ends on two
// cards, everything else ends on one.
func (s *PostBattleScene) drawSettled(gs *state.GlobalState, screen *ebiten.Image) {
	seats := settledSeats(gs, 1)
	if s.copied {
		seats = settledSeats(gs, 2)
	}

	at := flyingTo(s.arrivedFrom, seats[0], s.arrival)

	// The card that was picked. While a copy is being made it is the original, untouched, and it is
	// drawn plainly — the morph in the second seat is the whole of what is happening.
	if s.copied {
		drawCard(gs, screen, at, cards.Hand, s.before, heldByRun(gs, s.before), true, false)
		drawMorph(gs, screen, seats[1].Min, s.change)
		return
	}

	// **An eaten card leaves nothing** *(owner's call, 2026-09-08)*. It left an outlined hole until
	// then, on the argument that a blank gap reads as a layout fault — which is true of a seat that
	// was never filled, and not of this one: the player has just watched the card come apart square
	// by square, so the emptiness is the thing they were shown rather than something to explain. An
	// outline redrawn over the space the dissolve had just cleared put the card's silhouette back on
	// the table the frame after eating it.
	//
	// The morph is what draws the absence, by having no second face to hand over to. Nothing here
	// asks whether this was a removal.
	drawMorph(gs, screen, at, s.change)
}

func (s *PostBattleScene) title() string {
	switch s.stage {
	case settled:
		if s.removes {
			return "EATEN"
		}
		return "CHANGED"
	case narrate:
		return ""
	default:
		return "WORMS FLEE"
	}
}

// drawPrizeCard draws one prize where the row says it goes.
func drawPrizeCard(gs *state.GlobalState, screen *ebiten.Image, at image.Point,
	p prize, enabled bool) {

	drawWormCard(gs, screen, at, p.worm, enabled)
}

// drawWorms puts the offer up as cards. **A worm is a card because it is a thing you are given**,
// and the game already has one visual language for that — a name, a border colour and a line
// saying what it does.
//
// It borrows `cards.Hand` rather than taking a style of its own: a worm has no cost and no form,
// which that style draws as nothing at all, so what is left is exactly the name and the text. A
// dedicated style is what this wants once a worm has art.
// **A prize is lit exactly when clicking it would take it** *(2026-09-06)*, which is what makes
// select-then-click readable: with no card selected the whole row is dim, and selecting one lights
// the worms that could eat it. It is the same rule the parasite pane is under — see
// consumableTarget — and the same rule the offer row itself already followed in the other
// direction.
func (s *PostBattleScene) drawWorms(gs *state.GlobalState, screen *ebiten.Image) {
	for i, p := range s.prizes {
		drawPrizeCard(gs, screen, s.wormArrivingAt(gs, i), p, s.wormSpendable(gs, p))
	}
}

func (s *PostBattleScene) hint(gs *state.GlobalState) string {
	switch s.stage {
	case settled:
		if s.removes {
			return "one fewer card to draw"
		}
		return "into the deck it goes"
	default:
		if s.picksLeft > 1 {
			return fmt.Sprintf("take %d - %d cards in your deck, %d vitae in hand",
				s.picksLeft, gs.Run.Size(), gs.Run.Vitae())
		}
		return fmt.Sprintf("take one - %d cards in your deck, %d vitae in hand",
			gs.Run.Size(), gs.Run.Vitae())
	}
}

// updateRelicRow runs the drag over the worn row in the build band.
//
// **A click on a relic does nothing here**, as on the combat screen: this screen's clicks belong to
// the worms it is offering, and a relic that did something on a press would be a second meaning for
// the gesture that reorders it.
func (s *PostBattleScene) updateRelicRow(gs *state.GlobalState) {
	row := buildRelicRow(gs, nil)

	if !gs.CursorAllowed() {
		s.relicDrag.cancel(row)
		return
	}

	s.relicDrag.update(gs, row)
}
