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
	"github.com/hajimehoshi/ebiten/v2"
)

// The bridge between this screen and internal/cards.
//
// internal/cards draws a card into a plain Go image so that `go run ./tools/cardsheet`
// can render one without a window. That is the right trade for a review tool and the
// wrong one for a game loop: every pixel of the shape is written in Go and the text is
// rasterised, so building a card costs far too much to do sixty times a frame. This file
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

func faces(gs *state.GlobalState) *cards.Faces {
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
// 2026-08-26)*, damage-wise. A ring's multiplier was written into the figure and coloured pink from
// 2026-08-21 until today; what took it off is that the figure stopped being stable. A growing ring
// steps between the cards of one blow, so the same Strike is worth one thing queued first and
// another queued third — and a face stating either would be wrong somewhere. The owner's call went
// further than the accumulator: **no ring reaches the printed damage at all**, growing or not, so a
// Strike reads `1x DMG` whatever is on the fingers. What the rings did is shown where it happens,
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
// **The cost is passed in rather than read off the card** *(2026-08-17)*, because a discount ring
// makes it a property of the pairing: the same card costs 2 to a duelist wearing the discount and 3
// to one who is not. Every caller names the wearer it is drawing for, which is what keeps an enemy's
// queued card out of the player's discounts.
func cardSpec(c actionCard, h held, enabled, selected bool) cards.Spec {
	return cards.Spec{
		Name:     c.Label(),
		Form:     form(c.Form()),
		Cost:     h.cost,
		Element:  artFor(c.Element),
		Text:     cardEffect(c),
		Enabled:  enabled,
		Selected: selected,
	}
}

// boostInk is what a figure a ring has changed is written in. **The ring pink** — `cards.Ring` is
// the border colour a ring card carries, so the colour already means "a ring did this" everywhere
// else on screen, and spending a second hue on the same fact would be saying it twice.
var boostInk = cards.BorderOf(cards.Ring)

// held is the pairing a card is drawn in: what it costs the holder, what the holder hits for, and
// which rings the holder is wearing.
//
// **Cost travelled alone until 2026-08-21 and that was already the same idea** — a discount ring
// makes a cost a property of the pairing rather than of the card, and a damage ring does exactly
// that to the figure on the face. Grouping them is what stops the two drifting apart at a call site
// that remembered one and not the other.
//
// **The zero value is a card nobody is holding**: no rings, no strength, and its own printed cost.
// `tools/cardsheet` and any panel drawing the catalogue want that, and so does an enemy's queued
// card — rings are the duelist's only.
type held struct {
	cost int

	// dmg is the holder's DMG, and **zero means nobody on this screen knows one**. The tooltip says
	// what an attack is worth when it has a figure and states the multiplier alone when it does not;
	// the reward and shop screens are the second case, since a run's stats belong to a fight.
	dmg int

	worn []combat.WornRing
}

// heldBy is the pairing for a card in a duelist's hands, which is what every call site inside a
// fight has.
func heldBy(d combat.Duelist, c actionCard) held {
	return held{cost: d.CardCost(c), dmg: d.DMG, worn: ungrown(d.WornRings())}
}

// ungrown is a worn set with every accumulator at zero.
//
// **The tooltip explains a card's rings at their record, never at how far one has counted**
// *(owner's call, 2026-08-26)*. The card's *face* carries no ring at all now — see cardSpec — and
// what is left reading a worn set is the hover, which is the one place a player can ask what their
// rings do to a card before committing it. The accumulator is kept out of that answer for the reason
// it was kept off the face: it depends on where in the turn the card is counted, so any figure
// quoted before the turn is resolved would be wrong somewhere. The growth is said in the sum, beside
// the term it priced — see combat.GrowthScale and the hand dialog.
//
// **Cost is untouched by this**, because no growing ring adjusts a cost and a discount does not move
// with the queue.
func ungrown(worn []combat.WornRing) []combat.WornRing {
	if len(worn) == 0 {
		return nil
	}

	out := make([]combat.WornRing, len(worn))
	for i, w := range worn {
		w.Grown = 0
		out[i] = w
	}
	return out
}

// heldByRun is the pairing for a card drawn between fights, where there is a run and no duelist:
// the run's rings price it and no strength is known.
func heldByRun(gs *state.GlobalState, c actionCard) held {
	if gs.Run == nil {
		return held{cost: c.Cost()}
	}
	return held{cost: gs.Run.CardCost(c), worn: ungrown(gs.Run.WornRings())}
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

// artworkCache holds the decoded pictures that go *on* a card — enemy portraits, ring art —
// keyed by their assets name.
//
// **Decoded once and held**, for the same reason the cards themselves are cached: these are
// 320-pixel PNGs and `image.Decode` is not a per-frame operation. They are handed to
// internal/cards as plain `image.Image`, which is why they come out of `LoadImageData` as
// bytes rather than out of `LoadAssets` as *ebiten.Image — a card is drawn with no graphics
// context.
//
// **One cache for both, rather than one per kind of art.** It was `portraitCache` until the
// ring pane arrived on 2026-08-11; a second map would have been the same six lines keyed the
// same way, and the thing they have in common — a file that has to be decoded before it can
// be drawn into a card — is the whole of what either needs.
//
// A failure is cached as nil so a bad file logs once rather than sixty times a second, and
// the card then draws with no picture rather than not at all.
var artworkCache = map[string]image.Image{}

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
// **A badge belongs to the status and not to the ring that switches it on**, which is why the key
// sits in the status record: a status arriving by some other route — an affix, a boss rule — has to
// draw the same picture, and reading the art key off a ring the enemy is not wearing would be the
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
// **DMG is the stat, printed** *(2026-08-16)*. It used to be `combat.Strike.Damage(DMG)`, on the
// grounds that the figure should follow the ladder rather than the stat — which was worth doing
// while the ladder was a switch statement with Strike on its middle rung. A card declares its own
// multiplier now, so 1x is the definition rather than one card's entry, and asking a particular
// card what it deals would make this figure move when that card was retuned.
//
// AP is the live budget, `BonusAP` included, so a Prepare banked last round shows up on the
// card before it is spent. **It is passed in rather than asked of the combatant** *(2026-08-19)*,
// for the reason `life` is: a Prepare resolving mid-round sends its points to this card as a
// figure, and the line has to move when that figure lands rather than when the round's end state
// is adopted a whole opposing turn later. `shownBank` is the addition; the model underneath is
// untouched. Vitae is passed in because it is run-level state that does not live on a duelist
// yet — see startingVitae.
//
// Every distinct set of figures is a cache entry, like the enemy's life. Bounded by how many
// values a fight passes through, which is a handful.
// `life` is passed in for the reason enemySpec's is — the bar lags a figure still on its way.
func duelistSpec(c *entities.Combatant, name string, vitae, life, ap int) cards.Spec {
	spec := cards.Spec{
		Name:    name,
		Element: cards.Basic,
		Life:    life,
		MaxLife: c.MaxLife,
		Enabled: true,
	}
	spec.Stats[0] = cards.StatLine{Label: "DMG", Value: strconv.Itoa(c.DMG)}
	spec.Stats[1] = cards.StatLine{Label: "AP", Value: strconv.Itoa(ap)}
	spec.Stats[2] = cards.StatLine{Label: "Vitae", Value: strconv.Itoa(vitae), ValueInk: vitaeInk}
	return spec
}

// ringSpec is an equipped ring as a card: its name and its artwork, and nothing else.
//
// **The element on the record does not reach the Spec**, deliberately. `cards.Ring` is the
// element a ring card carries, which paints the border pink whatever the ring is about — the
// one thing that must never happen is reaching for a ring thinking it is a card you can play.
// `RingData.Element` says which element the ring will eventually *discount*; it is a rule, not
// a colour, and it has nowhere to be read yet.
//
// No cost, no category, no damage: a ring is not played from a hand and has no phase.
func ringSpec(gs *state.GlobalState, r data.RingData, counter string, enabled, lit bool) cards.Spec {
	return cards.Spec{
		Name:     r.FaceName(),
		Element:  cards.Ring,
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
// An unrecognised name falls back to the triangle and says so once. A back is cosmetic;
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
// colour, which is a visual bug; crashing mid-duel over one would be worse.
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
	case combat.FormPlan:
		return cards.FormPlan
	default:
		return cards.FormNone
	}
}

// Compile-time assurance that a card still answers everything a Spec needs. If combat.Card loses
// one of these, this fails here rather than in a card that silently renders blank.
var _ = func(c combat.Card, dmg int) (string, string, string, int, int) {
	return c.Label(), c.Category().String(), c.Form().String(), c.Damage(dmg), c.Cost()
}

// wormSpec is a worm drawn as a card: a name, a line of what it does, and the colour of whatever
// it grants.
//
// **It borrows `cards.Hand` at the call site rather than taking a style of its own**, because a
// worm has no cost and no form and that style draws both as nothing — no dashes for a zero cost,
// no mark for FormNone. What is left is exactly the name and the text, which is the whole of
// what a worm has to say. A style of its own is what this wants the day a worm has art.
//
// **The border carries the element for the same reason a card's does**: an Ember Worm is red
// because what it hands you is red. The ones that take a card away rather than colour it are
// basic, which is the mid grey `cards.BorderOf` gives that element — deliberately not a fifth hue,
// since removal is the absence of a colour rather than one of its own.
func wormSpec(gs *state.GlobalState, w session.Worm, enabled bool) cards.Spec {
	return cards.Spec{
		Name:    w.Name,
		Form:    cards.FormNone,
		Cost:    0,
		Element: artFor(w.Element),
		Art:     artwork(gs, wormArtKey),
		Text:    w.Text,
		Enabled: enabled,
	}
}

// wormArtKey is the picture every worm draws today: `assets/worm/default-worm.png`, a placeholder
// of its own rather than a borrowed ring.
//
// **Keys are not file paths** — `LoadImageData` files this under `defaultworm_png`, which is what a
// lookup has to spell. Writing the filename here is why the first version of this drew nothing and
// logged `no artwork named "default-ring"`.
//
// It is one constant rather than a field on the record because when worms get art it becomes a key
// per worm, and that change should be a `data/worms.json` field appearing, not a fallback being
// unpicked.
const wormArtKey = "defaultworm_png"
