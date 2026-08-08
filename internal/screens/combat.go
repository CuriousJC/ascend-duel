package screens

import (
	"fmt"
	"image"
	"math/rand"

	"github.com/curiousjc/ascend-duel/internal/combat"
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

// The order enemies are fought in. There is no run structure yet — no tower, no Session,
// no floors — so this is the smallest thing that lets the four fighting styles actually be
// met rather than only existing in data. Beat one and the next steps up; lose and the same
// one comes round again.
//
// **It is scaffolding and the tower replaces it wholesale.** MECHANICS.md already decides
// 8 floors x 3 fights with doors between them, so nothing here is a design decision being
// made early — it is a list, in a constant, standing in for a generator.
//
// What it does fix now is a genuine dead end: before this, winning left the screen with
// every button dark and no way to play on short of restarting the process.
var enemyRoster = []string{"Monster1", "Swarm1", "Warden1", "Tactician1"}

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

	// The card currently being dragged, if any. See combat_actionbox.go.
	drag *dragState

	// showDeck toggles the deck overlay. While it is up the cards underneath do not
	// respond, so reading the deck cannot accidentally re-plan the round.
	showDeck bool

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
	deckButton    *models.Button
}

// Init prepares a fresh duel. Safe to re-enter: the combatants and the button are
// built once, everything else resets every visit.
func (s *CombatScene) Init(gs *state.GlobalState) {
	if s.fighter == nil {
		s.fighter = combatantFromRecord(gs, "Fighter1")
	}

	// The enemy is rebuilt every visit rather than once, because fightIndex may have moved
	// since the last one. Init is how the next fight starts, not only how the screen is
	// entered — see nextFight.
	s.enemy = combatantFromRecord(gs, enemyRoster[s.fightIndex%len(enemyRoster)])

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
	if s.deckButton == nil {
		s.deckButton = models.NewButton(138, 50, "Deck", s.toggleDeck)
		s.deckButton.BaseColor = color.RGBA{R: 70, G: 130, B: 230, A: 255} // blue
	}

	// Discard and DUEL! sit together because they are the same choice: both act on the
	// selection, and pressing one is deciding what that selection was for. They sit
	// directly under the hand, next to the cards being selected, so the choice is made
	// where it is expressed. Deck is off at the far end — it changes nothing and belongs
	// nowhere near them. All three share one band under the row, so it reads as one strip.
	s.discardButton.ScreenX = gs.PctX(20)
	s.discardButton.ScreenY = gs.PctY(95)
	s.duelButton.ScreenX = gs.PctX(33)
	s.duelButton.ScreenY = gs.PctY(95)
	s.deckButton.ScreenX = gs.PctX(88)
	s.deckButton.ScreenY = gs.PctY(95)

	s.showDeck = false
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

	// Planned up front only so the enemy pane has something in it before the first
	// DUEL!. startRound re-plans it every round regardless, so this is display, not a
	// commitment the resolver ever reads.
	s.enemyActions = combat.PlanFor(s.enemy.Style, s.enemy.Duelist)

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
		s.fightIndex+1, enemyRoster[s.fightIndex%len(enemyRoster)], s.enemy.Style,
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
		systems.UpdateButton(gs, s.deckButton)
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
	systems.UpdateButton(gs, s.deckButton)
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
	s.enemyActions = combat.PlanFor(s.enemy.Style, s.enemy.Duelist)

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
// bars; the rest are for the caption and, later, animation cues.
func (s *CombatScene) applyEvent(e combat.Event) {
	if e.Kind != combat.KindDamage {
		return
	}

	if e.Target == combat.SideA {
		s.fighter.CurrentLife = e.Life
	} else {
		s.enemy.CurrentLife = e.Life
	}
}

// caption describes where the round has got to, so the duel is legible before the
// action box exists.
func (s *CombatScene) caption() string {
	// The end of a fight has to say what happens next, not just what happened. The button
	// beside it changed its own label to match, and a caption that stopped at "you win!"
	// left the only live control on screen unexplained.
	if !s.enemy.Alive() {
		if s.fightIndex+1 < len(enemyRoster) {
			return fmt.Sprintf("The monster falls in round %d - press Next for %s",
				s.round, enemyRoster[s.fightIndex+1])
		}
		// The roster wraps rather than ending, because there is no run structure for it to
		// end into yet. Say so instead of implying a victory screen that does not exist.
		return fmt.Sprintf("The monster falls in round %d - that is all of them, Next starts over",
			s.round)
	}
	if !s.fighter.Alive() {
		return fmt.Sprintf("You fall in round %d. Press Retry.", s.round)
	}

	// Between rounds: show the plan and what it costs. Over budget it must not say "press
	// DUEL!", because DUEL! is dead — it says what to do about it instead.
	if s.cursor >= len(s.log) {
		spent, budget := combat.CostOf(s.fighterActions), s.fighter.ActionPoints()
		tail := "press DUEL!"
		if spent > budget {
			tail = fmt.Sprintf("%d over - discard or deselect", spent-budget)
		}
		return fmt.Sprintf("Round %d - your plan: %s  (%d/%d AP)   %s",
			s.round+1, planLabel(s.fighterActions), spent, budget, tail)
	}

	// **During playback the caption says nothing, on purpose** *(2026-08-07)*. It used to
	// narrate one event at a time, which meant the whole account of a round existed only as a
	// quarter-second flash — a combo forming was unreadable, and the block that halved a Heavy
	// went past before it could be noticed. That job belongs to the Resolution pane now, which
	// keeps every line instead of replacing it.
	//
	// Leaving the caption to also narrate would put the newest line on screen twice, in two
	// places, which is the thing the pane was added to fix. So the two have one job each: the
	// pane records what happened, the caption proposes what to do next.
	return ""
}

func (s *CombatScene) Draw(gs *state.GlobalState, screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 50, G: 50, B: 50, A: 255})

	s.drawHandRow(gs, screen)
	// EXPERIMENT 2026-08-07: Action Flow is not drawn, and Resolution has taken its column as
	// well as its own. drawActionFlow and actionFlowRows are deliberately left in place and
	// unwired so this is one line to put back.
	//
	// **What this gives up, if it stays:** the enemy's queued shape while planning. Those
	// `??? (attack)` rows are the tell — see the concealment section of the combat-screen
	// skill — and Resolution is empty until DUEL! is pressed, so nothing on screen says what
	// the opponent is about to do.
	s.drawResolution(gs, screen)
	s.drawCaptionBox(gs, screen)
	s.drawFighterBlock(gs, screen)

	// The fighter's sprite and health bar are gone — the block above carries its state as
	// numbers instead. The enemy keeps both for now, moved up with the pane: at 50% its
	// health bar, which hangs 100px below it, ran into the top of the card row.
	s.drawCombatant(gs, screen, s.enemy, float64(gs.PctX(88)), float64(gs.PctY(34)))
	systems.DrawButton(gs, screen, s.duelButton)
	systems.DrawButton(gs, screen, s.discardButton)
	systems.DrawButton(gs, screen, s.deckButton)

	// Last, so the card in hand rides over the panes and the button it passes across.
	s.drawDraggedCard(gs, screen)

	// The overlay covers everything, card in flight included — and then Deck is drawn
	// again on top of it. While the deck is open it is the only control that still does
	// anything, so it is the only one that still looks like it does.
	if s.showDeck {
		s.drawDeckOverlay(gs, screen)
		systems.DrawButton(gs, screen, s.deckButton)
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
	trace.Rect("actionFlowPane", image.Rect(
		gs.PctX(actionFlowPane.leftPct), gs.PctY(paneTopPct),
		gs.PctX(actionFlowPane.rightPct), gs.PctY(paneBottomPct)))
	trace.Rect("resolutionPane", image.Rect(
		gs.PctX(resolutionPane.leftPct), gs.PctY(paneTopPct),
		gs.PctX(resolutionPane.rightPct), gs.PctY(paneBottomPct)))
	trace.Rect("captionBox", image.Rect(
		band.Min.X, gs.PctY(captionTopPct),
		band.Max.X, gs.PctY(captionTopPct)+captionHeight))
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
	}{{"discard", s.discardButton}, {"duel", s.duelButton}, {"deck", s.deckButton}} {
		trace.Logf("layout", "%-18s centre %4d,%-4d", "button "+b.name, b.b.ScreenX, b.b.ScreenY)
	}
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

// combatantFromRecord resolves a combatant record and its sprite sheet out of global
// state, then hands both to the entity constructor.
func combatantFromRecord(gs *state.GlobalState, record string) *entities.Combatant {
	d := gs.Combatants[record]
	return entities.NewCombatantFrom(d, gs.Assets[d.SpriteSheet])
}
