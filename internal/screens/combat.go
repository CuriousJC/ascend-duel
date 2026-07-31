package screens

import (
	"fmt"

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
// the screen for three seconds, and the damage and guard events that belong to it pass
// in a quick beat. Splitting three seconds across every event instead would make a
// Guard — which emits nothing but its own event — take as long as a Heavy that lands.
const (
	ticksPerSecond   = 60
	actionDwellTicks = 3 * ticksPerSecond
	beatTicks        = ticksPerSecond / 4
)

// dwellFor is how long one event holds the screen.
func dwellFor(e combat.Event) int {
	if e.Kind == combat.KindAction {
		return actionDwellTicks
	}
	return beatTicks
}

// CombatScene runs one duel: the player plans a set of actions against an action
// point budget, presses DUEL!, watches the round play out, and plans again.
//
// The fields below are the screen's own working state. None of it belongs to any
// other screen, and the action box will add more of it.
type CombatScene struct {
	fighter *entities.Combatant
	enemy   *entities.Combatant

	// The queued sets for the coming round. fighterActions is what the action box
	// will eventually write; enemyActions is re-planned by the opponent each round.
	fighterActions []combat.ActionKind
	enemyActions   []combat.ActionKind

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

	duelButton *models.Button
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

	// The scene builds its own widget and wires it to its own method, so no other
	// package needs to know this screen has a button or what pressing it means.
	if s.duelButton == nil {
		s.duelButton = models.NewButton(138, 50, "DUEL!", s.startRound)
		s.duelButton.BaseColor = color.RGBA{R: 220, G: 20, B: 60, A: 255} // crimson
	}
	s.duelButton.ScreenX = gs.PctX(20)
	s.duelButton.ScreenY = gs.PctY(85) // centred in the 80–90% band

	// Placeholder queue until the action box is draggable. It must fit the AP budget.
	s.fighterActions = defaultFighterPlan(s.fighter.Duelist)

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

func (s *CombatScene) Update(gs *state.GlobalState) error {
	systems.UpdateButton(gs, s.duelButton)
	s.advancePlayback()
	return nil
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
	if s.cursor >= len(s.log) {
		s.fighter.Duelist = s.fighterAfter
		s.enemy.Duelist = s.enemyAfter
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

	s.drawActions(gs, screen)
	s.drawActionBox(gs, screen)
	s.drawResolution(gs, screen)
	s.drawEnemyActions(gs, screen)
	s.drawCombatant(screen, s.fighter, float64(gs.PctX(10)), float64(gs.PctY(50)))
	s.drawCombatant(screen, s.enemy, float64(gs.PctX(75)), float64(gs.PctY(50)))
	systems.DrawButton(gs, screen, s.duelButton)
}

// The three panes share one vertical band and differ only in their horizontal slot.
const (
	paneTopPct    = 20
	paneBottomPct = 70

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

var (
	availableActionsPane = panePlacement{leftPct: 20, rightPct: 30, title: "Player", color: color.RGBA{R: 60, G: 200, B: 90, A: 255}}
	chosenActionsPane    = panePlacement{leftPct: 35, rightPct: 45, title: "Chosen", color: color.RGBA{R: 70, G: 130, B: 230, A: 255}}
	resolutionPane       = panePlacement{leftPct: 55, rightPct: 65, title: "Resolution", color: color.RGBA{R: 235, G: 105, B: 170, A: 255}}
	enemyActionsPane     = panePlacement{leftPct: 85, rightPct: 95, title: "Enemy", color: color.RGBA{R: 225, G: 200, B: 60, A: 255}}
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

func (s *CombatScene) drawActionBox(gs *state.GlobalState, screen *ebiten.Image) {
	//This will be the box that contains all the selected actions
	s.drawPane(gs, screen, chosenActionsPane, actionRows(s.fighterActions))
}

func (s *CombatScene) drawActions(gs *state.GlobalState, screen *ebiten.Image) {
	//This will be the box of available actions that can be dragged into the Action Box
	s.drawPane(gs, screen, availableActionsPane, paletteRows())
}

// drawEnemyActions shows what the opponent has queued. Read-only forever — nothing is
// ever dragged into this one.
func (s *CombatScene) drawEnemyActions(gs *state.GlobalState, screen *ebiten.Image) {
	s.drawPane(gs, screen, enemyActionsPane, actionRows(s.enemyActions))
}

// drawResolution shows the two queues merged into play order.
func (s *CombatScene) drawResolution(gs *state.GlobalState, screen *ebiten.Image) {
	s.drawPane(gs, screen, resolutionPane, s.resolutionRows(s.fighterActions, s.enemyActions))
}

// drawPane draws one column: a dim fill in its identifying colour, a full-strength
// border, a title and a row per action.
func (s *CombatScene) drawPane(gs *state.GlobalState, screen *ebiten.Image, p panePlacement, rows []paneRow) {
	x := float32(gs.PctX(p.leftPct))
	y := float32(gs.PctY(paneTopPct))
	w := float32(gs.PctX(p.rightPct)) - x
	h := float32(gs.PctY(paneBottomPct)) - y

	// Dim fill, full-strength border: the pane reads as green or blue or yellow at a
	// glance without drowning the text drawn on top of it.
	vector.DrawFilledRect(screen, x, y, w, h, systems.ColorAtStrength(p.color, 25), false)
	vector.StrokeRect(screen, x, y, w, h, 2, p.color, false)

	face := &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 16}

	titleOp := &text.DrawOptions{}
	titleOp.GeoM.Translate(float64(x+w/2), float64(y+paneTitleInset))
	titleOp.PrimaryAlign = text.AlignCenter
	text.Draw(screen, p.title, face, titleOp)

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

// paletteRows lists every action a duelist can queue, with its cost, in the order the
// combat package says the UI should offer them.
func paletteRows() []paneRow {
	rows := make([]paneRow, 0, len(combat.AllActions))
	for _, a := range combat.AllActions {
		rows = append(rows, paneRow{label: fmt.Sprintf("%s %d", a, a.Cost())})
	}
	return rows
}

// actionRows renders a queued set one action per row.
func actionRows(actions []combat.ActionKind) []paneRow {
	if len(actions) == 0 {
		return []paneRow{{label: "(empty)"}}
	}

	rows := make([]paneRow, 0, len(actions))
	for _, a := range actions {
		rows = append(rows, paneRow{label: a.String()})
	}
	return rows
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
func (s *CombatScene) resolutionRows(fighter, enemy []combat.ActionKind) []paneRow {
	if len(fighter) == 0 && len(enemy) == 0 {
		return []paneRow{{label: "(empty)"}}
	}

	playingSide, playingOrdinal, playing := s.currentAction()
	row := func(a combat.ActionKind, side combat.Side, ordinal int, swatch color.RGBA) paneRow {
		return paneRow{
			label:       a.String(),
			swatch:      swatch,
			highlighted: playing && side == playingSide && ordinal == playingOrdinal,
		}
	}

	rows := make([]paneRow, 0, len(fighter)+len(enemy))
	for i := 0; i < len(fighter) || i < len(enemy); i++ {
		if i < len(fighter) {
			rows = append(rows, row(fighter[i], combat.SideA, i, chosenActionsPane.color))
		}
		if i < len(enemy) {
			rows = append(rows, row(enemy[i], combat.SideB, i, enemyActionsPane.color))
		}
	}
	return rows
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

// defaultFighterPlan spends the budget on a Guard plus whatever attacks still fit.
// It stands in for the player's choices until the action box exists.
func defaultFighterPlan(d combat.Duelist) []combat.ActionKind {
	plan := []combat.ActionKind{combat.Guard}
	for combat.CostOf(append(plan, combat.Strike)) <= d.ActionPoints() {
		plan = append(plan, combat.Strike)
	}
	for combat.CostOf(append(plan, combat.Quick)) <= d.ActionPoints() {
		plan = append(plan, combat.Quick)
	}
	return plan
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
