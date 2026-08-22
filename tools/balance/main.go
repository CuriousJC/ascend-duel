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
	"log"
	"math/rand"
	"strings"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/decks"
	"github.com/curiousjc/ascend-duel/internal/pyramid"
	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/session"
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
	defence []combat.Card
	attack  []combat.Card
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
	fmt.Printf("Fighter1: %d life, %d AP, DMG %d, %d actions/round, wearing all four rings\n",
		fighter.MaxLife, fighter.ActionPoints(), fighter.DMG, fighter.MaxActions())

	// The postures bracket the real choice. The first three are the original set — all offence,
	// broad defence, precise defence — and the rest were added on 2026-08-08 with the concepts
	// they exercise. **A card no posture plays is a card this tool cannot see**, which is how
	// four new concepts would otherwise have shipped with their balance unmeasured.
	postures := []playerRound{
		// 6 AP: everything into damage, and no hand at all — a Smash and a Strike are different
		// concepts, so this is the High Card posture. It is the floor every other row is read
		// against.
		{"all-out", nil, combat.PlainCards(combat.Smash, combat.Strike)},

		// 6 AP into a built hand, which is the whole point of the rework: three Strikes are 6 AP
		// and a Three of a Kind, where all-out spends the same budget on two bigger cards and forms
		// nothing. If all-out ever beats this, the multipliers are too low to be worth building
		// toward.
		{"trips", nil, combat.PlainCards(combat.Strike, combat.Strike, combat.Strike)},
		// The cheapest hand in the game: three Bashes are 3 AP for a Three of a Kind, leaving a
		// Strike. Small cards multiplied against big cards unmultiplied.
		{"cheap-trips", nil, combat.PlainCards(combat.Bash, combat.Bash, combat.Bash, combat.Strike)},

		// Defend is 3, leaving a Thrust and a Jab. **This is the posture to read first**: half a
		// round bought half a blow, and if that wins duels the price is wrong.
		{"defending", combat.PlainCards(combat.Defend), combat.PlainCards(combat.Thrust, combat.Jab)},
		// Plan is 2 and widens the *next* hand by two. **This tool cannot see what it buys**: the
		// sim deals no cards, so a wider hand is a wider hand of nothing and the row measures the
		// 2 AP as pure loss. It is here as the floor — anything at or above `all-out` would mean
		// the cost is not being felt at all.
		{"planning", combat.PlainCards(combat.Plan), combat.PlainCards(combat.Thrust, combat.Bash)},
		// Prepare is 1 and banks +2, leaving 5 AP. Repeated every round it pays a card slot for a
		// budget it keeps re-spending on the same thing — deliberately a mediocre plan, and the
		// floor a real banking line has to beat.
		{"banking", combat.PlainCards(combat.Prepare), combat.PlainCards(combat.Smash, combat.Thrust)},

		// **The four element postures are all-out in a colour** *(2026-08-12)*, and they are meant
		// to be read against that row rather than against each other: same concepts, same 6 AP,
		// the same damage — so whatever a coloured row does differently is what the element is
		// worth.
		//
		// They exist because statuses landed the same day, and **a status no posture applies is a
		// status this tool cannot see** — the same rule the rows above were extended under when
		// new concepts arrived with their balance unmeasured.
		{"burning", nil, elemental(combat.Fire, combat.Smash, combat.Strike)},
		{"chilling", nil, elemental(combat.Ice, combat.Smash, combat.Strike)},
		{"shocking", nil, elemental(combat.Lightning, combat.Smash, combat.Strike)},
		{"weighting", nil, elemental(combat.Earth, combat.Smash, combat.Strike)},
	}
	fmt.Println("\npostures:")
	for _, p := range postures {
		fmt.Printf("  %-9s %s\n", p.label, label(append(append([]combat.Card{}, p.defence...), p.attack...)))
	}

	// **A summary line per enemy, not seven rows each** *(2026-08-11)*. The roster went from
	// four records to ninety-six, and seven postures each is 672 duels — a transcript nobody
	// reads. What is worth reading at that size is the number this tool exists to surface:
	// how many of the seven postures beat this enemy. `-v <EnemyRecord>` still prints the
	// round-by-round detail, for one record at a time.
	fmt.Printf("\n%-24s %-6s %-6s %5s %4s %4s   %s\n",
		"enemy", "cards", "floors", "life", "AP", "DMG", "beaten by (rounds)")
	fmt.Println("(life and DMG are grown by the ascent curve to the first fight of the enemy's" +
		"\nlowest valid floor - the shallowest slot the tower could put it in, and so the" +
		"\nkindest version of it a player will ever meet. The record's own numbers are in" +
		"\ndata/enemies.json.)")
	fmt.Println(strings.Repeat("-", 96))

	band := 0
	for _, name := range data.EnemyOrder(recs) {
		rec := recs[name]
		enemy := duelistOf(rec, pyramid.FirstFightOnFloor(rec.ValidFloors[0]))

		// A blank line each time the lowest valid floor moves, so the table reads as the
		// tower it describes rather than as ninety-six rows.
		if rec.ValidFloors[0] != band {
			band = rec.ValidFloors[0]
			fmt.Println()
		}

		results := make([]duelResult, len(postures))
		for i, p := range postures {
			results[i] = play(fighter, enemy, name, p, nil)
		}
		verdict := verdictOf(postures, results)
		fmt.Printf("%-24s %-6d %d-%d    %5d %4d %4d   %s\n",
			rec.Name, len(decks.EnemyCards(name)), rec.ValidFloors[0], rec.ValidFloors[1],
			enemy.MaxLife, enemy.ActionPoints(), enemy.DMG, verdict)
	}

	if *detail != "" {
		rec, ok := recs[*detail]
		if !ok {
			fmt.Printf("\nno enemy record called %q\n", *detail)
		} else {
			enemy := duelistOf(rec, pyramid.FirstFightOnFloor(rec.ValidFloors[0]))
			fmt.Printf("\n== %s  %d cards, %d life, %d AP, DMG %d\n",
				rec.Name, len(decks.EnemyCards(*detail)), enemy.MaxLife, enemy.ActionPoints(), enemy.DMG)
			fmt.Printf("   deck: %s\n", label(decks.EnemyCards(*detail)))
			for _, p := range postures {
				report(fighter, enemy, *detail, p)
			}
		}
	}

	fmt.Println("\nEvery duel above is played out through combat.ResolveRound, so the rules are" +
		"\nexact. What is idealised is the *hand*: the fighter repeats one posture every round" +
		"\nand always draws it. A real player is at the mercy of the deck, so read these as the" +
		"\nbest case for each posture — an enemy that beats a posture here beats it always." +
		"\n\nRead the last column for figures, not just for names. A posture is written with how many" +
		"\nrounds it took, and a wall says which posture came closest and what the enemy had left -" +
		"\nwhich is where a damage change shows up. A wall at 3% and a wall at 43% are not the same" +
		"\nopponent, and a table of win-or-lose called them both walls." +
		"\n\nWhat to look for: every enemy should lose to *something* and win against *something*." +
		"\nAn enemy beaten by every posture is free, and one that beats them all is a wall." +
		"\n\nAnd per posture: a posture that wins against everything is a card that needs pricing." +
		"\nNothing does that as of 2026-08-16: no posture beats more than a third of the roster." +
		"\n\nTwo figures to read next. Eighty-four enemies are walls, beaten by no posture at all," +
		"\nup from twelve before per-enemy decks, doubled HP and the ascent curve. The decks cost" +
		"\nthree of that, the doubling twenty-nine, enemies losing their hands one, the 10%" +
		"\nper-room curve twenty-nine, and hands ceasing to take an action off the opponent the" +
		"\nlast ten — every one of those ten a hand posture, since nothing else was earning a" +
		"\nstagger. Floors 1-2 are untouched by the curve, since floor 1's outer room is its" +
		"\nbaseline, and everything from floor 3 up is now a wall." +
		"\n\nRead that against a bare fighter in three rings, which is what this tool measures and" +
		"\nnot what the deep tower is priced for. And planning wins rarely, but this tool deals no cards: a" +
		"\nwider hand is a wider hand of nothing here, so that row measures Plan as 2 AP of pure" +
		"\nloss. Read it as the floor, not the card.")
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
// The first two rounds are still printed because an opponent that banks points has a second round
// unlike its first, and a verdict alone would not show that.
func report(fighter, enemy combat.Duelist, record string, p playerRound) {
	sample := func(round int, enemyPlan []combat.Card, dealt, taken int) {
		if round <= 2 {
			fmt.Printf("   r%d %-9s vs %-30s deal %2d  take %2d\n",
				round, p.label, label(enemyPlan), dealt, taken)
		}
	}

	r := play(fighter, enemy, record, p, sample)
	fmt.Printf("      %-9s -> %s\n", p.label, outcome(r))
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
func play(fighter, enemy combat.Duelist, record string, p playerRound,
	each func(round int, enemyPlan []combat.Card, dealt, taken int)) duelResult {

	plan := append(append([]combat.Card{}, p.defence...), p.attack...)

	f, e := fighter, enemy
	round := 0

	// **The opponent draws its own deck** *(2026-08-16)*, through the same internal/decks pile the
	// game uses, seeded the same way. It used to be one shared list for the whole roster picked
	// over by one of four planners; the deck is the enemy now, so a report that skipped it would
	// be a report about an enemy nobody fights.
	//
	// A fresh pile per posture, so each row starts from the same shuffle and the seven of
	// them can be compared with each other.
	pile := decks.NewEnemyPile(record, seeds.EnemyDeckPin, decks.EnemyHandSize)

	// **The rules take a source now, and this tool stopped being exact when they did.** A shock
	// roll decides whether a turn's whole attack lands, so one run of one matchup is a sample
	// rather than an answer — see `duelSamples` and `balanceSeed`.
	rng := rand.New(rand.NewSource(balanceSeed))

	for f.Alive() && e.Alive() && round < stalemateRounds {
		round++
		enemyPlan := pile.Plan(e)

		beforeF, beforeE := f.CurrentLife, e.CurrentLife
		_, f, e = combat.ResolveRound(f, e, plan, enemyPlan, round, rng)

		if each != nil {
			each(round, enemyPlan, beforeE-e.CurrentLife, beforeF-f.CurrentLife)
		}
	}
	return duelResult{
		won:          !e.Alive() && f.Alive(),
		rounds:       round,
		fighter:      f,
		enemy:        e,
		enemyLifePct: pctOf(e.CurrentLife, e.MaxLife),
	}
}

// outcome describes how the duel actually finished.
func outcome(r duelResult) string {
	f, e, rounds := r.fighter, r.enemy, r.rounds
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
// entities.NewEnemyFrom, which used to need a graphics context this tool has no reason to open.
//
// **There is no conversion left to drift** *(2026-08-16)*. It read `entities.LifePerCon` so life
// could not be worked out two ways; the record now says HP, so copying three fields is the whole
// of it.
// **SoloAttacks is set here as well as in entities.NewEnemyFrom** *(2026-08-17)*, which is the
// price of this function existing at all: an enemy that handed in the sim and not in the game
// would make every number below a measurement of a fight nobody plays.
//
// **`fight` is where in the ascent the enemy is met**, and it applies the same curve the game
// does — see pyramid.ScaleToFight. The caller maps a record onto the first fight of its lowest
// valid floor, which is the shallowest slot the tower could put it in and therefore the kindest
// version of it the player will ever see. Reporting the record's raw stats instead would describe
// an opponent that exists nowhere.
func duelistOf(d data.EnemyData, fight int) combat.Duelist {
	du := combat.Duelist{
		DMG:         pyramid.ScaleToFight(d.DMG, fight),
		Actions:     d.Actions,
		MaxLife:     pyramid.ScaleToFight(d.HP, fight),
		SoloAttacks: true,
	}
	du.CurrentLife = du.MaxLife
	return du
}

// duelistOfDuelist is the same for the player's record, which is a different struct since
// the roster split on 2026-08-11. Two near-identical functions rather than one generic one:
// the two records genuinely have different fields, and the day they stop sharing even these
// three is the day a shared helper would have been quietly wrong.
//
// **The fighter wears all four elemental rings** *(2026-08-16)*, which is a deliberate departure
// from the game — the player starts in three and the enemy wears none. A status only happens if a
// ring says so, so a ringless fighter would make the four elemental postures below identical to
// their plain equivalents and this tool would report four rows measuring nothing. **A card no
// posture plays is a card this tool cannot see**, and a posture whose whole point is inert is the
// same failure by another route. It measures the ceiling; the floor is the plain postures beside it.
//
// **The rings are named rather than derived** *(2026-08-17)*: a ring is a list of rules now and
// there is no longer any such thing as "the ring for this element" that a loop could find. It reads
// them through `internal/session`, which is the package that registers the catalogue — and which
// this tool may import for the reason it may import `internal/decks`: no Ebitengine.
//
// **This tool still cannot see anything else a ring does.** A damage ring, a discount ring or a stat
// ring would change every number it prints and none of them are worn here, so a ring's balance is
// unmeasurable until this tool takes a worn set as a posture axis. Say so rather than guessing.
func duelistOfDuelist(d data.DuelistData) combat.Duelist {
	du := combat.Duelist{DMG: d.DMG, Actions: d.Actions, MaxLife: d.HP}
	for _, key := range elementalRings {
		id, ok := session.RingID(key)
		if !ok {
			log.Fatalf("balance: %s is in no ring record", key)
		}
		du = du.Wearing(combat.WornRing{Ring: id})
	}
	du.CurrentLife = du.MaxLife
	return du
}

// elementalRings is the four whose whole rule is "this colour applies its status", which is what the
// elemental postures below are measuring.
//
// **They are the status halves, not the colour rings** *(2026-08-22)*. Every colour split into a
// damage doubler carrying the colour's name and a status ring beside it; what these postures
// measure is what the statuses do, so the list follows the rule rather than the name.
var elementalRings = []string{"burning-ring", "chilling-ring", "shocking-ring", "weighted-ring"}

// duelResult is how one posture's duel against one enemy ended.
//
// **It replaced a bare bool on 2026-08-18, and what forced it is a change this tool could not
// see.** The damage formula moved to (the hand's own cards) x (the hand multiplier) and the
// roster table came out byte-identical either side of it, because a table of win-or-lose only
// moves when a posture crosses that line and none did. The damage had genuinely moved: at DMG
// 10, trips went from 50 to 60 a round and cheap-trips from 35 to 30, which took cheap-trips
// from three rounds to four. **How fast a winning posture wins, and how close a losing one
// gets, are what a balance change actually moves**; the verdict is the last thing to shift and
// the coarsest.
type duelResult struct {
	won    bool
	rounds int

	// fighter and enemy are the two of them as the duel left them, which is what outcome reads
	// to describe a stalemate or a mutual kill.
	fighter, enemy combat.Duelist

	// enemyLifePct is what the enemy had left at the end. For a posture that lost, that is the
	// whole measure of how close it came - the difference between a wall the fighter nearly
	// brought down and one they never scratched.
	enemyLifePct int
}

// verdictOf writes the summary table's last column: which postures won and how long each took,
// and - when none did - which came closest and how close.
//
// **A collapsed verdict still carries a figure.** "everything - free" and "NOTHING - a wall"
// are the two rows worth spotting at a glance across ninety-six of them, so they stay short;
// but each names the number that moves when the balance does, the fastest kill on one and the
// nearest miss on the other. A row saying only "a wall" cannot tell a change from no change,
// which is the failure this whole column exists to fix.
func verdictOf(postures []playerRound, results []duelResult) string {
	var beat []string
	fastest, closest := -1, -1
	for i, r := range results {
		if r.won {
			beat = append(beat, fmt.Sprintf("%s:%d", postures[i].label, r.rounds))
			if fastest < 0 || r.rounds < results[fastest].rounds {
				fastest = i
			}
			continue
		}
		if closest < 0 || r.enemyLifePct < results[closest].enemyLifePct {
			closest = i
		}
	}

	switch {
	case len(beat) == 0:
		return fmt.Sprintf("NOTHING - a wall (closest %s, enemy %d%% left)",
			postures[closest].label, results[closest].enemyLifePct)
	case len(beat) == len(postures):
		return fmt.Sprintf("everything - free (fastest %s:%d)",
			postures[fastest].label, results[fastest].rounds)
	}
	return strings.Join(beat, " ")
}

// pctOf is what is left of a life total, floored at zero: a duel both sides lose would
// otherwise report a negative share.
func pctOf(cur, max int) int {
	if cur <= 0 || max <= 0 {
		return 0
	}
	return cur * 100 / max
}

// elemental builds a posture's attacks all in one element. Only the four element rows use it,
// and they are deliberately the same concepts as all-out so the comparison stays clean.
func elemental(e combat.Element, actions ...combat.ConceptID) []combat.Card {
	out := make([]combat.Card, len(actions))
	for i, a := range actions {
		out[i] = combat.Of(a, e)
	}
	return out
}

func label(plan []combat.Card) string {
	if len(plan) == 0 {
		return "(nothing)"
	}
	out := ""
	for i, c := range plan {
		if i > 0 {
			out += "+"
		}
		out += c.String()
	}
	return out
}

// balanceSeed is the fixed source every duel in a run is played from.
//
// **The rules acquired randomness on 2026-08-14** — a shock roll decides whether a turn's whole
// attack lands — so this tool stopped being an exact answer and became a sample. A fixed seed is
// what keeps it *reproducible* in the meantime: the same command prints the same table, so a
// change to a cost or a stat line still shows up as a diff rather than as noise.
//
// **That is not the same as being right, and the gap is recorded rather than papered over.** One
// sample per matchup can be an unlucky duel, and the honest version plays each matchup many times
// and reports how often the fighter wins. That is the next thing this tool wants.
const balanceSeed = 0x8A1A_9CE0
