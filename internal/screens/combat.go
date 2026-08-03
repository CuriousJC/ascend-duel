package screens

import (
	"fmt"
	"math/rand"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/entities"
	"github.com/curiousjc/ascend-duel/internal/models"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"image/color"
)

// Playback pacing at 60 TPS. Destined to become the game-speed setting.
//
// The budget is spent per action rather than spread evenly over events: an action holds
// the screen for a beat and a half, and the damage and guard events that belong to it pass
// quickly. Splitting one dwell across every event instead would make a Guard — which emits
// nothing but its own event — take as long as a Heavy that lands.
//
// Halved from three seconds on 2026-08-02: at three seconds a round of six actions took
// twenty seconds to watch, and the pause between a move and its consequence read as the
// game hesitating rather than as pacing.
const (
	ticksPerSecond   = 60
	actionDwellTicks = 3 * ticksPerSecond / 2
	beatTicks        = ticksPerSecond / 4
)

// dwellFor is how long one event holds the screen.
func dwellFor(e combat.Event) int {
	if e.Kind == combat.KindAction {
		return actionDwellTicks
	}
	return beatTicks
}

// The starting deck: 20 cards, and a hand of five drawn from it each round. Quick is not
// in it — the deck is what the player owns, and which of the actions the rules define
// actually appear is a deck-building question rather than a rules one.
//
// The 10/6/4 split is a first guess at "mostly attacks, some defence, a few big swings"
// and has not been played against anything. It is the obvious thing to tune once the
// fights are worth measuring.
const handSize = 5

var startingDeck = []deckEntry{
	{combat.Strike, 10},
	{combat.Guard, 6},
	{combat.Heavy, 4},
}

// deckEntry is one line of a deck list: a card and how many copies of it.
type deckEntry struct {
	action combat.ActionKind
	count  int
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
	deck    []combat.ActionKind
	hand    []paletteCard
	discard []combat.ActionKind

	// The shuffle source. Explicit and carried on state rather than the math/rand
	// package-level functions, which draw from a global shared with every other caller
	// and would make a run unreproducible. Seeded once in Init.
	rng *rand.Rand

	// The card currently being dragged, if any. See combat_actionbox.go.
	drag *dragState

	// showDeck toggles the deck overlay. While it is up the cards underneath do not
	// respond, so reading the deck cannot accidentally re-plan the round.
	showDeck bool

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
	if s.enemy == nil {
		s.enemy = combatantFromRecord(gs, "Monster1")
	}

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
	// nowhere near them. All three share the 80–90% band, so the row reads as one strip.
	s.discardButton.ScreenX = gs.PctX(20)
	s.discardButton.ScreenY = gs.PctY(85)
	s.duelButton.ScreenX = gs.PctX(33)
	s.duelButton.ScreenY = gs.PctY(85)
	s.deckButton.ScreenX = gs.PctX(88)
	s.deckButton.ScreenY = gs.PctY(85)

	s.showDeck = false

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
	s.enemyActions = combat.PlanGreedy(s.enemy.Duelist)

	s.fighter.CurrentLife = s.fighter.MaxLife
	s.enemy.CurrentLife = s.enemy.MaxLife
	s.fighter.Guarded = false
	s.enemy.Guarded = false

	s.log = nil
	s.cursor = 0
	s.ticks = 0
	s.round = 0
}

// discardSelected throws the selected cards away: they leave the hand for the discard
// pile, and the hand is dealt back up to size from the draw pile.
//
// Selection does double duty here — it is both "queue this for the round" and "this is
// the one I mean" — so throwing a card out costs the action points it was holding until
// the moment it leaves. That is a consequence of overloading selection rather than a
// designed cost; see TODO.md.
func (s *CombatScene) discardSelected() {
	if !s.planning() {
		return
	}

	kept := s.hand[:0]
	for _, c := range s.hand {
		if c.selected {
			s.discard = append(s.discard, c.action)
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
}

// resetDeck builds the starting deck, shuffles it, empties the discard and deals an
// opening hand.
func (s *CombatScene) resetDeck() {
	s.deck = s.deck[:0]
	for _, e := range startingDeck {
		for i := 0; i < e.count; i++ {
			s.deck = append(s.deck, e.action)
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
		s.hand = append(s.hand, paletteCard{action: s.deck[last]})
		s.deck = s.deck[:last]
	}
}

// discardHand sends the whole hand to the discard pile, played or not, and deals a fresh
// one. Nothing carries between rounds: a hand kept back would let a plan be prepared once
// and repeated, and every round being a real decision is the point.
func (s *CombatScene) discardHand() {
	for _, c := range s.hand {
		s.discard = append(s.discard, c.action)
	}
	s.hand = s.hand[:0]
	s.drawHand()
	s.syncQueue()
}

func (s *CombatScene) Update(gs *state.GlobalState) error {
	// The overlay swallows card interaction, so reading the deck cannot re-plan the round
	// through the panel covering it. The buttons stay live — one of them is how it closes.
	if !s.showDeck {
		s.updateActionBox(gs)
	}

	// Both act on the selection, so both need one — pressing either is deciding what the
	// selection was for. An empty queue is mechanically legal for DUEL! — ResolveRound
	// handles it — but it means standing still while being hit, which is not something to
	// offer by accident.
	//
	// Both go dead while the deck is open. It is a dialog: Deck is the only live control
	// on the screen until it is pressed again.
	live := s.planning() && len(s.fighterActions) > 0 && !s.showDeck
	setEnabled(s.duelButton, live)
	setEnabled(s.discardButton, live)

	systems.UpdateButton(gs, s.duelButton)
	systems.UpdateButton(gs, s.discardButton)
	systems.UpdateButton(gs, s.deckButton)
	s.advancePlayback()
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

	s.round++
	s.enemyActions = combat.PlanGreedy(s.enemy.Duelist)

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

	fmt.Printf("Round %d: %d events (fighter %d AP, enemy %d AP)\n",
		s.round, len(log),
		s.fighter.ActionPoints(), s.enemy.ActionPoints())
}

// advancePlayback walks the round's event log one entry at a time, applying each to
// the on-screen combatants. This is the whole of the screen's combat logic: the round
// was already decided by combat.ResolveRound, so playback can never disagree with it.
func (s *CombatScene) advancePlayback() {
	if s.cursor >= len(s.log) {
		return
	}

	s.ticks++
	if s.ticks < dwellFor(s.log[s.cursor]) {
		return
	}
	s.ticks = 0

	s.applyEvent(s.log[s.cursor])
	s.cursor++

	// Playback has caught up with the resolver. Adopt the authoritative end-of-round
	// state and hand control back to the player to plan the next round.
	//
	// The hand is spent here rather than at resolve time, and the ordering matters:
	// discardHand rebuilds fighterActions from the new hand, and the Resolution pane
	// draws fighterActions to narrate the round. Discarding while playback was still
	// running would empty the pane mid-round.
	if s.cursor >= len(s.log) {
		s.fighter.Duelist = s.fighterAfter
		s.enemy.Duelist = s.enemyAfter
		s.discardHand()
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
	if !s.enemy.Alive() {
		return fmt.Sprintf("The monster falls in round %d — you win!", s.round)
	}
	if !s.fighter.Alive() {
		return fmt.Sprintf("You fall in round %d. The monster wins.", s.round)
	}

	// Between rounds: show the plan and what it costs.
	if s.cursor >= len(s.log) {
		return fmt.Sprintf("Round %d — your plan: %s  (%d/%d AP)   press DUEL!",
			s.round+1,
			planLabel(s.fighterActions),
			combat.CostOf(s.fighterActions),
			s.fighter.ActionPoints())
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

	headingOp := &text.DrawOptions{}
	headingOp.GeoM.Translate(50, 50)
	text.Draw(screen, "Duel to the Top of the Pyramid",
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 20}, headingOp)

	captionOp := &text.DrawOptions{}
	captionOp.GeoM.Translate(50, 100)
	text.Draw(screen, s.caption(),
		&text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 16}, captionOp)

	s.drawPalette(gs, screen)
	s.drawResolution(gs, screen)
	s.drawCombatant(screen, s.fighter, float64(gs.PctX(9)), float64(gs.PctY(50)))
	s.drawCombatant(screen, s.enemy, float64(gs.PctX(88)), float64(gs.PctY(50)))
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
const (
	deckPanelLeftPct   = 4
	deckPanelRightPct  = 96
	deckPanelTopPct    = 4
	deckPanelBottomPct = 78

	deckRowHeight = 52
)

// drawDeckOverlay covers the screen with what is left to draw.
//
// Counts by kind, never the order. The draw pile is shuffled, and listing it in order
// would hand the player their next five cards and make the shuffle pointless. The discard
// is shown beside it because those cards are coming back — a reshuffle folds the pile in,
// so "what is left" honestly means both piles.
//
// Every kind is listed even at zero, which is how the absence of Quick from the deck is
// visible rather than merely implied.
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
	face := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 22}
	small := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 14}

	title := &text.DrawOptions{}
	title.GeoM.Translate(float64(left+width/2), float64(top+44))
	title.PrimaryAlign = text.AlignCenter
	text.Draw(screen, "What is left", heading, title)

	// Two columns: what can be drawn now, and what returns when the pile is reshuffled.
	drawCol := left + width*0.12
	discardCol := left + width*0.56

	header := func(x float32, label string, total int) {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x), float64(top+130))
		text.Draw(screen, fmt.Sprintf("%s — %d cards", label, total), small, op)
	}
	header(drawCol, "DRAW PILE", len(s.deck))
	header(discardCol, "DISCARD", len(s.discard))

	deckCounts, discardCounts := pileCounts(s.deck), pileCounts(s.discard)

	// AllActions rather than ranging the maps: Go randomises map order, and a list that
	// reshuffled itself every frame would be unreadable.
	for i, action := range combat.AllActions {
		y := top + 190 + float32(i*deckRowHeight)

		row := func(x float32, count int) {
			name := &text.DrawOptions{}
			name.GeoM.Translate(float64(x), float64(y))
			text.Draw(screen, action.String(), face, name)

			num := &text.DrawOptions{}
			num.GeoM.Translate(float64(x+width*0.30), float64(y))
			num.PrimaryAlign = text.AlignEnd
			text.Draw(screen, fmt.Sprintf("%d", count), face, num)
		}
		row(drawCol, deckCounts[action])
		row(discardCol, discardCounts[action])
	}

	hint := &text.DrawOptions{}
	hint.GeoM.Translate(float64(left+width/2), float64(top+height-40))
	hint.PrimaryAlign = text.AlignCenter
	text.Draw(screen, "Deck again to close", small, hint)
}

// pileCounts totals a pile by action kind.
func pileCounts(pile []combat.ActionKind) map[combat.ActionKind]int {
	counts := make(map[combat.ActionKind]int, len(combat.AllActions))
	for _, a := range pile {
		counts[a]++
	}
	return counts
}

// Both panes share one vertical band and differ only in their horizontal slot. The band is
// taller than it was because the palette now stacks two zones of cards rather than one.
const (
	paneTopPct    = 18
	paneBottomPct = 74

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

// Two panes now, flanked by the duelists: fighter / palette / resolution / enemy. The
// Chosen pane folded into the palette as a second zone under the available cards, and the
// Enemy pane went entirely — the Resolution pane already shows the opponent's actions, in
// a better order than a column of its own could. Resolution inherits the freed width
// because it is the centrepiece: it is the only pane that has to grow when exchanges
// acquire structure.
var (
	palettePane    = panePlacement{leftPct: 15, rightPct: 39, title: "Actions", color: color.RGBA{R: 60, G: 200, B: 90, A: 255}}
	resolutionPane = panePlacement{leftPct: 45, rightPct: 78, title: "Resolution", color: color.RGBA{R: 235, G: 105, B: 170, A: 255}}
)

// enemySwatch marks the opponent's rows in the Resolution pane. There is no enemy pane to
// take a colour from any more, but the round still has two sides and they have to be told
// apart at a glance. The player's rows carry the palette's green, so the whole screen
// reads as two colours: green is you, yellow is them.
var enemySwatch = color.RGBA{R: 225, G: 200, B: 60, A: 255}

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

// drawPaneFrame draws a column's fill, border and title, and reports its rectangle.
// Split out because the card panes fill themselves rather than drawing text rows.
func (s *CombatScene) drawPaneFrame(gs *state.GlobalState, screen *ebiten.Image, p panePlacement) (x, y, w, h float32) {
	x = float32(gs.PctX(p.leftPct))
	y = float32(gs.PctY(paneTopPct))
	w = float32(gs.PctX(p.rightPct)) - x
	h = float32(gs.PctY(paneBottomPct)) - y

	// Dim fill, full-strength border: the pane reads as green or blue or yellow at a
	// glance without drowning what is drawn on top of it.
	vector.DrawFilledRect(screen, x, y, w, h, systems.ColorAtStrength(p.color, 25), false)
	vector.StrokeRect(screen, x, y, w, h, 2, p.color, false)

	titleOp := &text.DrawOptions{}
	titleOp.GeoM.Translate(float64(x+w/2), float64(y+paneTitleInset))
	titleOp.PrimaryAlign = text.AlignCenter
	text.Draw(screen, p.title, &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 16}, titleOp)

	return x, y, w, h
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
func (s *CombatScene) resolutionRows(fighter, enemy []combat.ActionKind, concealEnemy bool) []paneRow {
	order := combat.ResolutionOrder(fighter, enemy)
	if len(order) == 0 {
		return []paneRow{{label: "(empty)"}}
	}

	playingSide, playingOrdinal, playing := s.currentAction()

	rows := make([]paneRow, 0, len(order))
	for _, slot := range order {
		label, swatch := slot.Action.String(), palettePane.color
		if slot.Side == combat.SideB {
			swatch = enemySwatch
			if concealEnemy {
				label = concealedLabel(slot.Action)
			}
		}

		rows = append(rows, paneRow{
			label:       label,
			swatch:      swatch,
			highlighted: playing && slot.Side == playingSide && slot.Index == playingOrdinal,
		})
	}
	return rows
}

// concealedLabel is what a hidden action shows instead of its name. Initiative is
// deliberately not hidden: it is what decides where the action sits in the order, so
// withholding it would make the Resolution pane unreadable rather than merely uncertain
// — the player could not tell why the rows are arranged as they are.
//
// This is the first cut at graded reveal rather than the finished scheme. What else
// leaks per action — whether it damages, whether it applies a status — is still open;
// see TODO.md.
func concealedLabel(a combat.ActionKind) string {
	return fmt.Sprintf("??? (i%d)", a.Initiative())
}

// currentAction reports which queued action the playback cursor is inside: its side and
// its index within that side's queue. It is derived by counting the action events walked
// so far rather than tracked in a field, so it cannot drift out of step with the cursor.
//
// ok is false before the first action of a round and once playback has finished.
func (s *CombatScene) currentAction() (combat.Side, int, bool) {
	if s.cursor >= len(s.log) {
		return combat.SideA, 0, false
	}

	fighterSeen, enemySeen := -1, -1
	side := combat.SideA
	found := false

	for _, e := range s.log[:s.cursor+1] {
		if e.Kind != combat.KindAction {
			continue
		}
		if e.Side == combat.SideA {
			fighterSeen++
		} else {
			enemySeen++
		}
		side = e.Side
		found = true
	}

	if !found {
		return combat.SideA, 0, false
	}
	if side == combat.SideA {
		return combat.SideA, fighterSeen, true
	}
	return combat.SideB, enemySeen, true
}

// drawCombatant draws one duelist and its health bar. The fighter and the monster
// differed only in which coordinate they were placed at, so they share this.
func (s *CombatScene) drawCombatant(screen *ebiten.Image, c *entities.Combatant, hPosition, vPosition float64) {
	var cm colorm.ColorM

	op := &colorm.DrawImageOptions{}
	op.GeoM.Translate(-float64(c.Sprite.Bounds().Dx())/2, -float64(c.Sprite.Bounds().Dy())/2) //center our origin
	op.GeoM.Translate(hPosition, vPosition)                                                   //position
	colorm.DrawImage(screen, c.Sprite, cm, op)

	DrawHealthBar(screen, hPosition, vPosition, c.CurrentLife, c.MaxLife)
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

func DrawHealthBar(screen *ebiten.Image,
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

	//TODO: Add the current/max health on to the health bar

	healthBarOp := &ebiten.DrawImageOptions{}
	healthBarOp.GeoM.Translate(-float64(rectWidth)/2, -float64(rectHeight)/2) //center our origin
	healthBarOp.GeoM.Translate(hPosition, vPosition)                          //position
	screen.DrawImage(healthBar, healthBarOp)
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
