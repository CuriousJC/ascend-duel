package screens

import (
	"fmt"
	"image"
	"math"
	"math/rand"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/decks"
	"github.com/curiousjc/ascend-duel/internal/entities"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/scenario"
	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/trace"
	"github.com/hajimehoshi/ebiten/v2"

	"image/color"
)

// Playback pacing. **The speed itself is not here** — it is `beatTicks` in clock.go, which is the
// game's rather than this screen's, and what lives here is how a round *spends* it: one multiplier
// per event kind.
//
// victoryHoldTicks is how long a won fight sits finished before the post-battle screen takes over
// by itself. A fraction of the one playback speed like every other clock here, and **the one
// number to move if the pause reads as too long or too short** — the picture it holds up is the
// last round of a won duel, cards on the table and an empty enemy bar. See holdVictory.
var victoryHoldTicks = beat(4, 1)

// eventDwells is each kind's share of the playback speed: a multiplier on `beatTicks`,
// where 1 is the ordinary beat every event gets.
//
// **One speed setting and a table of proportions, rather than a table of durations**
// *(2026-08-19, owner's call)*. Playback as a whole is `beatTicks`; whether a chill should
// hold longer than a card firing is this table. Written as ticks, the two questions could not be
// asked separately — every retune of the speed meant re-deriving every entry, and an entry that
// had drifted out of proportion looked exactly like one that had been chosen.
//
// **Everything is 1 as of 2026-08-19**, deliberately: the beats were being tuned against a dwell
// that was itself wrong, so the speed came down and the proportions were flattened to see what
// that alone does. **A row that is not 1 needs a sentence saying why**, the way the theatre table's
// entries carry a reason — a multiplier nobody can account for is the per-kind pacing this screen
// removed once already.
//
// **It is a table with an entry per kind and no default arm, and that shape is the whole point.**
// This screen had three dwells selected by a `switch` with a `default` once, and the default was
// the shortest of them — so every event kind added after that switch was written silently
// inherited a quarter-second flash. `KindNegated` landed there and a Defend blunting a heavy blow
// went past faster than the round-start beat. A map plus `TestEveryEventKindHasADwell` cannot do
// that: a kind added without a line here fails the tests rather than quietly picking up whatever
// the last arm said.
//
// **Nothing in here can change an outcome.** The round was resolved before a frame of it was
// drawn; these decide only how long the player looks at each part of it. Same constraint as the
// debug flags and every card in flight.
var eventDwells = map[combat.EventKind]float64{
	combat.KindRoundStart: 1,
	combat.KindAction:     1,
	combat.KindGathered:   1,
	combat.KindDrew:       1,
	combat.KindNegated:    1,
	combat.KindDamage:     1,
	combat.KindDefeated:   1,
	combat.KindHand:       1,
	combat.KindChilled:    1,
	combat.KindStatus:     1,
	combat.KindMissed:     1,
	combat.KindBurned:     1,
	combat.KindRoundEnd:   1,
}

// eventDwell is how long one kind is held, in ticks: the speed times the kind's multiplier.
//
// **A kind with no entry takes the plain beat** rather than nothing — too long is noticed and too
// short is missed, so that is the safe direction to be wrong in, and the test is what stops it
// staying wrong. **Never less than one tick**, or a multiplier small enough to round to zero would
// make an event nobody can see.
func eventDwell(kind combat.EventKind) int {
	mult, ok := eventDwells[kind]
	if !ok {
		mult = 1
	}
	if ticks := int(math.Round(float64(speedTicks()) * mult)); ticks > 1 {
		return ticks
	}
	return 1
}

// dwellForCurrent is the hold owed by the event currently on screen, which is the one *behind* the
// cursor: the cursor names the event about to be applied. Before the first event there is nothing
// on screen yet, so the lead-in takes the plain beat.
func (s *CombatScene) dwellForCurrent() int {
	if s.cursor <= 0 || s.cursor > len(s.log) {
		return speedTicks()
	}
	return eventDwell(s.log[s.cursor-1].Kind)
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

	// run is the run these piles were dealt out of, or nil for the callers that deal a hand
	// without one — `OpeningHand`, `tools/seeds` and the flight tests.
	//
	// **It is here for the element flip and for the panel that shows it.** A flip fires as a card
	// is drawn, and the two things that needs are the worn rings and the card as the run owns it:
	// the first so the draw knows what to recolour, the second so a card coming back out of the
	// discard is put back the way it was found instead of being flipped a second time. The deck
	// panel reads it for the same reason — the alterations toggle is a choice between two faces of
	// one card, and only the run holds the other one.
	//
	// **Nothing else on this screen may reach through it.** The scene takes the deck it is handed
	// and the run does not change while a fight is on; a rule read off it mid-fight would be a
	// second opinion about a fight the engine has already been given.
	run *session.Session

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
	// here so an enemy deck can be built without a window — see that package's header.
	//
	// **It carries its own shuffle source, separate from s.rng above**, and the reason is
	// the seed catalogue. CLAUDE.md names "card shuffles" as one determinism stream; sharing
	// it between the two sides would make the player's opening hand a function of how many
	// cards the enemy happened to draw, and every entry in seeds.go would break the first
	// time an enemy deck was retuned. A named hand has to stay a fact about the player's
	// deck alone.
	enemyPile *decks.EnemyPile

	// The press in progress over the hand, and the card it has lifted out of the row. See
	// carddrag.go for the lifecycle and combat_actionbox.go for the row it runs on.
	drag   cardDrag
	lifted paletteCard

	// The press in progress over the worn ring row. **Its own controller rather than the hand's**,
	// because the two rows are live at once and under different conditions: the hand is dead while
	// a round resolves and the ring row is not.
	ringDrag cardDrag

	// ringShake is each worn seat's shake and cardShake each played card's, with shakeItem the item
	// of the hand dialog's script that was running when the last one was started — which is how one
	// beat starts one shake rather than a new one every frame the box sits on the same figure.
	ringShake [combat.MaxWornRings]travel
	cardShake []travel
	shakeItem int

	// deckView is how the deck overlay is being read — the alterations and FULL/PLAYED toggles
	// along its bottom edge. **Not reset by Init**, exactly like sortMode: a reading preference is
	// not a fact about a duel, and snapping back every fight would make it something the player
	// re-presses.
	deckView deckView

	// showDeck toggles the deck overlay. While it is up the cards underneath do not
	// respond, so reading the deck cannot accidentally re-plan the round.
	showDeck bool

	// tip is the panel explaining whatever the cursor is resting on — a card's arithmetic, a
	// ring's rule, a status nobody has anywhere else to read. Aimed once a tick by `hover`, in
	// combat_hover.go, and hidden by the tick it is not aimed.
	tip models.Tooltip

	// showLog toggles the fight log. The second dialog in the game, and it obeys the first
	// one's rules — see combat_log.go.
	showLog bool

	// closer is the red X on whichever of this screen's two older dialogs is up — the deck overlay
	// or the fight log. **One between them**, because only one can be open at a time and two
	// buttons in the same corner is one too many. The hands panel carries its own, inside its
	// toggle.
	closer modalCloser

	// hands is the third dialog: every rung of the hand ladder, written as a sum. It carries
	// its own button and its own open flag, because it arrived after the shared modal chrome
	// existed and there was no reason to give the screen a fourth pair of fields by hand.
	hands handsToggle

	// logButton opens and closes it. Held on the scene rather than built in Draw because it
	// is a widget with hover and press state, like every other button here.
	logButton *models.Button

	// rounds is every round of this fight that has finished, oldest first, kept as the event
	// logs the resolver produced.
	//
	// **The round in progress is not in here** — it is still `log`, and the log dialog reads
	// the two together. A round moves across in startRound, as the previous one is replaced,
	// which is the one moment `log` is about to stop being the current round. Appending at
	// the *end* of playback instead would double the round the feed is still showing during
	// the planning phase.
	//
	// **It holds events rather than finished lines.** The prose is generated from the events
	// by logRows, so storing sentences would freeze this fight's account against the wording
	// of the day it was played — and would be a second copy of something the events already
	// say. Events are what the engine produced; everything else is a reading of them.
	rounds [][]combat.Event

	// theatre is everything this screen has moving on it: the cards in the air, the two rows on
	// the table, the damage figures, the banked points, the hand's name and the sum it flies into.
	//
	// **Eleven flat fields until 2026-08-21**, with the rules that govern all of them repeated as
	// comments across six files. See theatre.go for those rules and why they are a type now. The
	// one this grouping actually buys: a theatre is taken down all at once, so `Init` clears it in
	// a line rather than in six statements that each had to be remembered.
	theatre combatTheatre

	// The fighter's own resources, drawn in the character block. discardsLeft refills
	// every round. **Vitae is the run's, not the screen's** *(2026-08-17)* — see session.Session,
	// which is what the post-battle screen pays into.
	discardsLeft int

	// fightIndex is which room of the climb the player is in.
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

	// victoryHeld counts the frames a won fight has been sitting finished, and it is what raises
	// `won` without anybody pressing anything. See victoryHoldTicks.
	victoryHeld int

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

	// tut is Bob, when a run is being taught. **A field on the scene rather than global state**,
	// because the widget is this screen's — the two buttons and where the bubble last sat. What
	// survives a fight is the step cursor, and that is on the run. See tutorial.go.
	tut tutorialOverlay
}

// Init prepares a fresh duel. Safe to re-enter: the combatants and the button are
// built once, everything else resets every visit.
func (s *CombatScene) Init(gs *state.GlobalState) {
	// **The fighter is rebuilt from the record on every visit, and then re-equipped**
	// *(2026-08-17)*. It used to be built once, which was fine while a ring was a flag that never
	// changed — a growing ring's accumulator moves between fights, so equipping once would have
	// left every fight after the first paying fight one's figure. Rebuilding first is what stops
	// the stat rings stacking on themselves instead.
	//
	// Nothing is lost by rebuilding: a duel already restores full life below, and everything else
	// on the combatant comes out of the record.
	s.fighter = duelistFromRecord(gs, playerRecord)

	// **What the player is wearing is part of hydrating them**, not part of resetting a duel:
	// rings are run-level and a fight does not take them off. **The run puts them on**, which is
	// also the `fight-start` moment — a stat ring's DMG and HP arrive here, and a growing one
	// arrives with whatever it has accumulated. The screen used to parse an element off each
	// record and set a flag; the grammar is in `session.Equip` now and this is one call.
	if gs.Run != nil {
		s.fighter.Duelist = gs.Run.Equip(s.fighter.Duelist)
	}

	// The enemy is rebuilt every visit rather than once, because fightIndex may have moved
	// since the last one. Init is how the next fight starts, not only how the screen is
	// entered — see nextFight.
	// The run says which room this is; the scene keeps a copy for the frame. See fightIndex.
	s.fightIndex = gs.Run.Fight()

	// **A scenario may name who is standing in the room**, so an interaction can be looked at
	// against a chosen enemy rather than whoever the climb dealt. Compiled out of every normal
	// build; see internal/scenario.
	enemyKey := gs.Run.Enemy()
	if scenario.Active() && scenario.Enemy() != "" {
		enemyKey = scenario.Enemy()
	}
	s.enemy = enemyFromRecord(gs, enemyKey, s.fightIndex)

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

	// The Log button, beside the draw pile. Built once and placed every visit, like the strip
	// below — its position is a function of the pile, which is a function of the screen.
	if s.logButton == nil {
		s.buildLogButton()
	}
	{
		r := logButtonRect(gs)
		s.logButton.ScreenX = r.Min.X + r.Dx()/2
		s.logButton.ScreenY = r.Min.Y + r.Dy()/2
	}

	discardX, duelX := buttonStripSlots(gs, s.discardButton.Width, s.duelButton.Width)
	s.discardButton.ScreenX = discardX
	s.discardButton.ScreenY = gs.PctY(buttonStripPct)
	s.duelButton.ScreenX = duelX
	s.duelButton.ScreenY = gs.PctY(buttonStripPct)

	s.showDeck = false
	s.showLog = false
	s.hands.init(handsButtonPlace)
	s.tip = models.Tooltip{DwellTicks: tipDwell}

	// **The whole stage comes down, and that is one line on purpose** *(2026-08-21)*. It was eight
	// statements, each added after something was found still on screen at the start of the next
	// fight — a figure in the air belongs to the fight that raised it, a Prepare's points are a
	// fact about one round of one fight, and the played cards drew over the new table and blanked
	// the hand slots they claimed. All of those are the same bug: a settled duel freezes rather
	// than spending its hand, so anything tidied up only by the end-of-round spend is still there.
	// A mover added tomorrow is covered without anybody remembering. See combatTheatre.clear.
	s.theatre.clear()

	s.restart = false
	s.victoryHeld = 0
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
	s.drag = cardDrag{}
	s.ringDrag = cardDrag{}
	s.ringShake, s.cardShake, s.shakeItem = [combat.MaxWornRings]travel{}, nil, 0

	// A fresh shuffled deck for the opponent too, dealt before it plans, off its own stream.
	s.enemyPile = decks.NewEnemyPile(s.enemy.Record, enemySeed, decks.EnemyHandSize)

	// A fresh duel: full life, no standing defenses, and no action points banked by a
	// Prepare from a duel that has been walked away from.
	s.fighter.CurrentLife = s.fighter.MaxLife
	s.enemy.CurrentLife = s.enemy.MaxLife
	s.fighter.Duelist = resetCombatState(s.fighter.Duelist)
	s.enemy.Duelist = resetCombatState(s.enemy.Duelist)

	s.log = nil
	s.rounds = nil
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
	d.Statuses = [combat.MaxStatuses]combat.Status{}
	return d
}

// duelSettled reports that the fight is over *and* has finished being watched. Both halves
// matter: life reaches zero partway through playback, and taking the way out before the killing
// blow has been drawn would cut the round short at its most interesting moment. That is as true
// of the exit a won fight takes by itself as it was of the button — see holdVictory, which counts
// from here.
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
// victoryPending reports a won fight winding down: settled, the enemy dead, and the screen holding
// its last picture until the post-battle scene takes over on its own. It is what the button strip
// and `Draw` read, so "the fight is over and there is nothing to press" is asked in one place.
func (s *CombatScene) victoryPending() bool {
	return s.duelSettled() && !s.enemy.Alive()
}

// holdVictory is what carries a won fight into the post-battle screen **without the player
// pressing anything** *(2026-08-19, owner's call)*. It counts frames since the last event was
// drawn and raises the same flag Next does once `victoryHoldTicks` have passed.
//
// **The hold is the whole of the design, not a delay before one.** The screen freezes a settled
// duel deliberately — the cards stay on the table, the hand keeps its gaps, and the picture the
// player is looking at when the killing blow lands is the picture that stays up (see endOfRound).
// Leaving on the frame the last figure arrives would throw that away, so the pause is kept and
// only the press is dropped.
//
// **Next stays live and still works**, so the hold is a floor rather than a wait: a player who has
// seen enough presses on. And **the count stops while a dialog is up** — the fight log can be
// opened on a won fight, and a screen that changed out from under an open panel would be reading
// material snatched away.
//
// A defeat is untouched: `Retry` is a decision to play the same fight again, and a screen that
// restarted a lost duel by itself would take that decision away.
func (s *CombatScene) holdVictory() {
	if s.modalUp() {
		return
	}
	s.victoryHeld++
	if s.victoryHeld >= victoryHoldTicks {
		s.nextFight()
	}
}

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
		// **Before WonFight**, because a `grow-on-hit` ring grew the fighter's own copy during the
		// duel and this is the last tick that copy exists.
		gs.Run.AbsorbGrowth(s.fighter.Duelist)
		gs.Run.WonFight(s.fighter.CurrentLife)

		// **The first duel ever won, recorded on the profile.** It fires on every win and the
		// profile keeps one — see save.go, and profile.AchievementFirstSteps for why the key is
		// fixed from here on.
		awardFirstSteps(gs)

		advanceRun(gs)
		return nil
	}

	// **Everything moving, and the pile the cards come from. Both are outside every branch below
	// on purpose**: a flight that started before the killing blow should still land, and the deck
	// stack is the only control that survives its own overlay — it is what closes it.
	//
	// **It is also before advancePlayback and never inside it.** The cursor waits on a figure
	// still in the air, so a tick that only ran when playback was allowed to advance would leave
	// the flight frozen at age zero and hang the round on itself. The figures used to tick further
	// down for that reason, below the settled-duel branch; one call here is strictly earlier and
	// strictly more often, and it cannot reach a figure that was in the air when the duel settled
	// because playback does not finish until every figure has landed.
	// **The tutorial runs before anything on this screen it might close.** The gate it sets is
	// what every widget below reads, so running it afterwards would hand the player one live
	// frame per step on controls the lesson has shut. It is after the two exits above because
	// those change screen and there is nothing to point at on the way out.
	s.tut.update(gs, s)

	s.theatre.tick()
	// **Every opener is inert while any dialog is up, and the X on the panel is the exit**
	// *(owner's call, 2026-08-24)*. It used to be the other way round — each control survived its
	// own overlay because it was the only thing that closed one — and that stopped working when a
	// panel covered its own button. See modalCloser.
	if !s.modalUp() {
		s.updateDeckStack(gs)
	}
	s.updateLogButton(gs)

	// The deck panel's own two toggles, live only while it is up. They are a view over a picture
	// of the deck and can change nothing about the round underneath — see deckView.
	if s.showDeck {
		s.deckView.update(gs, s.fightContents())
	}

	// The X, run while either of the two older dialogs is up. The hands panel closes itself.
	if s.showDeck || s.showLog {
		if s.closer.update(gs) {
			s.showDeck, s.showLog = false, false
		}
	}

	// **The hands button is dead under the other two dialogs**, for the reason each of them is
	// dead under the other: a dialog whose exit is not the brightest thing on screen is a trap,
	// and two live exits is two.
	s.hands.block(s.showDeck || s.showLog)
	s.hands.update(gs)

	// Tell the frame a dialog is up, so the game's own chrome stands down rather than sitting
	// live on top of it. Written unconditionally from what the screen already knows, never
	// toggled — see state.ModalOpen. There are two dialogs: the deck overlay and the log.
	gs.ModalOpen = s.modalUp()

	// The overlay swallows card interaction, so reading the deck cannot re-plan the round
	// through the panel covering it. The buttons stay live — one of them is how it closes.
	if !s.modalUp() {
		s.updateActionBox(gs)
	}

	s.updateRingRow(gs)
	s.tickShakes(gs)

	// Above the branch below, because the column is live under exactly one condition and it is
	// its own: the hand may be rearranged whenever it may be edited. It goes dead once the duel
	// is settled for the same reason it goes dead during playback — see updateSortButtons.
	s.updateSortButtons(gs)

	// Once the duel is decided the DUEL! button becomes the way onward, in place. Reusing
	// the same button rather than adding a fourth is the point: it is the same slot for
	// "commit and move the game forward", and a control that appears only at the end of a
	// fight would be a control nobody has learned.
	if s.duelSettled() {
		// **A won fight has no control at all** *(2026-08-19, owner's call)*. It holds its last
		// picture and then leaves, so there is nothing to press and nothing to label: a `Next`
		// standing lit under a screen that is about to change by itself is an offer of a choice
		// the player has not got. See holdVictory and victoryPending — the button is not drawn
		// either.
		if s.victoryPending() {
			s.holdVictory()
			setEnabled(s.discardButton, false)
			systems.UpdateButton(gs, s.discardButton)
			return nil
		}

		// A defeat keeps its button: Retry is a decision to play the same fight again, and it is
		// the player's.
		s.duelButton.Text, s.duelButton.OnClick = "Retry", s.nextFight
		setEnabled(s.duelButton, !s.modalUp())
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
	live := s.planning() && len(s.fighterActions) > 0 && !s.modalUp()
	setEnabled(s.duelButton, live && !s.overBudget())
	setEnabled(s.discardButton, live && s.discardsLeft > 0)

	systems.UpdateButton(gs, s.duelButton)
	systems.UpdateButton(gs, s.discardButton)

	s.advancePlayback(gs)

	// **Last, after everything that could have moved a card.** The pass reads the same slot
	// functions the drawing does, so asking before a flight or a re-sort had settled would point at
	// where a card was rather than where it is.
	s.hover(gs)
	systems.UpdateTooltip(gs, &s.tip)

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

	// **The name of the hand is captured here, while the preview can still be asked for it.**
	// `previewAttack` is gated on `planning()`, which stops being true the moment the log below is
	// adopted — so this is the last frame the screen can name what the player built. From here it
	// is the banner's, and it flies down into the hand row as the cards fly up to the table. See
	// handBanner.
	s.theatre.banner.clear()
	if blow, ok := s.previewAttack(); ok {
		s.theatre.banner = handBanner{
			name: handShout(blow.Hand.Name),
			// The multiplier travels with the name, so what the hand is worth is written down
			// from the moment it was chosen rather than first met when the figure flies out of
			// the word. See handBanner.
			mult:   handMultiplierLine(blow.Multiplier),
			flight: newTravel(0, bannerFlyTicks),
			flying: true,
		}
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
	// **The round being replaced moves into the fight log.** This is the one moment `log` is
	// about to stop being the current round, so it is where the handover belongs — and the
	// feed goes on drawing the round that just ended right up to it, off `log`, so a round in
	// both places for a frame would be a round said twice. See combat_log.go.
	if len(s.log) > 0 {
		s.rounds = append(s.rounds, s.log)
	}
	s.log = log
	s.cursor = 0
	s.ticks = 0

	// Both hands go to the table now, not as the round plays out. The opponent's is known in
	// full at this moment and is drawn from enemyActions directly; the player's is dealt out of
	// the hand by the flights seatPlayedCards raises. Nothing here decides anything — the round
	// above is already resolved. See combat_table.go.
	s.theatre.firingSeats, s.theatre.enemyFiringSeats = nil, nil
	s.theatre.mathBox.clear()
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
		return fmt.Sprintf("status      %v puts %d %v on %v", e.Side, e.Amount, combat.StatusOf(e.Status).Key, e.Target)
	case combat.KindMissed:
		return fmt.Sprintf("missed      %v's %v never lands - shocked", e.Side, e.Action)
	case combat.KindBurned:
		return fmt.Sprintf("burned      %v takes %d from %v, leaving %d",
			e.Target, e.Amount, combat.StatusOf(e.Status).Key, e.Life)
	case combat.KindGathered:
		return fmt.Sprintf("prepared    %v banks %d AP for next round", e.Side, e.Amount)
	case combat.KindNegated:
		return fmt.Sprintf("negated     %v's %v cuts it to %d", e.Side, e.Action, e.Amount)
	case combat.KindDamage:
		return fmt.Sprintf("damage      %v hits %v for %d, leaving %d", e.Side, e.Target, e.Amount, e.Life)
	case combat.KindHand:
		return fmt.Sprintf("attack      %v forms %s (x%d.%02d)",
			e.Side, handName(e), e.Multiplier/100, e.Multiplier%100)
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
func (s *CombatScene) advancePlayback(gs *state.GlobalState) {
	if s.cursor >= len(s.log) {
		return
	}

	// **The hand dialog owns the beat while it runs, and it is the one exception to the dwell
	// below** *(2026-08-18)*. A sum revealed a figure at a time takes several seconds and cannot
	// be fitted inside one event's dwell, so rather than the box racing playback the cursor waits
	// for it. Everything else that moves on this screen runs on its own clock alongside the log.
	//
	// **It still cannot change an outcome.** The round was decided before a frame of this was
	// drawn; what waits is the drawing of it.
	if s.theatre.mathBox.running() {
		s.theatre.mathBox.tick()
		// **The banner goes when its own figure sets off** *(2026-08-19, owner's call)*. The
		// multiplier flies out of the second line under the hand's name and into the sum, so from
		// that frame the number is in the line and the banner is a copy of something that has
		// moved. The name goes with it rather than a beat later: it has been carried down, said,
		// and spent, and leaving it lit over the hand while the sum finishes and the enemy swings
		// back is a word breathing at the player long after it has anything left to tell them.
		if s.theatre.mathBox.at >= s.theatre.mathBox.multAt {
			s.theatre.banner.clear()
		}
		return
	}

	// **A landing figure holds the cursor** *(2026-08-18)*. A damage figure crosses half the screen
	// with the target's health bar waiting on it, and a banked figure crosses to the fighter card
	// with its AP line waiting — so letting playback run on would drop the bar, or raise the
	// budget, before the number got there, which is the picture the flights exist to remove.
	//
	// **Which movers hold the round is the theatre's answer, not this function's** — see
	// combatTheatre.running. It changes pacing and cannot change an outcome.
	if s.theatre.running() {
		return
	}

	// How long the event already on screen is held before the next one lands. **It is keyed on the
	// event *behind* the cursor**, which is the one the player is looking at — the cursor names
	// the event about to arrive. See eventDwell.
	s.ticks++
	if s.ticks < s.dwellForCurrent() {
		return
	}
	s.ticks = 0

	// **The finished sum stays on screen until the event after it lands, and that is a handoff
	// rather than a hold** *(2026-08-18)*. It used to be cleared on the tick after the script
	// stopped running, which put an empty band on screen for the whole of the dwell that follows —
	// so the damage figure, whose whole job is to be *that total* travelling into the card, set off
	// from a space the total had left a second and a quarter earlier. Clearing it here means the
	// last frame of the sum and the first frame of the flight are the same frame, at the same
	// point, in the same colour and at the same size. See combat_hits.go.
	//
	// A turn that misses rather than landing clears it the same way, on its `KindMissed`. When the
	// strike-through arrives that event will want this same handoff, so keep them together.
	if s.theatre.mathBox.active && !s.theatre.mathBox.running() {
		s.theatre.mathBox.clear()
	}

	s.applyEvent(s.log[s.cursor])
	s.startHandMath(gs, s.log[s.cursor])
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
// blow lands is the picture they should still be looking at while the screen holds**, so nothing
// moves — the played cards stay on the table, the hand keeps its gaps, and `Init` clears all of
// it when the next fight starts.
//
// It is a branch here rather than a rule inside `drawHand`, because what has to stop is not the
// *drawing*, it is the whole end-of-round movement. A hand that was spent but not refilled still
// reflows.
func (s *CombatScene) endOfRound() {
	s.fighter.Duelist = s.fighterAfter
	s.enemy.Duelist = s.enemyAfter

	// **The adoption above is where banked points become `BonusAP`**, so what the cards have been
	// drawing on top of it since the figures landed is now in the model and has to stop being
	// added. Before the early return below: a settled duel adopts its end state like any other.
	s.theatre.bankShown = [2]int{}

	if s.duelSettled() {
		return
	}

	// **The banner is usually gone by now**, cleared on the frame its multiplier flew into the sum
	// — see advancePlayback. This is the round that scored no hand at all: nothing took the name
	// down because nothing ever asked for it.
	s.theatre.banner.clear()

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
	s.theatre.firingSeats, s.theatre.enemyFiringSeats = nil, nil

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

	// A hand has formed: leave raised only the cards the engine says formed it.
	s.noteHand(e)

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

	// A Prepare has banked: send its points to the fighter card whose budget they raise. The
	// engine has already counted them into `GatheredAP`, so this is a ghost of something that has
	// happened — see combat_bank.go, and noteHit below, which keeps the same division.
	if e.Kind == combat.KindGathered {
		s.noteBank(e)
		return
	}

	// **A burn changes a life total without anybody acting**, so it has to be applied here
	// alongside damage rather than being a consequence of a card. Missing it would leave the two
	// fighter cards showing a life the engine has already spent — and a duelist who dies to a
	// fire tick would fall with a health bar that never moved.
	if e.Kind != combat.KindDamage && e.Kind != combat.KindBurned {
		return
	}

	// **The life the bar is showing right now, read before it is overwritten.** That is what the
	// figure in flight has to keep drawing, and it cannot be reconstructed afterwards: `e.Life` is
	// clamped at zero, so a killing blow's `Life + Amount` is the *size of the blow*, not the life
	// that was there. A 60-damage hand on a duelist with 30 left read as 60 — a bar that jumped
	// *up* for the length of a flight and then emptied, on the killing blow and nowhere else.
	before := s.enemy.CurrentLife
	if e.Target == combat.SideA {
		before = s.fighter.CurrentLife
	}

	if e.Target == combat.SideA {
		s.fighter.CurrentLife = e.Life
	} else {
		s.enemy.CurrentLife = e.Life
	}

	// **The figure is raised after the life has already moved**, so it is a ghost of something
	// that has happened rather than a thing in progress — the same division every card in flight
	// keeps. What lags is the bar's *drawing*, through `shownLife`. A burn is not flown from here:
	// it has its own source, the badge it ticks off, and its own row in the theatre table.
	if e.Kind == combat.KindDamage {
		s.noteHit(e, before)
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
	if e.Status < 0 || int(e.Status) >= combat.StatusCount() {
		return
	}
	target.Statuses[e.Status] = combat.Status{Amount: e.Amount, Rounds: 1}
}

// **There is no caption box**, and since 2026-08-18 there is no Resolution feed either — the
// band above the hand is empty except while a hand is previewed or a blow is being acted out.
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
	// **Nothing is drawn in the DUEL! slot on a won fight.** The screen is holding its last
	// picture and leaving by itself, so the slot stands empty for the second or so that takes —
	// see victoryPending. A defeat still shows Retry there.
	if !s.victoryPending() {
		systems.DrawButton(gs, screen, s.duelButton)
	}
	systems.DrawButton(gs, screen, s.discardButton)
	s.drawSortButtons(gs, screen)
	s.drawDiscardsLeft(gs, screen)
	s.drawDeckStack(gs, screen)
	s.drawLogButton(gs, screen)

	// **Order below is contested, and the ranking is written down because it will be
	// re-broken otherwise** *(2026-08-11, narrowed 2026-08-18)*. Two things want to be on top
	// of each other and they cannot both win:
	//
	//  1. A selected card over whatever is written in the band above the hand. Selection lifts
	//     a card 26px into that band's bottom 21 — see mathBandGapAboveCards — and the card is
	//     the thing being acted on. Hence the hand row after anything drawn in the band.
	//  2. A firing card over the inert hand row, at full size. Unchanged, and why the resolved
	//     pile is still drawn after the row.
	//
	// **The Resolution feed was the third and is gone** *(2026-08-18)*: it wanted to be over
	// the enemy card, because a box held open reached 12% and the opponent sits at 34%. The
	// band it left is claimed by the hand dialog and the planned hand's name, both of which
	// are drawn much later and neither of which is ever held open.
	s.drawHandRow(gs, screen)

	// Over the panes and the button it passes across.
	s.drawDraggedCard(gs, screen)

	// The table: the round as a confrontation. The opponent's queued cards right-aligned, the
	// player's played cards left-aligned facing them, and whichever of them is firing lifted
	// out of its row. Over the hand, which is inert during playback, and under the overlay like
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

	// **The hand dialog, over everything but the deck overlay.** It is the loudest thing on the
	// screen for the few seconds it is up, and it is deliberately over both rows of cards: the
	// shout is written across the hand row, which is inert while it is up, and the sum across the
	// band above it. See combat_mathbox.go.
	s.drawHandMath(gs, screen)

	// **The planned hand's name, in the middle of the half of the table its cards are about to
	// land in.** It and the shout above can never both be up: this one is gated on `planning()`
	// and that one only runs during playback. See drawPlannedHand.
	s.drawPlannedHand(gs, screen)

	// **The damage figures, over the sum they came out of and the card they are flying into.**
	// They are drawn after the dialog because a figure leaving the sum has to be on top of it —
	// underneath, the first frames of the flight would be hidden by the number it left.
	s.drawHits(gs, screen)

	// **The banked figures, over the card they are flying out of and the fighter card they raise.**
	// Beside the damage figures because they are the same gesture in the other direction, and after
	// the table for the reason those are: a figure leaving a card has to be on top of it.
	s.drawBanks(gs, screen)

	// The hands panel, under the other two: they are mutually exclusive, and this is drawn
	// first so that opening either of the others covers this one's button along with everything
	// else. Its own draw puts its button back on top of its own panel.
	s.hands.draw(gs, screen, s.fightHands())

	// The overlay covers everything, card in flight included, and the X goes on top of it. The
	// two are mutually exclusive rather than stacked — neither opener is live while the other's
	// panel is up — so they share one closing button.
	if s.showDeck {
		drawDeckPanel(gs, screen, &s.deckView, s.fightContents())
	}
	if s.showLog {
		s.drawLogOverlay(gs, screen)
	}
	if s.showDeck || s.showLog {
		s.closer.draw(gs, screen)
	}

	// **Over every overlay, because it explains what is on top.** The deck panel's cards are the
	// last thing drawn before this and they are the thing being asked about.
	systems.DrawTooltip(gs, screen, &s.tip)

	// **Bob goes over all of it, and the spotlight with him.** The scrim dims what is already on
	// the screen, so anything drawn after this would sit on top of the dimming and read as the
	// one lit thing — which is the job of the anchor and nothing else.
	s.tut.draw(gs, screen, s)

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
	// The band above the hand. Nothing is drawn there at rest — the feed left it on
	// 2026-08-18 — but the hand dialog and the planned hand's name are both laid out
	// against it, so it is worth a rectangle in the dump.
	trace.Rect("mathBand", image.Rect(
		tableInset, gs.PctY(handTopPct)-mathBandGapAboveCards-mathBandHeight,
		gs.ScreenWidth-tableInset, gs.PctY(handTopPct)-mathBandGapAboveCards))
	trace.Rect("duelistCard", s.duelistCardRect(gs))
	trace.Rect("enemyCard", s.enemyCardRect(gs))
	trace.Rect("ringPane", s.ringPaneRect(gs))
	trace.Rect("ringPane backing", s.ringPaneBackRect(gs))
	// The slots as they currently stand, not as they would at the cap: the pitch is a function
	// of how many rings are worn, so a dump of five would describe a row that is not on screen.
	for i := 0; i < len(wornRings(gs)); i++ {
		at := ringSlotAt(s.ringPaneRect(gs), i, len(wornRings(gs)))
		trace.Rect(fmt.Sprintf("ringSlot[%d]", i), image.Rect(
			at.X, at.Y, at.X+cards.RingStyle.Width, at.Y+cards.RingStyle.Height))
	}
	trace.Rect("apBar", image.Rect(
		band.Min.X, band.Max.Y+apBarBelow,
		band.Max.X, band.Max.Y+apBarBelow+apBarHeight))
	trace.Rect("deckPanel", image.Rect(
		gs.PctX(modalPanelLeftPct), gs.PctY(modalPanelTopPct),
		gs.PctX(modalPanelRightPct), gs.PctY(modalPanelBottomPct)))

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
// see pyramid.ScaleToFight. `fightIndex` is the whole of what the ascent curve reads, which is
// the same counter the floor and room under the duelist card are derived from.
//
// **No sheet to look up any more** — the enemy is a card, so its picture is a portrait key
// that internal/cards decodes when it draws one.
// **A stairway record comes from the boss pool** *(2026-08-23)*, which is the one place the two
// pools have to be told apart on this side: `data.BossData.Enemy()` hands back the enemy shape, so
// nothing below this line knows which one it got.
func enemyFromRecord(gs *state.GlobalState, record string, fight int) *entities.Combatant {
	if boss, ok := gs.Bosses[record]; ok {
		return entities.NewEnemyFrom(boss.Enemy(), fight)
	}
	return entities.NewEnemyFrom(gs.Enemies[record], fight)
}

// duelistFromRecord resolves a playable duelist. **No sheet to look up** — the character
// block replaced the fighter's sprite, so a duelist record has no picture in it.
func duelistFromRecord(gs *state.GlobalState, record string) *entities.Combatant {
	return entities.NewDuelistFrom(gs.Duelists[record])
}
