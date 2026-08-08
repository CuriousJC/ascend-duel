// Command balance prints what every shipped enemy actually does to the shipped fighter, so
// the roster can be judged from numbers rather than from the style names sounding varied.
//
//	go run ./tools/balance
//
// It exists because the first version of the roster shipped an unwinnable enemy and nobody
// could see it. Warden1 guards every round, which halves everything the player deals, and at
// 120 life that made it a 24-round grind against a fighter who dies in 10 — arithmetic that
// is obvious in a table and invisible while playing, because losing slowly looks a lot like
// losing to bad draws.
//
// **It reads the real data and the real rules.** Records come from data.LoadCombatants and
// every number below is produced by combat.ResolveRound, so this cannot drift from the game
// the way a spreadsheet would. It is the same trick as tools/glyphsheet: a picture of
// something otherwise only visible by playing.
//
// Needs no window. data is plain JSON and internal/combat imports no graphics, which is the
// property that makes tooling like this cheap and is worth protecting.
package main

import (
	"fmt"
	"sort"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/entities"
)

// playerRound is a posture the fighter can take, and what it costs them to hold it. The
// three below bracket the real choice: all offence, broad defence, or precise defence.
//
// attack is what the fighter deals in that posture, worked out from what is left of their
// budget after paying for the defence. It is a hand-built estimate rather than a simulated
// draw — the deck is on the screen, not in the rules — so treat the damage columns as the
// shape of a fight rather than its transcript.
type playerRound struct {
	label   string
	defence []combat.ActionKind
	attack  []combat.ActionKind
}

func main() {
	recs := data.LoadCombatants()

	fighter := duelistOf(recs["Fighter1"])
	fmt.Printf("Fighter1: %d life, %d AP, Str %d, %d actions/round\n",
		fighter.MaxLife, fighter.ActionPoints(), fighter.Str, fighter.MaxActions())

	// The postures bracket the real choice. The first three are the original set — all offence,
	// broad defence, precise defence — and the rest were added on 2026-08-08 with the concepts
	// they exercise. **A card no posture plays is a card this tool cannot see**, which is how
	// four new concepts would otherwise have shipped with their balance unmeasured.
	postures := []playerRound{
		// 6 AP: everything into damage.
		{"all-out", nil, []combat.ActionKind{combat.Heavy, combat.Strike}},
		// Guard is 3, leaving a Strike.
		{"guarding", []combat.ActionKind{combat.Guard}, []combat.ActionKind{combat.Strike}},
		// Two Dodges are 4, leaving a Strike.
		{"dodging", []combat.ActionKind{combat.Dodge, combat.Dodge}, []combat.ActionKind{combat.Strike}},

		// Mirror is 4, leaving a Jab. **This is the posture to read first**: Mirror reflects
		// every attack it stops, so its value is set entirely by how much the opponent commits.
		// Against a swarm it should be devastating and against a warden nearly worthless, and if
		// it is strong against everything then the reflect fraction is wrong.
		{"mirroring", []combat.ActionKind{combat.Mirror}, []combat.ActionKind{combat.Jab}},
		// Three Braces are 3, leaving a Strike — the cheap partial defence spread wide, against
		// two Dodges' precise negation.
		{"bracing", []combat.ActionKind{combat.Brace, combat.Brace, combat.Brace}, []combat.ActionKind{combat.Strike}},
		// Feint is 3 and a Strike is 2. Only distinguishable from all-out against an opponent
		// holding negations, which is what makes it the anti-Riposte reading.
		{"feinting", nil, []combat.ActionKind{combat.Feint, combat.Strike}},
		// Ritual is 4, leaving a Jab, and banks +5. Repeated every round it is a posture that
		// pays 4 AP a round for a budget it never gets to spend — deliberately a bad plan, and
		// the floor a real Ritual line has to beat.
		{"ritual", []combat.ActionKind{combat.Ritual}, []combat.ActionKind{combat.Jab}},
	}
	fmt.Println("\npostures:")
	for _, p := range postures {
		fmt.Printf("  %-9s %s\n", p.label, label(append(append([]combat.ActionKind{}, p.defence...), p.attack...)))
	}

	for _, name := range enemyNames(recs) {
		rec := recs[name]
		enemy := duelistOf(rec)
		style, known := combat.ParsePlanStyle(rec.PlanStyle)

		warn := ""
		if !known {
			warn = fmt.Sprintf("  ** PlanStyle %q not recognised, defaulted **", rec.PlanStyle)
		}
		fmt.Printf("\n== %s  %v  %d life, %d AP, Str %d%s\n",
			name, style, enemy.MaxLife, enemy.ActionPoints(), enemy.Str, warn)

		for _, p := range postures {
			report(fighter, enemy, style, p)
		}
	}

	fmt.Println("\nEvery duel above is played out through combat.ResolveRound, so the rules are" +
		"\nexact. What is idealised is the *hand*: the fighter repeats one posture every round" +
		"\nand always draws it. A real player is at the mercy of the deck, so read these as the" +
		"\nbest case for each posture — an enemy that beats a posture here beats it always." +
		"\n\nWhat to look for: every enemy should lose to *something* and win against *something*." +
		"\nAn enemy beaten by every posture is free, and one that beats them all is a wall." +
		"\n\nAnd per posture: a posture that wins against everything is a card that needs pricing." +
		"\nMirror is the one to watch — it reflects what it stops, so it should read as devastating" +
		"\nagainst a swarm and near-useless against a warden. Strong everywhere means wrong.")
}

// stalemateRounds is where a duel is called a draw. A fight nobody can finish is as broken
// as one nobody can win, and both need a bound to be reported rather than hung on.
const stalemateRounds = 40

// report plays the posture against the enemy for a whole duel, prints the first two rounds
// as a sample of the exchange, and then says how it actually ended.
//
// **The verdict comes from playing it out, not from a rate.** An earlier version divided
// life by the last round's damage, which quietly lied twice: it read a killing round as the
// steady state, and it flattened the tactician's setup/payoff rhythm into whichever half it
// happened to sample. Running the duel costs nothing here and cannot be wrong about a rule,
// because it is the same ResolveRound the game calls.
//
// The first two rounds are still printed because a tactician's character is that its second
// round is not its first, and a verdict alone would not show that.
func report(fighter, enemy combat.Duelist, style combat.PlanStyle, p playerRound) {
	plan := append(append([]combat.ActionKind{}, p.defence...), p.attack...)

	f, e := fighter, enemy
	round := 0

	for f.Alive() && e.Alive() && round < stalemateRounds {
		round++
		enemyPlan := combat.PlanFor(style, e)

		beforeF, beforeE := f.CurrentLife, e.CurrentLife
		_, f, e = combat.ResolveRound(f, e, plan, enemyPlan, round)

		if round <= 2 {
			fmt.Printf("   r%d %-9s vs %-30s deal %2d  take %2d\n",
				round, p.label, label(enemyPlan),
				beforeE-e.CurrentLife, beforeF-f.CurrentLife)
		}
	}

	fmt.Printf("      %-9s -> %s\n", p.label, outcome(f, e, round))
}

// outcome describes how the duel actually finished.
func outcome(f, e combat.Duelist, rounds int) string {
	switch {
	case !e.Alive() && !f.Alive():
		return fmt.Sprintf("both fall in round %d", rounds)
	case !e.Alive():
		return fmt.Sprintf("fighter WINS in %d rounds, %d/%d life left",
			rounds, f.CurrentLife, f.MaxLife)
	case !f.Alive():
		return fmt.Sprintf("fighter LOSES in %d rounds, enemy on %d/%d",
			rounds, e.CurrentLife, e.MaxLife)
	default:
		return fmt.Sprintf("STALEMATE after %d rounds - fighter %d/%d, enemy %d/%d",
			rounds, f.CurrentLife, f.MaxLife, e.CurrentLife, e.MaxLife)
	}
}

// duelistOf hydrates the stats half of a record. It deliberately does not go through
// entities.NewCombatantFrom, which needs an *ebiten.Image and therefore a graphics context
// this tool has no reason to open — but it reads LifePerCon from entities so the conversion
// cannot drift from the game's.
func duelistOf(d data.CombatantData) combat.Duelist {
	du := combat.Duelist{Con: d.Constitution, Str: d.Strength, Spd: d.Speed}
	du.MaxLife = du.Con * entities.LifePerCon
	du.CurrentLife = du.MaxLife
	return du
}

// enemyNames is every record that is not the player, sorted. Sorted because LoadCombatants
// returns a map and Go randomises that order — the same rule the game itself follows.
func enemyNames(recs map[string]data.CombatantData) []string {
	names := make([]string, 0, len(recs))
	for n := range recs {
		if n != "Fighter1" {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	return names
}

func label(plan []combat.ActionKind) string {
	if len(plan) == 0 {
		return "(nothing)"
	}
	out := ""
	for i, a := range plan {
		if i > 0 {
			out += "+"
		}
		out += a.String()
	}
	return out
}
