package screens

import (
	"fmt"
	"image"
	"math/rand"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/decks"
	"github.com/curiousjc/ascend-duel/internal/entities"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/trace"
	"github.com/hajimehoshi/ebiten/v2"

	"image/color"
)

// Playback pacing at 60 TPS. Destined to become the game-speed setting, and the one number
// that setting will scale.
//
// **Every event holds the screen for exactly this long.** There is deliberately no per-kind
// table and no dwellFor function to add a case to — the previous version had three dwells
// selected by a switch with a `default` arm, and the default was the shortest of them. Every
// event kind added after that switch was written therefore inherited a quarter-second flash
// without anyone choosing it: KindNegated landed there on 2026-08-06 and a Defend blunting a
// Heavy — one of the most consequential things that can happen in a round — went past faster
// than the round-start beat. That is not a tuning mistake, it is a shape that produces
// tuning mistakes, so the shape is gone.
//
// The cost is honest and worth stating: a round-start marker now holds as long as a killing
// blow, and a round is longer to watch than it was. One constant is the price of never
// having a new event kind quietly decide its own pacing.
//
// History for whoever tunes this. Three seconds per action was the original; halved on
// 2026-08-02 because a six-action round took twenty seconds and the pause between a move and
// its consequence read as the game hesitating. Damage was split out on 2026-08-04 for the
// opposite reason — the announcement held for a second and a half while the number it
// produced flashed past in a quarter of one. Both of those were real observations about
// *content*, and both are better answered by one readable dwell than by ranking events.
const (
	ticksPerSecond = 60

	// eventDwellTicks is a second and a quarter — three quarters of a second until
	// 2026-08-07, when it went up by half a second because **an event now carries a sentence
	// rather than a card name**. The Resolution pane made every beat something to read, and
	// three quarters of a second is a glance, not a read. The pane accumulating rather than
	// replacing takes some of the pressure off — a line missed is still on screen — but the
	// live row is the one being followed, and it has to hold long enough to follow.
	eventDwellTicks = 5 * ticksPerSecond / 4
)

// The order enemies are fought in: **every record in the roster** *(2026-08-11)*, shallowest
// floor first, where it was a hand-written list of four. Beat one and the next steps up; lose
// and the same one comes round again.
//
// **It is scaffolding and the tower replaces it wholesale.** MECHANICS.md already decides
// 8 floors x 3 fights with doors between them, so nothing here is a design decision being
// made early — it is a list standing in for a generator. Sorting by `ValidFloors` is what
// makes walking it feel like climbing: a floor-one Goblin comes before a floor-eight
// Bio-Titan because the data says where each belongs, not because someone typed them in that
// order.
//
// **Shuffled inside each floor band since 2026-08-11, so a run opens on a different opponent
// every launch.** The floor ordering is kept and the order *within* a floor is rolled from the
// run seed: fight one is some floor-one enemy rather than always the alphabetically first one,
// and playing over time reaches all 96 by a different route each run.
//
// **It shuffles within the bands rather than shuffling the list**, and that is the whole
// design. A flat shuffle would put a floor-eight Bio-Titan up as the opening fight, which is
// not a hard first fight but an unwinnable one — the failure `tools/balance` exists because of,
// and one that looks exactly like losing to bad draws. Sorted-then-shuffled keeps the climb and
// still varies the group.
//
// **It is still not the tower's randomiser.** Floor generation will pick from the records a
// floor allows off this same stream; this is one shuffle of a stand-in list, and it goes when
// the generator arrives.
//
// Built in Init rather than at package scope: it reads the loaded roster and the run seed out
// of global state, neither of which exists until main has run. Rolled once per launch — a
// death and a retry walk the same order, because there is no Session yet to re-roll a seed.
func (s *CombatScene) roster(gs *state.GlobalState) []string {
	if s.enemyRoster == nil {
		s.enemyRoster = shuffleWithinFloors(data.EnemyOrder(gs.Enemies), gs.Enemies,
			rand.New(rand.NewSource(seeds.For(gs.RunSeed, seeds.EnemySelect))))
	}
	return s.enemyRoster
}

// shuffleSeeds says what the two decks are shuffled from for the fight about to start: the
// pinned pair if deckSeed is set, otherwise a fresh pair rolled from the run seed.
//
// **The salts and the stride left this file on 2026-08-17** for `internal/seeds`, which is the
// one place every stream is now derived and the only place a duplicate salt can be seen. What
// stays here is the *policy* — which is a screen's business — and it is the pin, not the
// derivation.
//
// **Pinning is all-or-nothing.** A pinned player deck against a rolled opponent reproduces
// half a duel, which is worse than reproducing none of it — the hand looks right and the fight
// still differs, so a problem chased with the pin on cannot be trusted to be the same problem.
//
// **Unpinned it is a function of the fight index, not a running stream**, so a defeat and a
// retry deal the same fight again rather than a different one. That matches the enemy roster,
// which is rolled once per launch and walked in the same order after a death — the run has no
// Session yet to re-roll anything, so a retry is a replay of the same fight and both halves
// of it say so.
func (s *CombatScene) shuffleSeeds(gs *state.GlobalState) (player, enemy int64) {
	if deckSeed != 0 {
		return deckSeed, seeds.EnemyDeckPin
	}
	return seeds.ForFight(gs.RunSeed, seeds.PlayerDeck, s.fightIndex),
		seeds.ForFight(gs.RunSeed, seeds.EnemyDeck, s.fightIndex)
}

// shuffleWithinFloors reorders a floor-sorted roster inside each run of equal ValidFloors,
// leaving the bands themselves where they are.
//
// It takes the source rather than reaching for one, so the caller owns which stream is being
// advanced, and it is a plain function of its arguments so a test can hand it a fake roster
// and a fixed seed. Never `rand.Shuffle` — the package-level one draws from a global source
// shared with every other caller and would make a run unreproducible.
func shuffleWithinFloors(names []string, recs map[string]data.EnemyData, rng *rand.Rand) []string {
	out := append([]string(nil), names...)

	for start := 0; start < len(out); {
		end := start + 1
		for end < len(out) && recs[out[end]].ValidFloors == recs[out[start]].ValidFloors {
			end++
		}

		band := out[start:end]
		rng.Shuffle(len(band), func(i, j int) { band[i], band[j] = band[j], band[i] })

		start = end
	}
	return out
}

// The selection is capped at `s.fighter.MaxActions()` cards **regardless of what they
// cost** — the constant that used to live here moved into internal/combat on 2026-08-06.
//
// It moved because it is a **rule**: a round is bounded by cost and by count independently,
// and the opponent's planner has to obey the count exactly as the player's selection does.
// A cap enforced only by the screen was a cap the enemy could ignore. Being a method on
// Duelist also gives a ring or a brand raising it somewhere to bite, which MECHANICS.md
// asks for.
//
// The cap replaced the action-point budget as the gate on selection, and that is the whole
// point: you may now select more than you can afford. Selection had been doing two jobs —
// "queue this for the round" and "this is the one I mean" — and the AP gate made the second
// job impossible, because a hand you could not afford was a hand you could not throw away.
// Over-allocating is how you grab cards to discard.
//
// What stops it being a cheat is that DUEL! goes dead while the selection is over budget.
// The AP rule is never actually broken; it is enforced one step later, at the point of
// playing rather than the point of picking up. See overBudget.

// discardsPerRound caps how many times cards can be thrown back in one round, and refills
// at the end of each round. One press costs one discard however many cards were selected,
// which is what makes the size of the selection a decision rather than a formality: four
// presses of one card and one press of four cost the same.
//
// It matters more since the hand stopped emptying every round. Discard used to be a way of
// getting a fresh deal slightly early; it is now the *only* way an unwanted card leaves your
// hand, so this number is the rate at which a hand can be steered rather than a convenience.
// Four against a hand of eight is deliberately generous for now — a number to play against,
// not a balanced one.
const discardsPerRound = 4

// screenGround is what the whole combat screen is painted on, and **it went cream on
// 2026-08-14**, from the {50,50,50} dark grey it had been since the screen existed.
//
// **Everything drawn straight onto it had assumed a dark ground**, which is the cost of the
// change and the reason it is a named constant now rather than a literal in Draw. Three
// figures were white and are now `groundInk`; the action-point bar's empty cells were
// `ColorAtStrength(apBarColor, 20)`, which scales toward black and therefore came out *louder*
// than the filled cells on a light ground — exactly the failure `systems.ColorToward` was
// written for; and the ring row's backing had to stop being one step *lighter* than the ground
// and become one step darker.
//
// **It is deeper than the cards stand on** — `cards.Surface` is {240,239,234} and the
// Resolution pane's fill is {234,230,224} — because a card, a pane and the table cannot all be
// the same off-white or the objects stop having edges. The warmth is where the separation
// comes from: the ground is the yellowest of the three.
var screenGround = color.RGBA{R: 226, G: 208, B: 176, A: 255}

// groundInk is for text written straight onto the ground rather than onto a card, a pane or a
// button — the action-point figure, the draw pile's count and the ring row's fraction. Near
// black and slightly warm, so it belongs to the cream rather than sitting on it.
//
// Anything drawn on a surface of its own takes that surface's ink instead; this is only for
// the three figures with nothing behind them.
var groundInk = color.RGBA{R: 44, G: 40, B: 34, A: 255}

// apBarColor is the action-point bar's blue. It is deliberately not the palette's green:
// the bar reports the budget rather than belonging to the cards, and giving it its own
// colour stops it reading as a summary of the list underneath it.
var apBarColor = color.RGBA{R: 70, G: 130, B: 230, A: 255}

// apOverColor paints the part of the selection the budget will not cover. Red rather than a
// dimmer blue: over-allocation is a state you have to leave before you can duel, so it reads
// as a warning rather than as more of the same bar.
var apOverColor = color.RGBA{R: 225, G: 60, B: 60, A: 255}

// CombatScene runs one duel: the player plans a set of actions against an action
// point budget, presses DUEL!, watches the round play out, and plans again.
//
// The fields below are the screen's own working state. None of it belongs to any
// other screen, and the action box will add more of it.
type CombatScene struct {
	fighter *entities.Combatant
	enemy   *entities.Combatant

	// The queued sets for the coming round. fighterActions is derived from the hand by
	// syncQueue and never written directly; enemyActions is re-planned each round.
	fighterActions []combat.Card
	enemyActions   []combat.Card

	// The player's deck, in three piles. hand is what the action box draws and the only
	// one the player touches; deck is the draw pile and discard is what has been spent.
	// A card played this round moves to discard when the round resolves.
	deck    []actionCard
	hand    []paletteCard
	discard []actionCard

	// The shuffle source. Explicit and carried on state rather than the math/rand
	// package-level functions, which draw from a global shared with every other caller
	// and would make a run unreproducible. Seeded once in Init.
	rng *rand.Rand

	// The rules' own source, handed to ResolveRound. **A sixth stream, and separate from every
	// other one on purpose** — it is advanced per attack phase by the shock roll, so sharing it
	// with either shuffle would make a hand a function of how many attacks had been rolled
	// against, and every entry in seeds.go would break the first time lightning landed.
	//
	// Seeded from the run seed with its own salt, so a replayed run rolls the same shocks.
	combatRNG *rand.Rand

	// The opponent's deck, in the same three piles. It lives in internal/decks rather than
	// here because tools/balance plays whole duels headlessly and cannot import this
	// package — see that package's header.
	//
	// **It carries its own shuffle source, separate from s.rng above**, and the reason is
	// the seed catalogue. CLAUDE.md names "card shuffles" as one determinism stream; sharing
	// it between the two sides would make the player's opening hand a function of how many
	// cards the enemy happened to draw, and every entry in seeds.go would break the first
	// time an enemy deck was retuned. A named hand has to stay a fact about the player's
	// deck alone.
	enemyPile *decks.EnemyPile

	// The card currently being dragged, if any. See combat_actionbox.go.
	drag *dragState

	// showDeck toggles the deck overlay. While it is up the cards underneath do not
	// respond, so reading the deck cannot accidentally re-plan the round.
	showDeck bool

	// How many ticks the mouse has been held down on the Resolution feed. The box is
	// expanded while this is past longPressTicks — a count rather than a bool, because
	// "expanded" is then derived and there is no second state to fall out of step. See
	// updateFeed.
	feedPressTicks int

	// Cards currently travelling to or from the draw pile. Purely something to look at:
	// every one of them is a ghost of a card that has already moved. See combat_flight.go.
	flights []cardFlight

	// Cards currently moving from one slot in the hand to another — a sort, or the row
	// closing up after cards were spent. Separate from flights rather than a fourth flag on
	// one, because a slide is the only mover whose journey begins and ends in the row, and it
	// is the only one that needs a row size at each end.
	slides []handSlide

	// The player's side of the table: the cards played this round, in resolution order, flying
	// out of the hand and into a row on the left facing the opponent's. Dealt in full the
	// moment the round starts — see seatPlayedCards — and what a combo brackets.
	//
	// Cleared when the hand is spent, which is the moment those cards actually leave.
	resolved []resolvedCard

	// The opponent's side of the table: their whole queue for the coming round, flying in from
	// their own card in the top-right corner and settling into a row on the right.
	//
	// **It is laid out when the opponent plans, which is the start of the planning phase**, so
	// the player picks their round against a hand they can see. Re-seated once per round; see
	// planEnemyRound.
	enemyDealt []dealtCard

	// firingSeats and enemyFiringSeats are which cards on each side of the table are resolving
	// right now, empty for none. **Playback drives which cards are lit, not which cards exist**:
	// both hands are on the table from the moment DUEL! is pressed.
	//
	// Two fields rather than a side-plus-seat pair, because both rows are drawn independently
	// and each only ever asks about itself. `noteResolved` writes both on every action, so only
	// one side is ever lit at a time.
	//
	// **A list rather than one seat, because the attack phase is one blow** *(2026-08-14)*. The
	// cards announce one after another and stay up as they do, so the whole hand is raised by the
	// time the combo lands on it — and the combo then drops whichever of them earned nothing. A
	// single lit seat could only ever say "this card", which is the reading the one-blow rule
	// exists to stop.
	firingSeats      []int
	enemyFiringSeats []int

	// The fighter's own resources, drawn in the character block. discardsLeft refills
	// every round. **Vitae is the run's, not the screen's** *(2026-08-17)* — see session.Session,
	// which is what the post-battle screen pays into.
	discardsLeft int

	// fightIndex is how far along enemyRoster the player has got.
	//
	// **The run owns this now** *(2026-08-17)* — `session.Session.Fight()` — because the
	// post-battle screen seeds its offer from it and a number living on one scene is invisible to
	// every other. This is a per-visit copy, read in Init, so the draw paths and the seed
	// arithmetic can go on reading a plain int.
	//
	// restart is what a defeat raises: a retry re-enters this screen rather than changing to
	// another one. A flag rather than a direct call because it is raised from a button's OnClick,
	// which takes no arguments and so cannot reach the global state Init needs.
	fightIndex int
	restart    bool

	// won is the other exit: the fight was won, so the run advances and the post-battle screen
	// takes over. Same shape as restart and for the same reason — raised by a button, consumed by
	// Update with the global state in hand.
	won bool

	// enemyRoster is the fight order, built once from the loaded records. On the scene
	// rather than at package scope because it reads global state, which does not exist
	// until main has run. See roster.
	enemyRoster []string

	// tracedHand is the hand size the last layout dump described. The whole bottom band is
	// a function of that number — the pitch, the row, the AP bar, the caption box — so a
	// change to it is exactly when the dump is worth repeating. Watching the size rather
	// than flagging every place that changes it means no call site has to remember.
	tracedHand int

	// The resolved round and the playback cursor walking it. The screen never
	// computes an outcome — it replays this.
	log    []combat.Event
	cursor int
	ticks  int
	round  int

	// The authoritative end-of-round state, adopted once playback catches up. Guard
	// flags in particular only exist here, since no event carries them.
	fighterAfter combat.Duelist
	enemyAfter   combat.Duelist

	duelButton    *models.Button
	discardButton *models.Button

	// How the hand is arranged, and the column of buttons that chooses it. See
	// combat_sort.go — including why this is the one field on the scene that Init does not
	// reset: it is a reading preference rather than a fact about a duel.
	sortMode    handSort
	sortButtons []*models.Button
}

// Init prepares a fresh duel. Safe to re-enter: the combatants and the button are
// built once, everything else resets every visit.
func (s *CombatScene) Init(gs *state.GlobalState) {
	if s.fighter == nil {
		s.fighter = duelistFromRecord(gs, "Fighter1")

		// **What the player is wearing is part of hydrating them**, not part of resetting a
		// duel: rings are run-level and a fight does not take them off. See startingRings for
		// why the set is a constant here rather than something a Session hands over.
		s.fighter.Duelist = equipRings(gs, s.fighter.Duelist)
	}

	// The enemy is rebuilt every visit rather than once, because fightIndex may have moved
	// since the last one. Init is how the next fight starts, not only how the screen is
	// entered — see nextFight.
	// The run says which room this is; the scene keeps a copy for the frame. See fightIndex.
	s.fightIndex = gs.Run.Fight()
	s.enemy = enemyFromRecord(gs, s.roster(gs)[s.fightIndex%len(s.roster(gs))], s.fightIndex)

	// The scene builds its own widgets and wires them to its own methods, so no other
	// package needs to know this screen has buttons or what pressing them means.
	if s.duelButton == nil {
		s.duelButton = models.NewButton(stripButtonWidth, stripButtonHeight, "DUEL!", s.startRound)
		s.duelButton.BaseColor = color.RGBA{R: 220, G: 20, B: 60, A: 255} // crimson
	}
	if s.discardButton == nil {
		s.discardButton = models.NewButton(stripButtonWidth, stripButtonHeight, "Discard", s.discardSelected)
		s.discardButton.BaseColor = color.RGBA{R: 225, G: 200, B: 60, A: 255} // yellow
	}
	// **The bottom strip is one row of four things, spaced rather than placed** *(2026-08-11)*:
	// the AP figure at the hand's left edge, the two buttons, and the deck pile at the right.
	// The buttons no longer sit at percentages of the screen — `buttonStripSlots` divides
	// what is left between the figure and the pile into three equal gaps, so the strip stays
	// evenly spread if any of the three fixed things moves.
	//
	// **They are deliberately not adjacent any more.** Discard and DUEL! were side by side
	// because they are the same choice made two ways; they are separate choices now, and the
	// spacing says so. Discard briefly sat on the hand's left edge, which is the AP figure's
	// column — the figure came back on 2026-08-11 and wanted it.
	//
	// **There is no third button.** Deck was one until 2026-08-10 and is now the pile itself.
	// See combat_flight.go.
	// The sort column, beside the cards rather than on the strip below them: it arranges the
	// hand and commits nothing, so it belongs against the thing it arranges. **s.sortMode is
	// deliberately not reset here** — see combat_sort.go.
	if s.sortButtons == nil {
		s.buildSortButtons()
	}
	for i, b := range s.sortButtons {
		at := sortButtonCentre(gs, i)
		b.ScreenX, b.ScreenY = at.X, at.Y
	}

	discardX, duelX := buttonStripSlots(gs, s.discardButton.Width, s.duelButton.Width)
	s.discardButton.ScreenX = discardX
	s.discardButton.ScreenY = gs.PctY(buttonStripPct)
	s.duelButton.ScreenX = duelX
	s.duelButton.ScreenY = gs.PctY(buttonStripPct)

	s.showDeck = false
	s.feedPressTicks = 0
	s.flights = nil
	s.slides = nil
	s.firingSeats, s.enemyFiringSeats = nil, nil

	// **The table is cleared here as well as by the spend** *(2026-08-16)*. `s.resolved` used to
	// be emptied in exactly two places — `seatPlayedCards` at the start of a round and
	// `spendSelected` at the end of one — and that covered every case only because every round
	// ended in a spend. A settled duel now freezes instead (see endOfRound), so the last round's
	// cards were still seated when the next fight started: they drew over the new table, and
	// `resolvedInHand` blanked the hand slots they claimed, so the fresh hand came up with holes
	// in it. `enemyDealt` goes with it — `planEnemyRound` below rebuilds it, and a scene that
	// clears one half of the table and not the other is a trap for the next change.
	s.resolved = nil
	s.enemyDealt = nil
	s.restart = false
	s.discardsLeft = discardsPerRound

	// A fresh deck every visit rather than continuing a run that has been abandoned, and a
	// fresh *shuffle* per fight — the seeds come from the run seed unless deckSeed pins them.
	playerSeed, enemySeed := s.shuffleSeeds(gs)
	s.rng = rand.New(rand.NewSource(playerSeed))
	s.combatRNG = rand.New(rand.NewSource(seeds.For(gs.RunSeed, seeds.CombatRoll)))
	s.resetDeck(gs.Run)

	// The queue starts empty every visit and is derived from what is selected in hand.
	// DUEL! is disabled until something is in it.
	s.fighterActions = nil
	s.drag = nil

	// A fresh shuffled deck for the opponent too, dealt before it plans, off its own stream.
	s.enemyPile = decks.NewEnemyPile(s.enemy.Record, enemySeed, decks.EnemyHandSize)

	// A fresh duel: full life, no standing defenses, and no action points banked by a
	// Prepare from a duel that has been walked away from.
	s.fighter.CurrentLife = s.fighter.MaxLife
	s.enemy.CurrentLife = s.enemy.MaxLife
	s.fighter.Duelist = resetCombatState(s.fighter.Duelist)
	s.enemy.Duelist = resetCombatState(s.enemy.Duelist)

	s.log = nil
	s.cursor = 0
	s.ticks = 0
	s.round = 0

	// **The opponent plans for round one here, and this is a commitment now** *(2026-08-12)*.
	// It used to be display only — startRound re-planned every round regardless — and it is the
	// plan that will actually resolve. See planEnemyRound.
	//
	// **It has to come after the reset above, and both halves of that matter.** planEnemyRound
	// refuses to plan for a dead duelist, and a screen re-entered after a defeat still has a
	// corpse on it until the line above brings it back — so planning first would deal the new
	// fight an opponent with no cards. The budget the planner reads has to be the fresh one too,
	// or round one is planned against a Prepare banked in a duel that is over.
	//
	// **It spends cards from the opponent's hand**, which is why Init deals that hand first.
	// A plan is a commitment on either side.
	s.planEnemyRound()

	trace.Logf("scene", "fight %d: %s, %d cards, %d life, %d AP",
		s.fightIndex+1, s.enemy.Name, len(decks.EnemyCards(s.enemy.Record)),
		s.enemy.MaxLife, s.enemy.ActionPoints())
	trace.Logf("scene", "combat init: deck %d hand %d discard %d, seeds player %d enemy %d",
		len(s.deck), len(s.hand), len(s.discard), playerSeed, enemySeed)
	s.tracedHand = len(s.hand)
	s.traceLayout(gs)
}

// resetCombatState clears everything a duel accumulates, leaving the stats a combatant was
// hydrated with. Init re-enters a screen that may have been left mid-duel, so a standing
// defence or a banked Prepare would otherwise be inherited by the next one.
//
// It sets the fields by name rather than rebuilding the struct: Con/DMG/Spd/MaxLife come
// from the data record and must survive, and a zero literal here would quietly wipe them
// the first time someone re-entered the screen.
//
// **The defences are cleared by `combat.ClearDefenses` rather than field by field.** This
// function listed them by hand until 2026-08-14 and had fallen behind the rules package, so a
// raised defence survived into the next fight — which is exactly the failure a screen
// enumerating another package's state invites. The two fields below are the same hazard and are
// listed here because they are the screen's to reset, not the engine's.
// **The statuses go too, and the rings stay** *(2026-08-16)*. A burn is something one duel did
// to you; a ring is something you are wearing, and clearing it here would strip the player between
// fights. Both are fields on `combat.Duelist` and the difference between them is what this
// function exists to know.
func resetCombatState(d combat.Duelist) combat.Duelist {
	d = combat.ClearDefenses(d)
	d.BonusAP = 0
	d.GatheredAP = 0
	d.BonusDraw = 0
	d.DrewCards = 0
	d.Statuses = [combat.ElementCount]combat.Status{}
	return d
}

// duelSettled reports that the fight is over *and* has finished being watched. Both halves
// matter: life reaches zero partway through playback, and offering the way out before the
// killing blow has been drawn would cut the round short at its most interesting moment.
func (s *CombatScene) duelSettled() bool {
	return s.cursor >= len(s.log) && (!s.fighter.Alive() || !s.enemy.Alive())
}

// nextFight arms the restart. It cannot re-init the screen itself — a button's OnClick takes
// no arguments and Init needs the global state — so it raises a flag that Update acts on
// with the pointer already in hand.
//
// **Winning leaves the screen; losing stays on it.** A win goes to the post-battle scene, which
// offers one alteration to the deck and sends the player back here for the next room — so the
// advance along the roster is the run's (`WonFight`), not this screen's. A defeat has nothing to
// award, so it re-enters directly and puts the same opponent back up.
func (s *CombatScene) nextFight() {
	if !s.duelSettled() {
		return
	}
	if s.enemy.Alive() {
		s.restart = true
		return
	}
	s.won = true
}

func (s *CombatScene) Update(gs *state.GlobalState) error {
	// A restart re-enters this screen rather than changing to another one, so it calls Init
	// directly instead of setting gs.NewScreen — that flag belongs to screen changes, and
	// Init is documented as safe to re-enter.
	// The scripted-demo driver, empty in every build but `-tags demoplay`.
	s.demoUpdate(gs)

	if s.restart {
		s.restart = false
		s.Init(gs)
		return nil
	}

	// A won fight advances the run and hands over to the post-battle screen, which comes back
	// here when the player has taken their alteration. The advance is here rather than in
	// nextFight so the run moves once, on the tick the screen actually leaves.
	if s.won {
		s.won = false
		gs.Run.WonFight()
		gs.ActiveScreen = state.PostBattle
		gs.NewScreen = true
		return nil
	}

	// Cards in the air, and the pile they come from. Both are outside every branch below on
	// purpose: a flight that started before the killing blow should still land, and the deck
	// stack is the only control that survives its own overlay — it is what closes it.
	s.updateFlights()
	s.updateSlides()
	s.updateResolved()
	s.updateDeckStack(gs)

	// Tell the frame a dialog is up, so the game's own chrome stands down rather than sitting
	// live on top of it. Written unconditionally from what the screen already knows, never
	// toggled — see state.ModalOpen. The deck overlay is the only dialog in the game.
	gs.ModalOpen = s.showDeck

	// The long press on the Resolution feed. Outside every branch below for the same reason
	// the flights are: reading back what just happened is not an action, and it has to work
	// while a round plays and after one side is down.
	s.updateFeed(gs)

	// The overlay swallows card interaction, so reading the deck cannot re-plan the round
	// through the panel covering it. The buttons stay live — one of them is how it closes.
	if !s.showDeck {
		s.updateActionBox(gs)
	}

	// Above the branch below, because the column is live under exactly one condition and it is
	// its own: the hand may be rearranged whenever it may be edited. It goes dead once the duel
	// is settled for the same reason it goes dead during playback — see updateSortButtons.
	s.updateSortButtons(gs)

	// Once the duel is decided the DUEL! button becomes the way onward, in place. Reusing
	// the same button rather than adding a fourth is the point: it is the same slot for
	// "commit and move the game forward", and a control that appears only at the end of a
	// fight would be a control nobody has learned.
	if s.duelSettled() {
		s.duelButton.Text, s.duelButton.OnClick = "Retry", s.nextFight
		if !s.enemy.Alive() {
			s.duelButton.Text = "Next"
		}
		setEnabled(s.duelButton, !s.showDeck)
		setEnabled(s.discardButton, false)

		systems.UpdateButton(gs, s.duelButton)
		systems.UpdateButton(gs, s.discardButton)
		return nil
	}
	s.duelButton.Text, s.duelButton.OnClick = "DUEL!", s.startRound

	// Both act on the selection, so both need one — pressing either is deciding what the
	// selection was for. An empty queue is mechanically legal for DUEL! — ResolveRound
	// handles it — but it means standing still while being hit, which is not something to
	// offer by accident.
	//
	// Both go dead while the deck is open. It is a dialog: Deck is the only live control
	// on the screen until it is pressed again.
	//
	// The two no longer share a rule. Discard needs a discard left this round; DUEL! needs
	// the selection to be inside the action-point budget. Each has its own reason to go dead
	// and each reason is visible somewhere on screen — the count in the character block, the
	// bar going red — so a dark button is never unexplained.
	//
	// This is what makes over-allocating safe to allow: the budget is enforced here, at the
	// point of playing, rather than at the point of picking a card up.
	live := s.planning() && len(s.fighterActions) > 0 && !s.showDeck
	setEnabled(s.duelButton, live && !s.overBudget())
	setEnabled(s.discardButton, live && s.discardsLeft > 0)

	systems.UpdateButton(gs, s.duelButton)
	systems.UpdateButton(gs, s.discardButton)
	s.advancePlayback()

	if trace.Enabled() && len(s.hand) != s.tracedHand {
		s.tracedHand = len(s.hand)
		s.traceLayout(gs)
	}
	return nil
}

// setEnabled flips a button between disabled and normal without clobbering a hover or
// press it is in the middle of.
func setEnabled(b *models.Button, enabled bool) {
	if !enabled {
		b.State = models.ButtonStateDisabled
		return
	}
	if b.State == models.ButtonStateDisabled {
		b.State = models.ButtonStateNormal
	}
}

// startRound resolves a single round and hands playback an event log. It does not
// run the duel to a conclusion — control returns to the player to re-plan.
func (s *CombatScene) startRound() {
	// Ignore the press while a round is still playing back, or once someone is down.
	if s.cursor < len(s.log) {
		return
	}
	if !s.fighter.Alive() || !s.enemy.Alive() {
		return
	}
	// The button is already dead while over budget; this is the rule itself rather than the
	// reporting of it, so a round can never resolve a queue the fighter cannot pay for.
	if s.overBudget() {
		return
	}

	s.round++

	// **The opponent is not planned here any more** *(2026-08-12)*. It planned at the start of
	// this planning phase, its cards have been face up on the table the whole time the player
	// was choosing, and re-planning at the press would make the row a picture of a round that
	// never happened. Nothing about the plan changes by waiting — PlanFor never sees the
	// player's queue, and the opponent's own state has not moved since the last round ended —
	// so this is purely a change to *when* the player is told.
	log, fighterAfter, enemyAfter := combat.ResolveRound(
		s.fighter.Duelist, s.enemy.Duelist,
		s.fighterActions, s.enemyActions,
		s.round,
		s.combatRNG,
	)

	s.fighterAfter = fighterAfter
	s.enemyAfter = enemyAfter
	s.log = log
	s.cursor = 0
	s.ticks = 0

	// Both hands go to the table now, not as the round plays out. The opponent's is known in
	// full at this moment and is drawn from enemyActions directly; the player's is dealt out of
	// the hand by the flights seatPlayedCards raises. Nothing here decides anything — the round
	// above is already resolved. See combat_table.go.
	s.firingSeats, s.enemyFiringSeats = nil, nil
	s.seatPlayedCards()

	// The whole round, not a count of it. ResolveRound already decided every one of these
	// before a frame of playback ran, so this is the authoritative account of what is about
	// to happen on screen — and if the screen ever disagrees with it, the screen is wrong.
	if trace.Enabled() {
		trace.Section(fmt.Sprintf("round %d", s.round))
		trace.Logf("round", "fighter %d AP, plan %s", s.fighter.ActionPoints(), planLabel(s.fighterActions))
		trace.Logf("round", "enemy   %d AP, plan %s",
			s.enemy.ActionPoints(), planLabel(s.enemyActions))
		for i, e := range log {
			trace.Logf("event", "%2d %s", i, eventLabel(e))
		}
	}
}

// eventLabel renders one event, **printing only the fields that kind actually sets**.
//
// combat.Event is one struct covering six kinds, so most of its fields are zero on any
// given event: Action is set on KindAction alone, Amount and Target on the damage-ish ones.
// Printing them all made every round-start read "side A Strike amount 0 life 0", which is
// four facts of which none were true. A trace that invents detail is worse than one that
// omits it — the whole reason for having it is to be believed.
func eventLabel(e combat.Event) string {
	switch e.Kind {
	case combat.KindRoundStart:
		return fmt.Sprintf("round-start round %d", e.Round)
	case combat.KindRoundEnd:
		return fmt.Sprintf("round-end   round %d", e.Round)
	case combat.KindAction:
		return fmt.Sprintf("action      %v plays %v %v (%v)", e.Side, e.Element, combat.ConceptOf(e.Action).Label, combat.Plain(e.Action).Category())
	case combat.KindStatus:
		return fmt.Sprintf("status      %v puts %d %v on %v", e.Side, e.Amount, e.Element, e.Target)
	case combat.KindMissed:
		return fmt.Sprintf("missed      %v's %v never lands - shocked", e.Side, e.Action)
	case combat.KindBurned:
		return fmt.Sprintf("burned      %v takes %d, leaving %d", e.Target, e.Amount, e.Life)
	case combat.KindGathered:
		return fmt.Sprintf("prepared    %v banks %d AP for next round", e.Side, e.Amount)
	case combat.KindNegated:
		return fmt.Sprintf("negated     %v's %v cuts it to %d", e.Side, e.Action, e.Amount)
	case combat.KindDamage:
		return fmt.Sprintf("damage      %v hits %v for %d, leaving %d", e.Side, e.Target, e.Amount, e.Life)
	case combat.KindCombo:
		return fmt.Sprintf("attack      %v forms %s (x%d.%02d)",
			e.Side, comboName(e), e.Multiplier/100, e.Multiplier%100)
	case combat.KindChilled:
		return fmt.Sprintf("chilled     %v loses its %v", e.Side, e.Action)
	case combat.KindDefeated:
		return fmt.Sprintf("defeated    %v falls to %v", e.Target, e.Side)
	default:
		return fmt.Sprintf("kind %d?", e.Kind)
	}
}

// advancePlayback walks the round's event log one entry at a time, applying each to
// the on-screen combatants. This is the whole of the screen's combat logic: the round
// was already decided by combat.ResolveRound, so playback can never disagree with it.
func (s *CombatScene) advancePlayback() {
	if s.cursor >= len(s.log) {
		return
	}

	// One dwell, every event, no exceptions. See eventDwellTicks for why this is not a
	// lookup on the event's kind.
	s.ticks++
	if s.ticks < eventDwellTicks {
		return
	}
	s.ticks = 0

	s.applyEvent(s.log[s.cursor])
	s.cursor++

	// Playback has caught up with the resolver: hand control back to the player, or freeze
	// the screen if there is nobody left to play against. See endOfRound.
	if s.cursor >= len(s.log) {
		s.endOfRound()
	}
}

// endOfRound is what happens the moment playback catches up with the resolver: the
// authoritative end-of-round state is adopted, the hand is spent and refilled, and the
// opponent plans its answer.
//
// **A settled duel does none of the last two, and freezes instead** *(2026-08-16)*. Spending
// the hand takes cards out of the row, and the row is the thing half the lower screen is
// measured from — `handBand` is a function of how many cards are in it, the AP bar spans that
// band and the Resolution feed's bottom edge comes off the same row. So the last frame of a
// won fight used to reflow: the hand collapsed to a narrow centred huddle, the bar shrank to
// match, and the feed moved under it. **The picture the player is looking at when the killing
// blow lands is the picture they should still be looking at when they press Next**, so nothing
// moves — the played cards stay on the table, the hand keeps its gaps, and `Init` clears all of
// it when the next fight starts.
//
// It is a branch here rather than a rule inside `drawHand`, because what has to stop is not the
// *drawing*, it is the whole end-of-round movement. A hand that was spent but not refilled still
// reflows.
func (s *CombatScene) endOfRound() {
	s.fighter.Duelist = s.fighterAfter
	s.enemy.Duelist = s.enemyAfter

	if s.duelSettled() {
		return
	}

	// The hand is spent here rather than at resolve time, and the ordering matters:
	// endRoundHand rebuilds fighterActions from what is left, and the Resolution pane draws
	// fighterActions to narrate the round. Spending it while playback was still running would
	// empty the pane mid-round.
	s.endRoundHand()

	// **The opponent plans the next round the instant this one is over**, so its cards are
	// on the table while the player chooses their answer. It has to happen *after* the two
	// duelists above adopt their end-of-round state, or the plan is made against a budget
	// that no longer exists — a chill landing this round has to be in the AP the planner
	// reads.
	s.planEnemyRound()
}

// planEnemyRound asks the opponent for its round and lays the cards on the table.
//
// **The plan is a commitment made at the start of the planning phase, not at DUEL!**
// *(2026-08-12)*. Two things follow and both are the point:
//
//   - The player picks their round against a hand they can see, which is what the table was
//     built to make possible and what the opponent's row was missing.
//   - The opponent's intentions stop being hidden information. `concealEnemy` still governs the
//     Action Flow pane, but with the cards face up there is nothing left for it to hide.
//
// **Nothing about the plan itself changed.** `PlanFor` never sees the player's queue and the
// opponent's own state does not move between the end of one round and the press of DUEL!, so
// the same cards are chosen either way — what moved is when the player is shown them.
//
// A dead duelist plans nothing: the last round's row stays on the table, which is the round the
// player is looking at the result of.
func (s *CombatScene) planEnemyRound() {
	if !s.fighter.Alive() || !s.enemy.Alive() {
		return
	}
	// **Nothing on a freshly dealt row is firing.** The seat lists are written by playback and
	// nothing else clears them between rounds, so the last turn's raised cards would come back up
	// under the new plan — cards the player had not committed, standing as though they had — and
	// drop again at DUEL!, which is where the lists were being cleared. They are cleared here, on
	// the same line as the row they describe.
	//
	// **After the alive check, not before it.** A finished duel keeps its last round on the table
	// with the killing blow still raised; that row is the result the player is looking at.
	s.firingSeats, s.enemyFiringSeats = nil, nil

	s.enemyActions = s.enemyPile.Plan(s.enemy.Duelist)
	s.seatEnemyCards()
}

// applyEvent moves the visible state to match one event. Only damage moves the health
// bars; the other two arms are the animation cues this always anticipated.
//
// **None of this decides anything.** ResolveRound settled the whole round before playback
// began, so these are three ways of showing the same log and the screen could stop calling
// any of them without changing a result.
func (s *CombatScene) applyEvent(e combat.Event) {
	// A card of the player's has fired: lift it out of the hand and start it toward the pile.
	s.noteResolved(e)

	// A combo has formed: bracket the cards the engine says formed it.
	s.noteCombo(e)

	// A status has landed: put its badge on the card at the beat it was announced.
	//
	// **This is a drawing, and it is overwritten a few frames later.** The authoritative statuses
	// arrive with `s.enemyAfter` when playback finishes; what is written here is the same fact
	// arriving early, so the badge appears on the line that says it landed rather than after the
	// round is over. `Rounds` is set to 1 for no better reason than that `Active()` needs one —
	// nothing on this screen reads a duration, and the engine's own count replaces it.
	//
	// The same argument as the burn below: the alternative is a card that disagrees with the
	// sentence next to it.
	if e.Kind == combat.KindStatus {
		s.applyStatusBadge(e)
		return
	}

	// **A burn changes a life total without anybody acting**, so it has to be applied here
	// alongside damage rather than being a consequence of a card. Missing it would leave the two
	// fighter cards showing a life the engine has already spent — and a duelist who dies to a
	// fire tick would fall with a health bar that never moved.
	if e.Kind != combat.KindDamage && e.Kind != combat.KindBurned {
		return
	}

	if e.Target == combat.SideA {
		s.fighter.CurrentLife = e.Life
	} else {
		s.enemy.CurrentLife = e.Life
	}
}

// applyStatusBadge shows one landed status on the target's card, mid-playback.
//
// **Only the enemy card draws badges today** and this writes to both anyway, because which card
// shows what is `EnemyStyle`'s business and not this function's — the duelist's statuses are
// already tracked on its duelist for every other purpose, and having the screen hold two
// different ideas of what is standing on a combatant is how the two come to disagree.
func (s *CombatScene) applyStatusBadge(e combat.Event) {
	target := s.enemy
	if e.Target == combat.SideA {
		target = s.fighter
	}
	if e.Element <= combat.Basic || int(e.Element) >= combat.ElementCount {
		return
	}
	target.Statuses[e.Element] = combat.Status{Amount: e.Amount, Rounds: 1}
}

// **There is no caption box**, and the slot above the hand is the Resolution feed instead.
//
// What that costs, stated rather than discovered later: nothing on screen says *why* a dark
// DUEL! button is dark. The AP bar turning red says that something is wrong, not what to do
// about it, and there is no prompt naming the next enemy — DUEL! relabelling itself Next or
// Retry is what carries that.
//
// Both were the owner's call, made knowing the cost. If either is wanted back it is a line
// somewhere else, not a box.

func (s *CombatScene) Draw(gs *state.GlobalState, screen *ebiten.Image) {
	screen.Fill(screenGround)

	// **The top of the screen is one row of three things** *(2026-08-12)*: the player's card
	// in the left corner, the enemy's in the right, and the rings filling everything between
	// them. All three share a top edge and the two cards are the same format, so the row reads
	// as the two sides of the fight with what you are wearing laid out between them.
	//
	// The player was a framed block of captions until this landed and the enemy was floating
	// at 88%,34%; see drawDuelistCard and drawEnemyCard for why each moved.
	s.drawDuelistCard(gs, screen)

	// Where in the tower this fight is, in the band the card leaves above the table. See
	// drawTowerPlace — it is under the duelist card because the floor is something about the
	// run rather than about either fighter, and that corner is where the run's own figures
	// already are.
	s.drawTowerPlace(gs, screen)

	// **The ring pane is a sketch** *(2026-08-11)* — it draws what `data/rings.json` defines
	// and nothing equips, buys or reads one. Its width is what the two cards leave; see
	// combat_rings.go.
	s.drawRingPane(gs, screen)

	s.drawEnemyCard(gs, screen)
	systems.DrawButton(gs, screen, s.duelButton)
	systems.DrawButton(gs, screen, s.discardButton)
	s.drawSortButtons(gs, screen)
	s.drawDiscardsLeft(gs, screen)
	s.drawDeckStack(gs, screen)

	// **Order below is contested, and the ranking is written down because it will be
	// re-broken otherwise** *(2026-08-11)*. Three things want to be on top of each other and
	// they cannot all win:
	//
	//  1. The feed over the enemy and its health bar. An expanded feed reaches 12% and the
	//     opponent sits at 34%, so a box the player is holding open to read would otherwise
	//     have a monster drawn through it. Hence Resolution after drawCombatant.
	//  2. A selected card over the feed. Selection lifts a card 26px into the box's bottom
	//     21 — see feedGapAboveCards — and the card is the thing being acted on. Hence the
	//     hand row after Resolution.
	//  3. A firing card over the inert hand row, at full size. Unchanged, and why the
	//     resolved pile is still drawn after the row.
	//
	// **What loses is a firing card passing over an expanded feed**, and it is the right one
	// to give up: 1 and 2 are on screen constantly, that is only during playback with the box
	// held open, and the card holds above y=467 so the newest lines stay clear of it.
	//
	// EXPERIMENT 2026-08-07: Action Flow is not drawn, and Resolution has taken its column as
	// well as its own. drawActionFlow and actionFlowRows are deliberately left in place and
	// unwired so this is one line to put back.
	//
	// **What this gives up, if it stays:** the enemy's queued shape while planning. Those
	// `??? (attack)` rows are the tell — see the concealment section of the combat-screen
	// skill — and Resolution is empty until DUEL! is pressed, so nothing on screen says what
	// the opponent is about to do.
	s.drawResolution(gs, screen)
	s.drawHandRow(gs, screen)

	// Over the panes and the button it passes across.
	s.drawDraggedCard(gs, screen)

	// The table: the round as a confrontation. The opponent's queued cards right-aligned, the
	// player's played cards left-aligned facing them, and the ring round any that formed a
	// combo. Over the hand, which is inert during playback, and under the overlay like
	// everything else. See combat_table.go.
	s.drawEnemyQueue(gs, screen)
	s.drawPlayedCards(gs, screen)

	// Cards travelling to and from the pile, over everything the dragged card rides over and
	// for the same reason. Under the overlay, which covers them along with the rest.
	//
	// The slides go with them and just under them: a card crossing the row is above the row,
	// and a card being dealt into a slot arrives over a card still shuffling out of it.
	s.drawSlides(gs, screen)
	s.drawFlights(gs, screen)

	// The overlay covers everything, card in flight included — and then Deck is drawn
	// again on top of it. While the deck is open it is the only control that still does
	// anything, so it is the only one that still looks like it does.
	if s.showDeck {
		s.drawDeckOverlay(gs, screen)
		s.drawDeckStack(gs, screen)
	}

	// Last of all, so a capture holds the finished frame rather than a half-drawn one.
	s.demoDraw(gs, screen)
}

// currentSlot reports how far into the round's resolution order playback has got: the
// index of the slot on screen right now. It is derived by counting the action events
// walked so far rather than tracked in a field, so it cannot drift out of step with the
// cursor.
//
// It counts *slots* rather than a side and a queue position, which is what phase
// resolution forced and what should have been here anyway. Slot.Index is where a card sits
// in the player's queue, and reordering by category means that is no longer where it lands
// in the round — matching on it would have lit the wrong row. Counting the order works
// under any ordering rule, including whatever replaces this one.
//
// ok is false before the first action of a round and once playback has finished.
func (s *CombatScene) currentSlot() (int, bool) {
	if s.cursor >= len(s.log) {
		return 0, false
	}

	// **A chilled action counts as a slot even though it never happened.** The pane draws
	// every slot ResolutionOrder produced, including ones a chill deleted, so counting only
	// the actions that resolved would leave the highlight one row short for the rest of the
	// round and light the wrong card. One beat per slot, whether it was taken or lost.
	played := -1
	for _, e := range s.log[:s.cursor+1] {
		if e.Kind == combat.KindAction || e.Kind == combat.KindChilled {
			played++
		}
	}

	if played < 0 {
		return 0, false
	}
	return played, true
}

// traceLayout dumps every rectangle the screen computes. Called from Init and again at the
// start of each round, which is when the hand size — and therefore the whole bottom band —
// can have changed.
//
// This exists because a layout bug is far more obvious as a number than as a picture: a
// glyph column measuring 224 inside a card 132 tall states the problem outright, where the
// same thing on screen just looks vaguely wrong.
func (s *CombatScene) traceLayout(gs *state.GlobalState) {
	if !trace.Enabled() {
		return
	}

	trace.Section("combat layout")
	trace.Logf("layout", "screen %dx%d  hand %d cards  pitch %d (full %d)",
		gs.ScreenWidth, gs.ScreenHeight, len(s.hand),
		handPitch(gs, s.laidOutCount()), cardWidth+cardGap)

	band := handBand(gs, s.laidOutCount())
	trace.Rect("handBand", band)
	trace.Rect("actionFlowPane", panePlacementRect(gs, actionFlowPane))

	// Both states, because the collapsed one is what is on screen and the expanded one is
	// the thing a long press has to land inside. A dump of only the box as it currently
	// stands would say nothing about where it goes.
	trace.Rect("resolutionFeed", s.feedRect(gs))
	trace.Rect("resolutionFeed expanded", image.Rect(
		band.Min.X, gs.PctY(feedExpandTopPct),
		band.Max.X, gs.PctY(handTopPct)-feedGapAboveCards))
	trace.Rect("duelistCard", s.duelistCardRect(gs))
	trace.Rect("enemyCard", s.enemyCardRect(gs))
	trace.Rect("ringPane", s.ringPaneRect(gs))
	trace.Rect("ringPane backing", s.ringPaneBackRect(gs))
	// The slots as they currently stand, not as they would at the cap: the pitch is a function
	// of how many rings are worn, so a dump of five would describe a row that is not on screen.
	for i := 0; i < len(equippedRings(gs)); i++ {
		at := ringSlotAt(s.ringPaneRect(gs), i, len(equippedRings(gs)))
		trace.Rect(fmt.Sprintf("ringSlot[%d]", i), image.Rect(
			at.X, at.Y, at.X+cards.RingStyle.Width, at.Y+cards.RingStyle.Height))
	}
	trace.Rect("apBar", image.Rect(
		band.Min.X, band.Max.Y+apBarBelow,
		band.Max.X, band.Max.Y+apBarBelow+apBarHeight))
	trace.Rect("deckPanel", image.Rect(
		gs.PctX(deckPanelLeftPct), gs.PctY(deckPanelTopPct),
		gs.PctX(deckPanelRightPct), gs.PctY(deckPanelBottomPct)))

	for i, c := range s.hand {
		trace.Rect(fmt.Sprintf("card[%d] %s", i, cardLabel(c.actionCard)), s.cardSlot(gs, i))
	}

	for _, b := range []struct {
		name string
		b    *models.Button
	}{{"discard", s.discardButton}, {"duel", s.duelButton}} {
		trace.Logf("layout", "%-18s centre %4d,%-4d", "button "+b.name, b.b.ScreenX, b.b.ScreenY)
	}
	trace.Rect("deck stack", deckStackBounds(gs))
}

// cardLabel names a card for a trace line: "Strike/fire", or just "Strike" when plain.
func cardLabel(c actionCard) string {
	if c.Element == combat.Basic {
		return c.Label()
	}
	return c.Label() + "/" + c.Element.String()
}

// handLabel renders the whole hand for a trace line, marking the selected ones.
func handLabel(hand []paletteCard) string {
	out := ""
	for i, c := range hand {
		if i > 0 {
			out += " "
		}
		if c.selected {
			out += "[" + cardLabel(c.actionCard) + "]"
			continue
		}
		out += cardLabel(c.actionCard)
	}
	return out
}

// planLabel renders a queued set as "Guard + Strike + Jab".
func planLabel(cards []combat.Card) string {
	if len(cards) == 0 {
		return "(nothing)"
	}

	label := cardLabel(cards[0])
	for _, c := range cards[1:] {
		label += " + " + cardLabel(c)
	}
	return label
}

// enemyFromRecord hydrates an enemy out of global state, **grown to the fight it is met at** —
// see entities.ScaleToFight. `fightIndex` is the whole of what the ascent curve reads, which is
// the same counter the floor and room under the duelist card are derived from.
//
// **No sheet to look up any more** — the enemy is a card, so its picture is a portrait key
// that internal/cards decodes when it draws one.
func enemyFromRecord(gs *state.GlobalState, record string, fight int) *entities.Combatant {
	return entities.NewEnemyFrom(gs.Enemies[record], fight)
}

// duelistFromRecord resolves a playable duelist. **No sheet to look up** — the character
// block replaced the fighter's sprite, so a duelist record has no picture in it.
func duelistFromRecord(gs *state.GlobalState, record string) *entities.Combatant {
	return entities.NewDuelistFrom(gs.Duelists[record])
}
