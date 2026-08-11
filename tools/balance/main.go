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
	"flag"
	"fmt"
	"strings"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/decks"
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

// detail names one record to print round by round, under the summary table. The table is
// what a 96-enemy roster wants; this is what a single suspect entry wants.
var detail = flag.String("v", "",
	"an EnemyRecord to print round by round, e.g. -v OgreWarlord")

func main() {
	flag.Parse()

	duelists := data.LoadDuelists()
	recs := data.LoadEnemies()

	fighter := duelistOfDuelist(duelists["Fighter1"])
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

	// **A summary line per enemy, not seven rows each** *(2026-08-11)*. The roster went from
	// four records to ninety-six, and seven postures each is 672 duels — a transcript nobody
	// reads. What is worth reading at that size is the number this tool exists to surface:
	// how many of the seven postures beat this enemy. `-v <EnemyRecord>` still prints the
	// round-by-round detail, for one record at a time.
	fmt.Printf("\n%-24s %-10s %-6s %5s %4s %4s   %s\n",
		"enemy", "style", "floors", "life", "AP", "Str", "beaten by")
	fmt.Println(strings.Repeat("-", 96))

	band := 0
	for _, name := range data.EnemyOrder(recs) {
		rec := recs[name]
		enemy := duelistOf(rec)
		style, known := combat.ParsePlanStyle(rec.PlanStyle)

		// A blank line each time the lowest valid floor moves, so the table reads as the
		// tower it describes rather than as ninety-six rows.
		if rec.ValidFloors[0] != band {
			band = rec.ValidFloors[0]
			fmt.Println()
		}

		var beat []string
		for _, p := range postures {
			if playerWins(fighter, enemy, style, p) {
				beat = append(beat, p.label)
			}
		}

		verdict := strings.Join(beat, " ")
		switch len(beat) {
		case 0:
			verdict = "NOTHING - a wall"
		case len(postures):
			verdict = "everything - free"
		}
		if !known {
			verdict += fmt.Sprintf("   ** PlanStyle %q not recognised **", rec.PlanStyle)
		}

		fmt.Printf("%-24s %-10v %d-%d    %5d %4d %4d   %s\n",
			rec.Name, style, rec.ValidFloors[0], rec.ValidFloors[1],
			enemy.MaxLife, enemy.ActionPoints(), enemy.Str, verdict)
	}

	if *detail != "" {
		rec, ok := recs[*detail]
		if !ok {
			fmt.Printf("\nno enemy record called %q\n", *detail)
		} else {
			enemy := duelistOf(rec)
			style, _ := combat.ParsePlanStyle(rec.PlanStyle)
			fmt.Printf("\n== %s  %v  %d life, %d AP, Str %d\n",
				rec.Name, style, enemy.MaxLife, enemy.ActionPoints(), enemy.Str)
			for _, p := range postures {
				report(fighter, enemy, style, p)
			}
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
	// The first two rounds are printed because a tactician's character is that its second
	// round is not its first, and a verdict alone would not show that.
	sample := func(round int, enemyPlan []combat.ActionKind, dealt, taken int) {
		if round <= 2 {
			fmt.Printf("   r%d %-9s vs %-30s deal %2d  take %2d\n",
				round, p.label, label(enemyPlan), dealt, taken)
		}
	}

	f, e, round := play(fighter, enemy, style, p, sample)
	fmt.Printf("      %-9s -> %s\n", p.label, outcome(f, e, round))
}

// play runs one posture against one enemy for a whole duel and hands back how it ended.
//
// Split out of report on 2026-08-11 so the summary table can ask for a verdict without
// printing a transcript. **One loop, two callers**: a summary that played the duel its own
// way could disagree with the detail view of the same matchup, which is exactly the kind of
// quiet lie this tool exists to catch elsewhere.
//
// `each` may be nil, and is called with the round number, what the opponent queued, and what
// each side lost in that round.
func play(fighter, enemy combat.Duelist, style combat.PlanStyle, p playerRound,
	each func(round int, enemyPlan []combat.ActionKind, dealt, taken int)) (combat.Duelist, combat.Duelist, int) {

	plan := append(append([]combat.ActionKind{}, p.defence...), p.attack...)

	f, e := fighter, enemy
	round := 0

	// **The opponent draws from the real deck** *(2026-08-11)*, through the same
	// internal/decks pile the game uses, seeded the same way. Before enemies had decks a
	// style conjured its cards and this loop needed nothing but the style; now a brute that
	// draws no Heavy does not swing one, and a report that skipped the deck would be a
	// report about an enemy nobody fights.
	//
	// A fresh pile per posture, so each row starts from the same shuffle and the seven of
	// them can be compared with each other.
	pile := decks.NewEnemyPile(decks.EnemySeed, decks.EnemyHandSize)

	for f.Alive() && e.Alive() && round < stalemateRounds {
		round++
		enemyPlan := pile.Plan(style, e)

		beforeF, beforeE := f.CurrentLife, e.CurrentLife
		_, f, e = combat.ResolveRound(f, e, plan, enemyPlan, round)

		if each != nil {
			each(round, enemyPlan, beforeE-e.CurrentLife, beforeF-f.CurrentLife)
		}
	}
	return f, e, round
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
// entities.NewEnemyFrom, which needs an *ebiten.Image and therefore a graphics context
// this tool has no reason to open — but it reads LifePerCon from entities so the conversion
// cannot drift from the game's.
func duelistOf(d data.EnemyData) combat.Duelist {
	du := combat.Duelist{Con: d.Constitution, Str: d.Strength, Spd: d.Speed}
	du.MaxLife = du.Con * entities.LifePerCon
	du.CurrentLife = du.MaxLife
	return du
}

// duelistOfDuelist is the same for the player's record, which is a different struct since
// the roster split on 2026-08-11. Two near-identical functions rather than one generic one:
// the two records genuinely have different fields, and the day they stop sharing even these
// three is the day a shared helper would have been quietly wrong.
func duelistOfDuelist(d data.DuelistData) combat.Duelist {
	du := combat.Duelist{Con: d.Constitution, Str: d.Strength, Spd: d.Speed}
	du.MaxLife = du.Con * entities.LifePerCon
	du.CurrentLife = du.MaxLife
	return du
}

// playerWins plays the posture out and reports only the verdict, for the summary table.
//
// It shares report's loop rather than duplicating it — see there for why the duel is played
// rather than a damage rate divided into a life total.
func playerWins(fighter, enemy combat.Duelist, style combat.PlanStyle, p playerRound) bool {
	f, e, _ := play(fighter, enemy, style, p, nil)
	return !e.Alive() && f.Alive()
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
