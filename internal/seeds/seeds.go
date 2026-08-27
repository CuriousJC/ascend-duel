package seeds

import "fmt"

// Stream is one concern that draws randomness. **Append-only, like `combat.Element` and
// `systems.GlyphKind`** — the ordinal indexes the table below, so inserting one mid-list
// re-points every stream after it at the wrong salt, which is a silent reroll of half the game.
type Stream int

const (
	// EnemySelect is the order the tower's opponents are met in, shuffled within each floor
	// band. Per run: rolled once per launch, so a defeat and a retry walk the same order.
	EnemySelect Stream = iota

	// CombatRoll is the rules' own stream — the lightning shock, and nothing else. Per run,
	// but it advances *per attack phase*, so a change early in a duel reshuffles every roll
	// after it.
	CombatRoll

	// PlayerDeck is the player's card shuffle. Per fight.
	PlayerDeck

	// EnemyDeck is the opponent's card shuffle, which must never share with PlayerDeck: the
	// player's opening hand would become a function of how many cards the opponent happened to
	// draw, and every named hand in `internal/screens/seeds.go` would break the first time an
	// enemy deck was retuned.
	EnemyDeck

	// RewardHand is the cards offered for alteration after a fight — a fresh deal off the whole
	// run deck, ignoring whatever the fight left in the piles. Per fight.
	//
	// **It must not share with PlayerDeck**, which is the question this package exists to make
	// people ask: sharing would make the offer a function of how many cards happened to be drawn
	// in the fight just won, so playing a longer duel would change what you were offered for it.
	RewardHand

	// WormOffer is which two alterations are offered after a fight. Per fight.
	//
	// **A separate stream from RewardHand, and the question was asked rather than assumed.** They
	// are drawn from different lists — one from the catalogue, one from the run deck — and they
	// change on different schedules: adding a worm to `data/worms.json` would otherwise reroll
	// which *cards* every fight of every run offered, which is a retune of the catalogue silently
	// changing the game around it. That is the exact failure the salts exist to prevent.
	WormOffer

	// ShopStock is which rings the shop puts up after a fight. Per fight.
	//
	// **Its own stream, and the question was asked.** Sharing WormOffer would draw the shop's
	// three rings off the same sequence as the two worms, so authoring a new worm would change
	// which rings every run was ever offered — the catalogue-retune failure this package exists to
	// prevent, one file over. Sharing RewardHand would make the shelf a function of the run deck's
	// size, which is a thing the player changes on the screen immediately before the shop.
	ShopStock

	// BagStock is which four stones a bag of rocks holds. Per fight.
	//
	// **Its own stream, and the question is the one ShopStock already answered.** Sharing
	// ShopStock would draw the bag off the same sequence as the shelf, so authoring a stone would
	// change which rings every run was ever offered. It is also drawn from a different list on a
	// different schedule: `data/stones.json` grows a record per hand, and `hands.json` is tuned
	// far more often than the rings are.
	BagStock

	// CanStock is which four worms a can of worms holds. Per fight.
	//
	// **Separate from WormOffer even though both draw worms from one catalogue**, which is the
	// case worth stating: they are two draws that happen in one run at two different stations, and
	// sharing the stream would make the shop's four a function of which two the reward screen had
	// already put up. Buying the can would then be able to *guarantee* the pair you had just
	// turned down, or to guarantee it could not appear — a rule nobody designed, arriving out of
	// an implementation detail.
	CanStock

	// BucketStock is which four parasites a bucket of parasites holds. Per fight.
	//
	// **Its own stream, on exactly the argument CanStock is under.** A parasite catalogue and a
	// worm catalogue grow on different schedules and are drawn at the same station, so sharing
	// either of the other two goods' streams would make one good's contents a function of the
	// other's — and authoring a parasite would silently reroll every bag every run has ever
	// opened.
	BucketStock
)

// stream is what the package knows about each one. A table rather than four switch statements,
// so adding a stream is one entry and the tests can walk them all.
type stream struct {
	name string

	// salt separates this stream from every other consumer of the run seed. Seeding two of
	// them from the bare run seed would make them identical sequences, which is the same bug
	// wearing a disguise.
	//
	// **The values are arbitrary and may be changed freely today.** Nothing persists a run, so
	// changing one changes nothing anybody can observe across launches. That stops being true
	// the moment a save file or a shareable seed exists, which is the reason this package was
	// built before those did.
	salt int64

	// perFight says whether the fight index is mixed in. A per-run stream is seeded once and
	// carried; a per-fight stream is a *function* of the fight, which is what makes a defeat
	// and a retry deal that fight again rather than a new one.
	perFight bool
}

var streams = [...]stream{
	EnemySelect: {name: "enemy-select", salt: 0x5EED_E9E3},
	CombatRoll:  {name: "combat-roll", salt: 0x5EED_5C0F},
	PlayerDeck:  {name: "player-deck", salt: 0x5EED_DEC4, perFight: true},
	EnemyDeck:   {name: "enemy-deck", salt: 0x5EED_F0E5, perFight: true},
	RewardHand:  {name: "reward-hand", salt: 0x5EED_A17E, perFight: true},
	WormOffer:   {name: "worm-offer", salt: 0x5EED_7A19, perFight: true},
	ShopStock:   {name: "shop-stock", salt: 0x5EED_5403, perFight: true},
	BagStock:    {name: "bag-stock", salt: 0x5EED_B0C5, perFight: true},
	CanStock:    {name: "can-stock", salt: 0x5EED_CA07, perFight: true},
	BucketStock: {name: "bucket-stock", salt: 0x5EED_B0CC, perFight: true},
}

// fightStride separates one fight's seed from the next within a run. A large odd number so
// consecutive fights are not consecutive seeds — adjacent seeds are fine for math/rand, but a
// stride this size also keeps fight N of one run away from fight N+1 of a run seeded one apart.
const fightStride int64 = 0x2545_F491_4F6C_DD1D

func (s Stream) String() string {
	if s < 0 || int(s) >= len(streams) {
		return fmt.Sprintf("stream(%d)", int(s))
	}
	return streams[s].name
}

// All is every stream in order, for anything that walks them. The tests do; nothing in the game
// needs it, and nothing should — a consumer names the one stream it owns.
func All() []Stream {
	out := make([]Stream, 0, len(streams))
	for i := range streams {
		out = append(out, Stream(i))
	}
	return out
}

// For is the seed for a per-run stream.
//
// **It panics on a per-fight stream**, rather than quietly seeding fight zero. That would look
// like it worked and would hand every fight the same shuffle, which is exactly the class of bug
// this package exists to make unwritable. A programmer error, caught by the first call.
func For(runSeed int64, s Stream) int64 {
	def := lookup(s)
	if def.perFight {
		panic("seeds: " + def.name + " is a per-fight stream, use ForFight")
	}
	return runSeed ^ def.salt
}

// ForFight is the seed for a per-fight stream, taking the **zero-based** fight index the scene
// already carries. The `+1` that keeps fight zero from cancelling the stride lives here rather
// than at the call site, which is where it used to be forgotten.
func ForFight(runSeed int64, s Stream, fightIndex int) int64 {
	def := lookup(s)
	if !def.perFight {
		panic("seeds: " + def.name + " is a per-run stream, use For")
	}
	return runSeed ^ def.salt ^ (int64(fightIndex+1) * fightStride)
}

func lookup(s Stream) stream {
	if s < 0 || int(s) >= len(streams) {
		panic(fmt.Sprintf("seeds: no stream %d", int(s)))
	}
	return streams[s]
}
