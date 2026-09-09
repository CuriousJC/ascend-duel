package screens

// **What a tooltip says.** The panel itself is a widget — `models.Tooltip` and
// `systems.DrawTooltip` — and knows nothing but a title and a list of lines. This is where the
// lines come from.
//
// It is a separate file from prose.go because the two answer different questions. That one narrates
// a round that has happened, in sentences; this one explains a thing the cursor is resting on, in
// terms. A tooltip is read while deciding, so it is arithmetic and short phrases rather than prose.
//
// **The rule every line here follows: say where a number came from.** A card's face states what an
// attack is worth; the tooltip states why. That is the whole reason it exists — a slash reading 4x
// with no explanation is a better lie than one reading 2x, because it is believable.
//
// **A card's tooltip opens with a stat block rather than with the derivation** *(owner's call,
// 2026-09-09)*, and the derivation is printed underneath it only when a ring or a worm has actually
// moved something. See cardTip. The block's wording lives in `internal/carddesc`, which is
// windowless, so the review sheets print the same strings the game does.
//
// **Nothing here recomputes a rule.** The multipliers come off `combat.RingContributionsAt`, the
// same walk `Duelist.CardDamage` compounds, and the costs off the same ring moment the AP bar reads.
// A tooltip that did its own arithmetic would be a second implementation of the engine, printed in
// a box, and it would be wrong on exactly the days it mattered.

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/carddesc"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
)

// tipDwell is how long the cursor has to rest before a panel appears.
//
// **A proportion of the game's one speed**, like everything else timed in this game — see clock.go,
// so the game-speed setting will move this with everything else when it exists.
//
// **A beat and a half, about six tenths of a second** *(owner's call, 2026-08-21)*. It was half a
// beat and fired the moment the cursor touched anything, which does not read as asking a question:
// a panel that appears before you have decided to want it is a flicker following the mouse around.
// The point of a dwell is that resting is deliberate where crossing is not.
var tipDwell = beat(3, 2)

// cardTip explains one card: what it is, what it costs, what it is worth, and what its upgrade
// adds — then, only when something has moved one of those figures, where that came from.
//
// **The stat block leads and the arithmetic follows** *(owner's call, 2026-09-09)*. It used to be
// arithmetic and nothing else: `5 DMG, yours` / `1x the card` / `= 5 DMG`, which is three lines
// deriving a number the player wanted to be told. The derivation is still the reason this panel
// exists — a slash reading 4x with no explanation is a better lie than one reading 2x — but it is
// the answer to "why is that one worth more", and that is the *second* question. The first is
// "what is this", and the block is that.
//
// **The chain is printed only when there is a chain.** A ringless Jab derives to itself, so a card
// nothing has touched says four lines and stops. Put a ring on and every term comes back.
//
// **The block itself is `internal/carddesc`**, which is windowless, so `tools/upgradesheet` prints
// the same strings this panel does rather than a snapshot of them.
func cardTip(c actionCard, h held) (string, []string) {
	lines := carddesc.Lines(c, h.cost, h.dmg, ringScale(c, h))
	lines = append(lines, damageChainLines(c, h)...)
	if c.AmountPct != 0 {
		lines = append(lines, "a worm changed this card")
	}
	lines = append(lines, costTipLines(c, h)...)

	return carddesc.Title(c), lines
}

// damageChainLines is the damage arithmetic: the card's own multiplier, every ring that matches, and
// the result — **or nothing at all when no ring matches.**
//
// **Silence is the common case and is the point.** Every card in a ringless deck derives to the
// figure the block above already printed, and three lines saying so is a tooltip that trains the
// player not to read it. What earns the space is a ring having changed something.
//
// **It is only ever asked of an attack.** A shield and a defence have no ring moment on their
// figure, so a chain under one would be a heading with nothing beneath it.
func damageChainLines(c actionCard, h held) []string {
	if c.Spec().Verb != combat.VerbAttack {
		return nil
	}
	contributions := combat.RingContributionsAt(h.worn, combat.MomentCardDamage, c)
	if len(contributions) == 0 {
		return nil
	}

	lines := []string{multiplierText(c.Amount()) + " the card"}
	for _, contribution := range contributions {
		lines = append(lines, multiplierText(contribution.Effect.Amount)+" "+
			combat.RingOf(contribution.Ring).Name)
	}

	// **The chain has no total, because the block already printed it** — see cardTip, where the
	// ring scale is handed to `carddesc` precisely so the headline figure is the one the card will
	// deal. A `= 24 DMG` under these terms would be the same number twice, and the moment the two
	// disagreed one of them would be the bug nobody could see.
	//
	// **The caveat is said whenever the chain is.** The figure at the top of the panel is a card's
	// own worth and says nothing about a hand, which is the largest multiplier in the game.
	return append(lines, "before the hand multiplies it")
}

// ringScale is every ring that reaches this card's damage, compounded, as a percentage.
//
// **It walks `combat.RingContributionsAt`, which is the same walk `Duelist.CardDamage` compounds.**
// A tooltip that did its own arithmetic would be a second implementation of the engine, printed in
// a box, and it would be wrong on exactly the days it mattered.
func ringScale(c actionCard, h held) int {
	scale := 100
	for _, contribution := range combat.RingContributionsAt(h.worn, combat.MomentCardDamage, c) {
		scale = scale * contribution.Effect.Amount / 100
	}
	return scale
}

// costTipLines explains a price a ring has moved. **Only when one has** — a card costing what it
// says needs no line saying so, and a tooltip that repeats the face is a tooltip nobody reads twice.
func costTipLines(c actionCard, h held) []string {
	var lines []string
	for _, contribution := range combat.RingContributionsAt(h.worn, combat.MomentCardCost, c) {
		lines = append(lines, fmt.Sprintf("%+d AP %s",
			contribution.Effect.Amount, combat.RingOf(contribution.Ring).Name))
	}
	if len(lines) == 0 {
		return nil
	}
	return append(lines, "costs "+strconv.Itoa(h.cost)+" AP to you")
}

// ringTip explains a ring: the authored line from `rings.json`, and where it sits in the firing
// order when it is being worn.
//
// **The authored text rather than a sentence generated from the rules.** The rules would always be
// true and would read like a compiler — "card-damage, form slash, scale 200" — where the line in the
// file is written for a player. The risk is drift, and it is a real one: the file is the only place
// that says what a ring does in words, so a rule changed without its Text is a ring that lies.
func ringTip(record data.RingData, wornAt, wornOf int) (string, []string) {
	// **The authored text, split on its own line breaks.** A newline in `rings.json` is an authored
	// break for the *card face*, and a tooltip draws its own lines one at a time — handing the whole
	// string to one line draws every line of it at the same y, which reads as garbled text rather
	// than as a missing break. Same treatment `parasiteTipLines` gives a parasite.
	lines := strings.Split(record.Text, "\n")

	if wornAt >= 0 && wornOf > 1 {
		// **Worn order is a rule** — rings fire left to right and compound — so where one sits is
		// information about what it does, not about where it is drawn.
		lines = append(lines, fmt.Sprintf("fires %s of %d, left to right",
			ordinal(wornAt+1), wornOf))
	}
	return record.Name, lines
}

// shopRingTip is ringTip with the price under it, for a ring on the shelf.
func shopRingTip(record data.RingData) (string, []string) {
	title, lines := ringTip(record, -1, 0)

	if price, ok := session.RingPrice(record.RingRecord); ok {
		lines = append(lines, fmt.Sprintf("%d vitae, sells back for %d",
			price, session.SellValue(record.RingRecord)))
	}
	return title, lines
}

// duelistTip explains one of the two fighters: what they hit for, what is left of them, and every
// status standing on them.
//
// **This is where a badge is read.** The row of pictures along the bottom of the enemy's card is the
// only thing on screen that says a status is running, and nothing anywhere says what one *does* —
// `statuses.json` has carried the sentence since the day statuses became data, with nowhere to print
// it.
func duelistTip(name string, d combat.Duelist) (string, []string) {
	lines := []string{
		strconv.Itoa(d.DMG) + " DMG",
		fmt.Sprintf("%d of %d HP", d.CurrentLife, d.MaxLife),
	}

	for _, id := range combat.AllStatuses() {
		st := d.Statuses[id]
		if !st.Active() {
			continue
		}
		spec := combat.StatusOf(id)
		lines = append(lines, "", spec.Name+" - "+statusRounds(st.Rounds))
		if text := statusText(spec.Key); text != "" {
			lines = append(lines, text)
		}
	}
	return name, lines
}

func statusRounds(n int) string {
	if n == 1 {
		return "1 round left"
	}
	return strconv.Itoa(n) + " rounds left"
}

// statusText is the authored line for a status, out of `statuses.json`.
//
// **Read here rather than carried on `combat.StatusSpec`**, exactly as the badge key is: what a
// status is worth and how long it lasts are rules, and the sentence describing it to a player is
// this layer's business. Same division the ring's art key draws.
func statusText(key string) string { return statusLines[key] }

var statusLines = statusTexts()

func statusTexts() map[string]string {
	out := map[string]string{}
	for _, record := range data.LoadStatuses() {
		out[record.StatusRecord] = record.Text
	}
	return out
}

// **A worm has no tooltip** *(owner's call, 2026-09-05)*. The card's own face says what it does,
// one word to a line, and a hover repeating that sentence beside it was the same words twice.

// ordinal is 1st, 2nd, 3rd — for the five positions a ring can be worn in, and nothing else. Written
// out rather than generalised, because the row is capped at five and a general one would be a rule
// about English nobody here needs.
func ordinal(n int) string {
	switch n {
	case 1:
		return "1st"
	case 2:
		return "2nd"
	case 3:
		return "3rd"
	}
	return strconv.Itoa(n) + "th"
}

// attackWord is "one attack" or "three attacks", written out because a tooltip is a sentence and a
// numeral in the middle of one reads as a stat.
func attackWord(n int) string {
	names := [...]string{"no", "one", "two", "three", "four", "five"}
	if n < 0 || n >= len(names) {
		return strconv.Itoa(n) + " attacks"
	}
	if n == 1 {
		return "one attack"
	}
	return names[n] + " attacks"
}

// roundTimerTip explains the bar under the tower place: what the cells are, and what happens when
// the last one lights.
//
// **The bar has no legend anywhere else**, which is the objection TODO.md already files against
// every other figure written straight onto the table — so it arrives with one rather than joining
// the list. The sentence names the consequence in full: a player who has read this cannot be
// surprised by the death, and a timer that killed without having said so would be the worst kind
// of hidden rule.
func roundTimerTip(spent, limit int) (string, []string) {
	return "The Clock", []string{
		fmt.Sprintf("Round %d of %d.", spent+1, limit),
		"",
		fmt.Sprintf("Every duel lasts %d rounds.", limit),
		"Still standing when the last one",
		"ends, and the tower takes you.",
	}
}
