package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Button struct {
	game                *Game
	x, y, width, height int
	label               string
	onClick             func(g *Game)
	color               colorm.ColorM
}

func NewButton(game *Game, x, y, width, height int, label string, onClick func(g *Game)) Button {
	return Button{
		game:    game,
		x:       x,
		y:       y,
		width:   width,
		height:  height,
		label:   label,
		onClick: onClick,
		color:   colorm.ColorM{},
	}
}

func (b *Button) Update() {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		cursorX, cursorY := ebiten.CursorPosition()
		if cursorX >= b.x && cursorX <= b.x+b.width &&
			cursorY >= b.y && cursorY <= b.y+b.height {
			if b.onClick != nil {
				b.onClick(b.game)
			}
		}
	}
}

func (b *Button) Draw(game *Game, screen *ebiten.Image) {
	op := &colorm.DrawImageOptions{}
	op.GeoM.Translate(float64(b.x), float64(b.y))

	hue := float64(game.buttonHue128) * 2 * math.Pi / 128
	saturation := float64(game.buttonSaturation128) / 128
	value := float64(game.buttonValue128) / 128
	var c colorm.ColorM
	c.ChangeHSV(hue, saturation, value)

	colorm.DrawImage(screen, b.game.Assets["frozenring_png"], c, op)
	ebitenutil.DebugPrintAt(screen, b.label, b.x+10, b.y+10)

}
