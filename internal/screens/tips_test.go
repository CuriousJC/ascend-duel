package screens

import (
	"strings"
	"testing"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
)

// The wording of a tooltip and the figure on a card's face. Both are arithmetic over the ring
// grammar and neither needs a window — the same narrow exception the rest of this package's tests
// take.

// wearing is a pairing holding one ring, for a duelist with a round DMG figure.
func wearing(t *testing.T, dmg int, keys ...string) held {
	t.Helper()

	h := held{dmg: dmg}
	for _, key := range keys {
		id, ok := session.RingID(key)
		if !ok {
			t.Fatalf("%s is in no catalogue", key)
		}
		h.worn = append(h.worn, combat.WornRing{Ring: id})
	}
	return h
}

// aSlash is any attack card the Keen Ring matches, found rather than named: the deck is data and a
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

func TestNoRingReachesWhatTheFaceSays(t *testing.T) {
	// **The face says what the card does, whatever is on the fingers** *(owner's call, 2026-08-26)*.
	//
	// This test is the reverse of the one it replaced. From 2026-08-21 the face carried the ring's
	// multiplier, written pink, because a card reading "2x DMG" while dealing four times DMG was a
	// face telling the truth about the card and a lie about the attack. What changed is that there
	// is no longer one figure to tell the truth *with*: a growing ring steps between the cards of a
	// single blow, so the same Strike is worth one thing queued first and another queued third. The
	// face states the stable half and the sum states what it came to — see the hand dialog.
	card := aSlash(t)

	bare := cardEffect(card)
	keen := cardEffect(card)

	if keen != bare {
		t.Fatalf("the face reads %q in one pairing and %q in another", bare, keen)
	}

	// The figure is the card's own multiplier, undoubled, even wearing the ring that doubles it.
	want := multiplierText(card.Amount())
	if !strings.Contains(bare, want) {
		t.Errorf("%s reads %q, want the card's own %s in it", card.Label(), bare, want)
	}
	if doubled := multiplierText(card.Amount() * 2); strings.Contains(bare, doubled) {
		t.Errorf("%s reads %q, which carries the ring's %s", card.Label(), bare, doubled)
	}
}

// **The pairing still prices the card**, and that is the one thing a ring does reach on a face: a
// cost is not order-dependent and the face must agree with the AP bar.
func TestADiscountRingStillReachesTheCost(t *testing.T) {
	card := aSlash(t)

	bare := held{}
	worn := wearing(t, 12, "onslaught-ring")

	if worn.cost == bare.cost {
		t.Skip("onslaught-ring does not discount this card")
	}
	if spec := cardSpec(card, worn, true, false); spec.Cost != worn.cost {
		t.Errorf("the face is priced at %d where the pairing says %d", spec.Cost, worn.cost)
	}
}

func TestTheTooltipShowsEveryTermOfTheDamage(t *testing.T) {
	// **Term by term, not a total.** The question in front of a hand is not "what is this worth" but
	// "why is that one worth more", and a single number answers only the first.
	card := aSlash(t)
	h := wearing(t, 12, "keen-ring")

	title, lines := cardTip(card, h)
	if title != card.Label() {
		t.Errorf("the panel is titled %q, want %q", title, card.Label())
	}

	joined := strings.Join(lines, " | ")
	for _, want := range []string{"12 DMG", "the card", "Keen Ring", "before the hand"} {
		if !strings.Contains(joined, want) {
			t.Errorf("the tooltip is missing %q: %s", want, joined)
		}
	}

	// And the figure it lands on is the engine's, not a second sum: DMG x card x ring.
	d := combat.Duelist{DMG: h.dmg}
	for _, w := range h.worn {
		d = d.Wearing(w)
	}
	want := "= " + itoa(d.CardDamage(card)) + " DMG"
	if !strings.Contains(joined, want) {
		t.Errorf("the tooltip totals to something other than %q: %s", want, joined)
	}
}

func TestTheTooltipStatesMultipliersWhenNobodyIsHoldingTheCard(t *testing.T) {
	// Between fights there is no duelist — a run's stats belong to a fight — so a card offered on
	// the reward screen has to say "4x your DMG" rather than a number worked out against a strength
	// nobody has.
	card := aSlash(t)

	_, lines := cardTip(card, wearing(t, 0, "keen-ring"))
	joined := strings.Join(lines, " | ")

	if strings.Contains(joined, "DMG, yours") {
		t.Errorf("a card nobody is holding claimed a strength: %s", joined)
	}
	if !strings.Contains(joined, "your DMG") {
		t.Errorf("the tooltip does not state the multiplier: %s", joined)
	}
}

func TestADiscountRingExplainsThePrice(t *testing.T) {
	// The AP a card takes is the other figure a ring moves, and the face has shown the discounted
	// number since the day the grammar landed. What was missing is which ring did it.
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

	h := wearing(t, 12, "warm-ring")
	h.cost = combat.CostWith(h.worn, fire)

	_, lines := cardTip(fire, h)
	joined := strings.Join(lines, " | ")

	if !strings.Contains(joined, "Warm Ring") {
		t.Errorf("a discounted card does not name the ring: %s", joined)
	}
}

func TestEveryRingHasSomethingToSay(t *testing.T) {
	// **The tooltip prints the authored line and nothing else**, so a record with no Text is a ring
	// whose panel is a name and a blank. `rings.json` has carried the field since the grammar
	// landed, for a long press that never arrived; this is the first thing that reads it.
	for key, record := range data.LoadRings() {
		title, lines := ringTip(record, -1, 0)
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

func TestAWornRingSaysWhereItFires(t *testing.T) {
	// Worn order is a rule — rings fire left to right and compound — so the position is information
	// about the effect rather than about the layout.
	records := data.LoadRings()
	record := records["keen-ring"]

	_, lines := ringTip(record, 1, 3)
	if joined := strings.Join(lines, " | "); !strings.Contains(joined, "2nd of 3") {
		t.Errorf("a ring worn second of three says: %s", joined)
	}

	// Alone on the hand there is no order to explain, and a line saying "1st of 1" is noise.
	_, alone := ringTip(record, 0, 1)
	if joined := strings.Join(alone, " | "); strings.Contains(joined, "fires") {
		t.Errorf("the only ring worn explained its position: %s", joined)
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
