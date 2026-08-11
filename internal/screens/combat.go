package screens

import (
	"fmt"
	"image"
	"math/rand"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/decks"
	"github.com/curiousjc/ascend-duel/internal/entities"
	"github.com/curiousjc/ascend-duel/internal/models"
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
// without anyone choosing it: KindNegated landed there on 2026-08-06 and a Dodge stopping a
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
// **It is not the randomiser and does not pretend to be.** Floor generation picks from the
// records a floor allows, off its own determinism stream; this walks all 96 in order so every
// one of them can be reached by playing.
//
// Built in Init rather than at package scope: it reads the loaded roster out of global state,
// which does not exist until main has run.
func (s *CombatScene) roster(gs *state.GlobalState) []string {
	if s.enemyRoster == nil {
		s.enemyRoster = data.EnemyOrder(gs.Enemies)
	}
	return s.enemyRoster
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

// startingVitae is a placeholder. Vitae is a run-level resource that will live on Session
// state once that exists; until then the block shows a fixed 5 so the field has a shape on
// screen and somewhere to be read from.
const startingVitae = 5

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
	fighterActions []combat.ActionKind
	enemyActions   []combat.ActionKind

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

	// The player's cards that have resolved this round, in the order they fired: each one
	// rises out of the hand, holds where it can be read, then stacks in the bottom-left
	// corner. The pile is the round's own history, and it is what a combo brackets.
	//
	// Cleared when the hand is spent, which is the moment those cards actually leave.
	resolved []resolvedCard

	// The fighter's own resources, drawn in the character block. discardsLeft refills
	// every round; vitae is a placeholder that never moves yet.
	discardsLeft int
	vitae        int

	// fightIndex is how far along enemyRoster the player has got. It survives Init because
	// Init is also how the *next* fight starts — see nextFight, which advances it and then
	// asks for a re-init rather than resetting the screen itself.
	//
	// restart is that request. It is a flag rather than a direct call because it is raised
	// from a button's OnClick, which takes no arguments and so cannot reach the global state
	// Init needs.
	fightIndex int
	restart    bool

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
}

// Init prepares a fresh duel. Safe to re-enter: the combatants and the button are
// built once, everything else resets every visit.
func (s *CombatScene) Init(gs *state.GlobalState) {
	if s.fighter == nil {
		s.fighter = duelistFromRecord(gs, "Fighter1")
	}

	// The enemy is rebuilt every visit rather than once, because fightIndex may have moved
	// since the last one. Init is how the next fight starts, not only how the screen is
	// entered — see nextFight.
	s.enemy = enemyFromRecord(gs, s.roster(gs)[s.fightIndex%len(s.roster(gs))])

	// The scene builds its own widgets and wires them to its own methods, so no other
	// package needs to know this screen has buttons or what pressing them means.
	if s.duelButton == nil {
		s.duelButton = models.NewButton(138, 50, "DUEL!", s.startRound)
		s.duelButton.BaseColor = color.RGBA{R: 220, G: 20, B: 60, A: 255} // crimson
	}
	if s.discardButton == nil {
		s.discardButton = models.NewButton(138, 50, "Discard", s.discardSelected)
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
	discardX, duelX := buttonStripSlots(gs, s.discardButton.Width, s.duelButton.Width)
	s.discardButton.ScreenX = discardX
	s.discardButton.ScreenY = gs.PctY(buttonStripPct)
	s.duelButton.ScreenX = duelX
	s.duelButton.ScreenY = gs.PctY(buttonStripPct)

	s.showDeck = false
	s.feedPressTicks = 0
	s.flights = nil
	s.restart = false
	s.discardsLeft = discardsPerRound
	s.vitae = startingVitae

	// A fresh deck every visit, shuffled from the same seed, so re-entering the screen
	// deals the same opening hand rather than continuing a run that has been abandoned.
	s.rng = rand.New(rand.NewSource(deckSeed))
	s.resetDeck()

	// The queue starts empty every visit and is derived from what is selected in hand.
	// DUEL! is disabled until something is in it.
	s.fighterActions = nil
	s.drag = nil

	// A fresh shuffled deck for the opponent too, dealt before it plans.
	s.enemyPile = decks.NewEnemyPile(decks.EnemySeed, decks.EnemyHandSize)

	// Planned up front only so the enemy pane has something in it before the first
	// DUEL!. startRound re-plans it every round regardless, so this is display, not a
	// commitment the resolver ever reads.
	//
	// **It spends cards from the opponent's hand**, which is why Init has to deal that hand
	// first. A plan is a commitment on either side.
	s.enemyActions = s.enemyPile.Plan(s.enemy.Style, s.enemy.Duelist)

	// A fresh duel: full life, no standing defenses, and no action points banked by a
	// Gather from a duel that has been walked away from.
	s.fighter.CurrentLife = s.fighter.MaxLife
	s.enemy.CurrentLife = s.enemy.MaxLife
	s.fighter.Duelist = resetCombatState(s.fighter.Duelist)
	s.enemy.Duelist = resetCombatState(s.enemy.Duelist)

	s.log = nil
	s.cursor = 0
	s.ticks = 0
	s.round = 0

	trace.Logf("scene", "fight %d: %s, style %v, %d life, %d AP",
		s.fightIndex+1, s.enemy.Name, s.enemy.Style,
		s.enemy.MaxLife, s.enemy.ActionPoints())
	trace.Logf("scene", "combat init: deck %d hand %d discard %d, seed %d",
		len(s.deck), len(s.hand), len(s.discard), deckSeed)
	s.tracedHand = len(s.hand)
	s.traceLayout(gs)
}

// resetCombatState clears everything a duel accumulates, leaving the stats a combatant was
// hydrated with. Init re-enters a screen that may have been left mid-duel, so a standing
// guard or a banked Gather would otherwise be inherited by the next one.
//
// It sets the fields by name rather than rebuilding the struct: Con/Str/Spd/MaxLife come
// from the data record and must survive, and a zero literal here would quietly wipe them
// the first time someone re-entered the screen.
func resetCombatState(d combat.Duelist) combat.Duelist {
	d.Guarded = false
	d.Ripostes = 0
	d.Dodges = 0
	d.BonusAP = 0
	d.GatheredAP = 0
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
// Winning advances along the roster; losing puts the same opponent back up.
func (s *CombatScene) nextFight() {
	if !s.duelSettled() {
		return
	}
	if !s.enemy.Alive() {
		s.fightIndex++
	}
	s.restart = true
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

	// Cards in the air, and the pile they come from. Both are outside every branch below on
	// purpose: a flight that started before the killing blow should still land, and the deck
	// stack is the only control that survives its own overlay — it is what closes it.
	s.updateFlights()
	s.updateResolved()
	s.updateDeckStack(gs)

	// The long press on the Resolution feed. Outside every branch below for the same reason
	// the flights are: reading back what just happened is not an action, and it has to work
	// while a round plays and after one side is down.
	s.updateFeed(gs)

	// The overlay swallows card interaction, so reading the deck cannot re-plan the round
	// through the panel covering it. The buttons stay live — one of them is how it closes.
	if !s.showDeck {
		s.updateActionBox(gs)
	}

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
	s.enemyActions = s.enemyPile.Plan(s.enemy.Style, s.enemy.Duelist)

	log, fighterAfter, enemyAfter := combat.ResolveRound(
		s.fighter.Duelist, s.enemy.Duelist,
		s.fighterActions, s.enemyActions,
		s.round,
	)

	s.fighterAfter = fighterAfter
	s.enemyAfter = enemyAfter
	s.log = log
	s.cursor = 0
	s.ticks = 0

	// The whole round, not a count of it. ResolveRound already decided every one of these
	// before a frame of playback ran, so this is the authoritative account of what is about
	// to happen on screen — and if the screen ever disagrees with it, the screen is wrong.
	if trace.Enabled() {
		trace.Section(fmt.Sprintf("round %d", s.round))
		trace.Logf("round", "fighter %d AP, plan %s", s.fighter.ActionPoints(), planLabel(s.fighterActions))
		trace.Logf("round", "enemy   %d AP, %v plan %s",
			s.enemy.ActionPoints(), s.enemy.Style, planLabel(s.enemyActions))
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
		return fmt.Sprintf("action      %v plays %v (%v)", e.Side, e.Action, e.Action.Category())
	case combat.KindGathered:
		return fmt.Sprintf("prepared    %v banks %d AP for next round", e.Side, e.Amount)
	case combat.KindNegated:
		return fmt.Sprintf("negated     %v's %v stops %v cold", e.Side, e.Action, e.Target)
	case combat.KindGuarded:
		return fmt.Sprintf("guarded     %v halves it to %d (target on %d)", e.Target, e.Amount, e.Life)
	case combat.KindBraced:
		return fmt.Sprintf("braced      %v halves it to %d (target on %d)", e.Target, e.Amount, e.Life)
	case combat.KindStripped:
		return fmt.Sprintf("stripped    %v's feint removes %v's %v", e.Side, e.Target, e.Action)
	case combat.KindDamage:
		return fmt.Sprintf("damage      %v hits %v for %d, leaving %d", e.Side, e.Target, e.Amount, e.Life)
	case combat.KindCombo:
		name := "?"
		if c, ok := combat.ComboByID(e.Combo); ok {
			name = c.Name
		}
		return fmt.Sprintf("combo       %v forms %s", e.Side, name)
	case combat.KindStaggered:
		return fmt.Sprintf("staggered   %v loses its %v", e.Side, e.Action)
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

	// Playback has caught up with the resolver. Adopt the authoritative end-of-round
	// state and hand control back to the player to plan the next round.
	//
	// The hand is spent here rather than at resolve time, and the ordering matters:
	// endRoundHand rebuilds fighterActions from what is left, and the Resolution pane
	// draws fighterActions to narrate the round. Spending it while playback was still
	// running would empty the pane mid-round.
	if s.cursor >= len(s.log) {
		s.fighter.Duelist = s.fighterAfter
		s.enemy.Duelist = s.enemyAfter
		s.endRoundHand()
	}
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

	if e.Kind != combat.KindDamage {
		return
	}

	if e.Target == combat.SideA {
		s.fighter.CurrentLife = e.Life
	} else {
		s.enemy.CurrentLife = e.Life
	}
}

// **The caption box is gone** *(2026-08-11)*, and the slot above the hand is the Resolution
// feed instead. It held the plan line, its action-point cost and the tail that said what to
// press; `caption()` and `drawCaptionBox` went with it.
//
// What that gives up, stated rather than discovered later: the sentence explaining a dark
// DUEL! button. "2 over - discard or deselect" was the only place on screen that said *why*
// the button had gone dead — the AP bar turning red says that something is wrong, not what to
// do about it. The end-of-fight prompt naming the next enemy went the same way; DUEL!
// relabelling itself Next or Retry is what is left of it.
//
// Both were the owner's call, made knowing the cost. If either is wanted back it is a line
// somewhere else, not a box — see TODO.md.

func (s *CombatScene) Draw(gs *state.GlobalState, screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 50, G: 50, B: 50, A: 255})

	s.drawFighterBlock(gs, screen)

	// **The enemy is a card now** *(2026-08-11)*, centred where its sprite stood. It was the
	// last thing on this screen drawn as a loose picture on the background, with a health bar
	// hanging under it — and everything else the duel is made of is a card, including the one
	// you are fighting. Its portrait, name, bar and life-as-a-fraction are all one cached
	// image from internal/cards; see drawEnemyCard.
	s.drawEnemyCard(gs, screen, image.Pt(gs.PctX(88), gs.PctY(34)))
	systems.DrawButton(gs, screen, s.duelButton)
	systems.DrawButton(gs, screen, s.discardButton)
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

	// The round's own history: the cards that have fired, on their way to the corner or
	// parked in it, and the ring round any that formed a combo. Over the hand, which is
	// inert during playback, and under the overlay like everything else.
	s.drawResolvedCards(gs, screen)

	// Cards travelling to and from the pile, over everything the dragged card rides over and
	// for the same reason. Under the overlay, which covers them along with the rest.
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

	// **A staggered action counts as a slot even though it never happened.** The pane draws
	// every slot ResolutionOrder produced, including ones a stagger deleted, so counting only
	// the actions that resolved would leave the highlight one row short for the rest of the
	// round and light the wrong card. One beat per slot, whether it was taken or lost.
	played := -1
	for _, e := range s.log[:s.cursor+1] {
		if e.Kind == combat.KindAction || e.Kind == combat.KindStaggered {
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
	trace.Rect("fighterBlock", image.Rect(
		gs.PctX(blockLeftPct), gs.PctY(blockTopPct),
		gs.PctX(blockRightPct), gs.PctY(blockTopPct)+blockHeight))
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
	if c.element == elementBasic {
		return c.action.String()
	}
	return c.action.String() + "/" + c.element.String()
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
func planLabel(actions []combat.ActionKind) string {
	if len(actions) == 0 {
		return "(nothing)"
	}

	label := actions[0].String()
	for _, a := range actions[1:] {
		label += " + " + a.String()
	}
	return label
}

// enemyFromRecord hydrates an enemy out of global state.
//
// **No sheet to look up any more** — the enemy is a card, so its picture is a portrait key
// that internal/cards decodes when it draws one.
func enemyFromRecord(gs *state.GlobalState, record string) *entities.Combatant {
	return entities.NewEnemyFrom(gs.Enemies[record])
}

// duelistFromRecord resolves a playable duelist. **No sheet to look up** — the character
// block replaced the fighter's sprite, so a duelist record has no picture in it.
func duelistFromRecord(gs *state.GlobalState, record string) *entities.Combatant {
	return entities.NewDuelistFrom(gs.Duelists[record])
}
