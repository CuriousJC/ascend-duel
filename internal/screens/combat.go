package screens

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/entities"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"image/color"
)

// InitCombatScreen is run at the initialization of the combat screen
func InitCombatScreen(gs *state.GlobalState) {

	if gs.Fighter == nil {
		gs.Fighter = entities.NewCombatant()
		gs.Fighter.Str = gs.Combatants["Fighter1"].Strength
		gs.Fighter.Spd = gs.Combatants["Fighter1"].Speed
		gs.Fighter.Con = gs.Combatants["Fighter1"].Constitution
		gs.Fighter.SpriteRect = image.Rect(gs.Combatants["Fighter1"].SpriteRect[0], gs.Combatants["Fighter1"].SpriteRect[1], gs.Combatants["Fighter1"].SpriteRect[2], gs.Combatants["Fighter1"].SpriteRect[3])
		gs.Fighter.Sprite = gs.Assets[gs.Combatants["Fighter1"].SpriteSheet].SubImage(gs.Fighter.SpriteRect).(*ebiten.Image)
	}

	if gs.Enemy == nil {
		gs.Enemy = entities.NewCombatant()
		gs.Enemy.Str = gs.Combatants["Monster1"].Strength
		gs.Enemy.Spd = gs.Combatants["Monster1"].Speed
		gs.Enemy.Con = gs.Combatants["Monster1"].Constitution
		gs.Enemy.SpriteRect = image.Rect(gs.Combatants["Monster1"].SpriteRect[0], gs.Combatants["Monster1"].SpriteRect[1], gs.Combatants["Monster1"].SpriteRect[2], gs.Combatants["Monster1"].SpriteRect[3])
		gs.Enemy.Sprite = gs.Assets[gs.Combatants["Monster1"].SpriteSheet].SubImage(gs.Fighter.SpriteRect).(*ebiten.Image)
	}
}

func UpdateCombatScreen(gs *state.GlobalState) error {

	if gs.NewScreen {
		InitCombatScreen(gs)
		gs.NewScreen = false
	}

	return nil
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

	DrawActionBox(gs, screen)
	DrawActions(gs, screen)
	DrawCharacter(gs, screen)
	DrawEnemy(gs, screen)

	//TODO: Create the DUEL! button which will perform the calculations and run through the actions

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

	DrawHealthBar(gs, screen, hPosition, vPosition, gs.Enemy.Life_current, gs.Enemy.Life_max)
}

func DrawCharacter(gs *state.GlobalState, screen *ebiten.Image) {
	hPosition := float64(gs.FirstQuarterX)
	vPosition := float64(gs.HalfwayY)
	var c colorm.ColorM

	characterOp := &colorm.DrawImageOptions{}
	characterOp.GeoM.Translate(-float64(gs.Fighter.Sprite.Bounds().Dx())/2, -float64(gs.Fighter.Sprite.Bounds().Dy())/2) //center our origin
	characterOp.GeoM.Translate(hPosition, vPosition)                                                                     //position
	colorm.DrawImage(screen, gs.Fighter.Sprite, c, characterOp)

	DrawHealthBar(gs, screen, hPosition, vPosition, gs.Fighter.Life_current, gs.Fighter.Life_max)

}

func DrawCombatButton(gs *state.GlobalState, screen *ebiten.Image) {
	//This will be the button for performing the combat calculations and actions in the action box

}

func DrawHealthBar(gs *state.GlobalState, screen *ebiten.Image,
	hPositionEntity float64, vPositionEntity float64,
	currentLife int, maxLife int) {

	rectWidth := 150
	rectHeight := 25
	hPosition := hPositionEntity
	vPosition := vPositionEntity + 100                      //move down 100 px from the position
	rectColor := color.RGBA{A: 255, R: 96, G: 37, B: 37}    // Crimson red
	maskColor := color.RGBA{A: 255, R: 255, G: 192, B: 203} // mask color
	maskFill := color.RGBA{A: 0, R: 255, G: 255, B: 255}    //fill transparent

	//Rounded corners mask Image
	mask := ebiten.NewImage(rectWidth, rectHeight)
	mask.Fill(maskFill)
	CreateRoundedRecMask(mask, 0, 0, float32(rectWidth), float32(rectHeight), 10, maskColor)

	//Red Rectangle
	redRect := ebiten.NewImage(rectWidth, rectHeight)
	vector.DrawFilledRect(redRect, 0, 0, float32(rectWidth), float32(rectHeight), rectColor, false)

	//Blend red rectangle into the mask.  The source is the red rectangle
	//  and the mask will only display that portion of the source that is blended into the non-transparent alpha
	rectMaskOp := &ebiten.DrawImageOptions{}
	rectMaskOp.Blend = ebiten.BlendSourceIn
	mask.DrawImage(redRect, rectMaskOp) //blend source redRect into destination mask
	healthBar := mask                   //mask is now filled with transparent maskFill and the maskColor was overwritten with redRect

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
	vector.DrawFilledRect(mask, x, y+radius, radius+width, height-2*radius, maskColor, false)        //left edge
	vector.DrawFilledRect(mask, x+width-radius, y+radius, radius, height-2*radius, maskColor, false) //right edge
}
