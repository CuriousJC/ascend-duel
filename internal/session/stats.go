package session

// **The run's statistics: what this journey has come to so far, by name.**
//
// A run keeps one profile.Tally for the whole of its length, saved with it, and the profile is
// handed it once — when the run ends, through screens.endRun. That ordering is what makes the
// figures safe to keep:
//
//   - **A resume cannot count a fight twice.** A duel's rounds are held in a tally of their own and
//     join the run's only when the duel settles, and only the run's is saved — so a duel quit
//     halfway, and fought again from the top on resume, is counted once. What is spent mid-duel is
//     counted straight away, because the spend is saved with the run and is not undone by a resume.
//   - **The chosen-seed split is decided once**, at the end, by the one flag the run already carries.
//   - **Everything here is testable without a window.** Purchases and uses are counted inside the
//     session methods that perform them, and a round is counted off its resolved event log — the
//     same rule the play counts and the purse are under: presentation may never change an outcome,
//     and a statistic is an outcome.
//
// **The names are the contract with every profile already on disk**, on the package's
// name-never-an-ordinal rule: `relic:<record key>`, `dealt:element:fire`. Rename one and every
// player's figure for it reads zero.

import (
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/profile"
)

// The totals. Each is summed across the run, and then across every run.
const (
	StatRunsDefeat    = "runs:" + EndedInDefeat
	StatRunsAbandoned = "runs:" + EndedByChoice
	StatRunsClaimed   = "runs:" + EndedInVictory

	StatFightsWon  = "fights:won"
	StatFightsLost = "fights:lost"
	StatRounds     = "rounds"

	StatDealt   = "dealt"
	StatTaken   = "taken"
	StatHealed  = "healed"
	StatHits    = "hits"
	StatBlocked = "blocked"
	StatPlayed  = "played"

	StatVitaeEarned = "vitae:earned"
	StatVitaeSpent  = "vitae:spent"
)

// The records. A Most is the highest figure any one run, fight, round or blow has reached; a Least
// the lowest.
const (
	StatBestHit       = "hit"
	StatBestRound     = "round"
	StatBestFight     = "fight"
	StatBestRunDealt  = "run:dealt"
	StatBestRunWins   = "run:wins"
	StatBestRealm     = "realm"
	StatBestVitaeHeld = "vitae:held"

	// StatFastestWin is a Least: the fewest rounds a duel has been won in.
	StatFastestWin = "win:rounds"
)

// The families: a prefix and a record key. **A prefix rather than one name per record**, so a relic
// authored tomorrow is counted with nothing edited here.
const (
	StatRelicBought = "relic:"
	StatPotionDrunk = "potion:"
	StatTonicDrunk  = "tonic:"
	StatCantripCast = "cantrip:"
	StatEssenceUsed = "essence:"
	StatRuneUsed    = "rune:"
	StatStoneUsed   = "stone:"
	StatGoodBought  = "good:"
	StatHandFormed  = "hand:"

	// The traits: a relic worn with one — bought so, or a cantrip's relic cast — and a card taken out
	// of an action bundle carrying one.
	StatRelicTrait = "relic-trait:"
	StatCardTrait  = "card-trait:"

	// StatCardTaken is a card taken out of an action bundle, by concept key.
	StatCardTaken = "card-taken:"

	StatCreatureDefeated = "defeated:"
	StatCreatureKiller   = "killed-by:"

	// The same two, per element the creature was dealt as: `defeated-as:<record>:<element>`. **A
	// family of their own rather than a longer key under the two above**, so a sum over `defeated:`
	// counts each fight once. See CreatureAs.
	StatCreatureDefeatedAs = "defeated-as:"
	StatCreatureKillerAs   = "killed-by-as:"

	StatPlayedElement = "played:element:"
	StatPlayedForm    = "played:form:"
	StatAttackElement = "attacks:element:"
	StatAttackForm    = "attacks:form:"
	StatDealtElement  = "dealt:element:"
	StatDealtForm     = "dealt:form:"
)

// Stats is the run's tally so far. **A copy**, so a page reading it every frame cannot move it.
func (s *Session) Stats() profile.Tally {
	var out profile.Tally
	out.Merge(s.stats)
	return out
}

// StatsSoFar is the run's tally with the duel being fought added on: every round already played
// counts, where Stats waits for the duel to settle. **For reading only** — what is saved and what the
// profile is handed is Stats, which is why a resume cannot count a round twice. A copy, like Stats.
func (s *Session) StatsSoFar() profile.Tally {
	out := s.Stats()
	out.Merge(s.duel)
	return out
}

// count adds to one of the run's totals. Every recording in the package goes through it.
func (s *Session) count(name string, n int) { s.stats.Add(name, n) }

// CountCantrip records a cantrip cast. **The one count a screen raises**, because a cast is a scroll
// leaving the case and a cantrip-relic going onto a fighter the session never holds — see
// screens.castCantrip. The scroll leaving is DropScroll, which a sale also does.
func (s *Session) CountCantrip(key string) {
	s.count(StatCantripCast+key, 1)
	if c, ok := CantripByKey(key); ok {
		for _, t := range combat.RelicTraits() {
			if c.Worn().Traits.Has(t) {
				s.count(StatRelicTrait+t.String(), 1)
			}
		}
	}
}

// TallyRound adds one resolved round to the duel's statistics: what the player dealt and took, the
// hands formed, the cards played and the records broken.
//
// **The resolved log and the turn that was handed to the resolver**, read once, at adoption — the
// rule the purse and the play counts are written under. A hit names its card by Slot, an index into
// the turn as it was played, which is what lets damage be told by element and form without the event
// carrying either.
//
// **Into the duel's tally, not the run's.** See Session.duel for why, and TallyFight for the move.
func (s *Session) TallyRound(log []combat.Event, turn []combat.Card) {
	d := &s.duel
	d.Add(StatRounds, 1)

	for _, c := range turn {
		d.Add(StatPlayed, 1)
		d.Add(StatPlayedForm+c.Form().String(), 1)
		d.Add(StatPlayedElement+c.Element.String(), 1)
		if c.Spec().Verb == combat.VerbAttack {
			d.Add(StatAttackForm+c.Form().String(), 1)
			d.Add(StatAttackElement+c.Element.String(), 1)
		}
	}

	round := 0
	for _, e := range log {
		switch e.Kind {
		case combat.KindDamage:
			if e.Target == combat.SideA {
				d.Add(StatTaken, e.Amount)
				continue
			}
			round += e.Amount
			d.Add(StatDealt, e.Amount)
			d.Add(StatHits, 1)
			d.Max(StatBestHit, e.Amount)
			if e.Slot >= 0 && e.Slot < len(turn) {
				c := turn[e.Slot]
				d.Add(StatDealtForm+c.Form().String(), e.Amount)
				d.Add(StatDealtElement+c.Element.String(), e.Amount)
			}
		case combat.KindBlocked:
			if e.Target == combat.SideA {
				d.Add(StatBlocked, 1)
			}
		case combat.KindHealed, combat.KindDrained, combat.KindRegenerated:
			if e.Side == combat.SideA {
				d.Add(StatHealed, e.Amount)
			}
		case combat.KindHand:
			if e.Side != combat.SideA {
				continue
			}
			if h, ok := combat.HandByID(e.Hand); ok {
				d.Add(StatHandFormed+h.Key, 1)
			}
		}
	}
	d.Max(StatBestRound, round)
}

// TallyFight settles a duel into the run's statistics: everything its rounds came to, who it was
// against, whether it was won, and how many rounds it took.
//
// **Called once per duel, where the ledger closes it**, which is the one place a settled duel is
// written down whichever way it went. `element` is the colour the creature was dealt as, and an
// empty one counts the creature without it.
func (s *Session) TallyFight(creature, element string, won bool, rounds int) {
	d := s.duel
	s.duel = profile.Tally{}

	d.Max(StatBestFight, d.Total(StatDealt))
	if won {
		d.Add(StatFightsWon, 1)
		d.Add(StatCreatureDefeated+creature, 1)
		if element != "" {
			d.Add(CreatureAs(StatCreatureDefeatedAs, creature, element), 1)
		}
		d.Min(StatFastestWin, rounds)
	} else {
		d.Add(StatFightsLost, 1)
		d.Add(StatCreatureKiller+creature, 1)
		if element != "" {
			d.Add(CreatureAs(StatCreatureKillerAs, creature, element), 1)
		}
	}
	s.stats.Merge(d)
}

// CreatureAs is the name one creature dealt as one element is counted under, in one of the two
// per-element families.
func CreatureAs(family, creature, element string) string {
	return family + creature + ":" + element
}

// FinalStats is the run's tally as it goes onto the profile: everything counted along the way, plus
// how the run ended and the records only a finished run can set.
//
// **Taken from the summary rather than worked out again**, so the splash and the statistics cannot
// disagree about how far a run got or what it dealt.
func (s *Session) FinalStats(sum RunSummary) profile.Tally {
	out := s.Stats()
	out.Add("runs:"+sum.Ended, 1)
	out.Max(StatBestRealm, sum.Realm)
	out.Max(StatBestRunDealt, sum.Dealt)
	out.Max(StatBestRunWins, sum.Defeated)
	return out
}

// Favorites is the element and the form the run has played the most attack cards of, this duel's
// rounds included — what a Torrent keeps a streak in. **A tie is no favorite**, and so is a run
// that has attacked with nothing yet: Basic and FormNone, which leave a Torrent asleep rather than
// picking between two the player has used equally.
func (s *Session) Favorites() (combat.Element, combat.Form) {
	t := s.StatsSoFar()
	element, best, tied := combat.Basic, 0, false
	for _, e := range combat.AllElements {
		if e == combat.Basic {
			continue
		}
		switch n := t.Total(StatAttackElement + e.String()); {
		case n > best:
			element, best, tied = e, n, false
		case n == best && n > 0:
			tied = true
		}
	}
	if tied {
		element = combat.Basic
	}
	form, most, even := combat.FormNone, 0, false
	for _, f := range combat.Forms() {
		switch n := t.Total(StatAttackForm + f.String()); {
		case n > most:
			form, most, even = f, n, false
		case n == most && n > 0:
			even = true
		}
	}
	if even {
		form = combat.FormNone
	}
	return element, form
}
