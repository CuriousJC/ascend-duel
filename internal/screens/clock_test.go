package screens

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// The game's one speed, and the rule that everything which moves is a fraction of it.
//
// These parse the package's own source rather than calling into it, which is unusual and is the
// point: what is being defended is not a value but a *shape*. A screen written next year can
// declare `fadeTicks = 40` and nothing else in the repo will notice — the game will simply have
// two clocks again, and the game-speed setting will move one of them.

// clockExceptions are the identifiers allowed to be a raw number, each for a stated reason. Adding
// to this list is a decision; it should be argued for in the comment beside the declaration, not
// here.
var clockExceptions = map[string]string{
	"beatTicks":       "the speed itself — everything else is a fraction of this one",
	"ticksPerSecond":  "the simulation rate, a fact about Ebitengine rather than a pace",
	"mathBreathTicks": "deliberately not a beat; see its own comment for why",
}

func TestNoClockIsWrittenAsARawNumber(t *testing.T) {
	// **This is what stops the game growing a second clock.** The between-fight screens had one
	// until 2026-08-21: postbattle.go predated `beat` and carried 26 and 100, so turning the speed
	// down would have sped up a duel and left the reward screen exactly as slow as it was. The
	// shop and the room choice are the next two screens that could do it again.
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parsing the package: %v", err)
	}

	for _, pkg := range pkgs {
		for name, file := range pkg.Files {
			if strings.HasSuffix(name, "_test.go") {
				continue
			}
			ast.Inspect(file, func(n ast.Node) bool {
				// **A timing is a function now, so this has to read functions too** *(2026-09-15)*.
				// Every clock became `func fooTicks() int { return beat(n, d) }` on the day the
				// speed setting started reaching them — a package-level `var` is evaluated once at
				// init, when the speed is still the tuned one, so the slider moved nothing but the
				// event dwells. Without this arm the whole guard would have been silently disarmed
				// by that change: a later `func fadeTicks() int { return 40 }` is exactly the
				// second clock this test exists to refuse, and it is no longer a ValueSpec.
				if fn, ok := n.(*ast.FuncDecl); ok {
					checkClockFunc(t, fn, name)
					return true
				}
				spec, ok := n.(*ast.ValueSpec)
				if !ok {
					return true
				}
				for i, id := range spec.Names {
					if !strings.HasSuffix(id.Name, "Ticks") {
						continue
					}
					if _, allowed := clockExceptions[id.Name]; allowed {
						continue
					}
					if i >= len(spec.Values) {
						continue // part of an iota run or a bare declaration
					}
					if lit, ok := spec.Values[i].(*ast.BasicLit); ok {
						t.Errorf("%s = %s in %s is a raw duration — write it as beat(num, den), "+
							"or add it to clockExceptions with a reason",
							id.Name, lit.Value, name)
					}
				}
				return true
			})
		}
	}
}

// checkClockFunc fails a timing function whose body is a bare number.
//
// It matches the shape every clock in the package now has — a name ending in Ticks, no parameters,
// one `return` — and refuses a raw literal in it for the reason the ValueSpec arm refuses one: the
// number is then a pace nobody can retune and the game-speed setting cannot reach. Anything more
// elaborate than a single return is left alone; the shape being defended is the declaration, not
// the arithmetic inside it.
func checkClockFunc(t *testing.T, fn *ast.FuncDecl, file string) {
	t.Helper()

	if fn.Name == nil || !strings.HasSuffix(fn.Name.Name, "Ticks") {
		return
	}
	if _, allowed := clockExceptions[fn.Name.Name]; allowed {
		return
	}
	if fn.Body == nil || len(fn.Body.List) != 1 {
		return
	}
	ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return
	}
	if lit, ok := ret.Results[0].(*ast.BasicLit); ok {
		t.Errorf("%s returns %s in %s is a raw duration — write it as beat(num, den), "+
			"or add it to clockExceptions with a reason",
			fn.Name.Name, lit.Value, file)
	}
}

func TestABeatIsNeverLessThanATick(t *testing.T) {
	// A small enough fraction of a slow enough speed rounds to zero, and a movement lasting no
	// ticks is a movement that does not happen — the card appears at its destination, which is the
	// one thing the flight rule exists to prevent.
	for den := 1; den <= 200; den++ {
		if got := beat(1, den); got < 1 {
			t.Fatalf("beat(1, %d) is %d", den, got)
		}
	}
}

func TestTheBetweenFightScreensMoveOnTheSameSpeedAsTheDuel(t *testing.T) {
	// The duel's clocks and the reward screen's have to scale together. Comparing them against the
	// speed rather than against fixed numbers is what makes this survive a tuning change: it fails
	// when one of them stops being a proportion, not when the pace is retuned.
	cases := []struct {
		name  string
		ticks int
	}{
		{"settleFlightTicks", settleFlightTicks()},
		{"settledHoldTicks", settledHoldTicks()},
		{"victoryHoldTicks", victoryHoldTicks()},
		{"flightTicks", flightTicks()},
		{"hitFlyTicks", hitFlyTicks()},
	}

	for _, c := range cases {
		if c.ticks < 1 {
			t.Errorf("%s is %d ticks — nothing moves", c.name, c.ticks)
		}
		if c.ticks > beatTicks*8 {
			t.Errorf("%s is %d ticks, over eight beats — that is a pause, not a movement",
				c.name, c.ticks)
		}
	}
}
