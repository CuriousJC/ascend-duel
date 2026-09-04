package screens

import (
	"testing"

	"github.com/curiousjc/ascend-duel/internal/profile"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// **A fresh profile is windowed**, which is the zero value doing the right thing for once — see
// profile.Settings.Fullscreen. The other two settings are the ones whose zero value is a trap.
func TestAFreshProfileIsWindowed(t *testing.T) {
	if profile.Defaults().Fullscreen {
		t.Error("a player who has never opened the settings screen is put into fullscreen")
	}
}

// The three rows and the rule under them are laid out from one gap, so a row added or removed
// cannot leave the rule sitting on top of the last control.
func TestTheAbandonRuleClearsTheLastSetting(t *testing.T) {
	gs := &state.GlobalState{ScreenWidth: state.ScreenWidth, ScreenHeight: state.ScreenHeight}
	s := &SettingsScene{}
	s.Init(gs)

	if bottom := s.full.ScreenY + s.full.Height/2; s.abandonRuleY(gs) <= bottom {
		t.Errorf("the rule is at y=%d, on top of the fullscreen toggle ending at y=%d",
			s.abandonRuleY(gs), bottom)
	}
	if s.full.ScreenY <= s.speed.ScreenY {
		t.Errorf("the fullscreen toggle at y=%d is not below the speed bar at y=%d",
			s.full.ScreenY, s.speed.ScreenY)
	}
}
