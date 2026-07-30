package actions

import (
	"fmt"

	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/state"
)

func CombatButtonAction(gs *state.GlobalState) {
	gs.ActiveScreen = state.Combat
	gs.NewScreen = true
	fmt.Println("Combat button clicked!")
	gs.Debug1 = "Combat button clicked!"
	gs.Debug2 = gs.ActiveScreen.String()
}

func SettingsButtonAction(gs *state.GlobalState) {
	fmt.Println("Settings Button Clicked!")
	// No settings screen yet, and no screen change, so NewScreen stays as it is —
	// setting it would now cost a skipped Draw and a pointless re-Init.
}

// DuelButtonAction resolves a single round and hands the screen an event log to
// replay. It does not run the duel to a conclusion — control returns to the player
// to re-plan once playback finishes. Nothing here draws, and nothing downstream
// recomputes an outcome; the log is the record of what happened, and the returned
// duelists are the state it happened to.
func DuelButtonAction(gs *state.GlobalState) {
	// Ignore the press while a round is still playing back, or once someone is down.
	if gs.DuelCursor < len(gs.DuelLog) {
		return
	}
	if !gs.Fighter.Alive() || !gs.Enemy.Alive() {
		return
	}

	gs.DuelRound++

	// The opponent re-plans every round against its own budget.
	gs.EnemyActions = combat.PlanGreedy(gs.Enemy.Duelist)

	log, fighterAfter, enemyAfter := combat.ResolveRound(
		gs.Fighter.Duelist, gs.Enemy.Duelist,
		gs.FighterActions, gs.EnemyActions,
		gs.DuelRound,
	)

	// Playback walks the log and moves the health bars; these are the authoritative
	// end states, applied when the cursor reaches the end.
	gs.FighterAfter = fighterAfter
	gs.EnemyAfter = enemyAfter

	gs.DuelLog = log
	gs.DuelCursor = 0
	gs.DuelTicks = 0

	fmt.Printf("Round %d: %d events (fighter %d AP, enemy %d AP)\n",
		gs.DuelRound, len(log),
		gs.Fighter.ActionPoints(), gs.Enemy.ActionPoints())
}

func ExitButtonAction(gs *state.GlobalState) {
	fmt.Println("Exit Button Clicked")
	gs.ShouldClose = true
}
