package screens

import (
	"image"
	"strings"
	"testing"

	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
)

// The narration's arithmetic and its claims. **Nothing here draws** — the same narrow exception the
// rest of this package's tests take.

func wonRun(life int) *state.GlobalState {
	gs := &state.GlobalState{RunSeed: 20260822, Run: session.New(session.StartingDeck())}
	gs.Run.AddVitae(15)
	gs.Run.WonFight(life)
	return gs
}

// TestTheScriptNamesEveryFigureItPays. Each sentence that claims a part has to say that part's
// number, or the purse moves for a reason the player was never told.
func TestTheScriptNamesEveryFigureItPays(t *testing.T) {
	gs := wonRun(63)
	spoils := gs.Run.Spoils()
	lines := payoutLines(gs)

	joined := ""
	for _, l := range lines {
		joined += l.plain() + "\n"
	}

	for _, want := range []string{
		itoaTest(spoils.Propagated), itoaTest(spoils.FromLife), itoaTest(spoils.FromRoom),
		itoaTest(gs.Run.Vitae() + spoils.Total()),
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("the script never says %q:\n%s", want, joined)
		}
	}
}

// TestTypingAndSkippingPayTheSame. **Presentation may never change an outcome**: a player who
// clicks through the narration must end on exactly the purse a player who watched it ends on.
func TestTypingAndSkippingPayTheSame(t *testing.T) {
	watched := wonRun(63)
	w := typewriter{}
	w.setLines(payoutLines(watched))
	for i := 0; i < 20000 && !w.finished(); i++ {
		w.tick(watched, func(int) image.Point { return image.Point{} })
	}

	skipped := wonRun(63)
	s := typewriter{}
	s.setLines(payoutLines(skipped))
	s.skip(skipped)

	if watched.Run.Vitae() != skipped.Run.Vitae() {
		t.Errorf("watching paid %d and skipping paid %d",
			watched.Run.Vitae(), skipped.Run.Vitae())
	}
	if !s.finished() {
		t.Error("a skipped narration is not finished")
	}
	if got := watched.Run.Spoils().Total(); got != 0 {
		t.Errorf("a fully typed narration left %d vitae unclaimed", got)
	}
}

// TestALineIsTypedLeftToRight. The visible part of a line is a prefix of it — a sentence that
// revealed its runs out of order would be a sentence read wrong.
func TestALineIsTypedLeftToRight(t *testing.T) {
	gs := wonRun(63)
	w := typewriter{}
	w.setLines(payoutLines(gs))

	full := w.lines[0].plain()
	for i := 0; i < 200 && w.line == 0; i++ {
		runs, on := w.visible(0)
		if !on {
			t.Fatal("the first line is not on screen")
		}
		shown := ""
		for _, r := range runs {
			shown += r.text
		}
		if !strings.HasPrefix(full, shown) {
			t.Fatalf("typed %q, which is not a prefix of %q", shown, full)
		}
		w.tick(gs, func(int) image.Point { return image.Point{} })
	}
}

// TestNoInterestLineWithNoInterest. A sentence saying a purse swelled by nothing teaches the player
// the screen is not reading their run.
func TestNoInterestLineWithNoInterest(t *testing.T) {
	gs := &state.GlobalState{RunSeed: 1, Run: session.New(session.StartingDeck())}
	for gs.Run.Vitae() > 0 {
		gs.Run.SpendVitae(1)
	}
	gs.Run.WonFight(50)

	for _, l := range payoutLines(gs) {
		if strings.Contains(l.plain(), "resonates") {
			t.Errorf("a run earning no interest was told %q", l.plain())
		}
	}
}

func itoaTest(n int) string {
	if n == 0 {
		return "0"
	}
	out := ""
	for n > 0 {
		out = string(rune('0'+n%10)) + out
		n /= 10
	}
	return out
}
