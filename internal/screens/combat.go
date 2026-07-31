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

// duelTicksPerEvent is how long each event in the log is held on screen during
// playback, at 60 TPS. Destined to become the game-speed setting.
const duelTicksPerEvent = 8

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
		s.duelButton = models.NewButton(275, 100, "DUEL!", s.startRound)
	}
	s.duelButton.ScreenX = gs.HalfwayX
	s.duelButton.ScreenY = gs.ThirdQuarterY

	// Placeholder queue until the action box is draggable. It must fit the AP budget.
	s.fighterActions = defaultFighterPlan(s.fighter.Duelist)
	s.enemyActions = nil

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
	if s.ticks < duelTicksPerEvent {
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
	case combat.KindVolleyStart:
		return fmt.Sprintf("%s acts", who)
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

	s.drawActionBox(gs, screen)
	s.drawActions(gs, screen)
	s.drawCombatant(screen, s.fighter, float64(gs.FirstQuarterX), float64(gs.HalfwayY))
	s.drawCombatant(screen, s.enemy, float64(gs.ThirdQuarterX), float64(gs.HalfwayY))
	systems.DrawButton(gs, screen, s.duelButton)
}

func (s *CombatScene) drawActionBox(gs *state.GlobalState, screen *ebiten.Image) {
	//This will be the box that contains all the selected actions
}

func (s *CombatScene) drawActions(gs *state.GlobalState, screen *ebiten.Image) {
	//This will be the box of available actions that can be dragged into the Action Box
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
