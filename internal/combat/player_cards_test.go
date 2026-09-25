package combat

import "testing"

// The 2026-08-15 card set: three attack forms of three tiers, plus the three plans. Each of
// the plans gets the case that says what it does, and the ladder gets the structural cases that
// say the forms are three ways of doing the same thing at the same prices — which is the claim
// MECHANICS.md makes about the deck and the one a retuned cost quietly breaks.

// hasKind reports whether the log holds an event of the given kind.
func hasKind(events []Event, kind EventKind) bool {
	for _, e := range events {
		if e.Kind == kind {
			return true
		}
	}
	return false
}

// kindCount is how many events of a kind the log holds.
func kindCount(events []Event, kind EventKind) int {
	n := 0
	for _, e := range events {
		if e.Kind == kind {
			n++
		}
	}
	return n
}

func TestEachDefendCardRaisesItsOwnNumberOfShields(t *testing.T) {
	for _, tc := range []struct {
		card ConceptID
		want int
	}{{Brace, 1}, {Block, 2}, {Guard, 3}} {
		c := ConceptOf(tc.card)
		if c.Amount != tc.want || c.Cost != tc.want {
			t.Errorf("%s raises %d for %d AP, want %d for %d", c.Label, c.Amount, c.Cost, tc.want, tc.want)
		}

		events, after, _ := resolve(duelist(10, 6, 100), duelist(10, 6, 100), PlainCards(tc.card), nil, 1)
		if after.Shields != tc.want {
			t.Errorf("a %s left %d shields standing, want %d", c.Label, after.Shields, tc.want)
		}
		if !hasKind(events, KindRaised) {
			t.Errorf("a %s raised no KindRaised — the screen is never told to draw the pips", c.Label)
		}
	}
}

// **One shield eats one attack, whole.** This is the mechanic: a creature is a solo attacker, so
// its turn is several discrete blows and the player decides how many of them to take.
func TestAShieldEatsExactlyOneAttack(t *testing.T) {
	a := duelist(10, 6, 200)
	b := duelist(10, 6, 200)
	b.SoloAttacks = true

	// **Both in one round, because that is the turn a shield covers.** A raises at the end of its
	// own turn and B acts next; a shield still standing when A comes round again is one that
	// expires unspent — see TestUnspentShieldsLapseBeforeTheirOwnerActsAgain.
	events, a2, _ := resolve(a, b, PlainCards(Block), PlainCards(Bash, Bash, Bash), 1)

	if got := countKind(events, KindBlocked); got != 2 {
		t.Errorf("%d attacks blocked, want 2 — one shield is one attack", got)
	}
	landed := 0
	for _, e := range events {
		if e.Kind == KindDamage && e.Side == SideB {
			landed++
		}
	}
	if landed != 1 {
		t.Errorf("%d attacks landed, want 1 — the third Bash had no shield left to meet it", landed)
	}
	if a2.Shields != 0 {
		t.Errorf("%d shields left standing, want 0 — both were spent", a2.Shields)
	}
}

// **A blocked attack lands nothing at all**, which is what separates a shield from a guard: a
// guard leaves a figure and this leaves none.
func TestABlockedAttackDealsNoDamage(t *testing.T) {
	a := duelist(10, 6, 200)
	b := duelist(10, 6, 200)
	b.SoloAttacks = true

	_, a1, _ := resolve(a, b, PlainCards(Brace), PlainCards(Bash), 1)

	if a1.CurrentLife != a.CurrentLife {
		t.Errorf("a shielded duelist lost %d life to a blocked Bash, want none",
			a.CurrentLife-a1.CurrentLife)
	}
}

// **Shields last the turn after they were played and then lapse**, on the same schedule a raised
// guard does — up at the end of your turn, standing through the opponent's, gone before you act
// again. Stockpiling them across quiet rounds is the thing they were built not to do.
func TestUnspentShieldsLapseBeforeTheirOwnerActsAgain(t *testing.T) {
	a := duelist(10, 6, 200)
	b := duelist(10, 6, 200)
	b.SoloAttacks = true

	// Round 1: A raises three and B swings at nothing, so all three survive the round.
	_, a1, b1 := resolve(a, b, PlainCards(Guard), nil, 1)
	if a1.Shields != 3 {
		t.Fatalf("A ended round 1 with %d shields, want the Guard's three standing", a1.Shields)
	}

	// Round 2: they expire at the start of A's own turn, before anything in it resolves.
	events, a2, _ := resolve(a1, b1, nil, nil, 2)
	if a2.Shields != 0 {
		t.Errorf("A carried %d shields into round 3, want them lapsed", a2.Shields)
	}
	if !hasKind(events, KindExpired) {
		t.Error("no KindExpired — the pip row would keep drawing shields the engine had taken away")
	}
}

// **maxShields bounds one card and not a duelist** *(owner's call, 2026-09-09)*. Three Guards is
// nine shields and is meant to be: what stops a turn raising them is the action budget, exactly as
// it stops nine attacks. This test asserted the opposite for nine days — see Duelist.raiseShields
// for why the argument behind the clamp did not hold.
func TestATurnMayRaiseMoreShieldsThanOneCardCan(t *testing.T) {
	a := duelist(10, 12, 200)
	b := duelist(10, 6, 200)

	// Three Guards is nine shields' worth, paid for out of a budget that can afford it.
	_, after, _ := resolve(a, b, PlainCards(Guard, Guard, Guard), nil, 1)
	if after.Shields != 9 {
		t.Errorf("three Guards left %d standing, want the nine they raised", after.Shields)
	}
}

// **One card still cannot raise more than maxShields**, which is the half of the rule that
// survived. RegisterConcept refuses a concept declaring more, and Card.Amount clamps an essence that
// scaled one past it — a card whose face promised a sixth shield would be promising something its
// own turn can never see spent.
func TestOneCardStillCannotRaisePastTheCap(t *testing.T) {
	swollen := PlainCards(Guard)[0]
	swollen.AmountPct = 1000

	if got := swollen.Amount(); got != maxShields {
		t.Errorf("a Guard scaled ten times raises %d, want it held to %d", got, maxShields)
	}
}

// **Nothing in the shipped game gives an enemy a shield**, and that asymmetry is the whole reason
// a count is safe. The player forms hands and lands one figure a turn, so a single shield facing
// that would delete the whole turn.
func TestNoPlayerConceptGivesTheOpponentShields(t *testing.T) {
	for _, id := range PlayerConcepts() {
		if c := ConceptOf(id); c.Verb == VerbShield && c.Form != FormDefend {
			t.Errorf("%s raises shields but is not a defend card", c.Label)
		}
	}
}

func TestDefensesAreSpentWhetherOrNotTheyWereNeeded(t *testing.T) {
	// A defense answers the opponent's turn and then goes, spent or not. It cannot be banked
	// against a turn that queued no attack at all.
	a := duelist(10, 4, 100)
	b := duelist(10, 4, 100)

	// A raises two defenses into a turn with nothing to answer.
	_, a1, b1 := resolve(a, b, PlainCards(Guard, Guard), nil, 1)
	if a1.Shields != 6 {
		t.Fatalf("A ended round 1 holding %d shields, want the 6 two Guards raise", a1.Shields)
	}

	// Round two: A queues nothing, so its own turn expires them before B swings.
	events, _, _ := resolve(a1, b1, nil, PlainCards(Bash, Bash), 2)

	if hasKind(events, KindBlocked) {
		t.Error("a shield ate a blow in round 2 — A's own idle turn should have expired them")
	}
	open, _, _ := resolve(a, b, nil, PlainCards(Bash, Bash), 1)
	if got, want := firstDamage(t, events, SideB).Amount, firstDamage(t, open, SideB).Amount; got != want {
		t.Errorf("a blow into expired defends dealt %d, want the full %d", got, want)
	}
}

func TestClearDefensesClearsEveryDefensiveField(t *testing.T) {
	// ClearDefenses is the one place that has to know about every defensive field, and a new one
	// left out of it would stand forever — the worst failure mode available here, and a silent
	// one. Pin the whole set rather than only the fields this session added.
	//
	// **It cleared two mechanics until 2026-09-16 and clears one now.** The percentage guard was
	// deleted; the shields are what is left, and the test stays because the failure it guards
	// against is a *future* field being added and forgotten.
	var d Duelist
	d = d.raiseShields(3)

	got := ClearDefenses(d)

	if got.Shields != 0 {
		t.Errorf("ClearDefenses left something standing: %+v", got)
	}
}

func TestTheAttackLadderIsThreeFormsByFiveTiers(t *testing.T) {
	// One concept per form per tier, and **the tiers are identical across the forms** — same
	// cost, same damage. **Five rungs, not three** *(2026-08-24)*: the 0 AP and 4 AP ends ship at
	// zero copies and exist only for an essence to walk a card onto, but they are rungs of the same
	// ladder and have to match across the forms exactly as the dealt three do. That is the structural claim MECHANICS.md makes about the deck: a form
	// is which pair you are building, never a better or worse way to build one. It is also the
	// thing that quietly breaks the first time somebody makes a Cleave hit harder than a Skewer.
	//
	// It also catches a concept falling through Cost()'s default arm, which returns a mid-tier
	// price — a mistake that would otherwise hide behind an entirely plausible number.
	const dmg = 10

	attackForms := []Form{FormStab, FormSlash, FormCrush}
	tiers := map[Form]map[int]ConceptID{}

	for _, a := range PlayerConcepts() {
		fam := ConceptOf(a).Form
		if Plain(a).Category() != CategoryAttack || fam == FormNone {
			continue
		}
		if ConceptOf(a).Cost < 0 || ConceptOf(a).Cost > 4 {
			t.Errorf("%v costs %d, outside the 0-4 tiers the ladder is built on", a, ConceptOf(a).Cost)
		}
		if tiers[fam] == nil {
			tiers[fam] = map[int]ConceptID{}
		}
		if other, taken := tiers[fam][ConceptOf(a).Cost]; taken {
			t.Errorf("%v and %v are both %v at %d AP — the ladder holds one concept per rung", other, a, fam, ConceptOf(a).Cost)
		}
		tiers[fam][ConceptOf(a).Cost] = a
	}

	for _, fam := range attackForms {
		if len(tiers[fam]) != 5 {
			t.Errorf("%v has %d of 5 tiers filled", fam, len(tiers[fam]))
		}
	}

	// Every form's rung deals what Stab's rung deals.
	for tier := 0; tier <= 4; tier++ {
		want, ok := tiers[FormStab][tier]
		if !ok {
			continue
		}
		for _, fam := range attackForms[1:] {
			got, ok := tiers[fam][tier]
			if !ok {
				continue
			}
			if Plain(got).Damage(dmg) != Plain(want).Damage(dmg) {
				t.Errorf("at %d AP, %v deals %d and %v deals %d — the forms must ladder identically",
					tier, got, Plain(got).Damage(dmg), want, Plain(want).Damage(dmg))
			}
		}
	}
}

func TestDefensesDealNothingAndAttacksDoNot(t *testing.T) {
	// The two categories are what a card *is*, so each has to hold on its own side of the line. A
	// defense that dealt damage would be an attack wearing the wrong verb in the feed.
	const dmg = 10
	for _, a := range PlayerConcepts() {
		switch Plain(a).Category() {
		case CategoryDefend:
			if d := Plain(a).Damage(dmg); d != 0 {
				t.Errorf("%v is a defense and deals %d damage", a, d)
			}
		case CategoryAttack:
			if d := Plain(a).Damage(dmg); d <= 0 {
				t.Errorf("%v is an attack and deals %d damage", a, d)
			}
		}
	}
}

func TestEveryPlanIsInThePlanFormAndViceVersa(t *testing.T) {
	// Category has two values and is derivable from the form, which is exactly why the two must
	// not be able to disagree: the card face says the form and the feed says the category, and a
	// card whose corner and verb told different stories would be the screen contradicting itself.
	for _, a := range PlayerConcepts() {
		if (ConceptOf(a).Form == FormDefend) != (Plain(a).Category() == CategoryDefend) {
			t.Errorf("%v is form %v but category %v", a, ConceptOf(a).Form, Plain(a).Category())
		}
	}
}

func TestEveryPlayerCardHasAForm(t *testing.T) {
	// **FormNone is a real answer, not a fallthrough**, and after 2026-08-16 it belongs entirely
	// to the enemies: a form is the player's deck axis, the thing a pair is counted on and the
	// mark in the card's corner. A player's card landing there would draw with a blank corner and
	// be excluded from the deck panel's sort.
	//
	// It used to name the two shared enemy concepts as the permitted exceptions. There is no shared
	// enemy deck now — every enemy carries its own cards, all of them formless — so the rule is
	// simply that the player's side is fully covered.
	for _, a := range PlayerConcepts() {
		if ConceptOf(a).Form == FormNone {
			t.Errorf("%v has no form; only an enemy's cards may", ConceptOf(a).Label)
		}
	}
}

func TestEveryPlayerConceptIsFoundByItsLabel(t *testing.T) {
	// duelist_cards.json names cards by label, and the deck the screen builds looks each one up in
	// the registry the rules built from that same file. A label that does not round-trip is four
	// cards missing from the deck.
	for _, a := range PlayerConcepts() {
		label := ConceptOf(a).Label
		got, ok := ConceptByKey(label)
		if !ok {
			t.Errorf("no concept registered under %q", label)
			continue
		}
		if got != a {
			t.Errorf("ConceptByKey(%q) = %v, want %v", label, got, a)
		}
	}

	if _, ok := ConceptByKey("Cleeve"); ok {
		t.Error("the registry answered to a typo — it must report failure so a bad deck fails loudly")
	}
}

func TestParseFormRoundTripsEveryForm(t *testing.T) {
	// The deck lists declare a form and `CheckCostTiers` holds it against the rules, so the join
	// has to work in both directions exactly as ParseAction's does.
	for _, f := range Forms() {
		got, ok := ParseForm(f.String())
		if !ok {
			t.Errorf("ParseForm(%q) failed for a form the rules define", f.String())
			continue
		}
		if got != f {
			t.Errorf("ParseForm(%q) = %v, want %v", f.String(), got, f)
		}
	}

	if _, ok := ParseForm("stabby"); ok {
		t.Error("ParseForm accepted a typo")
	}
	// And FormNone is not in Forms(), so it cannot be parsed into by accident — but it still
	// names itself, because a deck list writes "none" for the opponent's cards.
	if got := FormNone.String(); got != "none" {
		t.Errorf("FormNone is named %q, want %q", got, "none")
	}
}

// TestAEssencesBoundsHold. The two per-card modifiers are the only way a card's numbers move, and
// both ends of each are a rule rather than a convenience.
func TestAEssencesBoundsHold(t *testing.T) {
	// Cost floors at zero and does not go negative. A free card is bounded by the count cap
	// instead of the budget, which is the trade that was taken deliberately.
	cheap := Card{Concept: Bash, CostDelta: -99}
	if got := cheap.Cost(); got != 0 {
		t.Errorf("a card cheapened past zero costs %d, want 0", got)
	}

	// Nothing stops a blow outright, however many essences are stacked on a Guard.
	wall := Card{Concept: Guard, AmountPct: 10000}
	if got := wall.Amount(); got >= 100 {
		t.Errorf("a defense scaled up reduces by %d%%, and nothing may reach 100", got)
	}

	// An amount cannot be scaled away to nothing: a reward that left a card doing zero would be
	// a punishment wearing a gift's clothes.
	crushed := Card{Concept: Block, AmountPct: 1}
	if got := crushed.Amount(); got < 1 {
		t.Errorf("a scaled-down card banks %d", got)
	}

	// The zero value is unmodified, which is what keeps every existing Card literal working.
	plain := Card{Concept: Block}
	if plain.Amount() != ConceptOf(Block).Amount {
		t.Errorf("an unmodified card reports %d against the concept's %d",
			plain.Amount(), ConceptOf(Block).Amount)
	}
}

// TestTheLadderWalksItsOwnForm. Promote and demote are derived from what duelist_cards.json
// declares rather than from a table beside it, so this pins that the derivation finds the right
// neighbor and stops at both ends.
func TestTheLadderWalksItsOwnForm(t *testing.T) {
	up, ok := Neighbor(Jab, 1)
	if !ok {
		t.Fatal("Jab cannot be promoted, and it is a middle rung of its ladder")
	}
	if ConceptOf(up).Form != ConceptOf(Jab).Form {
		t.Errorf("promoting a %v produced a %v", ConceptOf(Jab).Form, ConceptOf(up).Form)
	}
	if ConceptOf(up).Tier() != ConceptOf(Jab).Tier()+1 {
		t.Errorf("promoting moved from tier %d to %d", ConceptOf(Jab).Tier(), ConceptOf(up).Tier())
	}

	if down, ok := Neighbor(up, -1); !ok || down != Jab {
		t.Errorf("demoting the promotion gave %v, want Jab", down)
	}

	// A Jab now demotes, because the ladder grew an end below it. That is the whole point of the
	// zero-copy rungs: the essence reaches a card it used to be refused on.
	if down, ok := Neighbor(Jab, -1); !ok || down != Poke {
		t.Errorf("demoting a Jab gave %v, want Poke", down)
	}

	// Both ends still stop, one rung further out than they used to. A card at the top of its form
	// cannot be promoted, and the screen asks before it offers so the player is never shown an essence
	// that would do nothing.
	if _, ok := Neighbor(Poke, -1); ok {
		t.Error("the bottom of a ladder was demoted")
	}
	if _, ok := Neighbor(Impale, 1); ok {
		t.Error("the top of a ladder was promoted")
	}

	// The defenses are a ladder too, and Grow and Shrink walk it exactly as they walk an attack
	// form. Block sits in the middle of Flinch / Brace / Block / Guard.
	if up, ok := Neighbor(Block, 1); !ok || up != Guard {
		t.Errorf("promoting a Block gave %v, want Guard", up)
	}
	if down, ok := Neighbor(Brace, -1); !ok || down != Flinch {
		t.Errorf("demoting a Brace gave %v, want Flinch", down)
	}
	if _, ok := Neighbor(Flinch, -1); ok {
		t.Error("the bottom of the defend ladder was demoted")
	}
	if _, ok := Neighbor(Guard, 1); ok {
		t.Error("the top of the defend ladder was promoted")
	}

	// The two ladders never meet: a defense promoted stays a defense, and an enemy card has no
	// form and therefore no ladder at all.
	if up, _ := Neighbor(Block, 1); ConceptOf(up).Verb != VerbShield {
		t.Error("promoting a defense produced an attack")
	}
}

// TestTheWrappingLadderJoinsItsTwoEnds. An essence is something the player aims, so the ends of a
// ladder are joined rather than refusing the pick — and the plain Neighbor, which a relic reads,
// still stops.
func TestTheWrappingLadderJoinsItsTwoEnds(t *testing.T) {
	for _, id := range []ConceptID{Poke, Flinch} {
		rungs := Ladder(id)
		if len(rungs) < 2 {
			t.Fatalf("%v stands on a ladder of %d", ConceptOf(id).Label, len(rungs))
		}
		bottom, top := rungs[0], rungs[len(rungs)-1]

		if got, ok := NeighborWrapping(bottom, -1); !ok || got != top {
			t.Errorf("demoting %v gave %v, want the top rung %v",
				ConceptOf(bottom).Label, ConceptOf(got).Label, ConceptOf(top).Label)
		}
		if got, ok := NeighborWrapping(top, 1); !ok || got != bottom {
			t.Errorf("promoting %v gave %v, want the bottom rung %v",
				ConceptOf(top).Label, ConceptOf(got).Label, ConceptOf(bottom).Label)
		}

		// Every rung in between steps to the next one along, so the wrap is the two ends and
		// nothing else. Read off the ladder rather than off Neighbor, because a test elsewhere in
		// this package may have registered a concept sharing a rung.
		for i := 0; i < len(rungs)-1; i++ {
			if got, ok := NeighborWrapping(rungs[i], 1); !ok || got != rungs[i+1] {
				t.Errorf("promoting %v gave %v, want the rung above it %v",
					ConceptOf(rungs[i]).Label, ConceptOf(got).Label, ConceptOf(rungs[i+1]).Label)
			}
		}

		// A ladder and its rungs never leave the form or the verb they belong to.
		for _, r := range rungs {
			if ConceptOf(r).Form != ConceptOf(id).Form || ConceptOf(r).Verb != ConceptOf(id).Verb {
				t.Errorf("%v stands on %v's ladder", ConceptOf(r).Label, ConceptOf(id).Label)
			}
		}
	}

	// The relic's door is unchanged: a card with no rung below it is left where it is, so a relic
	// that weakens a hand can never hand it the top rung instead.
	if _, ok := Neighbor(Poke, -1); ok {
		t.Error("the plain Neighbor demoted the bottom of a ladder")
	}

	// A card with no form has no ladder, so there is nothing to wrap it round.
	if _, ok := NeighborWrapping(NoConcept, 1); ok {
		t.Error("a card with no form was promoted")
	}
}
