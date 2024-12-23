package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Button struct {
	x, y, width, height int
	label               string
	onClick             func()
}

func NewButton(x, y, width, height int, label string, onClick func()) Button {
	return Button{
		x:       x,
		y:       y,
		width:   width,
		height:  height,
		label:   label,
		onClick: onClick,
	}
}

func (b *Button) Update() {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		cursorX, cursorY := ebiten.CursorPosition()
		if cursorX >= b.x && cursorX <= b.x+b.width &&
			cursorY >= b.y && cursorY <= b.y+b.height {
			if b.onClick != nil {
				b.onClick()
			}
		}
	}
}

func (b *Button) Draw(screen *ebiten.Image) {
	ebitenutil.DebugPrintAt(screen, b.label, b.x+10, b.y+10)
}
