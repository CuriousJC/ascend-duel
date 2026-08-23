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
// **Nothing here recomputes a rule.** The multipliers come off `combat.RingContributionsAt`, the
// same walk `Duelist.CardDamage` compounds, and the costs off the same ring moment the AP bar reads.
// A tooltip that did its own arithmetic would be a second implementation of the engine, printed in
// a box, and it would be wrong on exactly the days it mattered.

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/curiousjc/ascend-duel/data"
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

// cardTip explains one card: what it will deal, and every step between the holder's DMG and that
// figure.
//
// **The chain is printed term by term rather than as a total**, because the question a player has in
// front of a hand is not "what is this worth" but "why is that one worth more". A single number
// answers the first and hides the second, and the second is what a build is made of.
func cardTip(c actionCard, h held) (string, []string) {
	spec := c.Spec()
	var lines []string

	if spec.Verb == combat.VerbAttack {
		lines = attackTipLines(c, h)
	} else {
		lines = planTipLines(c)
	}

	if c.AmountPct != 0 {
		lines = append(lines, "a worm changed this card")
	}
	lines = append(lines, costTipLines(c, h)...)

	return c.Label(), lines
}

// attackTipLines is the damage arithmetic: strength, the card's own multiplier, every ring that
// matches, and the result.
//
// **With no DMG figure it states the multipliers and stops.** Between fights there is no duelist —
// a run's stats belong to a fight — so the honest answer is "four times your DMG" rather than a
// number worked out against a strength nobody has yet.
func attackTipLines(c actionCard, h held) []string {
	var lines []string

	if h.dmg > 0 {
		lines = append(lines, strconv.Itoa(h.dmg)+" DMG, yours")
	}
	lines = append(lines, multiplierText(c.Amount())+" the card")

	scale := 100
	for _, contribution := range combat.RingContributionsAt(h.worn, combat.MomentCardDamage, c) {
		scale = scale * contribution.Effect.Amount / 100
		lines = append(lines, multiplierText(contribution.Effect.Amount)+" "+
			combat.RingOf(contribution.Ring).Name)
	}

	total := c.Amount() * scale / 100
	if h.dmg > 0 {
		lines = append(lines, "= "+strconv.Itoa(h.dmg*total/100)+" DMG")
	} else {
		lines = append(lines, "= "+multiplierText(total)+" your DMG")
	}

	// **Said on every attack, not only on a big one.** The hand is the largest multiplier in the
	// game and it is decided by what else is selected, so a figure here that did not say it was
	// pre-hand would be the same kind of half-truth the face was telling before today.
	return append(lines, "before the hand multiplies it")
}

// planTipLines says what a plan card's figure buys, in the terms the round uses. **The face states
// the rule and this states the consequence** — "Bank 2 AP" is what the card does, "spend them next
// round" is why anyone would.
func planTipLines(c actionCard) []string {
	amount := c.Amount()

	switch c.Spec().Verb {
	case combat.VerbDefend:
		return []string{
			"takes " + strconv.Itoa(amount) + "% off one blow",
			"the round it is played",
		}
	case combat.VerbBank:
		return []string{
			"+" + strconv.Itoa(amount) + " AP next round",
			"on top of your usual budget",
		}
	case combat.VerbDraw:
		return []string{
			"+" + strconv.Itoa(amount) + " cards next round",
			"a wider hand to build from",
		}
	}
	return nil
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
	lines := []string{record.Text}

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

// wormTip explains an offered alteration: what it does, and that it is permanent.
//
// **"For the rest of the run" is the part the face does not say.** A worm reads as a reward and some
// of them are trades; the card states the change and this states how long you live with it.
//
// **The card's authored line breaks are the tooltip's line breaks** *(2026-08-23)*. `Text` may carry
// a newline saying where the *card* should break — see `cards.WrapText` — and a tooltip is a list of
// lines already, so splitting on it is what stops the escape reaching a player as a stray glyph.
func wormTip(w session.Worm) (string, []string) {
	return w.Name, append(strings.Split(w.Text, "\n"), "for the rest of the run")
}

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
