package entities

import (
	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/journey"
)

// Combatant is a duelist that can be drawn. The stats live in the embedded
// combat.Duelist so the rules engine can take them without ever seeing a sprite;
// the promoted fields mean gs.Fighter.DMG and friends still read the same.
type Combatant struct {
	combat.Duelist

	// Record is which roster entry this combatant was built from, and it is what names its deck
	// in `internal/decks` *(2026-08-16)*. It replaced `Style combat.PlanStyle`: an enemy's
	// behavior used to be a string picking one of four planners and is now the cards it holds,
	// so what a caller needs from this struct is the key to those cards.
	//
	// Empty for the player, whose deck is built by the screen.
	Record string

	// **There is no sprite here any more** *(2026-08-11)*. The fighter's went when the
	// character block replaced it, and the enemy's went when the enemy became a card — so
	// the field, the sheet slicing and the *ebiten.Image import all went with them. This
	// struct now holds nothing Ebitengine defines, which is worth keeping: it is one of the
	// two things standing between `entities` and needing a window.
	//
	// Portrait is the assets key for the picture on this combatant's card, carried through
	// from the data record rather than resolved here — entities cannot reach the asset map,
	// and the card is drawn by internal/cards from raw bytes rather than from an
	// *ebiten.Image anyway.
	//
	// Empty for the fighter, like Sprite is nil for it.
	Portrait string

	// Name is what this combatant is called on screen. Set for a duelist, whose record
	// carries one; an enemy's name comes from the roster in internal/screens rather than
	// from its record, and moving that is a separate change.
	Name string

	// Deck is the deck this duelist plays from — its back and how many discards a round allows —
	// resolved off the record's `Deck` key when the duelist is built.
	//
	// Zero for an enemy: enemies do not have a deck the player ever sees the back of, and nothing
	// discards from one.
	Deck data.DeckData

	// Element is which element this opponent was dealt as, by name — the realm's theme. Empty for
	// the player, whose cards each carry their own.
	//
	// **A string rather than a combat.Element**: what this package carries is what
	// the data file writes, and the parsing belongs where the cards are built.
	Element string
}

// NewEnemyFrom builds an opponent from a motif record, dealt as one element and **grown to the
// fight it is met at** — see journey.ScaleToFight. Fight 0 is the first room of the journey and
// takes the record's bases ungrown, set at the journey's HPScale and DMGScale.
//
// **The fight index is a parameter rather than something read later**, so an unscaled opponent
// cannot be built by accident: every caller has to say where on the growth curve this one stands.
//
// **The element is the realm's**, and it decides the picture this opponent wears, the colour of
// every card in its deck, and which of the player's hits fizzle on it — see combat.Duelist.Element.
// A record carries a picture per element it can be dealt as, so a fire goblin and an ice goblin
// are two drawings of one creature.
func NewEnemyFrom(r data.MotifRecord, element string, fight int, shape data.JourneyData) *Combatant {
	c := &Combatant{
		Duelist: combat.Duelist{
			// **Two of the three stats climb and one does not.** HP and DMG are what the curve is
			// made of, on their own growth rates; `Actions` is left alone because it is the budget
			// a *deck* is spent out of, and growing it would hand a realm-eight opponent more cards
			// rather than a harder version of its own. It is the dial to reach for on purpose, per
			// record, not one to move by arithmetic.
			DMG:     journey.ScaleToFight(r.DMG, fight, shape.DMGGrowth, shape.DMGScale),
			Actions: r.Actions,
			MaxLife: journey.ScaleToFight(r.HP, fight, shape.HPGrowth, shape.HPScale),

			// **Enemies do not form hands.** Their cards resolve one at a time, in the order the
			// planner chose them. It is set here because this is the one place an opponent is built
			// from a record — the same seat `Relics` deliberately leaves at its zero value for the
			// mirror-image reason.
			SoloAttacks: true,

			// **The realm's element is the creature's own**, and a hit of it fizzles — see
			// combat.fizzles. An element the rules cannot name leaves it Basic, which is no
			// element; the deck builder refuses such a record long before a fight gets here.
			Element: enemyElement(element),
		},
		Record:   r.Record,
		Name:     r.FullName(),
		Portrait: r.ArtKey(element),
		Element:  element,
	}
	c.CurrentLife = c.MaxLife
	return c
}

// enemyElement is the rules' element for the name a realm dealt, and Basic for one it cannot read.
func enemyElement(name string) combat.Element {
	e, _ := combat.ParseElement(name)
	return e
}

// NewDuelistFrom builds the player from a duelist record and the deck it plays from.
//
// **No sprite, no plan style, and neither is a gap.** The character block replaced the
// fighter's sprite on the combat screen, and a duelist is planned by whoever is holding the
// mouse — which is exactly why the two records split.
func NewDuelistFrom(d data.DuelistData, deck data.DeckData) *Combatant {
	c := &Combatant{
		Duelist: combat.Duelist{
			DMG:     d.DMG,
			Actions: d.Actions,
			MaxLife: d.HP,
		},
		Name: d.Name,
		Deck: deck,
	}
	c.CurrentLife = c.MaxLife
	return c
}
