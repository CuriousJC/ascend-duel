package session

// relic_trait.go: data/relic_traits.json resolved against the rules and registered with them —
// every relic trait, what it does, and how often the shelf sells a relic carrying it.
//
// **Parsed here rather than in `data`** for the reason relics.json is: a trait's Rules are the relic
// grammar, resolved by parseRelicRules, and `data` may not import `internal/combat`. A bad word, a
// rule the grammar refuses, or a fight-long trait the shelf would sell panics at load.

import (
	"fmt"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
)

// ShelfTrait is one relic trait the shelf may sell, its chance out of 100 per seat, and its record.
type ShelfTrait struct {
	Trait   combat.RelicTrait
	Percent int
	Record  data.RelicTraitData
}

// shelfTraits is every trait the file prices, in file order — the order each seat rolls them.
var shelfTraits = registerRelicTraits(data.LoadRelicTraits())

func registerRelicTraits(list []data.RelicTraitData) []ShelfTrait {
	out := make([]ShelfTrait, 0, len(list))
	for _, r := range list {
		if r.Text == "" {
			panic(fmt.Sprintf("relic_traits.json: %q says nothing about what it does", r.RelicTrait))
		}
		key := "trait." + r.RelicTrait
		rules, err := parseRelicRules(key, r.Rules)
		if err != nil {
			panic("relic_traits.json: " + err.Error())
		}
		slot := r.Slot == nil || *r.Slot
		t, err := combat.RegisterRelicTrait(r.RelicTrait, r.Name, slot, r.LastsFight(), rules)
		if err != nil {
			panic("relic_traits.json: " + err.Error())
		}
		if r.LastsFight() && r.Percent != 0 {
			panic(fmt.Sprintf("relic_traits.json: %q lasts one fight and is sold at %d%%, "+
				"and a bought relic gone after one fight is vitae spent on nothing", r.RelicTrait, r.Percent))
		}
		out = append(out, ShelfTrait{Trait: t, Percent: r.Percent, Record: r})
	}
	return out
}

// ShelfTraits is every relic trait with its shelf odds, in file order.
func ShelfTraits() []ShelfTrait { return shelfTraits }

// RelicTraitNamed is the trait a key names. **It panics on a key the file does not hold**, because
// every caller names one the game is built around — a cantrip-relic is "weightless" and "ephemeral" —
// and a file without it is a game that cannot cast.
func RelicTraitNamed(key string) combat.RelicTrait {
	t, ok := combat.ParseRelicTrait(key)
	if !ok {
		panic(fmt.Sprintf("relic_traits.json has no %q, and the game needs it", key))
	}
	return t
}
