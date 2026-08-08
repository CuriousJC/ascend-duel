//go:build !demoplay

package screens

import (
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
)

// The demo driver compiled out. Both functions are empty and the scripted plan, the frame
// captures and the PNG encoder are not in the binary at all — the same shape as
// internal/trace and internal/idle, and for the same reason: a game that plays itself must
// never be in something that ships.

func (s *CombatScene) demoUpdate(*state.GlobalState) {}

func (s *CombatScene) demoDraw(*state.GlobalState, *ebiten.Image) {}
