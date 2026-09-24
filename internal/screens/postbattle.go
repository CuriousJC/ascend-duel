package screens

// The post-battle screen: **pick an essence, then pick the card it eats.**
//
// It is the first of the between-fight scenes — a shop and a room choice come after it — and all
// of them are ordinary scenes in the registry rather than modes of the combat screen. That is
// what keeps `CombatScene` from growing a fourth phase, and it is why the chain is wired as
// screen changes: win → post-battle → combat.
//
// **The payout and the offer are one screen** *(owner's call, 2026-09-18)*. The win is read out in
// the first third of the width — interest, a tenth of the life you kept, what the room is worth,
// each figure flying to the duelist card as its sentence lands — and the essences and the way out
// stand in the two thirds beside it. It was a narration the player clicked past before anything was
// offered, so the two halves of "you won" were two screens with a click between them.
//
// **The columns are what makes that fit.** The payout is a block of short lines and the offer is a
// row of cards; side by side they each have their own width, where stacked they were each other's
// ceiling. See payoutColumnPct, essenceRowSeats and proseColumnMid.
//
// **The build is on screen the whole time**: the player's card in the corner and their relics
// beside it, so an essence is chosen against the thing it would be changing. See
// postbattle_prose.go and buildband.go.
//
// **The essences are the reward and they are up from the first frame** *(2026-08-17)*. Two are drawn
// from the catalog and offered as cards; a hand off the run deck is dealt below them to apply one
// to. It ran the other way round first — pick a card, then say what to do to it — and the reason it
// turned is that the *essence* is the reward, and a menu of verbs under a chosen card made the
// reward look like a property of the card.
//
// **An essence varies a card the game already defines.** It recolors it, removes it, or copies it;
// the concept is never touched, so nothing here can produce a card `internal/combat` cannot
// resolve. See `internal/session/essence.go` for the target vocabulary and why it is short.
//
// **The offer is indices into the run deck, not cards.** Alteration happens between fights, when
// no pile is live and nothing else is touching the list, so a position is unambiguous for exactly
// as long as the screen is up. That is what makes card identity unnecessary — see MECHANICS.md
// for what mid-fight alteration would cost instead.

import (
	"fmt"
	"image"
	"math/rand"
	"sort"

	"github.com/curiousjc/ascend-duel/internal/achieve"
	"github.com/curiousjc/ascend-duel/internal/ui"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/journal"
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

// essencesOffered is how many alterations a win puts up.
//
// **Two essences and nothing else** *(owner's call, 2026-08-22)*. A third card paying vitae used to
// stand beside them, so that declining a change to the deck was a choice among three with a price.
// The win pays vitae by itself now — read out at the top of this screen — so the money card was
// charging for something the player had already been given, and the offer is the two essences the
// prose says are bleeding from it. **Taking neither is a button again**, deliberately: the offer is free, so
// walking away costs nothing and does not need to look like a card.
const essencesOffered = 2

// prize is one of the cards on the table. **Every one of them is an essence** since the vitae card went
// — the struct survives because a prize is an essence *plus what this visit has done with it*, which
// the catalog record has no business carrying.
type prize struct {
	// taken is set once this prize has been picked. **It stays in the row rather than being removed
	// from it** — a card leaving would move the one beside it. Only a relic that adds a pick can
	// produce a row with a taken card still in it.
	taken   bool
	essence session.Essence
}

func (p prize) name() string { return p.essence.Name }

// Where the two rows sit and where the controls sit under them. Percentages anchor the groups;
// offsets inside a group stay in pixels, per CLAUDE.md.
const (
	// **The prose has done its job once the offer is up, and the two rows take the screen** — the
	// essences where the eye lands, the cards they may eat below them.
	//
	// **The essences used to sit at 58% under the payout** *(owner's call, 2026-08-22)*, on the
	// argument that what a win paid and what it is offering are one picture. That could not
	// survive both rows being on screen at once: an essence row at 58% is 280 tall against a button
	// strip at 88%, so there is nowhere for the cards to go. **The narration therefore clears when
	// the offer arrives rather than one stage later**, which is the cost of the reversed gesture
	// and was taken deliberately.
	essenceChosenRowPct = 34

	// payoutColumnPct is where the payout's column ends and the offer's begins.
	//
	// **A third of the width for the reading and two thirds for the choosing** *(owner's call,
	// 2026-09-18)*. The payout is five short lines — the widest measures about 520 pixels at 26pt,
	// which fits a 640-pixel column with room on both sides — and the essence row is two cards and a
	// button, 880 wide, which fits the rest. Neither column is centered on the screen, so the two
	// cannot drift into each other as either grows: the split is the one number they both read.
	payoutColumnPct = 33

	// **64 rather than 62 since 2026-09-06**, to buy the headroom a selected card lifts into. See
	// offerSelectedNudge: the row is a card tall and rises by 26 when one is picked, and at 62 the
	// lifted card's top edge landed inside the essence row above it.
	offerRowPct = 64

	// The title and the hint hang off the bottom of the build band, each by its own drop.
	// **Measured from the band rather than from the top of the screen**, so the next time the band
	// moves the type follows — it ends at 311 today and both of these were absolute pixels written
	// against a band that ended at 253, which drew the first line of the payout under a worn relic.
	offerTitleDrop = 9
	offerHintDrop  = 47

	// proseTextSize is the type the payout is set in, and proseLineGap is the pitch between its
	// lines. **The size is here rather than at the one call site that draws it** because the block
	// is laid out from its *last* line up — see proseTop — so its height is arithmetic over both
	// figures and neither may be written down twice.
	proseTextSize = 26
	proseLineGap  = 42

	// proseLineHeight is how tall one line of that type actually is, which is what the block's
	// bottom edge is measured against: a pitch is the distance between two lines and says nothing
	// about where the last one ends. `TestThePayoutsLineHeightIsTheFontsOwn` measures the real face
	// and fails if this drifts — the block would come to sit a few pixels off the row it is aligned
	// to, which is exactly the kind of wrong nobody sees and nobody can unsee.
	proseLineHeight = 22

	offerButtonsPct   = 88
	offerButtonWidth  = 400
	offerButtonHeight = 76

	// essenceRowGap is the air between the two essences, and between the second of them and the way
	// out.
	//
	// **The essences stand together and the button stands beside them** *(owner's call,
	// 2026-09-18)*. The two cards are the question, so they are read against each other without a
	// control cutting between them; taking neither is the answer that is not a card, so it sits off
	// the end of the row rather than inside it. **It is bottom-aligned to the cards** — a button is
	// half a card tall, and a control centered on a row of cards reads as floating between them.
	essenceRowGap = 40

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
	// choosing: everything is up — the payout reading itself out in its column, the essences, and
	// the cards they may eat. **Every visit starts here** *(owner's call, 2026-09-18)*: the payout
	// used to be a stage of its own that the offer waited behind, and a win is one thing rather
	// than two.
	//
	// **It was two stages until 2026-09-06** *(owner's call)*, essence first and then the card. The
	// order reversed with the rune's: **select the card, then click the essence**, so a consumable
	// is pointed at a card the same way everywhere in the game. One gesture in one order meant one
	// stage — the two rows are on screen together, because the player is choosing between them
	// rather than passing through them.
	choosing stage = iota

	// settled: the alteration is taken and the new card is shown alone, so the thing that was won
	// is looked at before it disappears into a deck of forty-eight.
	settled
)

// This screen's two clocks, **both proportions of the game's one speed** *(2026-08-21)*.
//
// They were raw tick counts — 26 and 100 — written before `beat` existed, which meant this screen
// was outside the setting the duel is paced by: turning the game's speed down would have sped up a
// round and left the reward screen exactly as slow as it was. See clock.go. The one behavior
// change is that the flight is 25 ticks rather than 26, which is a frame and a half.
//
// `var` rather than `const` because `beat` is a function, exactly like `victoryHoldTicks()`.
// settleFlightTicks() is how long the won card takes to cross to the middle.
//
// **Cards fly to where they are going, everywhere in this game.** A card that appears in the
// middle is a card that was never anywhere else, and the whole point of this screen is that a
// thing was *won* and has come to you.
func settleFlightTicks() int { return ui.Beat(1, 1) }

// settledHoldTicks() is how long the finished card is held before the screen leaves. **Long
// enough to read, short enough not to need a button** — the click that picked the card is the
// last input the player has to make.
func settledHoldTicks() int { return ui.Beat(4, 1) }

// PostBattleScene offers one alteration to the run deck.
type PostBattleScene struct {
	// prizes is the three cards on the table — two essences and the vitae — and chosen is which one,
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

	// entry is each offered essence's flight in from the side of the screen — one per prize, indexed
	// alike. **Cards fly; they never appear**, and an essence arriving from off-screen is the picture
	// the prose has just described: two essences bleeding from the enemy you beat.
	entry []ui.Travel

	// deck is the panel over the whole deck, opened by clicking the draw pile.
	//
	// **An essence is aimed at a card, so the deck is what the choice is about** *(owner's call,
	// 2026-09-19)*. The offer is a hand's worth off a deck of fifty-odd, and judging "do I want one
	// fewer earth Bash" against eight of them and nothing else was the one screen in the game where
	// the deck mattered most and could not be read. The same widget and the same pile the shop and
	// the sealed good carry — see deckpile.go.
	deck ui.DeckToggle

	// relicDrag is the press in progress over the worn relic row in the build band. **The row is
	// reorderable here like everywhere else** — worn order is a rule, and between fights is when a
	// player is thinking about their build.
	relicDrag ui.CardDrag

	// **Skipping is a button again** *(2026-08-22)*, after the vitae card that replaced it was
	// removed. It takes neither essence and pays nothing extra — the win has already paid — so it is
	// an exit rather than a third choice, which is exactly why it is not a card.
	skipButton *models.Button

	// tut is Bob, when a run is being taught. See tutorial.go, and combat.go for the same field.
	tut tutorialOverlay

	// selected is which offered cards are picked out, as row slots.
	//
	// **A set rather than one index, because an essence may take more than one card** *(owner's
	// call, 2026-09-19)*. One is the mechanic and a relic scales it — see essence_targets.go — so
	// what asks whether the selection is enough is consumableTarget, exactly as it does for a rune
	// naming two cards on the combat screen.
	//
	// **It is the offer row's counterpart of the hand's `selected` flag**, and it stays a different
	// shape: the hand's selection is also the round's queue, where this is only the cards an
	// essence is about to eat.
	selected []int

	// lands is what the essence did to each card it was pointed at: the seat each one flew out of,
	// the face it had, the face it became, and the change between them.
	//
	// **Computed once, when the cards are picked** rather than every frame: the essence is run
	// against a throwaway copy of the run, and doing that in Draw would be a screen that alters the
	// deck sixty times a second. See essence_spend.go.
	lands []essenceLanding

	// removes says the alteration has no "after" card, because the card is gone. What is left when
	// the dissolve finishes is an empty seat.
	//
	// **One flag for the whole spend rather than one per card**, because an essence that eats eats
	// every card it was aimed at: what an essence *did* is a fact about the essence.
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
	arrival     ui.Travel
	arrivedFrom image.Rectangle

	// applyNow is the confirmed alteration, run against the real deck once the settled stage is
	// over. **The deck is not touched while the result is on screen**, which is what lets the
	// after-card be drawn from a preview rather than from a deck already altered — and it means
	// leaving the screen any other way changes nothing.
	applyNow func(*session.Session)

	// pendingWhat is what the trace line will say once the alteration lands.
	pendingWhat string

	// tip explains whichever card the cursor is on: what an essence will do and how long it lasts, or
	// what one of the offered deck cards is worth.
	tip models.Tooltip

	// How the offer row is arranged, and the block of tabs that chooses it — the same widget the
	// combat screen's hand carries *(owner's call, 2026-09-05)*. **Eight overlapping cards are
	// eight overlapping cards wherever they are dealt**, so the row an essence is pointed at is read
	// the same way a hand is.
	//
	// **sortMode is the working copy of `gs.HandSort`**, exactly as CombatScene's is: the button
	// callback moves this, and Update writes it back. So a player who arranges by element in a
	// duel meets an offer already arranged by element.
	sortMode ui.HandSort
	sortTabs *ui.SortTabs

	// slides is the offer row rearranging itself, on the shared mover — see cardslide.go. **The
	// same widget behaves the same way on both screens** *(owner's call, 2026-09-05)*: a card that
	// changes where it is on screen travels there, and a sort that re-laid this row out instantly
	// while sliding the hand would be two controls wearing one set of labels.
	slides []ui.CardSlide

	// picksLeft is how many prizes this visit still owes the player, from `session.Picks` — the
	// `prizes-dealt` moment, which the Hungry relic is what moves off 1.
	//
	// **A second pick is another card out of the same row**, not a fresh row: the offer was dealt
	// once, and re-dealing it would make the second pick a different draw of the same fight. The card
	// offer *is* re-dealt, because an essence may have removed a card and every index after it has moved.
	picksLeft int
}

// Init deals both offers. **Re-entered on every visit**, because each fight earns its own.
func (s *PostBattleScene) Init(gs *state.GlobalState) {
	if s.skipButton == nil {
		s.skipButton = models.NewButton(offerButtonWidth, offerButtonHeight, "LET THEM ESCAPE",
			func() { s.skipping = true })
		s.skipButton.BaseColor = color.RGBA{R: 120, G: 132, B: 150, A: 255}
	}

	s.deck.InitAsPile()
	s.chosen, s.selected = -1, nil
	s.stage = choosing
	s.removes, s.copied, s.held = false, false, 0
	s.lands = nil
	s.arrival, s.arrivedFrom = ui.Travel{}, image.Rectangle{}
	s.pendingWhat, s.applyNow = "", nil
	s.prizes = dealPrizes(gs)
	s.entry = make([]ui.Travel, len(s.prizes))
	s.flyEssencesIn()
	s.prose.setLines(payoutLines(gs))
	s.skipping = false
	s.offer = dealOffer(gs)
	s.picksLeft = gs.Run.Picks()
	s.tip = models.Tooltip{DwellTicks: ui.TipDwell()}

	s.sortMode = ui.HandSortOf(gs)
	if s.sortTabs == nil {
		s.sortTabs = ui.NewSortTabs(s.sortTabRect, s.setSort)
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

// dealPrizes is the offer: two essences drawn from the catalog.
func dealPrizes(gs *state.GlobalState) []prize {
	out := make([]prize, 0, essencesOffered)
	for _, w := range dealEssences(gs) {
		out = append(out, prize{essence: w})
	}
	return out
}

// dealEssences picks which alterations are offered: a shuffle of the catalog, cut to two.
//
// **Its own stream** (`seeds.EssenceOffer`), separate from the cards. They are drawn from different
// lists and change on different schedules — adding an essence to the catalog would otherwise reroll
// which *cards* every fight of every run offered.
//
// **Distinct by construction**, since it shuffles the catalog rather than drawing twice: being
// offered the same essence as both options would be a choice that is not one.
func dealEssences(gs *state.GlobalState) []session.Essence {
	all := session.Essences()

	rng := rand.New(rand.NewSource(seeds.ForFight(gs.RunSeed, seeds.EssenceOffer, gs.Run.Fight())))
	rng.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })

	if len(all) > essencesOffered {
		all = all[:essencesOffered]
	}
	return all
}

// dealOffer picks which cards the essence may be applied to: a shuffle of every index in the run
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

// place puts the skip button in its seat in the essence row. **Read off essenceRowSeats**, so the button
// and the cards beside it cannot disagree about where the row is.
func (s *PostBattleScene) place(gs *state.GlobalState) {
	_, button := essenceRowSeats(gs, len(s.prizes))
	s.skipButton.ScreenX = button.Min.X + button.Dx()/2
	s.skipButton.ScreenY = button.Min.Y + button.Dy()/2
}

func (s *PostBattleScene) Update(gs *state.GlobalState) error {
	// **Before this screen's own input**, for the reason combat.go runs it first: the gate it
	// sets is what every widget below reads.
	s.tut.update(gs, s)

	// **The narration runs beside the offer rather than in front of it** *(owner's call,
	// 2026-09-18)*. It holds nothing up and gates nothing: it is a clock over figures
	// `session.WonFight` already froze, so a player who reads every line and one who takes an
	// essence on the first frame end the visit with the same purse. See claimThePayout, which is
	// what makes that true when the screen is left early.
	if s.stage == choosing {
		s.prose.tick(gs, func(i int) image.Point { return proseLineAt(gs, len(s.prose.lines), i) })
	}

	// **The deck panel runs before anything else and swallows the frame**, the shop's own order:
	// while it is up the rows underneath are dead, so a press meant for the panel cannot reach the
	// offer behind it.
	if s.deck.Update(gs, ui.OwnedContents(gs)) {
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
		if !s.arrival.Done() {
			s.arrival.Tick()
			return nil
		}
		// **The change is its own beat, and it does not start until the card has landed** — the
		// same rule the hold below follows, and for the same reason: a dissolve running over a
		// moving card would put the one thing worth watching on a target the eye is still chasing.
		if !tickLandings(s.lands) {
			return nil
		}
		s.held--
		if s.held <= 0 {
			if s.applyNow != nil {
				s.applyNow(gs.Run)
				trace.Logf("postbattle", "%s, deck now %d", s.pendingWhat, gs.Run.Size())
				s.applyNow = nil

				// **The alteration is a moment, raised where the deck actually changes.** An essence
				// that removed a card leaves nothing behind, so there is nothing to name and the
				// moment is not raised — see achieve.MomentCardAltered, which carries the resulting
				// card's label rather than the essence's, because several essences can arrive at one card.
				if !s.removes {
					for _, l := range s.lands {
						earnMoment(gs, achieve.CardAltered(l.after.Label()))
					}
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
		s.claimThePayout(gs)
		trace.Logf("postbattle", "took neither essence")
		advanceRun(gs)
		return nil
	}

	// The essences' arrival. **They are clickable while they fly** — a card is where its layout
	// function says it is, and the flight is a ghost over that seat, the same rule the combat
	// screen's hand follows.
	for i := range s.entry {
		s.entry[i].Tick()
	}

	s.click(gs)

	switch s.stage {
	case choosing:
		s.place(gs)
		systems.UpdateButton(gs, s.skipButton)

		// **Placed every tick rather than at Init**, unlike every other widget on this screen: the
		// block hangs off the offer row's right edge, and a second pick re-deals that row against a
		// deck an essence may have shortened. A block placed once would then stand beside a row that
		// had moved out from under it.
		s.sortTabs.Place(gs)
		s.sortTabs.Update(gs, true)
		ui.SetHandSort(gs, s.sortMode)
		s.sortOffer(gs)
		s.slides = ui.Advance(s.slides)
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
	// actually do" is the question an essence is picked by.
	if hoverBuildRelics(gs, at, &s.tip) {
		return
	}

	switch s.stage {
	case choosing:
		// **The prizes are tooltipped now that their faces are pictures** *(owner's call,
		// 2026-09-18)*. An essence used to print its whole rule across a scrim over its art, so a
		// panel repeating it would have been the card read twice; the sentence lives here instead.
		// What a dim prize means — "not for the card you have selected" — is still left to the row.
		if i := ui.HoveredSeat(at, len(s.prizes), func(i int) image.Rectangle {
			return s.essenceSlot(gs, i)
		}); i >= 0 {
			seat := s.essenceSlot(gs, i)
			title, lines := ui.EssenceTip(s.prizes[i].essence, s.reachNow(gs))
			s.tip.Point(seat, ui.TipLine(title), ui.TipLines(lines))
			return
		}

		// **ui.HoveredSeat, like every row in the game** — the offer is dealt at the hand's own
		// pitch, so it overlaps exactly when the hand does.
		if i := ui.HoveredSeat(at, len(s.offer), func(i int) image.Rectangle {
			return s.offerSlot(gs, i)
		}); i >= 0 {
			seat := s.offerSlot(gs, i)
			if card, ok := gs.Run.Card(s.offer[i]); ok {
				title, lines := ui.CardTip(card, ui.HeldByRun(gs, card))
				s.tip.Point(seat, ui.TipLine(title), ui.TipLines(lines))
			}
			return
		}
	}
}

// click is the press on the two rows, and on the column beside them.
func (s *PostBattleScene) click(gs *state.GlobalState) {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || !gs.CursorAllowed() {
		return
	}
	at := image.Pt(gs.MouseX, gs.MouseY)

	if s.stage != choosing {
		return
	}

	// **The pile is asked first**, because the panel it opens covers both rows: a press here while
	// the panel is up must not reach the offer underneath. See deckpile.go.
	if at.In(deckPileBounds(gs)) {
		s.deck.Toggle()
		s.tip.Forget()
		return
	}

	// **The card row is asked next**, because it is the row a click is most often meant for and
	// the two do not overlap. Selecting is free and reversible; clicking an essence spends the pick.
	// **Both go through ui.HoveredSeat**, so a click lands on the card the tooltip just described.
	if i := ui.HoveredSeat(at, len(s.offer), func(i int) image.Rectangle {
		return s.offerSlot(gs, i)
	}); i >= 0 {
		s.selectOffered(gs, i)
		return
	}

	if i := ui.HoveredSeat(at, len(s.prizes), func(i int) image.Rectangle {
		return s.essenceSlot(gs, i)
	}); i >= 0 {
		s.takePrize(gs, i)
		return
	}

	// **A click on nothing fills the payout** *(owner's call, 2026-09-18)*, which is the whole of
	// what the narration's own click became once it stopped owning the screen: a player who wants
	// the figures now still has a gesture for it, and it is the one gesture that was already
	// meaningless here. It cannot take the screen off them — nothing here releases anything.
	if !s.prose.filled() {
		s.prose.skip(gs)
	}
}

// claimThePayout pays whatever the narration has not read out yet.
//
// **The payout may not depend on how long the player looked at it.** The figures were frozen by
// `session.WonFight` before this screen existed, and the narration is a clock over them — so
// leaving while a line is still typing has to hand over the same purse that watching it would.
// `typewriter.skip` pays through the same claims rather than adding a total of its own, which is
// what stops the fast path and the slow one disagreeing.
func (s *PostBattleScene) claimThePayout(gs *state.GlobalState) {
	if !s.prose.filled() {
		s.prose.skip(gs)
	}
}

// selectOffered picks a card out of the offer row, or puts it back.
//
// **Clicking a selected card deselects it**, which is the hand row's own gesture — a card clicked
// into the queue is clicked out of it — so the one thing a player already knows how to undo works
// here too.
//
// **A full selection replaces its oldest card rather than refusing the click** *(owner's call,
// 2026-09-19)*. An essence takes a fixed number of cards, so at the cap there is nothing a further
// click could add, and a row that ignored it would leave a player who picked the wrong card having
// to work out that they must deselect one first. Replacing the oldest is also what keeps the
// single-target case behaving exactly as it always did: clicking a second card moves the pick.
func (s *PostBattleScene) selectOffered(gs *state.GlobalState, i int) {
	for k, sel := range s.selected {
		if sel == i {
			s.selected = append(s.selected[:k], s.selected[k+1:]...)
			s.tip.Forget()
			return
		}
	}

	if n := s.targets(gs); len(s.selected) >= n {
		s.selected = append([]int(nil), s.selected[len(s.selected)-n+1:]...)
	}
	s.selected = append(s.selected, i)
	s.tip.Forget()
}

// isSelected reports whether this row slot is one of the picked cards.
func (s *PostBattleScene) isSelected(i int) bool {
	for _, sel := range s.selected {
		if sel == i {
			return true
		}
	}
	return false
}

// targets is how many cards an essence takes on this screen — one, whatever the relics make of it,
// and never more than the offer is holding. See essence_targets.go.
func (s *PostBattleScene) targets(gs *state.GlobalState) int {
	return essenceTargetCount(gs, len(s.offer))
}

// reachNow is how many cards a click on an essence would change right now: what is selected, or the
// ceiling when nothing is. See essenceReach.
func (s *PostBattleScene) reachNow(gs *state.GlobalState) int {
	return essenceReach(len(s.selected), s.targets(gs))
}

// selectedSlots is the picked cards **in row order**, whatever order they were clicked in.
//
// **The order of a selection is the order of the row**, which is the rule the combat screen's
// consumables are already under: what the player reads left to right is what the settled row shows
// left to right, and there is no separate click order to learn.
func (s *PostBattleScene) selectedSlots() []int {
	out := append([]int(nil), s.selected...)
	sort.Ints(out)
	return out
}

// selectedDeckIndexes is the offer's current picks as indexes into the run deck, in row order.
func (s *PostBattleScene) selectedDeckIndexes() []int {
	out := make([]int, 0, len(s.selected))
	for _, slot := range s.selectedSlots() {
		if slot < 0 || slot >= len(s.offer) {
			return nil
		}
		out = append(out, s.offer[slot])
	}
	return out
}

// essenceSpendable is whether clicking this prize now would take it: at least one card and no more
// than the essence reaches is selected, and this essence can change every one of them.
//
// **A player may always take fewer.** The reach is a ceiling rather than a quota — see
// consumableTarget.fewest.
//
// **It is the same question the click asks and the same one the card's lit state reads**, which is
// what stops a prize looking available and doing nothing. See consumableTarget.
func (s *PostBattleScene) essenceSpendable(gs *state.GlobalState, p prize) bool {
	if p.taken || gs.Run == nil {
		return false
	}
	target := consumableTarget{
		needs:  s.targets(gs),
		fewest: 1,
		legal: func(idx []int) bool {
			for _, i := range idx {
				if !gs.Run.CanApply(p.essence, i) {
					return false
				}
			}
			return true
		},
	}
	return target.satisfiedBy(s.selectedDeckIndexes())
}

// takePrize is the click on the prize row: this essence is spent on the card that is selected.
//
// **It refuses rather than falling back**, on the predicate the card's lit state already read — a
// prize drawn dim cannot be taken, and a prize drawn lit always works.
func (s *PostBattleScene) takePrize(gs *state.GlobalState, i int) {
	if i < 0 || i >= len(s.prizes) || !s.essenceSpendable(gs, s.prizes[i]) {
		return
	}
	slots := s.selectedSlots()

	// **The purse is settled before the essence is**, because the settled stage returns early and
	// the narration stops ticking the moment the choosing stage ends. See claimThePayout.
	s.claimThePayout(gs)

	// **The essence and the cards it was aimed at, in one line.** The offer row's own clicks are
	// deliberately not journalled: a selection that is replaced before the prize is taken changed
	// nothing, and what has to be retraceable is which cards the essence actually landed on.
	gs.Journal.Write(journal.Record{
		Kind:    journal.KindEssence,
		Key:     s.prizes[i].essence.Record,
		Seat:    i,
		Targets: deckCardIDs(gs, s.selectedDeckIndexes()),
	})

	s.chosen = i
	s.tip.Forget()
	s.aimAt(gs, slots)
}

// rearm is what a second pick is: the taken prize is struck off, the row stays where it is, and the
// screen goes back to the top. It reports whether another pick is owed.
//
// **The card offer is re-dealt and the prize row is not.** An essence may have removed a card, so every
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

	s.chosen, s.selected = -1, nil
	s.stage = choosing
	s.removes, s.copied, s.held = false, false, 0
	s.lands = nil
	s.arrival, s.arrivedFrom = ui.Travel{}, image.Rectangle{}
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
	s.stage, s.held = settled, settledHoldTicks()
	s.arrival = ui.NewTravel(0, settleFlightTicks())
	s.arrivedFrom = from
}

// aimAt points the chosen essence at the offered cards and works out what each would become.
//
// **The preview runs the real essence against a throwaway copy of the run**, rather than a second
// implementation of what each target does. A preview computed by its own arithmetic is a preview
// that can disagree with the thing it is previewing, which is the one failure this screen exists
// to prevent. See essence_spend.go, which is that preview and the shop's vial's at once.
//
// **The commitment is by identity, not by position** *(2026-09-19)*. An essence may take several
// cards now, and removing one moves every deck position above it — so the positions this frame read
// are resolved to the cards they name before anything is promised, and `ApplyToAll` finds them
// again when the settled stage is over.
func (s *PostBattleScene) aimAt(gs *state.GlobalState, slots []int) {
	essence, ok := s.chosenEssence()
	if !ok || len(slots) == 0 || gs.Run == nil {
		return
	}

	at := make([]int, 0, len(slots))
	from := make([]image.Rectangle, 0, len(slots))
	ids := make([]int, 0, len(slots))
	for _, slot := range slots {
		if slot < 0 || slot >= len(s.offer) {
			return
		}
		card, ok := gs.Run.Card(s.offer[slot])
		if !ok {
			return
		}
		at = append(at, s.offer[slot])
		from = append(from, s.offerSlot(gs, slot))
		ids = append(ids, card.ID)
	}

	// **An essence that cannot take one of the cards is refused rather than shown** — the same
	// question the prize's lit state asked, so a prize drawn lit always works.
	lands, ok := previewEssence(gs, essence, at, from)
	if !ok {
		return
	}

	s.lands = lands
	s.removes = essence.Target == session.TargetRemove
	s.copied = essence.Target == session.TargetDuplicate

	// **The click is the commitment** *(owner's call, 2026-09-05)*. It was a preview with Take and
	// Back under it; picking the card is now the whole decision, and what follows is the result
	// being shown rather than a question about it. The deck is still not touched until the settled
	// stage is over — see applyNow — so the cards on screen are drawn from the trial run above.
	s.pendingWhat = fmt.Sprintf("%s on %d card(s) %v", essence.Record, len(at), at)
	s.applyNow = func(run *session.Session) { run.ApplyToAll(essence, ids) }
	s.tip.Forget()
	s.settle(gs, from[0])
}

// essenceSlot is where one offered essence is drawn, and the rectangle it is clicked in.
//
// **One function for both**, the same rule the hand follows: a card hit-tested against a
// rectangle it is not drawn in is exactly the bug this shape prevents.
//
// In the second stage the chosen essence stays on screen, alone and centered, so what is about to
// happen is still stated while the card is picked.
func (s *PostBattleScene) essenceSlot(gs *state.GlobalState, i int) image.Rectangle {
	// **The row sits where the chosen essence used to** *(2026-09-06)*. Both rows are up at once now,
	// so the essences take the seat the prose vacates and the cards they may eat go below them —
	// which is the layout the second stage already had, with every prize in it rather than one.
	seats, _ := essenceRowSeats(gs, len(s.prizes))
	if i < 0 || i >= len(seats) {
		return image.Rectangle{}
	}
	return seats[i]
}

// essenceRowSeats lays the whole essence row out: the prizes side by side, and the way out standing
// off the end of them.
//
// **One function for the cards and the button**, which is the rule every row in this game follows —
// a control placed by its own arithmetic beside cards placed by theirs is how the two come to
// overlap, which is exactly what happened when the button was left at 88%.
//
// **The prizes touch and the button comes last** *(owner's call, 2026-09-18)*. The essences are the
// question and are compared against each other, so nothing stands between them; the answer that is
// not a card stands where it cannot be mistaken for one. **Its bottom edge is the row's**, because
// a control two fifths of a card tall, centered, floats.
//
// **The row is centered in the two thirds the payout leaves, not on the screen**. Both columns are
// up at once, so a row centered on the screen would stand half in the payout's own column and be
// read as one thing with it.
func essenceRowSeats(gs *state.GlobalState, n int) (prizes []image.Rectangle, button image.Rectangle) {
	top, bottom := gs.PctY(essenceChosenRowPct), essenceRowBottom(gs)

	// The row is n cards and one button, with a gap between every pair.
	width := n*cardWidth + offerButtonWidth + n*essenceRowGap
	x := offerColumnMid(gs) - width/2

	prizes = make([]image.Rectangle, 0, n)
	for i := 0; i < n; i++ {
		prizes = append(prizes, image.Rect(x, top, x+cardWidth, top+cardHeight))
		x += cardWidth + essenceRowGap
	}
	button = image.Rect(x, bottom-offerButtonHeight, x+offerButtonWidth, bottom)
	return prizes, button
}

// essenceRowBottom is the edge everything in this band is aligned to: the essences, the button
// beside them and the last line of the payout in the column to their left.
//
// **One function, because three things share the edge.** A row, a control and a block of type each
// deriving "where does this band end" would be three numbers that agree until one of them moves.
func essenceRowBottom(gs *state.GlobalState) int {
	return gs.PctY(essenceChosenRowPct) + cardHeight
}

// proseColumnMid is the middle of the payout's column, which every narrated line is centered on and
// every figure sets off from.
func proseColumnMid(gs *state.GlobalState) int { return gs.PctX(payoutColumnPct) / 2 }

// offerColumnMid is the middle of what is left, which the essence row stands in.
//
// **Both are derived from the one split**, so widening the payout narrows the offer by exactly as
// much and neither can be written down twice.
func offerColumnMid(gs *state.GlobalState) int {
	left := gs.PctX(payoutColumnPct)
	return left + (gs.ScreenWidth-left)/2
}

// offerRow is the whole row of offered cards: what the seats are cut out of, and what the sort
// block is hung off. **One function rather than the arithmetic twice**, the reason cardBandWidth
// is one on the combat screen — a row and the tabs beside it disagreeing about where the row ends
// is a block standing in the middle of the cards.
func (s *PostBattleScene) offerRow(gs *state.GlobalState) image.Rectangle {
	return offerRowOf(gs, len(s.offer))
}

// offerRowOf is that row **for a stated number of cards**, which is what a slide needs: a row of
// eight is not centered where a row of seven is, so a card leaving one and landing in the other has
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
// that was not enough to see: with the essences lit by the selection, the screen said a card had been
// picked and did not say which one.
//
// **The lift is in this function rather than in the drawing**, so the protruding part of a card is
// clickable rather than merely visible — the drawn-here-clicked-there bug every row in this game is
// shaped to avoid.
func (s *PostBattleScene) offerSlot(gs *state.GlobalState, i int) image.Rectangle {
	at := s.offerSeat(gs, i, len(s.offer))
	if s.isSelected(i) {
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
// block follows — see sortTabRect there. This screen has no control column, and the row is centered
// rather than banded, so the anchor is the row's own right edge and sortColumnGap is the same air
// the hand leaves.
func (s *PostBattleScene) sortTabRect(gs *state.GlobalState, i int) image.Rectangle {
	row := s.offerRow(gs)
	left := row.Max.X + ui.SortColumnGap
	top := row.Min.Y + i*ui.ControlButtonHeight
	return image.Rect(left, top, left+ui.ControlColumnWidth(), top+ui.ControlButtonHeight)
}

// setSort is the press on a tab. **It records the mode and nothing else** — the row is rearranged
// by sortOffer on the same tick, which is where the global state a sort needs is available; a
// button's OnClick reaches none on any screen in this package.
func (s *PostBattleScene) setSort(mode ui.HandSort) { s.sortMode = mode }

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
		return ui.HandLess(s.sortMode, a, b)
	})

	sorted := make([]int, len(s.offer))
	for to, from := range order {
		sorted[to] = s.offer[from]
	}
	s.offer = sorted

	// Nothing in this row stands proud of it: there is no selection on this screen, so the lift
	// every slide carries is zero.
	s.slides = ui.SlidesFor(s.slides, order, func(i int) combat.Card {
		c, _ := card(s.offer[i])
		return c
	}, func(int) int { return 0 }, func(int) int { return 0 })
}

// drawSlides draws the offered cards moving within their row, on the shared mover.
//
// **A sliding card is drawn usable**, whatever the essence could do to it. The dimming says "this one
// cannot be picked", which is a fact about a card sitting in a seat waiting to be clicked; a card
// in flight is not being offered yet, and re-deriving it mid-slide would make the row flicker as
// cards crossed each other.
func (s *PostBattleScene) drawSlides(gs *state.GlobalState, screen *ebiten.Image) {
	ui.DrawCardSlides(gs, screen, s.slides,
		func(gs *state.GlobalState, i, count int) image.Point { return s.offerSeat(gs, i, count) },
		func(sl ui.CardSlide) cards.Spec {
			return ui.CardSpec(sl.Card, ui.HeldByRun(gs, sl.Card), true, false)
		})
}

func (s *PostBattleScene) chosenPrize() (prize, bool) {
	if s.chosen < 0 || s.chosen >= len(s.prizes) {
		return prize{}, false
	}
	return s.prizes[s.chosen], true
}

// chosenEssence is the essence the player picked, if they have.
func (s *PostBattleScene) chosenEssence() (session.Essence, bool) {
	p, ok := s.chosenPrize()
	if !ok {
		return session.Essence{}, false
	}
	return p.essence, true
}

func (s *PostBattleScene) Draw(gs *state.GlobalState, screen *ebiten.Image) {
	ui.FillGround(screen)

	heading := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 34}
	small := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 18}
	prose := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: proseTextSize}

	// **The build is on screen for the whole visit**, every stage of it: what the payout landed on,
	// and what an essence is about to change.
	drawBuildBand(gs, screen, gs.Run.Vitae(), &s.relicDrag, s.tip.Showing())
	drawDeckPile(gs, screen)

	// **The narration stays up for the whole visit, in its own column** — the payout is what the
	// essence beside it is being chosen with, so it does not clear when the offer arrives.
	s.drawProse(gs, screen, prose)

	line := func(y int, face *text.GoTextFace, msg string) {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(gs.PctX(50)), float64(y))
		op.PrimaryAlign = text.AlignCenter
		op.ColorScale.ScaleWithColor(ui.GroundInk)
		text.Draw(screen, msg, face, op)
	}

	// **Neither stage that offers a choice is titled** *(owner's call, 2026-09-05)*. The essences
	// arrive under the sentence saying they are bleeding out, and the card row under the essence that is
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

	s.drawEssences(gs, screen)

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
			if ui.SlideInto(s.slides, i) {
				continue
			}
			// **Every card is selectable and the essences are what go dim** *(2026-09-06)*. The row
			// used to dim a card the chosen essence could not change, which was the same rule read in
			// the other direction — with the card picked first there is no essence yet to ask, so the
			// legality lands on the prize row instead. See essenceSpendable.
			ui.DrawCard(gs, screen, s.offerSlot(gs, i).Min, cards.Hand, card, ui.HeldByRun(gs, card),
				true, s.isSelected(i))
		}
		s.drawSlides(gs, screen)
		s.sortTabs.Draw(gs, screen)
	}

	// The deck panel covers the screen, so nothing of this one may be drawn on top of it.
	s.deck.Draw(gs, screen, ui.OwnedContents(gs))

	// **Bob over everything, and the spotlight with him.** See combat.go's Draw, whose last line
	// this is the counterpart of: the scrim dims what is already drawn, so nothing may follow it.
	s.tut.draw(gs, screen, s)
}

// settledSeats lays the settled row out for n cards, centered.
func settledSeats(gs *state.GlobalState, n int) []image.Rectangle {
	if n < 1 {
		n = 1
	}
	top := gs.PctY(36)
	width := n*cardWidth + (n-1)*essenceRowGap
	x := gs.PctX(50) - width/2

	out := make([]image.Rectangle, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, image.Rect(x, top, x+cardWidth, top+cardHeight))
		x += cardWidth + essenceRowGap
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
// asks what the essence did: `change` was handed the two faces in aimAt and is the only thing that
// knows which of the three shapes this is. A removal ends on an empty seat, a duplicate ends on two
// cards, everything else ends on one.
// **An eaten card leaves nothing** *(owner's call, 2026-09-08)*. It left an outlined hole until
// then, on the argument that a blank gap reads as a layout fault — which is true of a seat that was
// never filled, and not of this one: the player has just watched the card come apart square by
// square, so the emptiness is the thing they were shown rather than something to explain.
func (s *PostBattleScene) drawSettled(gs *state.GlobalState, screen *ebiten.Image) {
	drawLandings(gs, screen, s.lands, s.arrival, s.copied)
}

func (s *PostBattleScene) title() string {
	switch s.stage {
	case settled:
		if s.removes {
			return "EATEN"
		}
		return "CHANGED"
	default:
		return "ESSENCES FLEE"
	}
}

// drawPrizeCard draws one prize where the row says it goes.
func drawPrizeCard(gs *state.GlobalState, screen *ebiten.Image, at image.Point,
	p prize, enabled bool) {

	ui.DrawEssenceCard(gs, screen, at, p.essence, enabled)
}

// drawEssences puts the offer up as cards. **An essence is a card because it is a thing you are given**,
// and the game already has one visual language for that — a name, a border color and a line
// saying what it does.
//
// It borrows `cards.Hand` rather than taking a style of its own: an essence has no cost and no form,
// which that style draws as nothing at all, so what is left is exactly the name and the text. A
// dedicated style is what this wants once an essence has art.
// **A prize is lit exactly when clicking it would take it** *(2026-09-06)*, which is what makes
// select-then-click readable: with no card selected the whole row is dim, and selecting one lights
// the essences that could eat it. It is the same rule the rune pane is under — see
// consumableTarget — and the same rule the offer row itself already followed in the other
// direction.
func (s *PostBattleScene) drawEssences(gs *state.GlobalState, screen *ebiten.Image) {
	for i, p := range s.prizes {
		drawPrizeCard(gs, screen, s.essenceArrivingAt(gs, i), p, s.essenceSpendable(gs, p))
	}
}

func (s *PostBattleScene) hint(gs *state.GlobalState) string {
	switch s.stage {
	case settled:
		if s.removes {
			if len(s.lands) > 1 {
				return fmt.Sprintf("%d fewer cards to draw", len(s.lands))
			}
			return "one fewer card to draw"
		}
		return "into the deck it goes"
	default:
		// **The essence's reach is said in words, because nothing else on the screen says it.**
		// A run wearing a Cloud Necklace is asked for two cards and told so; the row lights the
		// prizes only once it has them, which says when but never how many.
		take := "take one"
		if s.picksLeft > 1 {
			take = fmt.Sprintf("take %d", s.picksLeft)
		}
		if n := s.targets(gs); n > 1 {
			// **Up to, because the reach is a ceiling rather than a quota** — one card is always a
			// legal spend. See consumableTarget.fewest.
			take += fmt.Sprintf(", up to %d cards each", n)
		}
		return fmt.Sprintf("%s - %d cards in your deck, %d vitae in hand",
			take, gs.Run.Size(), gs.Run.Vitae())
	}
}

// updateRelicRow runs the drag over the worn row in the build band.
//
// **A click on a relic does nothing here**, as on the combat screen: this screen's clicks belong to
// the essences it is offering, and a relic that did something on a press would be a second meaning for
// the gesture that reorders it.
func (s *PostBattleScene) updateRelicRow(gs *state.GlobalState) {
	row := buildRelicRow(gs, nil)

	if !gs.CursorAllowed() {
		s.relicDrag.Cancel(row)
		return
	}

	s.relicDrag.Update(gs, row)
}
