package screens

import (
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/hajimehoshi/ebiten/v2"
)

// AscendScene will be the tower loop: 8 floors of 3 fights, with a loot choice after
// every fight and a floor choice after each floor's last one. Not built yet.
type AscendScene struct{}

func (s *AscendScene) Init(gs *state.GlobalState)                       {}
func (s *AscendScene) Update(gs *state.GlobalState) error               { return nil }
func (s *AscendScene) Draw(gs *state.GlobalState, screen *ebiten.Image) {}

// CreditsScene is not built yet.
type CreditsScene struct{}

func (s *CreditsScene) Init(gs *state.GlobalState)                       {}
func (s *CreditsScene) Update(gs *state.GlobalState) error               { return nil }
func (s *CreditsScene) Draw(gs *state.GlobalState, screen *ebiten.Image) {}
