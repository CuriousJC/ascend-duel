package screens

import (
	"fmt"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/entities"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"image/color"
)

// duelTicksPerEvent is how long each event in the log is held on screen during
// playback, at 60 TPS.
const duelTicksPerEvent = 8

// InitCombatScreen is run at the initialization of the combat screen
func InitCombatScreen(gs *state.GlobalState) {

	if gs.Fighter == nil {
		gs.Fighter = combatantFromRecord(gs, "Fighter1")
	}

	if gs.Enemy == nil {
		gs.Enemy = combatantFromRecord(gs, "Monster1")
	}

	// Placeholder queue until the action box is draggable — this is what the
	// drag-and-drop UI will eventually write, and it must fit the AP budget.
	gs.FighterActions = defaultFighterPlan(gs.Fighter.Duelist)
	gs.EnemyActions = nil

	gs.DuelButton.ScreenX = gs.HalfwayX
	gs.DuelButton.ScreenY = gs.ThirdQuarterY

	// A fresh duel: full health, no log, round zero.
	gs.Fighter.CurrentLife = gs.Fighter.MaxLife
	gs.Enemy.CurrentLife = gs.Enemy.MaxLife
	gs.Fighter.Guarded = false
	gs.Enemy.Guarded = false
	gs.DuelLog = nil
	gs.DuelCursor = 0
	gs.DuelTicks = 0
	gs.DuelRound = 0
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

// combatantFromRecord resolves a combatant record and its sprite sheet out of global
// state, then hands both to the entity constructor.
func combatantFromRecord(gs *state.GlobalState, record string) *entities.Combatant {
	d := gs.Combatants[record]
	return entities.NewCombatantFrom(d, gs.Assets[d.SpriteSheet])
}

func UpdateCombatScreen(gs *state.GlobalState) error {

	if gs.NewScreen {
		InitCombatScreen(gs)
		gs.NewScreen = false
	}

	systems.UpdateButton(gs, gs.DuelButton)
	advanceDuelPlayback(gs)

	return nil
}

// advanceDuelPlayback walks the round's event log one entry at a time, applying each
// to the on-screen combatants. This is the whole of the screen's combat logic: the
// round was already decided by combat.ResolveRound, so playback can never disagree
// with it.
func advanceDuelPlayback(gs *state.GlobalState) {
	if gs.DuelCursor >= len(gs.DuelLog) {
		return
	}

	gs.DuelTicks++
	if gs.DuelTicks < duelTicksPerEvent {
		return
	}
	gs.DuelTicks = 0

	applyDuelEvent(gs, gs.DuelLog[gs.DuelCursor])
	gs.DuelCursor++

	// Playback has caught up with the resolver. Adopt the authoritative end-of-round
	// state — the guard flags live only there, since no event carries them — and hand
	// control back to the player to plan the next round.
	if gs.DuelCursor >= len(gs.DuelLog) {
		gs.Fighter.Duelist = gs.FighterAfter
		gs.Enemy.Duelist = gs.EnemyAfter
	}
}

// applyDuelEvent moves the visible state to match one event. Only damage moves the
// health bars; the rest are for the caption and, later, animation cues.
func applyDuelEvent(gs *state.GlobalState, e combat.Event) {
	if e.Kind != combat.KindDamage {
		return
	}

	if e.Target == combat.SideA {
		gs.Fighter.CurrentLife = e.Life
	} else {
		gs.Enemy.CurrentLife = e.Life
	}
}

// duelCaption describes where the round has got to, so the duel is legible before the
// action box exists.
func duelCaption(gs *state.GlobalState) string {
	if !gs.Enemy.Alive() {
		return fmt.Sprintf("The monster falls in round %d — you win!", gs.DuelRound)
	}
	if !gs.Fighter.Alive() {
		return fmt.Sprintf("You fall in round %d. The monster wins.", gs.DuelRound)
	}

	// Between rounds: show the plan and what it costs.
	if gs.DuelCursor >= len(gs.DuelLog) {
		return fmt.Sprintf("Round %d — your plan: %s  (%d/%d AP)   press DUEL!",
			gs.DuelRound+1,
			planLabel(gs.FighterActions),
			combat.CostOf(gs.FighterActions),
			gs.Fighter.ActionPoints())
	}

	e := gs.DuelLog[gs.DuelCursor]
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

func DrawCombatScreen(gs *state.GlobalState, screen *ebiten.Image) {

	screen.Fill(color.RGBA{
		R: 50,
		G: 50,
		B: 50,
		A: 255,
	})

	combatHeadingOp := &text.DrawOptions{}
	combatHeadingOp.GeoM.Translate(50, 50)
	text.Draw(screen, "Duel to the Top of the Pyramid", &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 20}, combatHeadingOp)

	captionOp := &text.DrawOptions{}
	captionOp.GeoM.Translate(50, 100)
	text.Draw(screen, duelCaption(gs), &text.GoTextFace{Source: gs.Fonts["kubasta"], Size: 16}, captionOp)

	DrawActionBox(gs, screen)
	DrawActions(gs, screen)
	DrawCharacter(gs, screen)
	DrawEnemy(gs, screen)
	DrawCombatButton(gs, screen)

}

func DrawActionBox(gs *state.GlobalState, screen *ebiten.Image) {
	//This will be the box that contains all the selected actions

}

func DrawActions(gs *state.GlobalState, screen *ebiten.Image) {
	//This will be the box of available actions that can be dragged into the Action Box

}

func DrawEnemy(gs *state.GlobalState, screen *ebiten.Image) {
	hPosition := float64(gs.ThirdQuarterX)
	vPosition := float64(gs.HalfwayY)
	var c colorm.ColorM

	monsterOp := &colorm.DrawImageOptions{}
	monsterOp.GeoM.Translate(-float64(gs.Enemy.Sprite.Bounds().Dx())/2, -float64(gs.Enemy.Sprite.Bounds().Dy())/2) //center our origin
	monsterOp.GeoM.Translate(hPosition, vPosition)                                                                 //position
	colorm.DrawImage(screen, gs.Enemy.Sprite, c, monsterOp)

	DrawHealthBar(gs, screen, hPosition, vPosition, gs.Enemy.CurrentLife, gs.Enemy.MaxLife)
}

func DrawCharacter(gs *state.GlobalState, screen *ebiten.Image) {
	hPosition := float64(gs.FirstQuarterX)
	vPosition := float64(gs.HalfwayY)
	var c colorm.ColorM

	characterOp := &colorm.DrawImageOptions{}
	characterOp.GeoM.Translate(-float64(gs.Fighter.Sprite.Bounds().Dx())/2, -float64(gs.Fighter.Sprite.Bounds().Dy())/2) //center our origin
	characterOp.GeoM.Translate(hPosition, vPosition)                                                                     //position
	colorm.DrawImage(screen, gs.Fighter.Sprite, c, characterOp)

	DrawHealthBar(gs, screen, hPosition, vPosition, gs.Fighter.CurrentLife, gs.Fighter.MaxLife)

}

func DrawCombatButton(gs *state.GlobalState, screen *ebiten.Image) {
	//The DUEL! button. Position is set in InitCombatScreen; Draw only draws.
	systems.DrawButton(gs, screen, gs.DuelButton)
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
