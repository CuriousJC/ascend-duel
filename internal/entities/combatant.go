package entities

import (
	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
)

// Combatant is a duelist that can be drawn. The stats live in the embedded
// combat.Duelist so the rules engine can take them without ever seeing a sprite;
// the promoted fields mean gs.Fighter.DMG and friends still read the same.
type Combatant struct {
	combat.Duelist

	// Record is which roster entry this combatant was built from, and it is what names its deck
	// in `internal/decks` *(2026-08-16)*. It replaced `Style combat.PlanStyle`: an enemy's
	// behaviour used to be a string picking one of four planners and is now the cards it holds,
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

	// CardBack is the mark on the back of this duelist's cards, by name. A string rather
	// than a cards.BackMark because entities must not import the drawing package — the
	// screen parses it with cards.ParseBackMark, exactly as it parses an element.
	//
	// Empty for an enemy: enemies do not have a deck the player ever sees the back of.
	CardBack string
}

// NewEnemyFrom builds an opponent from an enemy record.
//
// **It takes no sprite sheet since 2026-08-11.** It used to slice a west-facing idle frame
// out of one, which is why the caller had to resolve an asset out of global state and pass
// it in; the enemy is a card now, so all that is left is the portrait's key.
func NewEnemyFrom(d data.EnemyData) *Combatant {
	c := &Combatant{
		Duelist: combat.Duelist{
			DMG:     d.DMG,
			Actions: d.Actions,
			MaxLife: d.HP,
		},
		Record:   d.EnemyRecord,
		Name:     d.Name,
		Portrait: d.Portrait,
	}
	c.CurrentLife = c.MaxLife
	return c
}

// NewDuelistFrom builds the player from a duelist record.
//
// **No sprite, no plan style, and neither is a gap.** The character block replaced the
// fighter's sprite on the combat screen, and a duelist is planned by whoever is holding the
// mouse — which is exactly why the two records split.
func NewDuelistFrom(d data.DuelistData) *Combatant {
	c := &Combatant{
		Duelist: combat.Duelist{
			DMG:     d.DMG,
			Actions: d.Actions,
			MaxLife: d.HP,
		},
		Name:     d.Name,
		CardBack: d.CardBack,
	}
	c.CurrentLife = c.MaxLife
	return c
}
