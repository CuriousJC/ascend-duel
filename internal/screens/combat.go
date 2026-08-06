package screens

import (
	"fmt"
	"image"
	"math/rand"
	"sort"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/entities"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/curiousjc/ascend-duel/internal/trace"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

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

	// eventDwellTicks is three quarters of a second: long enough that a negation or a
	// banked point can be read, short enough that a full round is not a cutscene. It sits
	// between the old action dwell and the old damage dwell on purpose.
	eventDwellTicks = 3 * ticksPerSecond / 4
)

// element is what a card is made of.
//
// **Still colour and nothing else in the code** — no rule reads it and ResolveRound never
// sees it, which is why it lives here on the screen rather than in internal/combat.
//
// That is no longer the design. Elements became mechanical on 2026-08-05: each applies a
// status to the opponent (ice cuts their AP, lightning adds a miss chance, fire is a damage
// over time, earth blunts their damage) and a matching ring discounts cards of that element.
// See MECHANICS.md. When that lands the type has to cross into internal/combat — cost stops
// being a property of the card and becomes a property of the pairing, the way Damage(str)
// already is — and ResolveRound, ResolutionOrder, the planners and every test in that package
// grow with it. Cheap to move now, less so later.
//
// Primary elements get cards; secondary ones (poison, force, hunger) do not, and where they
// appear is still open. Poison is in the starting deck only because it predates the split.
//
// This is what the reserved colour was reserved for: the glyphs were given one hueless
// palette on 2026-08-03 specifically so colour would still be free to mean something when
// elements arrived.
type element int

const (
	elementBasic element = iota

	// Primary. One card of each per concept, and a ring for each.
	elementFire
	elementIce
	elementLightning
	elementEarth

	// Secondary. No cards yet.
	elementPoison
)

// elementColors is the surface colour of a card, at full strength — drawCard scales it
// down for the resting, selected and unaffordable states.
//
// White is the *absence* of an element rather than a colour of its own. A plain card makes
// no claim, so the coloured ones are the ones that catch the eye, which is the whole point
// of painting them before the mechanic exists.
//
// An array rather than a map: nothing here iterates, but a map in this package is one
// refactor away from something that does, and Go randomises that order.
var elementColors = [...]color.RGBA{
	elementBasic:     {R: 235, G: 235, B: 235, A: 255},
	elementFire:      {R: 235, G: 120, B: 45, A: 255},
	elementIce:       {R: 80, G: 155, B: 230, A: 255},
	elementLightning: {R: 240, G: 205, B: 55, A: 255},
	elementEarth:     {R: 150, G: 105, B: 60, A: 255},
	elementPoison:    {R: 70, G: 140, B: 60, A: 255},
}

func (e element) color() color.RGBA { return elementColors[e] }

// elementNames is for tracing and for anything that has to say an element out loud. An
// array indexed by the constant rather than a map, for the same reason as elementColors.
var elementNames = [...]string{
	elementBasic:     "basic",
	elementFire:      "fire",
	elementIce:       "ice",
	elementLightning: "lightning",
	elementEarth:     "earth",
	elementPoison:    "poison",
}

func (e element) String() string {
	if int(e) >= len(elementNames) {
		return "?"
	}
	return elementNames[e]
}

// actionCard is one instance in the piles. A card used to *be* a combat.ActionKind, so two
// Strikes were the same value and indistinguishable; an element makes them differ, so the
// deck, hand and discard hold structs even while the element does nothing but paint.
type actionCard struct {
	action  combat.ActionKind
	element element
}

// The hand drawn from the deck each round. Quick is not in the deck — the deck is what the
// player owns, and which of the actions the rules define actually appear is a deck-building
// question rather than a rules one.
//
// Eight from five on 2026-08-04. Eight cards do not fit the screen side by side, which is
// what the overlap in handPitch is for — the hand is expected to go past eight sometimes.
// Eight against the 30-card deck below is roughly the ratio it was sized for.
const handSize = 8

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

// primaries is the element set that gets cards, in the order a concept's five are built.
// Secondary elements — poison, force, hunger — deliberately have none; where they appear is
// still open. See MECHANICS.md.
var primaries = []element{elementBasic, elementFire, elementIce, elementLightning, elementEarth}

// conceptDeck is one card of a concept in every primary element — the unit the deck is built
// from. Decided 2026-08-05: **a concept ships as five cards**, and that is the rule for
// adding concepts rather than a description of this particular deck. A new concept arrives as
// a set of five, never as a lone card.
func conceptDeck(action combat.ActionKind) []deckEntry {
	entries := make([]deckEntry, 0, len(primaries))
	for _, e := range primaries {
		entries = append(entries, deckEntry{actionCard{action, e}, 1})
	}
	return entries
}

// The starting deck: every concept, five cards each, grouped by the category it resolves in.
//
// **Six concepts, 30 cards**, split evenly 10 setup / 10 attack / 10 defend. That even split
// is a consequence of the concept list rather than a target — Guard moved from defend to
// setup on 2026-08-06 and Parry was dropped the same day, which is what turned the lopsided
// 1/2/4 the design started from into 2/2/2. The 5-setup/10-attack/20-defend shape recorded
// in MECHANICS.md is the thing that went; the two-thirds-defensive theory went with it.
//
// Quick is still homeless: it is an ActionKind with a cost and damage and no concept, so it
// has no five cards to arrive as.
var startingDeck = concat(
	// Setup.
	conceptDeck(combat.Prepare),
	conceptDeck(combat.Guard),

	// Attack.
	conceptDeck(combat.Strike),
	conceptDeck(combat.Heavy),

	// Defend.
	conceptDeck(combat.Dodge),
	conceptDeck(combat.Riposte),
)

func concat(groups ...[]deckEntry) []deckEntry {
	var out []deckEntry
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}

// deckEntry is one line of a deck list: a card and how many copies of it.
type deckEntry struct {
	card  actionCard
	count int
}

// deckSeed fixes the shuffle so every launch deals the same cards, which is what makes a
// layout problem reproducible while the screen is being built. It is a placeholder for the
// per-run seed described in TODO.md — when Session state exists this reads from there, and
// the rest of the deck code does not change.
const deckSeed = 1

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
	// Prepare from a duel that has been walked away from.
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
// guard or a banked Prepare would otherwise be inherited by the next one.
//
// It sets the fields by name rather than rebuilding the struct: Con/Str/Spd/MaxLife come
// from the data record and must survive, and a zero literal here would quietly wipe them
// the first time someone re-entered the screen.
func resetCombatState(d combat.Duelist) combat.Duelist {
	d.Guarded = false
	d.Ripostes = 0
	d.Dodges = 0
	d.BonusAP = 0
	d.PreparedAP = 0
	return d
}

// discardSelected throws the selected cards away: they leave the hand for the discard
// pile, and the hand is dealt back up to size from the draw pile.
//
// Selection does double duty here — it is both "queue this for the round" and "this is
// the one I mean" — so throwing a card out costs the action points it was holding until
// the moment it leaves. That is a consequence of overloading selection rather than a
// designed cost; see TODO.md.
func (s *CombatScene) discardSelected() {
	if !s.planning() || s.discardsLeft <= 0 {
		return
	}
	s.discardsLeft--
	s.spendSelected()

	trace.Logf("input", "discard pressed, %d left this round, hand now %s",
		s.discardsLeft, handLabel(s.hand))
}

// spendSelected moves every selected card to the discard pile, deals the hand back up to
// size, and rebuilds the queue from what is left.
//
// **Selected is the only thing that leaves a hand.** Both ways a card can go — thrown away
// by Discard, or played by DUEL! — are the same movement over the same predicate, so they
// are one function rather than two that have to be kept in agreement.
func (s *CombatScene) spendSelected() {
	// Filtering in place over the hand's own array. Safe because kept never runs ahead of
	// the read cursor, and it keeps the surviving cards in the order the player left them.
	kept := s.hand[:0]
	for _, c := range s.hand {
		if c.selected {
			s.discard = append(s.discard, c.actionCard)
			continue
		}
		kept = append(kept, c)
	}
	s.hand = kept

	s.drawHand()
	s.syncQueue()
}

// toggleDeck shows or hides the deck overlay.
func (s *CombatScene) toggleDeck() {
	s.showDeck = !s.showDeck
	trace.Logf("input", "deck overlay %v (draw %d, discard %d)",
		s.showDeck, len(s.deck), len(s.discard))
}

// resetDeck builds the starting deck, shuffles it, empties the discard and deals an
// opening hand.
func (s *CombatScene) resetDeck() {
	s.deck = s.deck[:0]
	for _, e := range startingDeck {
		for i := 0; i < e.count; i++ {
			s.deck = append(s.deck, e.card)
		}
	}

	s.discard = s.discard[:0]
	s.hand = s.hand[:0]

	s.shuffleDeck()
	s.drawHand()
}

// shuffleDeck shuffles the draw pile using the scene's own source. Never rand.Shuffle,
// which draws from the global source and would make the deal unreproducible.
func (s *CombatScene) shuffleDeck() {
	s.rng.Shuffle(len(s.deck), func(i, j int) {
		s.deck[i], s.deck[j] = s.deck[j], s.deck[i]
	})
}

// drawHand fills the hand up to handSize, reshuffling the discard back into the draw pile
// when it runs dry. A hand can come up short only if every card the player owns is already
// in it, which cannot happen with a deck larger than the hand.
func (s *CombatScene) drawHand() {
	for len(s.hand) < handSize {
		if len(s.deck) == 0 {
			if len(s.discard) == 0 {
				return
			}
			s.deck = append(s.deck, s.discard...)
			s.discard = s.discard[:0]
			s.shuffleDeck()
		}

		last := len(s.deck) - 1
		s.hand = append(s.hand, paletteCard{actionCard: s.deck[last]})
		s.deck = s.deck[:last]
	}
}

// endRoundHand spends what was played and refills. **Only the cards that were actually
// played leave** — anything still sitting unselected in the hand stays exactly where it is,
// and the draw tops the hand back up to size.
//
// This reverses the decision recorded in TODO.md that the whole hand discarded every round,
// played or not, on the grounds that a hand kept back would let a plan be prepared once and
// repeated. What that actually produced was a hand you could not build on: cards you had
// deliberately held were taken away for having been held, so the only way to keep anything
// was to play it. Refilling only what was used is what makes a hand something you shape
// across rounds rather than a fresh deal you react to.
//
// It also gives Discard a real job. A card you never want now sits in your hand until you
// throw it out, so the discard button is how you clear it rather than a shortcut for
// something the round boundary was going to do anyway.
func (s *CombatScene) endRoundHand() {
	s.spendSelected()
	s.discardsLeft = discardsPerRound
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
	case combat.KindPrepared:
		return fmt.Sprintf("prepared    %v banks %d AP for next round", e.Side, e.Amount)
	case combat.KindNegated:
		return fmt.Sprintf("negated     %v's %v stops %v cold", e.Side, e.Action, e.Target)
	case combat.KindGuarded:
		return fmt.Sprintf("guarded     %v halves it to %d (target on %d)", e.Target, e.Amount, e.Life)
	case combat.KindDamage:
		return fmt.Sprintf("damage      %v hits %v for %d, leaving %d", e.Side, e.Target, e.Amount, e.Life)
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

	e := s.log[s.cursor]
	who := "Fighter"
	if e.Side == combat.SideB {
		who = "Monster"
	}

	switch e.Kind {
	case combat.KindRoundStart:
		return fmt.Sprintf("Round %d!", e.Round)
	case combat.KindAction:
		return fmt.Sprintf("%s uses %s", who, e.Action)
	case combat.KindPrepared:
		return fmt.Sprintf("%s banks %d AP for next round", who, e.Amount)
	case combat.KindNegated:
		// who is the *defender* here — the event belongs to whoever's defense fired, not
		// to the attack it stopped.
		return fmt.Sprintf("%s's %s stops it dead", who, e.Action)
	case combat.KindGuarded:
		return "Guarded! Damage halved"
	case combat.KindDamage:
		return fmt.Sprintf("%s hits for %d", who, e.Amount)
	case combat.KindDefeated:
		return fmt.Sprintf("%s lands the finishing blow", who)
	default:
		return ""
	}
}

func (s *CombatScene) Draw(gs *state.GlobalState, screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 50, G: 50, B: 50, A: 255})

	s.drawHandRow(gs, screen)
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
}

// Deck overlay geometry. The panel is nearly the whole screen and stops above the button
// band, so the Deck button that closes it stays outside the panel as well as on top of it.
//
// The grid holds every card that is not in hand, which is at most deck minus handSize —
// 22 of the 30 today. Both piles are drawn into one fixed grid, so nothing below moves as
// cards shift from one pile to the other.
const (
	deckPanelLeftPct  = 4
	deckPanelRightPct = 96
	deckPanelTopPct   = 4
	// 92 rather than 86: at 86 the panel stopped short of the hand, so the tops of the cards
	// and the whole AP line sat below it, dimmed by the scrim but still visibly outside the
	// dialog. It still ends above the button band, which is what keeps the Deck button that
	// closes it outside the panel as well as drawn on top of it.
	deckPanelBottomPct = 92

	// Eight columns of 138 plus the gaps comes to 1160 inside the panel's 1177. Three rows
	// of 186-tall cards comes to 578 inside the ~713 the panel has below its heading, which
	// is 24 slots against the 22 cards that can be outside the hand.
	//
	// The third row is paid for by the card losing its initiative glyph: a two-glyph column
	// is 50 pixels shorter than a three-glyph one, and the deck doubled to 30 cards on the
	// same day. Without that the overflow line would fire on every look.
	deckGridColumns = 8
	deckGridGap     = 8
	deckGridRows    = 3

	// Offsets down from the panel's top edge.
	deckTitleTop  = 40
	deckCountsTop = 78
	deckLegendTop = 100
	deckGridTop   = 132
	deckHintUp    = 22 // hint's distance up from the panel's bottom edge
)

// drawDeckOverlay covers the screen with what is left to draw.
//
// The piles are drawn as the cards themselves at half size rather than as a table of
// counts. A count could say "six Strikes"; it could not say which of them are fire and
// which are plain, and with elements on the cards that is most of what the player wants to
// know. Twenty half-size cards fit on the panel at once, so the whole deck is one look.
//
// Both piles in one grid, with the discarded ones dimmed. The discard belongs here because
// those cards are coming back — a reshuffle folds the pile in, so "what is left" honestly
// means both piles — and merging them is what lets a card stay in place and simply dim
// when it is spent. See drawPileGrid.
func (s *CombatScene) drawDeckOverlay(gs *state.GlobalState, screen *ebiten.Image) {
	if !s.showDeck {
		return
	}

	// A scrim over everything, so the panel reads as covering the screen rather than
	// floating on it, and so the cards underneath look as inert as they now are.
	bounds := screen.Bounds()
	vector.DrawFilledRect(screen, 0, 0,
		float32(bounds.Dx()), float32(bounds.Dy()),
		color.RGBA{A: 190}, false)

	left, top := float32(gs.PctX(deckPanelLeftPct)), float32(gs.PctY(deckPanelTopPct))
	width := float32(gs.PctX(deckPanelRightPct)) - left
	height := float32(gs.PctY(deckPanelBottomPct)) - top

	vector.DrawFilledRect(screen, left, top, width, height,
		color.RGBA{R: 30, G: 30, B: 38, A: 255}, false)
	vector.StrokeRect(screen, left, top, width, height, 2, apBarColor, false)

	heading := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 28}
	small := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 14}

	title := &text.DrawOptions{}
	title.GeoM.Translate(float64(left+width/2), float64(top+deckTitleTop))
	title.PrimaryAlign = text.AlignCenter
	text.Draw(screen, "What is left", heading, title)

	// Hyphens, not em dashes. The kubasta font has no U+2014 and draws a missing-glyph box
	// for it — the middle dot is in the font, the dash is not.
	line := func(y float32, s string) {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(left+width/2), float64(top+y))
		op.PrimaryAlign = text.AlignCenter
		text.Draw(screen, s, small, op)
	}
	line(deckCountsTop, fmt.Sprintf("draw %d  ·  discard %d  ·  %d in hand",
		len(s.deck), len(s.discard), len(s.hand)))
	line(deckLegendTop, "dimmed cards are in the discard - they return on the next reshuffle")

	s.drawPileGrid(gs, screen, left+width/2, top+deckGridTop)

	hint := &text.DrawOptions{}
	hint.GeoM.Translate(float64(left+width/2), float64(top+height-deckHintUp))
	hint.PrimaryAlign = text.AlignCenter
	text.Draw(screen, "Deck again to close", small, hint)
}

// pileEntry is one card in the overlay and whether it can be drawn right now.
type pileEntry struct {
	card      actionCard
	available bool
}

// drawPileGrid lays every card outside the hand into one grid, centred on centerX.
//
// One grid rather than a section per pile. Two sections of full-height cards do not fit the
// panel, and the single grid turns out to be the better picture anyway: sorting by kind and
// element rather than by which pile a card is in means **a card does not move when it is
// discarded, it only dims**. Watching your deck drain in place reads far better than
// watching cards jump between two lists.
//
// Sorted, never in pile order. The draw pile is shuffled, and drawing it in order would hand
// the player their next five cards and make the shuffle pointless. This is a picture of what
// the piles hold, not of their sequence.
func (s *CombatScene) drawPileGrid(gs *state.GlobalState, screen *ebiten.Image, centerX, top float32) {
	entries := make([]pileEntry, 0, len(s.deck)+len(s.discard))
	for _, c := range s.deck {
		entries = append(entries, pileEntry{c, true})
	}
	for _, c := range s.discard {
		entries = append(entries, pileEntry{c, false})
	}

	// Availability is the last key rather than the first, so identical cards sit together
	// and the drawable one of a pair comes first. That is what keeps a card's position
	// steady as it moves from the draw pile to the discard.
	sort.Slice(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.card.action != b.card.action {
			return a.card.action < b.card.action
		}
		if a.card.element != b.card.element {
			return a.card.element < b.card.element
		}
		return a.available && !b.available
	})

	pitchX := deckCardStyle.width + deckGridGap
	pitchY := deckCardStyle.height + deckGridGap
	rowLeft := int(centerX) - (deckGridColumns*pitchX-deckGridGap)/2
	slots := deckGridColumns * deckGridRows

	for i, e := range entries {
		if i >= slots {
			break
		}

		at := image.Pt(
			rowLeft+(i%deckGridColumns)*pitchX,
			int(top)+(i/deckGridColumns)*pitchY)

		// enabled carries "can be drawn", not "can be afforded". Never selected: this is an
		// inventory, not a choice, and dimming by the round's remaining AP would say
		// something about a budget that has nothing to do with a pile you cannot play from.
		drawCard(gs, screen, at, deckCardStyle, e.card, e.available, false, s.fighter.Str)
	}

	// The grid holds 24 and the deck puts at most 22 outside the hand, so this cannot fire
	// today — but deckbuilding will grow the deck, and a panel that silently drops the
	// overflow would be a picture that lies about what you own.
	if over := len(entries) - slots; over > 0 {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(centerX), float64(int(top)+deckGridRows*pitchY))
		op.PrimaryAlign = text.AlignCenter
		text.Draw(screen, fmt.Sprintf("+%d more not shown", over),
			&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 14}, op)
	}
}

// The Resolution pane's vertical band. It reaches higher than it did — the heading text
// that used to sit at the top of the screen is gone — and stops well short of the bottom,
// because the hand took the lower third when the cards turned portrait and the caption box
// sits between the two.
const (
	paneTopPct    = 12
	paneBottomPct = 46

	paneTitleInset = 10 // gap from the pane's top edge to its title
	paneFirstRow   = 45 // gap from the top edge to the first action row
	paneRowHeight  = 30
	paneRowInset   = 10 // gap from the pane's left edge to a row's swatch
	swatchSize     = 16
	swatchGap      = 6 // gap between a swatch and its label
)

// panePlacement is one pane's horizontal slot, label and identifying colour. The
// colours are loud on purpose — these are placeholders for finding the layout, not a
// palette anyone has chosen yet.
type panePlacement struct {
	leftPct, rightPct int
	title             string
	color             color.RGBA
}

// One pane now. The Actions pane went with the move to portrait cards along the bottom:
// the hand is no longer a column and has no frame at all, so there was nothing left for a
// placement to hold. What the vacated 15–39% column does with the space is deliberately
// undecided — see TODO.md — so it is simply empty rather than filled with something
// arbitrary.
var resolutionPane = panePlacement{leftPct: 45, rightPct: 78, title: "Resolution", color: color.RGBA{R: 235, G: 105, B: 170, A: 255}}

// The two sides' colours. playerSwatch is what the cards and the player's resolution rows
// are drawn in; enemySwatch marks the opponent's rows. Neither has a pane to take a colour
// from any more, but the round still has two sides and they have to be told apart at a
// glance — green is you, yellow is them.
var (
	playerSwatch = color.RGBA{R: 60, G: 200, B: 90, A: 255}
	enemySwatch  = color.RGBA{R: 225, G: 200, B: 60, A: 255}
)

// paneRow is one line in a pane: a label, optionally preceded by a colour swatch
// saying whose action it is. A zero-alpha swatch means the row has none, in which case
// the label is centred instead of sitting in a column beside the squares.
type paneRow struct {
	label  string
	swatch color.RGBA

	// highlighted marks the row as the one happening right now, drawn lit against the
	// dim pane behind it.
	highlighted bool
}

// drawResolution shows the two queues merged into play order.
func (s *CombatScene) drawResolution(gs *state.GlobalState, screen *ebiten.Image) {
	s.drawPane(gs, screen, resolutionPane,
		s.resolutionRows(s.fighterActions, s.enemyActions, s.concealEnemy(gs)))
}

// concealEnemy reports whether the opponent's queued actions should be hidden from the
// player. True while planning, false once the round is playing back — an action that has
// happened is not a secret — and always false with DebugGameplay on.
//
// What concealment hides is *what* the enemy queued, not *how many* actions it queued:
// a concealed queue still occupies its real number of rows in both panes. That leaks the
// opponent's action-point spend, which against a greedy planner is most of the tell. It
// is deliberate rather than overlooked: collapsing the rows would hide the spend but would
// also destroy the Resolution pane's account of who acts when, and that alternation is a
// rule the player is meant to read and eventually manipulate. Revisit alongside the wider
// hidden-information decision — see TODO.md.
func (s *CombatScene) concealEnemy(gs *state.GlobalState) bool {
	return !gs.DebugGameplay && s.planning()
}

// drawBox draws a framed panel: dim fill, full-strength border, and a title centred on the
// top edge. Dim fill and a full-strength border means the box reads as green or pink at a
// glance without drowning what is drawn on top of it.
//
// This takes a rectangle rather than a panePlacement because the boxes along the bottom
// are sized by the hand rather than by a slice of the screen. An empty title draws none.
func drawBox(gs *state.GlobalState, screen *ebiten.Image, r image.Rectangle, c color.RGBA, title string) {
	x, y := float32(r.Min.X), float32(r.Min.Y)
	w, h := float32(r.Dx()), float32(r.Dy())

	vector.DrawFilledRect(screen, x, y, w, h, systems.ColorAtStrength(c, 25), false)
	vector.StrokeRect(screen, x, y, w, h, 2, c, false)

	if title == "" {
		return
	}
	titleOp := &text.DrawOptions{}
	titleOp.GeoM.Translate(float64(x+w/2), float64(y+paneTitleInset))
	titleOp.PrimaryAlign = text.AlignCenter
	text.Draw(screen, title, &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 16}, titleOp)
}

// drawPaneFrame draws a column's fill, border and title, and reports its rectangle.
// Split out because the card panes fill themselves rather than drawing text rows.
func (s *CombatScene) drawPaneFrame(gs *state.GlobalState, screen *ebiten.Image, p panePlacement) (x, y, w, h float32) {
	r := image.Rect(
		gs.PctX(p.leftPct), gs.PctY(paneTopPct),
		gs.PctX(p.rightPct), gs.PctY(paneBottomPct),
	)
	drawBox(gs, screen, r, p.color, p.title)

	return float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy())
}

// The caption box sits between the Resolution pane and the hand, and takes its width from
// the hand rather than from the pane above it — the same band the AP bar spans, so the two
// line up on both edges however many cards are held.
const (
	captionTopPct = 48
	captionHeight = 56
)

// drawCaptionBox narrates the round. The caption used to be loose text at a hardcoded
// 50,100 with nothing around it; giving it a frame in the middle column puts it where the
// player is already looking, between the order they planned and the cards they planned it
// with.
func (s *CombatScene) drawCaptionBox(gs *state.GlobalState, screen *ebiten.Image) {
	band := handBand(gs, s.laidOutCount())
	top := gs.PctY(captionTopPct)
	r := image.Rect(band.Min.X, top, band.Max.X, top+captionHeight)

	drawBox(gs, screen, r, resolutionPane.color, "")

	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(r.Min.X+r.Dx()/2), float64(r.Min.Y+r.Dy()/2))
	op.PrimaryAlign = text.AlignCenter
	op.SecondaryAlign = text.AlignCenter
	text.Draw(screen, s.caption(),
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 18}, op)
}

// The character block, in the column the Actions pane vacated. It replaced the fighter's
// sprite and health bar: a bar says roughly how hurt you are, and a duel decided in whole
// points of damage wants the exact number.
const (
	blockLeftPct = 4
	blockTopPct  = 12

	blockWidth  = 268
	blockHeight = 196

	blockInset     = 16 // gap from the block's edge to a label
	blockLifeTop   = 44 // "Health", small, above the figure
	blockLifeMid   = 84 // centre line of the life figure itself
	blockFirstRow  = 130
	blockRowHeight = 32
)

// lifeColor is the red the life fraction is written in. It is the one place on the screen
// that has to be found without being looked for, so it gets a colour nothing else uses.
var lifeColor = color.RGBA{R: 225, G: 65, B: 65, A: 255}

// drawFighterBlock draws what the player is: life, discards left this round, and vitae.
//
// Vitae is a placeholder reading a fixed 5 — it has no rule behind it yet. It is drawn
// anyway so the block has its real shape while the rest of the character's state is
// decided, rather than being retrofitted into a box already sized without it.
func (s *CombatScene) drawFighterBlock(gs *state.GlobalState, screen *ebiten.Image) {
	left, top := gs.PctX(blockLeftPct), gs.PctY(blockTopPct)
	r := image.Rect(left, top, left+blockWidth, top+blockHeight)

	drawBox(gs, screen, r, playerSwatch, "Fighter")

	small := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 13}
	row := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 18}
	life := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 34}

	label := &text.DrawOptions{}
	label.GeoM.Translate(float64(left+blockWidth/2), float64(top+blockLifeTop))
	label.PrimaryAlign = text.AlignCenter
	text.Draw(screen, "HEALTH", small, label)

	lifeOp := &text.DrawOptions{}
	lifeOp.GeoM.Translate(float64(left+blockWidth/2), float64(top+blockLifeMid))
	lifeOp.PrimaryAlign = text.AlignCenter
	lifeOp.SecondaryAlign = text.AlignCenter
	lifeOp.ColorScale.ScaleWithColor(lifeColor)
	text.Draw(screen, fmt.Sprintf("%d / %d", s.fighter.CurrentLife, s.fighter.MaxLife), life, lifeOp)

	// Label on the left, value on the right, so the numbers line up in a column and a
	// change in one of them is visible without reading the words beside it.
	counters := []struct {
		label string
		value int
	}{
		{"Discards", s.discardsLeft},
		{"Vitae", s.vitae},
	}
	for i, c := range counters {
		y := float64(top + blockFirstRow + i*blockRowHeight)

		nameOp := &text.DrawOptions{}
		nameOp.GeoM.Translate(float64(left+blockInset), y)
		text.Draw(screen, c.label, row, nameOp)

		valueOp := &text.DrawOptions{}
		valueOp.GeoM.Translate(float64(left+blockWidth-blockInset), y)
		valueOp.PrimaryAlign = text.AlignEnd
		text.Draw(screen, fmt.Sprintf("%d", c.value), row, valueOp)
	}
}

// drawPane draws a read-only column: the frame, then a row per action.
func (s *CombatScene) drawPane(gs *state.GlobalState, screen *ebiten.Image, p panePlacement, rows []paneRow) {
	x, y, w, _ := s.drawPaneFrame(gs, screen, p)

	face := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 16}

	for i, row := range rows {
		rowY := y + paneFirstRow + float32(i*paneRowHeight)
		rowOp := &text.DrawOptions{}

		// The row happening right now gets a lit bar the full width of the pane, so it
		// carries across the room at a glance rather than needing the label read.
		if row.highlighted {
			vector.DrawFilledRect(screen, x+2, rowY-4, w-4, paneRowHeight-2,
				systems.ColorAtStrength(p.color, 70), false)
		}

		if row.swatch.A == 0 {
			rowOp.GeoM.Translate(float64(x+w/2), float64(rowY))
			rowOp.PrimaryAlign = text.AlignCenter
		} else {
			// A swatch turns the row into a column: square on the left, label beside it,
			// so the squares line up down the pane and the alternation is readable as a
			// pattern rather than as text.
			//
			// Idle swatches are dimmed so the lit one is the brightest thing in the pane.
			swatch := row.swatch
			if !row.highlighted {
				swatch = systems.ColorAtStrength(swatch, 55)
			}
			vector.DrawFilledRect(screen, x+paneRowInset, rowY+2, swatchSize, swatchSize, swatch, false)
			rowOp.GeoM.Translate(float64(x+paneRowInset+swatchSize+swatchGap), float64(rowY))
		}

		text.Draw(screen, row.label, face, rowOp)
	}
}

// resolutionRows interleaves the two queued sets one action each, and marks the row for
// the action currently playing back. Each row is swatched in its side's pane colour, so
// who-acts-when reads as a pattern of squares before any of the labels are read.
//
// Whichever set is longer keeps going alone once the other runs out — a faster duelist
// buys more actions, and the tail is exactly where that advantage shows.
//
// This layout is the order combat.ResolveRound actually plays, so the highlight walks
// straight down the pane. Keep the two in step: the pane is the player's model of the
// round, and effects that reorder resolution will have to move both.
// concealEnemy replaces the opponent's labels with placeholders while leaving their rows
// in place, so the interleaving still reads correctly and only the content is withheld.
//
// This function needs no change when phase-based resolution lands — it draws whatever
// ResolutionOrder returns and never works the order out for itself, which is the whole
// point of that split. What it *will* need is a way to draw a combo spanning two or more
// slots that need not be adjacent; one row per slot with a single walking highlight has no
// way to say "these together did a thing". See MECHANICS.md.
func (s *CombatScene) resolutionRows(fighter, enemy []combat.ActionKind, concealEnemy bool) []paneRow {
	order := combat.ResolutionOrder(fighter, enemy)
	if len(order) == 0 {
		return []paneRow{{label: "(empty)"}}
	}

	playingSlot, playing := s.currentSlot()

	rows := make([]paneRow, 0, len(order))
	for i, slot := range order {
		label, swatch := slot.Action.String(), playerSwatch
		if slot.Side == combat.SideB {
			swatch = enemySwatch
			if concealEnemy {
				label = concealedLabel(slot.Action)
			}
		}

		rows = append(rows, paneRow{
			label:       label,
			swatch:      swatch,
			highlighted: playing && i == playingSlot,
		})
	}
	return rows
}

// concealedLabel is what a hidden action shows instead of its name. The category is
// deliberately not hidden: it is what decides where the action sits in the order, so
// withholding it would make the Resolution pane unreadable rather than merely uncertain —
// the player could not tell why the rows are arranged as they are. It replaced the
// initiative number in exactly that job when initiative was removed.
//
// This is the first cut at graded reveal rather than the finished scheme. What else
// leaks per action — whether it damages, whether it applies a status — is still open;
// see TODO.md.
func concealedLabel(a combat.ActionKind) string {
	return fmt.Sprintf("??? (%s)", a.Category())
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

	played := -1
	for _, e := range s.log[:s.cursor+1] {
		if e.Kind == combat.KindAction {
			played++
		}
	}

	if played < 0 {
		return 0, false
	}
	return played, true
}

// drawCombatant draws one duelist and its health bar. Only the enemy uses it now — the
// fighter's sprite and bar became the character block — but it stays shaped for either.
func (s *CombatScene) drawCombatant(gs *state.GlobalState, screen *ebiten.Image, c *entities.Combatant, hPosition, vPosition float64) {
	var cm colorm.ColorM

	op := &colorm.DrawImageOptions{}
	op.GeoM.Translate(-float64(c.Sprite.Bounds().Dx())/2, -float64(c.Sprite.Bounds().Dy())/2) //center our origin
	op.GeoM.Translate(hPosition, vPosition)                                                   //position
	colorm.DrawImage(screen, c.Sprite, cm, op)

	DrawHealthBar(gs, screen, hPosition, vPosition, c.CurrentLife, c.MaxLife)
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
	trace.Rect("resolutionPane", image.Rect(
		gs.PctX(resolutionPane.leftPct), gs.PctY(paneTopPct),
		gs.PctX(resolutionPane.rightPct), gs.PctY(paneBottomPct)))
	trace.Rect("captionBox", image.Rect(
		band.Min.X, gs.PctY(captionTopPct),
		band.Max.X, gs.PctY(captionTopPct)+captionHeight))
	trace.Rect("fighterBlock", image.Rect(
		gs.PctX(blockLeftPct), gs.PctY(blockTopPct),
		gs.PctX(blockLeftPct)+blockWidth, gs.PctY(blockTopPct)+blockHeight))
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

// planLabel renders a queued set as "Guard + Strike + Quick".
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

func DrawHealthBar(gs *state.GlobalState, screen *ebiten.Image,
	hPositionEntity float64, vPositionEntity float64,
	currentLife int, maxLife int) {

	rectWidth := 150
	rectHeight := 25
	hPosition := hPositionEntity
	vPosition := vPositionEntity + 100                      //move down 100 px from the position
	rectColor := color.RGBA{A: 255, R: 96, G: 37, B: 37}    // Crimson red
	trackColor := color.RGBA{A: 255, R: 38, G: 22, B: 22}   // the drained portion behind the fill
	maskColor := color.RGBA{A: 255, R: 255, G: 192, B: 203} // mask color
	maskFill := color.RGBA{A: 0, R: 255, G: 255, B: 255}    //fill transparent

	//Rounded corners mask Image
	mask := ebiten.NewImage(rectWidth, rectHeight)
	mask.Fill(maskFill)
	CreateRoundedRecMask(mask, 0, 0, float32(rectWidth), float32(rectHeight), 10, maskColor)

	//Track plus the red fill, scaled to the current life fraction
	lifeRect := ebiten.NewImage(rectWidth, rectHeight)
	lifeRect.Fill(trackColor)
	if maxLife > 0 && currentLife > 0 {
		fillWidth := float32(rectWidth) * float32(currentLife) / float32(maxLife)
		vector.DrawFilledRect(lifeRect, 0, 0, fillWidth, float32(rectHeight), rectColor, false)
	}

	//Blend the life rectangle into the mask.  The source is the life rectangle
	//  and the mask will only display that portion of the source that is blended into the non-transparent alpha
	rectMaskOp := &ebiten.DrawImageOptions{}
	rectMaskOp.Blend = ebiten.BlendSourceIn
	mask.DrawImage(lifeRect, rectMaskOp) //blend source lifeRect into destination mask
	healthBar := mask                    //mask is now filled with transparent maskFill and the maskColor was overwritten with lifeRect

	healthBarOp := &ebiten.DrawImageOptions{}
	healthBarOp.GeoM.Translate(-float64(rectWidth)/2, -float64(rectHeight)/2) //center our origin
	healthBarOp.GeoM.Translate(hPosition, vPosition)                          //position
	screen.DrawImage(healthBar, healthBarOp)

	// The figure, in the same "60 / 60" shape and the same red the fighter's block uses, so
	// both duelists state their life the same way even though only one of them has a bar.
	// Onto the screen after the bar rather than into the mask before it: the mask is
	// composited with BlendSourceIn, which would eat anything drawn into it first.
	lifeOp := &text.DrawOptions{}
	lifeOp.GeoM.Translate(hPosition, vPosition)
	lifeOp.PrimaryAlign = text.AlignCenter
	lifeOp.SecondaryAlign = text.AlignCenter
	lifeOp.ColorScale.ScaleWithColor(lifeColor)
	text.Draw(screen, fmt.Sprintf("%d / %d", currentLife, maxLife),
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 16}, lifeOp)
}

func CreateRoundedRecMask(mask *ebiten.Image, x, y, width, height, radius float32, maskColor color.Color) {
	//Round Corners
	vector.DrawFilledCircle(mask, x+radius, y+radius, radius, maskColor, false)
	vector.DrawFilledCircle(mask, x+width-radius, y+radius, radius, maskColor, false)
	vector.DrawFilledCircle(mask, x+radius, y+height-radius, radius, maskColor, false)
	vector.DrawFilledCircle(mask, x+width-radius, y+height-radius, radius, maskColor, false)

	//Rectangle Edges
	vector.DrawFilledRect(mask, x+radius, y, width-2*radius, radius, maskColor, false)               //top edge
	vector.DrawFilledRect(mask, x+radius, y+height-radius, width-2*radius, radius, maskColor, false) //bottom edge
	vector.DrawFilledRect(mask, x, y+radius, radius, height-2*radius, maskColor, false)              //left edge
	vector.DrawFilledRect(mask, x+width-radius, y+radius, radius, height-2*radius, maskColor, false) //right edge
}
