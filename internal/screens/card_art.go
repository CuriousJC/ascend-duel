package screens

import (
	"bytes"
	"image"
	_ "image/png"
	"log"
	"strconv"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/entities"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/systems"
	"github.com/hajimehoshi/ebiten/v2"
)

// The bridge between this screen and internal/cards.
//
// internal/cards draws a card into a plain Go image so that `go run ./tools/cardsheet`
// can render one without a window. That is the right trade for a review tool and the
// wrong one for a game loop: every pixel of the shape is written in Go and the text is
// rasterized, so building a card costs far too much to do sixty times a frame. This file
// is the cache that makes it affordable, and it is the only thing the game adds.

// cardKey identifies a rendered card. Two cards that key the same are the same picture.
//
// The Spec is the whole of what cards.Render looks at, so it can be the whole of the key
// — it is comparable, being strings, ints and bools. The style is not comparable in a
// useful way, so its footprint stands in for it; the two styles differ in size, and a
// third that did not would be the same card twice.
type cardKey struct {
	spec cards.Spec
	w, h int
}

// cardCache is package state rather than a field on CombatScene, because the pictures
// outlive any one visit to the screen and re-entering it should not repaint sixty cards.
// Bounded in practice by the deck: 60 cards times two sizes times three states.
var cardCache = map[cardKey]*ebiten.Image{}

// cardFaces is the font internal/cards sets its text with, built once on first use.
// nil after a failure, which is checked rather than retried — a font that will not parse
// will not parse the second time either, and retrying it every frame would turn one log
// line into thousands.
var (
	cardFaces  *cards.Faces
	facesTried bool
)

// cardImages is the image bank a card's own picture is looked up in, stashed here rather than
// threaded through cardSpec.
//
// **It is the shape cardFaces is already in, and for the same reason.** A card's face is built in
// a dozen places — the hand, the deal, the flights, a morph, the post-battle table — and several
// of them deliberately have no GlobalState at all: `resetDeck` takes a run, `OpeningHand` and the
// flight tests take nothing, and threading a screen's state into them to fetch a picture would put
// a graphics dependency inside the headless paths that exist precisely not to have one.
//
// **Nil is a working answer**, which is what makes the stash safe: a tool or a test that never
// went through a drawing door gets no pictures and draws every card exactly as the game drew them
// before this catalog existed. See data.DefaultCardArt, which is empty for the same reason.
var cardImages map[string][]byte

// useImages fills the stash from a screen's state.
//
// **Called at the door of a scene as well as from faces**, which is what fixes the deal: a hand's
// faces are captured in Init, before a single frame has been drawn, so a stash filled lazily on the
// first cardImage call was still nil when the opening hand's pictures were looked up. The cards
// then flew out of the pile with no artwork and grew one the moment the row took over drawing them.
func useImages(gs *state.GlobalState) {
	if cardImages == nil {
		cardImages = gs.ImageData
	}
}

func faces(gs *state.GlobalState) *cards.Faces {
	useImages(gs)
	if facesTried {
		return cardFaces
	}
	facesTried = true

	ttf := gs.FontData["kubasta"]
	if len(ttf) == 0 {
		log.Println("cards: no kubasta font data; cards will not be drawn")
		return nil
	}
	f, err := cards.NewFaces(ttf)
	if err != nil {
		log.Println(err)
		return nil
	}
	cardFaces = f
	return cardFaces
}

// cardSpec turns the screen's own types into the plain data internal/cards draws from.
//
// **The face says what the *card* does and nothing about who is holding it** *(owner's call,
// 2026-08-26)*, damage-wise. A relic's multiplier was written into the figure and colored pink from
// 2026-08-21 until today; what took it off is that the figure stopped being stable. A growing relic
// steps between the cards of one blow, so the same Bash is worth one thing queued first and
// another queued third — and a face stating either would be wrong somewhere. The owner's call went
// further than the accumulator: **no relic reaches the printed damage at all**, growing or not, so a
// Bash reads `1x DMG` whatever is on the fingers. What the relics did is shown where it happens,
// in the sum — see the hand dialog, `combat.GrowthScale` and `Event.HandGrowth`.
//
// **Cost is the exception and stays the pairing's** — see below. A discount is not order-dependent
// and the face must agree with the AP bar.
//
// **The card carries no damage figure** *(2026-08-14)*. It used to resolve `Damage(str)` here,
// because the number a card deals is a property of the pairing rather than of the concept —
// and that is exactly what made it worth removing once the effect text arrived: "Deal 2x DMG"
// is the rule, where "14" was the rule already multiplied out by this duelist's strength and
// was the same fact said twice. `combat.Card.Damage` is still what the engine resolves with, and
// the duelist card still shows a DMG stat.
// **The cost is passed in rather than read off the card** *(2026-08-17)*, because a discount relic
// makes it a property of the pairing: the same card costs 2 to a duelist wearing the discount and 3
// to one who is not. Every caller names the wearer it is drawing for, which is what keeps an enemy's
// queued card out of the player's discounts.
func cardSpec(c actionCard, h held, enabled, selected bool) cards.Spec {
	return cards.Spec{
		Name:       c.Label(),
		Form:       form(c.Form()),
		Cost:       h.cost,
		Element:    artFor(c.Element),
		Text:       cardEffect(c) + riderText(c),
		Highlights: cards.ElementHighlights(cardEffect(c) + riderText(c)),
		Upgrade:    upgradeOf(c),
		Enabled:    enabled,
		Selected:   selected,

		Badge:    cardBadge(c),
		BadgePct: cardBadgePct(c),
		Shields:  cardShields(c),

		Art: cardArtwork(c),
	}
}

// cardBadge is the MOCKUP figure in the card's bottom-left corner: an attack's damage multiplier.
// **Empty for everything else** — a defense says its number by stacking shields rather than by
// printing one, so a figure there would be a second unit in one badge. See cardShields.
//
// **The multiplier is the card's own and no relic reaches it**, exactly as the effect text's is:
// cardSpec's own note says why, and a badge disagreeing with the sentence beside it would be worse
// than either alone.
func cardBadge(c actionCard) string {
	if c.Spec().Verb == combat.VerbAttack {
		return damageMultiplier(c.Amount())
	}
	return ""
}

// cardBadgePct is the raw multiplier the badge art is keyed on — the same figure cardBadge words,
// and zero for anything that is not an attack.
func cardBadgePct(c actionCard) int {
	if c.Spec().Verb == combat.VerbAttack {
		return c.Amount()
	}
	return 0
}

// cardShields is how many shields this card raises, which the card draws as that many stacked
// shields rather than as a figure. **Zero for anything that raises none**, which is every attack
// and a creature's percentage guard.
func cardShields(c actionCard) int {
	if c.Spec().Verb == combat.VerbShield {
		return c.Amount()
	}
	return 0
}

// damageMultiplier writes a card's Amount — a percentage, where 50 is half DMG and 400 is four
// times it — as the multiplier a player reads.
//
// **A fraction for the rungs under one rather than a decimal**, because "1/4" is two glyphs where
// "0.25" is four, and a 32-pixel disc has room for two. The shipped ladder is 25, 50, 100, 300 and
// 400 — so the fractions are the bottom two rungs and there is no 2x rung at all.
//
// **Anything else falls back to the decimal**, which is the readable failure: a card at 150 says
// 1.5 rather than rounding to something it is not.
func damageMultiplier(pct int) string {
	if f, ok := damageFractions[pct]; ok {
		return f
	}
	if pct%100 == 0 {
		return strconv.Itoa(pct / 100)
	}
	return strconv.FormatFloat(float64(pct)/100, 'g', -1, 64)
}

// damageFractions is how the rungs below 1x are written. **A table rather than arithmetic**: a
// general fraction reducer would be three lines of Euclid to produce three strings, and these are
// the only denominators a card can reach.
var damageFractions = map[int]string{
	25: "1/4",
	50: "1/2",
	75: "3/4",
}

// upgradeOf is the visible alteration a card's riders amount to — the one place a rules value
// becomes a drawing one.
//
// **`internal/cards` may not do this and neither may `internal/systems`.** Neither knows what a
// rider is, and neither should: this is the same separation Spec.TextInk draws, where a relic
// becomes a color up here and the renderer is handed the answer. It is why an upgrade is a
// closed vocabulary in `systems` rather than a field on `combat.Rider`.
//
// **A card carries one rider and every kind of rider shows** *(owner's call, 2026-09-09)*. That is
// the whole grammar: form, element and action compose freely, and then there is one upgrade, which
// is a rider, and it is painted. The question of what two visible upgrades on one card look like
// cannot arise, because a card cannot carry two — see `combat.MaxCardRiders`.
//
// **The table is total on purpose, and TestEveryRiderKindIsDrawn is what holds it that way.** A
// rider with no upgrade would be a rune the player spent and cannot see they spent, which is
// the failure the whole mechanic is written to avoid.
func upgradeOf(c combat.Card) systems.Upgrade {
	return upgradeForRider[c.Rider().Kind]
}

// upgradeForRider is which upgrade paints each rider. **RiderNone is deliberately absent**, so an
// unridden card falls out as UpgradeNone — the zero value — rather than needing a case.
var upgradeForRider = map[combat.RiderKind]systems.Upgrade{
	combat.RiderWildElement:  systems.UpgradeWild,
	combat.RiderGolden:       systems.UpgradeGolden,
	combat.RiderSilver:       systems.UpgradeSilver,
	combat.RiderHealOnPlay:   systems.UpgradeHeal,
	combat.RiderShieldOnPlay: systems.UpgradeShield,
	combat.RiderDamageOnPlay: systems.UpgradeDamage,
	combat.RiderScaleInCombo: systems.UpgradeCombo,
	combat.RiderDamageInHand: systems.UpgradeHeldDamage,
	combat.RiderScaleInHand:  systems.UpgradeHeldScale,
	combat.RiderVitaeInHand:  systems.UpgradeHeldVitae,
}

// boostInk is what a figure a relic has changed is written in. **The relic pink** — `cards.Relic` is
// the border color a relic card carries, so the color already means "a relic did this" everywhere
// else on screen, and spending a second hue on the same fact would be saying it twice.
var boostInk = cards.BorderOf(cards.Relic)

// held is the pairing a card is drawn in: what it costs the holder, what the holder hits for, and
// which relics the holder is wearing.
//
// **Cost traveled alone until 2026-08-21 and that was already the same idea** — a discount relic
// makes a cost a property of the pairing rather than of the card, and a damage relic does exactly
// that to the figure on the face. Grouping them is what stops the two drifting apart at a call site
// that remembered one and not the other.
//
// **The zero value is a card nobody is holding**: no relics, no strength, and its own printed cost.
// `tools/cardsheet` and any panel drawing the catalog want that, and so does an enemy's queued
// card — relics are the duelist's only.
type held struct {
	cost int

	// dmg is the holder's DMG, and **zero means nobody on this screen knows one**. The tooltip says
	// what an attack is worth when it has a figure and states the multiplier alone when it does not;
	// the reward and shop screens are the second case, since a run's stats belong to a fight.
	dmg int

	worn []combat.WornRelic
}

// heldBy is the pairing for a card in a duelist's hands, which is what every call site inside a
// fight has.
func heldBy(d combat.Duelist, c actionCard) held {
	return held{cost: d.CardCost(c), dmg: d.DMG, worn: ungrown(d.WornRelics())}
}

// ungrown is a worn set with every accumulator at zero.
//
// **The tooltip explains a card's relics at their record, never at how far one has counted**
// *(owner's call, 2026-08-26)*. The card's *face* carries no relic at all now — see cardSpec — and
// what is left reading a worn set is the hover, which is the one place a player can ask what their
// relics do to a card before committing it. The accumulator is kept out of that answer for the reason
// it was kept off the face: it depends on where in the turn the card is counted, so any figure
// quoted before the turn is resolved would be wrong somewhere. The growth is said in the sum, beside
// the term it priced — see combat.GrowthScale and the hand dialog.
//
// **Cost is untouched by this**, because no growing relic adjusts a cost and a discount does not move
// with the queue.
func ungrown(worn []combat.WornRelic) []combat.WornRelic {
	if len(worn) == 0 {
		return nil
	}

	out := make([]combat.WornRelic, len(worn))
	for i, w := range worn {
		w.Grown = 0
		out[i] = w
	}
	return out
}

// heldByRun is the pairing for a card drawn between fights, where there is a run and no duelist:
// the run's relics price it and no strength is known.
func heldByRun(gs *state.GlobalState, c actionCard) held {
	if gs.Run == nil {
		return held{cost: c.Cost()}
	}
	return held{cost: gs.Run.CardCost(c), worn: ungrown(gs.Run.WornRelics())}
}

// cardImage returns the card for this spec, rendering and caching it on a miss.
//
// Returns nil rather than a placeholder when the font is missing. drawCard checks for
// that and draws nothing: a card-shaped hole is a bug someone will report, where a card
// drawn in a fallback font is one they will not notice until a screenshot looks wrong.
func cardImage(gs *state.GlobalState, spec cards.Spec, st cards.Style) *ebiten.Image {
	key := cardKey{spec: spec, w: st.Width, h: st.Height}
	if img, ok := cardCache[key]; ok {
		return img
	}

	f := faces(gs)
	if f == nil {
		return nil
	}
	rendered, err := cards.Render(spec, st, f)
	if err != nil {
		log.Println(err)
		cardCache[key] = nil // negative-cached, so a broken card logs once and not per frame
		return nil
	}

	img := ebiten.NewImageFromImage(rendered)
	cardCache[key] = img
	return img
}

// artworkCache holds the decoded pictures that go *on* a card — enemy portraits, relic art —
// keyed by their assets name.
//
// **Decoded once and held**, for the same reason the cards themselves are cached: these are
// 320-pixel PNGs and `image.Decode` is not a per-frame operation. They are handed to
// internal/cards as plain `image.Image`, which is why they come out of `LoadImageData` as
// bytes rather than out of `LoadAssets` as *ebiten.Image — a card is drawn with no graphics
// context.
//
// **One cache for both, rather than one per kind of art.** It was `portraitCache` until the
// relic pane arrived on 2026-08-11; a second map would have been the same six lines keyed the
// same way, and the thing they have in common — a file that has to be decoded before it can
// be drawn into a card — is the whole of what either needs.
//
// A failure is cached as nil so a bad file logs once rather than sixty times a second, and
// the card then draws with no picture rather than not at all.
var artworkCache = map[string]image.Image{}

// cardArt is the catalog of pictures, one per card per element. Loaded once; the engine reads
// none of it — see data/card_art_data.go.
var cardArt = data.LoadCardArt()

// cardArtwork is the picture for one card in the element it is wearing, or nil for a pairing
// nobody has drawn yet — which is almost all of them, and draws the card as it always looked.
//
// **The element is the card's own, not the one a relic flipped it to on the way to the table.**
// `Card.Element` is what the card *is*, and it is what the rest of the face already reads: the
// cost ticks and the border are drawn from it, so a picture keyed off anything else would be the
// one thing on the card disagreeing with them.
//
// **The form is the opposite, and that is the owner's call** *(2026-09-16)*. A form override moves
// the corner mark — a Slice told to be a crush wears the club — so a picture keyed off the card's
// own concept left the one figure on the card still swinging a sabre under a club. The art follows
// the mark instead: `combat.Counterpart` answers the concept standing at the same rung of the
// overridden form, and that concept's label is what the catalog is asked for. **The two axes
// therefore disagree on purpose**, and the reason is which of them the rest of the face agrees
// with: nothing on a card is drawn from a flipped element, and the mark is drawn from the
// overridden form.
//
// **A defense keeps its own picture, and that is decided rather than left over** *(owner's call,
// 2026-09-16)*. `combat.Counterpart` matches on the verb, so nothing on the attack ladders answers
// a Brace told to be a crush — and it should not: the card still raises shields, so a club in its
// hands would be the picture lying about what it does. Crossing the attack/defend line changes the
// mark and nothing else; the repaint is for a card moving between the three attack forms, where
// what it does is the same and only the weapon differs.
func cardArtwork(c actionCard) image.Image {
	rec, ok := cardArt[cardArtRecord(c)]
	if !ok {
		return nil
	}
	return artworkFrom(rec.ArtKey())
}

// cardArtRecord is which record of the catalog a card draws from. Split out of cardArtwork so the
// rule above can be tested without a picture: the catalog is mostly undrawn, so asserting on the
// image would be asserting on which pairings happen to have been painted.
func cardArtRecord(c actionCard) string {
	label := c.Label()
	if id, ok := combat.Counterpart(c.Concept, c.Form()); ok {
		label = combat.ConceptOf(id).Label
	}
	return data.CardArtKey(label, c.Element.String())
}

// artworkFrom is artwork() without a GlobalState, reading the stash instead. See cardImages.
func artworkFrom(key string) image.Image {
	if key == "" || cardImages == nil {
		return nil
	}
	if img, ok := artworkCache[key]; ok {
		return img
	}
	data := cardImages[key]
	if len(data) == 0 {
		log.Printf("cards: no artwork named %q", key)
		artworkCache[key] = nil
		return nil
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		log.Printf("cards: decoding artwork %q: %v", key, err)
		img = nil
	}
	artworkCache[key] = img
	return img
}

func artwork(gs *state.GlobalState, key string) image.Image {
	if key == "" {
		return nil
	}
	if img, ok := artworkCache[key]; ok {
		return img
	}

	data := gs.ImageData[key]
	if len(data) == 0 {
		log.Printf("cards: no artwork named %q", key)
		artworkCache[key] = nil
		return nil
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		log.Printf("cards: decoding artwork %q: %v", key, err)
		img = nil
	}
	artworkCache[key] = img
	return img
}

// enemySpec is the opponent as a card: its portrait, its name, and the life it has left.
//
// Life is on the Spec, so a point of damage produces a different cache entry — see the
// field's comment in internal/cards. Bounded by how many distinct life totals a fight passes
// through, which is a handful.
// **It carries the statuses standing on the opponent** *(2026-08-16)*, as a row of badges along
// the bottom edge — see `effectArt`. A status is invisible without it: a chill takes a card off
// a turn that has not been queued yet and a weight blunts a blow not yet swung, so a player with
// no badge to look at learns about either only by being surprised by it.
// **`life` is passed in rather than read off the combatant** *(2026-08-18)*, because the bar waits
// for the figure flying at it: while a hit is in the air the card keeps drawing what the duelist had
// before the blow, so the drop and the arrival are one event. The combatant is already correct
// underneath — see `CombatScene.shownLife`, which is a view over it and never a second copy.
func enemySpec(gs *state.GlobalState, c *entities.Combatant, name string, life int) cards.Spec {
	spec := cards.Spec{
		Name:    name,
		Element: cards.Basic,
		Art:     artwork(gs, c.Portrait),
		Life:    life,
		MaxLife: c.MaxLife,
		Enabled: true,
	}

	// **Walked in registration order, which is what makes the row stable.** A badge that moved along
	// the row as another status came and went would read as a different badge. `AllStatuses` is the
	// file order the determinism rules require.
	n := 0
	for _, id := range combat.AllStatuses() {
		if n == len(spec.Effects) || !c.Statuses[id].Active() {
			continue
		}
		img := effectArt(gs, id)
		if img == nil {
			continue
		}
		spec.Effects[n] = img
		n++
	}
	return spec
}

// statusBadges is the art key each status is drawn with, **read off `statuses.json`** rather than
// held in a table here *(2026-08-17)*.
//
// **A badge belongs to the status and not to the relic that switches it on**, which is why the key
// sits in the status record: a status arriving by some other route — an affix, a boss rule — has to
// draw the same picture, and reading the art key off a relic the enemy is not wearing would be the
// wrong lookup by construction. It was a table keyed by element until statuses stopped being
// elements, at which point the table would have had to be keyed by the record anyway — so the
// record carries it.
//
// A status whose badge is empty or unknown falls back to `defaulteffect_png`, so one nobody has made
// art for shows a shape nobody has learned rather than nothing at all.
var statusBadges = badgeKeys()

func badgeKeys() map[string]string {
	out := map[string]string{}
	for _, s := range data.LoadStatuses() {
		if s.Badge != "" {
			out[s.StatusRecord] = s.Badge
		}
	}
	return out
}

func effectArt(gs *state.GlobalState, id combat.StatusID) image.Image {
	key, ok := statusBadges[combat.StatusOf(id).Key]
	if !ok {
		key = "defaulteffect_png"
	}
	return artwork(gs, key)
}

// duelistSpec is the player as a card: their name, three figures, and the life they have
// left.
//
// **DMG is the stat, printed** *(2026-08-16)*. It used to be `combat.Bash.Damage(DMG)`, on the
// grounds that the figure should follow the ladder rather than the stat — which was worth doing
// while the ladder was a switch statement with Bash on its middle rung. A card declares its own
// multiplier now, so 1x is the definition rather than one card's entry, and asking a particular
// card what it deals would make this figure move when that card was retuned.
//
// AP is the round's budget, which is the duelist's own stat and nothing else — nothing adds to it
// any more. **It is still passed in rather than asked of the combatant**, so the one caller that
// wants a different figure has somewhere to put it, exactly as `life` and `shields` do. Vitae is
// passed in because it is run-level state that does not live on a duelist yet — see
// startingVitae.
//
// Every distinct set of figures is a cache entry, like the enemy's life. Bounded by how many
// values a fight passes through, which is a handful.
// `life` is passed in for the reason enemySpec's is — the bar lags a figure still on its way.
// **`fight` is where the run has got to, and the last two rows are drawn from it**
// *(owner's call, 2026-09-15)*. The floor and the room were two lines on the ground under the
// card, which made them the one fact about the duelist not written on the duelist; they are stat
// rows now, on the same terms as DMG and AP — a label against a figure, true for the whole fight.
//
// **They are derived here rather than passed in**, unlike every other figure in this signature.
// The others are quantities a caller may want to draw differently from the model — a bar lagging a
// figure in flight, a build band with no shields standing — and where the run *is* has no such
// reading: `towerFloor` and `towerRoom` are the whole of it.
func duelistSpec(gs *state.GlobalState, c *entities.Combatant, name string,
	dmg, vitae, life, maxLife, ap, fight, shields int, els ...cards.Element) cards.Spec {
	spec := cards.Spec{
		Name:    name,
		Element: cards.Basic,
		Life:    life,
		MaxLife: maxLife,
		Enabled: true,
	}
	spec.Stats[0] = cards.StatLine{Label: "DMG", Value: strconv.Itoa(dmg)}
	spec.Stats[1] = cards.StatLine{Label: "AP", Value: strconv.Itoa(ap)}
	spec.Stats[2] = cards.StatLine{Label: "VITAE", Value: strconv.Itoa(vitae), ValueInk: vitaeInk}
	spec.Stats[3] = cards.StatLine{Label: "FLOOR", Value: strconv.Itoa(towerFloor(fight))}
	spec.Stats[4] = cards.StatLine{Label: "ROOM", Value: towerRoom(fight)}

	// **One pip per shield, in the seat the enemy's status badges sit in.** They are drawn with
	// the defend form's own mark, so what the player raised and what is standing are the same
	// picture — a second drawing for the same idea is how a row of pips comes to mean something
	// slightly different from the card that bought it.
	//
	// **`combat` caps a duelist at as many shields as this row holds**, so a count that would
	// overflow cannot exist rather than being silently trimmed here — see Duelist.raiseShields.
	// **Each pip keeps the element of the card that raised it** *(owner's call, 2026-09-02)*, which
	// is the shield it was drawn as while it flew. Cosmetic: nothing about a shield depends on the
	// element behind it, and a pip that changed on landing would say the opposite. A pip with no
	// element — a shield standing from a round nobody watched, or one drawn outside a duel — is the
	// neutral mark.
	if shields > 0 {
		for i := 0; i < shields && i < len(spec.Effects); i++ {
			e := cards.Basic
			if i < len(els) {
				e = els[i]
			}
			if img := shieldPip(gs, e); img != nil {
				spec.Effects[i] = img
			}
		}
	}
	return spec
}

// shieldPip is the picture one standing shield is drawn as: the defend form's corner mark in that
// shield's own element, the same file a Brace, a Block and a Guard carry.
//
// **It goes through `artwork` rather than `systems.ArtMark`** because a pip is scaled into a
// twenty-pixel badge box like every other thing in that row, and the badge row takes an
// `image.Image` — `internal/cards` has no graphics context, which is the whole reason the badges
// are bytes.
//
// **Nothing is tinted** *(2026-09-16)*. There were four form marks and five elements, so a pip was
// one near-white drawing multiplied by a color; there are now twenty marks plus a neutral set, so
// the pip is simply the drawing the card is showing. The tinted-pip cache went with it.
func shieldPip(gs *state.GlobalState, e cards.Element) image.Image {
	return artwork(gs, shieldPipKey(e))
}

// relicSpec is an equipped relic as a card: its name and its artwork, and nothing else.
//
// **The element on the record does not reach the Spec**, deliberately. `cards.Relic` is the
// element a relic card carries, and what it paints the border from is the *rarity*
// *(owner's call, 2026-09-13)* — the pink it used to paint is in cards.rarityBorders' history.
// `RelicData.Element` says which element the relic will eventually *discount*; it is a rule, not
// a color, and it has nowhere to be read yet.
//
// **The rarity does reach it, and it is the only record field that becomes a color here.** A
// relic is bought off a shelf, so how scarce it is the fact worth carrying on the face.
//
// No cost, no category, no damage: a relic is not played from a hand and has no phase.
func relicSpec(gs *state.GlobalState, r data.RelicData, counter string, enabled, lit bool) cards.Spec {
	return cards.Spec{
		Name:     r.Name,
		Element:  cards.Relic,
		Rarity:   r.Rarity,
		Art:      artwork(gs, r.ArtKey()),
		Counter:  counter,
		Enabled:  enabled,
		Selected: lit,
	}
}

// backSpec is a face-down card of this duelist's deck.
//
// **A duelist and a card back go together** *(2026-08-11)*: the plan is to offer different
// duelists as different decks, and the mark on the back is how you tell at a glance whose
// deck is on the table. The name comes from `data/duelists.json` and is parsed here rather
// than at load, because `internal/entities` must not import the drawing package — the same
// separation the element mapping below exists for.
//
// An unrecognized name falls back to the triangle and says so once. A back is cosmetic;
// refusing to draw the draw pile over one would be a worse outcome than the wrong shape.
func (s *CombatScene) backSpec() cards.Spec {
	mark, ok := cards.ParseBackMark(s.fighter.CardBack)
	if !ok && s.fighter.CardBack != "" && !warnedBack {
		warnedBack = true
		log.Printf("cards: duelist card back %q is not a mark; using %v",
			s.fighter.CardBack, mark)
	}
	return cards.Spec{FaceDown: true, Back: mark}
}

// warnedBack keeps a bad name in duelists.json to one log line rather than one per frame.
var warnedBack bool

// artFor maps the rules' element onto the drawing package's.
//
// **The two enums stay separate**, and the reason changed on 2026-08-12 rather than expiring.
// It used to be that `element` was a screen type on its way into `internal/combat` and a drawing
// package should not stand in the way of the move. The move has happened, and the separation is
// now the ordinary one `category` below has: `internal/cards` knows how to paint a border and
// nothing about what an element does to a duelist. The cost is this switch;
// TestEveryElementHasItsOwnArt keeps it honest.
//
// A free function rather than a method, because a method cannot be hung on another package's
// type — which is the one thing the collapse to `combat.Element` cost.
//
// The default is Basic rather than a panic. An unmapped element is a card in the wrong
// color, which is a visual bug; crashing mid-duel over one would be worse.
func artFor(e combat.Element) cards.Element {
	switch e {
	case combat.Fire:
		return cards.Fire
	case combat.Ice:
		return cards.Ice
	case combat.Lightning:
		return cards.Lightning
	case combat.Earth:
		return cards.Earth
	case combat.Arcane:
		return cards.Arcane
	default:
		return cards.Basic
	}
}

// form maps the rules' form onto the drawing package's, which is drawn in the card's corner.
//
// Two enums again, and for the same reason as the elements — internal/cards knows how to
// draw a card and nothing about how a round resolves. The default is FormNone, which
// draws no mark at all rather than guessing at one, and it is what the opponent's cards get.
func form(f combat.Form) cards.Form {
	switch f {
	case combat.FormStab:
		return cards.FormStab
	case combat.FormSlash:
		return cards.FormSlash
	case combat.FormCrush:
		return cards.FormCrush
	case combat.FormDefend:
		return cards.FormDefend
	default:
		return cards.FormNone
	}
}

// Compile-time assurance that a card still answers everything a Spec needs. If combat.Card loses
// one of these, this fails here rather than in a card that silently renders blank.
var _ = func(c combat.Card, dmg int) (string, string, string, int, int) {
	return c.Label(), c.Category().String(), c.Form().String(), c.Damage(dmg), c.Cost()
}

// essenceSpec is an essence drawn as a card: a name, a line of what it does, and the color of whatever
// it grants.
//
// **It borrows `cards.Hand` at the call site rather than taking a style of its own**, because a
// essence has no cost and no form and that style draws both as nothing — no dashes for a zero cost,
// no mark for FormNone. What is left is exactly the name and the text, which is the whole of
// what an essence has to say. A style of its own is what this wants the day an essence has art.
//
// **The picture comes off the record** *(2026-09-12)*, through `data.EssenceData.ArtKey`, which is
// already resolved by the time a `session.Essence` exists — so an essence nobody has drawn wears the
// placeholder and one that has been drawn wears its own, and this call site does not know which.
//
// **The border carries the element for the same reason a card's does**: an Ember Essence is red
// because what it hands you is red. The ones that take a card away rather than color it are
// basic, which is the mid gray `cards.BorderOf` gives that element — deliberately not a fifth hue,
// since removal is the absence of a color rather than one of its own.
func essenceSpec(gs *state.GlobalState, w session.Essence, enabled bool) cards.Spec {
	return cards.Spec{
		Name:       w.Name,
		Form:       cards.FormNone,
		Cost:       0,
		Element:    artFor(w.Element),
		Art:        artwork(gs, w.Art),
		Text:       w.Text,
		Highlights: cards.ElementHighlights(w.Text),
		Enabled:    enabled,
	}
}

// stoneSpec is a stone drawn as a card: a name, the rung it raises, and what it is worth.
//
// **The figure is computed rather than authored** *(2026-08-27)*. What a stone adds is a tenth of
// its rung's catalog multiplier, so `+11` written into `data/stones.json` would be a number that
// went stale the first time `hands.json` was tuned — silently, since nothing reads a card's text.
// The record carries the sentence and this carries the arithmetic, which is the same split a
// card's face already makes between its label and its damage.
//
// **The picture is the record's own, resolved through `data.StoneData.ArtKey`**, which answers the
// catalog's default face for a stone nobody has painted — so this and tools/stonesheet cannot
// disagree about what an undrawn one looks like.
//
// **Basic, not an element.** A stone raises a rung of the ladder, and a rung is not a color — the
// axis a hand counts on is not one of the five. So its border is the mid gray `cards.BorderOf`
// gives `basic`, exactly as a Devour essence's is.
func stoneSpec(gs *state.GlobalState, st session.Stone, enabled bool) cards.Spec {
	return cards.Spec{
		Name:    st.Name,
		Form:    cards.FormNone,
		Cost:    0,
		Element: artFor(combat.Basic),
		Art:     stoneFace(gs, st),
		Enabled: enabled,
	}
}

// goodSpec is one of the shop's two sealed goods as a card: the bag of rocks, or the vial of essence.
//
// **A sealed good is a card that says what is inside without saying which**, and since 2026-09-15
// it says it in the tooltip rather than on its face *(owner's call)*. It carried "4 stones / keep
// 1" across the lower half of its picture until then — the shape of the offer and never its
// contents, which is still exactly what the tooltip says, in the room it takes to say it properly.
// The mechanic is unchanged: what a player can know before paying is the vessel and the count, and
// the four inside are drawn when it is opened.
//
// **All three are painted as of 2026-09-15**: a bag, a vial and a sack, each drawn as the vessel
// its Name says it is. They borrowed the picture of whatever was inside them until then — the
// boulder, the essence default, the rune default — on the argument that a third picture was a
// third thing to recognize for no gain. What that missed is that a borrowed face makes the vial
// and the essence card in it the same picture, so the shelf says "essences" twice and says nothing
// about the *vessel* being what is bought. The borrow is still the empty case; see goodArt.
func goodSpec(gs *state.GlobalState, name string, art image.Image, enabled bool) cards.Spec {
	return cards.Spec{
		Name:    name,
		Form:    cards.FormNone,
		Cost:    0,
		Element: artFor(combat.Basic),
		Art:     art,
		Enabled: enabled,
	}
}

// stoneFace is the picture one stone draws.
//
// **There is no fallback here any more** *(2026-09-16)*. `session.Stone.Art` is resolved through
// `data.StoneData.ArtKey`, which answers the catalog's default face for a record nobody has
// painted — so the decision lives in `data/` beside the relics' and the essences', rather than in
// a screen the review tools cannot reach. What it replaced was the generated boulder, which was
// the last thing keeping a silhouette in `internal/systems` that nothing else drew.
func stoneFace(gs *state.GlobalState, st session.Stone) image.Image {
	return artwork(gs, st.Art)
}
