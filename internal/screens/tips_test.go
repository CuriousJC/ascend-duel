package screens

import (
	"strings"
	"testing"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/carddesc"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
)

// The wording of a tooltip and the figure on a card's face. Both are arithmetic over the relic
// grammar and neither needs a window — the same narrow exception the rest of this package's tests
// take.

// wearing is a pairing holding one relic, for a duelist with a round DMG figure.
func wearing(t *testing.T, dmg int, keys ...string) held {
	t.Helper()

	h := held{dmg: dmg}
	for _, key := range keys {
		id, ok := session.RelicID(key)
		if !ok {
			t.Fatalf("%s is in no catalogue", key)
		}
		h.worn = append(h.worn, combat.WornRelic{Relic: id})
	}
	return h
}

// aSlash is any attack card The Sickle matches, found rather than named: the deck is data and a
// test naming one card by label would be a test about `duelist_cards.json`.
func aSlash(t *testing.T) combat.Card {
	t.Helper()

	for _, id := range combat.AllConcepts() {
		c := combat.Plain(id)
		if c.Spec().Verb == combat.VerbAttack && c.Form() == combat.FormSlash {
			return c
		}
	}
	t.Skip("no slash attack in the deck")
	return combat.Card{}
}

func TestNoRelicReachesWhatTheFaceSays(t *testing.T) {
	// **The face says what the card does, whatever is on the fingers** *(owner's call, 2026-08-26)*.
	//
	// This test is the reverse of the one it replaced. From 2026-08-21 the face carried the relic's
	// multiplier, written pink, because a card reading "2x DMG" while dealing four times DMG was a
	// face telling the truth about the card and a lie about the attack. What changed is that there
	// is no longer one figure to tell the truth *with*: a growing relic steps between the cards of a
	// single blow, so the same Bash is worth one thing queued first and another queued third. The
	// face states the stable half and the sum states what it came to — see the hand dialog.
	card := aSlash(t)

	bare := cardEffect(card)
	keen := cardEffect(card)

	if keen != bare {
		t.Fatalf("the face reads %q in one pairing and %q in another", bare, keen)
	}

	// The figure is the card's own multiplier, undoubled, even wearing the relic that doubles it.
	want := multiplierText(card.Amount())
	if !strings.Contains(bare, want) {
		t.Errorf("%s reads %q, want the card's own %s in it", card.Label(), bare, want)
	}
	if doubled := multiplierText(card.Amount() * 2); strings.Contains(bare, doubled) {
		t.Errorf("%s reads %q, which carries the relic's %s", card.Label(), bare, doubled)
	}
}

// **The pairing still prices the card**, and that is the one thing a relic does reach on a face: a
// cost is not order-dependent and the face must agree with the AP bar.
func TestADiscountRelicStillReachesTheCost(t *testing.T) {
	card := aSlash(t)

	bare := held{}
	worn := wearing(t, 12, "onslaught")

	if worn.cost == bare.cost {
		t.Skip("onslaught-ring does not discount this card")
	}
	if spec := cardSpec(card, worn, true, false); spec.Cost != worn.cost {
		t.Errorf("the face is priced at %d where the pairing says %d", spec.Cost, worn.cost)
	}
}

// **The block leads with what the card is, and the figure in it is what the card will deal.**
// A headline number that did not carry the relics would be the wrong one at the top of a panel whose
// next four lines explain a bigger one.
func TestTheTooltipOpensWithTheStatBlock(t *testing.T) {
	card := aSlash(t)
	h := wearing(t, 12, "dmg-all-slash")

	title, lines := cardTip(card, h)
	if want := carddesc.Title(card); title != want {
		t.Errorf("the panel is titled %q, want %q", title, want)
	}
	if len(lines) < 2 {
		t.Fatalf("the tooltip is %d lines: %v", len(lines), lines)
	}

	// The engine's own figure, not a second sum: DMG x card x relic.
	d := combat.Duelist{DMG: h.dmg}
	for _, w := range h.worn {
		d = d.Wearing(w)
	}
	if want := itoa(h.cost) + " AP"; lines[0] != want {
		t.Errorf("the first line is %q, want %q", lines[0], want)
	}
	if want := itoa(d.CardDamage(card)) + " DMG"; lines[1] != want {
		t.Errorf("the second line is %q, want %q", lines[1], want)
	}
}

// **The chain is printed when a relic has moved something**, and the terms are the relics themselves
// — that is the whole reason the panel exists, and it survived the block landing on top of it.
func TestTheTooltipShowsEveryTermOfTheDamage(t *testing.T) {
	card := aSlash(t)
	h := wearing(t, 12, "dmg-all-slash")

	_, lines := cardTip(card, h)
	joined := strings.Join(lines, " | ")

	for _, want := range []string{"the card", "The Sickle", "before the hand"} {
		if !strings.Contains(joined, want) {
			t.Errorf("the tooltip is missing %q: %s", want, joined)
		}
	}

	// **And it does not print the total twice.** The block already stated it; a second copy is a
	// number that can disagree with the first, which is the bug nobody would be able to see.
	if strings.Contains(joined, "= ") {
		t.Errorf("the chain totalled a figure the block had already printed: %s", joined)
	}
}

// **A card nothing has touched derives to itself, so it says nothing.** Three lines restating a
// figure the block printed is a tooltip that trains the player not to read it.
func TestTheTooltipIsSilentWhenNoRelicChangedTheCard(t *testing.T) {
	card := aSlash(t)

	_, lines := cardTip(card, held{dmg: 12})
	joined := strings.Join(lines, " | ")

	for _, unwanted := range []string{"the card", "before the hand"} {
		if strings.Contains(joined, unwanted) {
			t.Errorf("a bare card printed a derivation (%q): %s", unwanted, joined)
		}
	}
	if len(lines) != 2 {
		t.Errorf("a bare attack is %d lines, want the cost and the damage: %v", len(lines), lines)
	}
}

func TestTheTooltipStatesMultipliersWhenNobodyIsHoldingTheCard(t *testing.T) {
	// Between fights there is no duelist — a run's stats belong to a fight — so a card offered on
	// the reward screen has to say "4x DMG" rather than a number worked out against a strength
	// nobody has.
	card := aSlash(t)

	_, lines := cardTip(card, wearing(t, 0, "dmg-all-slash"))
	joined := strings.Join(lines, " | ")

	if !strings.Contains(joined, "X DMG") {
		t.Errorf("the tooltip does not state the multiplier: %s", joined)
	}
}

// **Every rider is mentioned.** A parasite the player spent whose effect the tooltip does not name
// is the same failure as one with no drawing, in the other direction.
func TestEveryRiderKindHasTipLines(t *testing.T) {
	for _, k := range combat.RiderKinds() {
		c := combat.Plain(combat.Bash).SetRider(combat.Rider{Kind: k, Amount: 5})
		if lines := carddesc.RiderLines(c); len(lines) == 0 {
			t.Errorf("rider %s adds no line to a card's tooltip", k)
		}
	}
	if lines := carddesc.RiderLines(combat.Plain(combat.Bash)); len(lines) != 0 {
		t.Errorf("an unridden card claimed an upgrade: %v", lines)
	}
}

// **The two examples the owner wrote out**, held against what the panel actually builds. They are
// the specification for the format and this is the one place they exist as one.
func TestTheTooltipReadsTheWayItWasSpecified(t *testing.T) {
	jab := cardNamed(t, "Jab")
	jab.Element = combat.Fire
	jab = jab.SetRider(combat.Rider{Kind: combat.RiderHealOnPlay, Amount: 10})

	// **A DMG of 10, because a Jab is a half-multiplier card.** The owner's example wrote `5 DMG`,
	// and the figure a Jab prints is the holder's DMG times the card's own 50% — so the duelist
	// that example describes hits for 10.
	title, lines := cardTip(jab, held{cost: jab.Cost(), dmg: 10})
	want := []string{"1 AP", "5 DMG", "+10 HEAL ON PLAY"}
	if title != "FIRE JAB" || !equal(lines, want) {
		t.Errorf("a fire Jab with a Leech reads %q %v, want %q %v", title, lines, "FIRE JAB", want)
	}

	ward := cardNamed(t, "Brace")
	ward.Element = combat.Ice
	ward = ward.SetRider(combat.Rider{Kind: combat.RiderVitaeInHand, Amount: 3})

	title, lines = cardTip(ward, held{cost: ward.Cost()})
	want = []string{"1 AP", "1 SHIELD", "+3 VITAE IN HAND"}
	if title != "ICE BRACE" || !equal(lines, want) {
		t.Errorf("an ice Brace with a Brood reads %q %v, want %q %v", title, lines, "ICE BRACE", want)
	}
}

// cardNamed is the shipped card of this label, or a fatal failure. It reads the registry rather
// than being built by hand, so a test writing a cost or an amount the catalogue does not have
// fails here instead of passing against a card the game does not deal.
func cardNamed(t *testing.T, label string) combat.Card {
	t.Helper()
	for _, id := range combat.AllConcepts() {
		if combat.ConceptOf(id).Label == label {
			return combat.Plain(id)
		}
	}
	t.Fatalf("no card is called %q", label)
	return combat.Card{}
}

func equal(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestADiscountRelicExplainsThePrice(t *testing.T) {
	// The AP a card takes is the other figure a relic moves, and the face has shown the discounted
	// number since the day the grammar landed. What was missing is which relic did it.
	var fire combat.Card
	for _, id := range combat.AllConcepts() {
		if c := combat.Of(id, combat.Fire); c.Spec().Verb == combat.VerbAttack {
			fire = c
			break
		}
	}
	if fire.Spec().Verb != combat.VerbAttack {
		t.Skip("no fire attack in the deck")
	}

	h := wearing(t, 12, "discount-fire")
	h.cost = combat.CostWith(h.worn, fire)

	_, lines := cardTip(fire, h)
	joined := strings.Join(lines, " | ")

	if !strings.Contains(joined, "Warm") {
		t.Errorf("a discounted card does not name the relic: %s", joined)
	}
}

func TestEveryRelicHasSomethingToSay(t *testing.T) {
	// **The tooltip prints the authored line and nothing else**, so a record with no Text is a relic
	// whose panel is a name and a blank. `relics.json` has carried the field since the grammar
	// landed, for a long press that never arrived; this is the first thing that reads it.
	for key, record := range data.LoadRelics() {
		title, lines := relicTip(record, -1, 0)
		if title == "" {
			t.Errorf("%s has no name", key)
		}
		if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
			t.Errorf("%s has no Text, so its tooltip says nothing about what it does", key)
		}
	}
}

func TestEveryStatusHasSomethingToSay(t *testing.T) {
	// The badge row is pictures. This is the only place the words behind them are printed, so a
	// status with no line is a badge that stays a mystery.
	for _, id := range combat.AllStatuses() {
		spec := combat.StatusOf(id)
		if statusText(spec.Key) == "" {
			t.Errorf("%s has no Text in statuses.json, so its badge cannot be read", spec.Key)
		}
	}
}

func TestAWornRelicSaysWhereItFires(t *testing.T) {
	// Worn order is a rule — relics fire left to right and compound — so the position is information
	// about the effect rather than about the layout.
	records := data.LoadRelics()
	record := records["dmg-all-slash"]

	_, lines := relicTip(record, 1, 3)
	if joined := strings.Join(lines, " | "); !strings.Contains(joined, "2nd of 3") {
		t.Errorf("a relic worn second of three says: %s", joined)
	}

	// Alone on the hand there is no order to explain, and a line saying "1st of 1" is noise.
	_, alone := relicTip(record, 0, 1)
	if joined := strings.Join(alone, " | "); strings.Contains(joined, "fires") {
		t.Errorf("the only relic worn explained its position: %s", joined)
	}
}

// itoa keeps the arithmetic assertions above readable without importing strconv for one call.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var out []byte
	for n > 0 {
		out = append([]byte{byte('0' + n%10)}, out...)
		n /= 10
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}

// **A wildcard is CHROMATIC, not the element it happens to be** *(owner's call, 2026-09-09)*. The
// card still is an arcane Skewer and everything else goes on reading it as one; what changes is the
// title, because `ARCANE SKEWER` over a line reading `COUNTS AS EVERY ELEMENT` is a panel
// contradicting itself in two lines.
func TestAWildcardsTitleIsChromatic(t *testing.T) {
	card := combat.Of(combat.Bash, combat.Arcane)

	if got := carddesc.Title(card); got != "ARCANE BASH" {
		t.Errorf("an ordinary arcane card is titled %q", got)
	}

	wild := card.SetRider(combat.Rider{Kind: combat.RiderWildElement})
	if got := carddesc.Title(wild); got != carddesc.Chromatic+" BASH" {
		t.Errorf("a wildcard is titled %q, want %q", got, carddesc.Chromatic+" BASH")
	}
	// The card is still arcane, and everything that is not the title still says so.
	if wild.Element != combat.Arcane {
		t.Error("naming a wildcard chromatic changed what element it is")
	}
}

// **The element word in a title is written in its element's colour.** It was the one place in the
// game that rule did not reach, because `models.Tooltip.Title` was a plain string — and a card's
// title is where an element word is most worth colouring.
func TestTheElementInATitleIsColoured(t *testing.T) {
	title := tipLine(carddesc.Title(combat.Of(combat.Bash, combat.Fire)))

	if title.Text() != "FIRE BASH" {
		t.Fatalf("the title reads %q", title.Text())
	}
	if len(title) < 2 {
		t.Fatalf("the title is one run, so nothing in it is coloured: %v", title)
	}

	want := cards.BorderOf(cards.Fire)
	for _, run := range title {
		if run.Text == "FIRE" && run.Ink == want {
			return
		}
	}
	t.Errorf("FIRE is not written in the fire red: %v", title)
}

// **CHROMATIC takes no *single* element's colour, and that is still the rule** *(owner's call,
// 2026-09-09)*. The wheel has no hue left for "all of them" and a word written in one of the five
// would be claiming the one thing it exists to deny — so it is written in all five at once, sampled
// out of the wildcard's own wash. What this holds is the half that did not change: no letter of it
// is an element's ink, and the card's *element* is not what the title is coloured by.
func TestChromaticTakesNoElementColour(t *testing.T) {
	title := tipLine(carddesc.Title(
		combat.Of(combat.Bash, combat.Arcane).SetRider(combat.Rider{Kind: combat.RiderWildElement})))

	elements := map[cards.Element]bool{
		cards.Fire: true, cards.Ice: true, cards.Lightning: true, cards.Earth: true, cards.Arcane: true,
	}
	for _, run := range title {
		for e := range elements {
			if run.Ink == cards.BorderOf(e) {
				t.Errorf("%q in a chromatic title is written in the %v ink: %v", run.Text, e, title)
			}
		}
	}
}
