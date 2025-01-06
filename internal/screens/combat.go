package screens

import (
	"image"

	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"image/color"
)

func UpdateCombatScreen(gs *state.GlobalState) error {

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

	var c colorm.ColorM

	// Character sprite
	characterRect := image.Rect(98, 142, 191, 222)
	characterSprite := gs.Assets["tyrian_ship_sprites_png"].SubImage(characterRect).(*ebiten.Image)
	characterWidth := characterRect.Dx()
	characterHeight := characterRect.Dy()

	characterOp := &colorm.DrawImageOptions{}
	characterOp.GeoM.Translate(
		float64(gs.HalfwayX)-(float64(characterWidth)/2),  // Center horizontally
		float64(gs.HalfwayY)-(float64(characterHeight)/2), // Center vertically
	)
	colorm.DrawImage(screen, characterSprite, c, characterOp)

	// Opponent sprite
	monsterRect := image.Rect(5, 58, 119, 152)
	monsterSprite := gs.Assets["tyrian_monster_sprites_png"].SubImage(monsterRect).(*ebiten.Image)
	monsterWidth := monsterRect.Dx()
	monsterHeight := monsterRect.Dy()

	monsterOp := &colorm.DrawImageOptions{}
	monsterOp.GeoM.Translate(
		float64(gs.ScreenWidth)*0.85-(float64(monsterWidth)/2), // 15% away from the right edge, centered horizontally
		float64(gs.HalfwayY)-(float64(monsterHeight)/2),        // Center vertically
	)
	colorm.DrawImage(screen, monsterSprite, c, monsterOp)

}
