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
	gs.Run.WonFight(life, life)
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
	if !s.filled() {
		t.Error("a skipped narration is not filled")
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

	// The first claim, which has both a left part and an amount to type.
	const row = 1
	for i := 0; i < 2000 && w.line < row; i++ {
		w.tick(gs, func(int) image.Point { return image.Point{} })
	}
	full := w.lines[row].plain()
	for i := 0; i < 400 && w.line == row; i++ {
		left, right, on := w.visible(row)
		if !on {
			t.Fatal("the first claim is not on screen")
		}
		shown := ""
		for _, r := range append(left, right...) {
			shown += r.text
		}
		if !strings.HasPrefix(full, shown) {
			t.Fatalf("typed %q, which is not a prefix of %q", shown, full)
		}
		w.tick(gs, func(int) image.Point { return image.Point{} })
	}
}

// The interest line is always read out, +0 included, so a purse too small to earn any still says
// what holding more would have paid.
func TestTheInterestLineIsThereWithNoInterest(t *testing.T) {
	gs := &state.GlobalState{RunSeed: 1, Run: session.New(session.StartingDeck())}
	for gs.Run.Vitae() > 0 {
		gs.Run.SpendVitae(1)
	}
	gs.Run.WonFight(50, 50)

	want := payoutGain(0).text
	for _, l := range payoutLines(gs) {
		if strings.Contains(l.plain(), itoaTest(session.PropagationPer)) && l.pays != nil {
			if len(l.right) != 1 || l.right[0].text != want {
				t.Errorf("a run earning no interest was told %q, want it to end on %q", l.plain(), want)
			}
			return
		}
	}
	t.Error("a run earning no interest was not told about interest at all")
}

// TestEveryClaimIsARowAndItsAmountIsAFigure. The account reads as a rectangle only if every claim
// puts its amount on the right edge, and the amount carries the vitae mark only if it is drawn on
// the figure sheet — the prose sheet has no `¤`.
func TestEveryClaimIsARowAndItsAmountIsAFigure(t *testing.T) {
	for _, l := range payoutLines(wonRun(63)) {
		if l.pays == nil {
			continue
		}
		if l.centered || len(l.right) != 1 {
			t.Errorf("claim %q is not a row ending on one amount", l.plain())
			continue
		}
		if !l.right[0].figure {
			t.Errorf("claim %q sets its amount %q in prose", l.plain(), l.right[0].text)
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

// TestATypedOutNarrationWaitsToBeDismissed. **Nothing advances on its own** — a block that finished
// typing without ever being clicked still holds the screen, so the essences cannot arrive over a total
// the player is still reading.
func TestATypedOutNarrationWaitsToBeDismissed(t *testing.T) {
	gs := wonRun(63)
	w := typewriter{}
	w.setLines(payoutLines(gs))

	for i := 0; i < 20000; i++ {
		w.tick(gs, func(int) image.Point { return image.Point{} })
		if w.finished() {
			t.Fatalf("the narration released itself after %d ticks", i)
		}
	}
	if !w.filled() {
		t.Fatal("the narration never finished typing")
	}

	w.release()
	if !w.finished() {
		t.Error("a released narration is not finished")
	}
	if got := gs.Run.Spoils().Total(); got != 0 {
		t.Errorf("a fully typed narration left %d vitae unclaimed", got)
	}
}

// TestFillingDoesNotLeaveTheNarration. **The click that fills the block may not also be the click
// that leaves it** *(owner's call, 2026-09-08)*. A player asking to see the whole payout is asking
// to read it, not to be past it, so the essences wait for a second click.
func TestFillingDoesNotLeaveTheNarration(t *testing.T) {
	gs := wonRun(63)
	w := typewriter{}
	w.setLines(payoutLines(gs))
	w.skip(gs)

	for i := 0; i < 600; i++ {
		w.tick(gs, func(int) image.Point { return image.Point{} })
		if w.finished() {
			t.Fatalf("a filled narration released itself after %d ticks", i)
		}
	}

	w.release()
	if !w.finished() {
		t.Error("a released narration is not finished")
	}
}
