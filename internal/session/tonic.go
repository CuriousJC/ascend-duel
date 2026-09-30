package session

// Tonics: the run's own rules, bought one at a time and never taken back.
//
// **A potion changes the duelist's body; a tonic changes the container the run is played in** —
// what the shop charges, what a card costs, how long a fight lasts, how many cards a hand holds, how
// many relics may be worn. It is drunk on the spot and lasts for the rest of the run.
//
// **The shop offers one per realm.** The catalog is shuffled once off the run seed
// (`seeds.TonicOrder`) and each realm takes the first tonic in that order the run can still be
// offered. A tonic is offered at most once: bought or passed over, it never comes back. A tonic the
// run is not yet allowed — one whose `Requires` has not been drunk — is skipped rather than spent,
// so it stays in the order and is offered by a later realm once it is allowed.
//
// **The order is a permutation of the catalog, not a fixed-length list.** Walking it from the top
// every realm and skipping what has been offered is what lets a skipped tonic come back, and a
// catalog of N can never be walked past the end — a run simply stops being offered tonics once
// every one has been.
//
// **What a run carries is the list it drank and the list it was offered, and every effect is read
// off the first.** Nothing here writes to the run's own round limit or relic slots, so a resumed
// run cannot apply a tonic twice: the figures are the base plus what the drunk list adds, asked
// fresh each time.

import (
	"fmt"
	"math/rand"
	"sort"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/journey"
	"github.com/curiousjc/ascend-duel/internal/seeds"
)

// TonicEffect is which of the run's rules a tonic moves. **A closed vocabulary**, in Go rather than
// in the file: a new effect is a rule the run has to learn how to apply.
type TonicEffect int

const (
	// TonicDiscount takes Amount percent off every price the shop charges, rounded up to at least
	// one vitae, and never below a price of one.
	TonicDiscount TonicEffect = iota

	// TonicCostCut takes Amount action points off every card the duelist plays, never below one.
	TonicCostCut

	// TonicPressure takes pressureRounds off every fight's round limit and multiplies every vitae
	// the run earns by Amount.
	TonicPressure

	// TonicHandSize deals Amount more cards into every hand.
	TonicHandSize

	// TonicDiscards gives Amount more discards every round.
	TonicDiscards

	// TonicRelicSlots lets the run wear Amount more relics.
	TonicRelicSlots
)

// pressureRounds is how many rounds a Pressure tonic takes off the clock.
const pressureRounds = 1

// Tonic is one record, resolved against the vocabulary above. Comparable, like a Potion.
type Tonic struct {
	Record string
	Name   string

	// Family and Art are read by the shelf and the review sheets and by nothing that resolves a
	// fight. Art is already resolved through data.TonicData.ArtKey.
	Family string
	Art    string

	Effect   TonicEffect
	Amount   int
	Requires string
	Price    int
}

// tonics is the validated catalog, in file order, built once at package init. **A bad record
// panics at init**, as a bad potion does.
var tonics = loadTonics()

// Tonics is the whole catalog, in file order. **Nothing that decides an outcome walks it** — the
// offer walks tonicOrder.
func Tonics() []Tonic {
	out := make([]Tonic, len(tonics))
	copy(out, tonics)
	return out
}

// TonicByKey finds one by its record key.
func TonicByKey(key string) (Tonic, bool) {
	for _, t := range tonics {
		if t.Record == key {
			return t, true
		}
	}
	return Tonic{}, false
}

func loadTonics() []Tonic {
	recs := data.LoadTonics()
	if len(recs) == 0 {
		panic("tonics.json: the catalog is empty, and the shop has a seat to fill")
	}

	seen := map[string]bool{}
	out := make([]Tonic, 0, len(recs))
	for _, rec := range recs {
		t, err := resolveTonic(rec)
		if err != nil {
			panic("tonics.json: " + err.Error())
		}
		if seen[t.Record] {
			panic("tonics.json: two records keyed " + t.Record)
		}
		seen[t.Record] = true
		out = append(out, t)
	}

	// **A requirement naming nothing is refused**, because a tonic waiting on a record that does
	// not exist is a tonic no run is ever offered, and nothing else would notice.
	for _, t := range out {
		if t.Requires == "" {
			continue
		}
		if t.Requires == t.Record {
			panic("tonics.json: " + t.Record + " requires itself")
		}
		if !seen[t.Requires] {
			panic(fmt.Sprintf("tonics.json: %s requires %q, which is in no tonic record", t.Record, t.Requires))
		}
	}
	return out
}

// resolveTonic turns one record into a Tonic, or says why it cannot.
func resolveTonic(rec data.TonicData) (Tonic, error) {
	if rec.TonicRecord == "" {
		return Tonic{}, fmt.Errorf("a record with no TonicRecord")
	}
	effect, err := parseTonicEffect(rec.Effect)
	if err != nil {
		return Tonic{}, fmt.Errorf("%s: %w", rec.TonicRecord, err)
	}
	if rec.Amount <= 0 {
		return Tonic{}, fmt.Errorf("%s: an amount of %d does nothing", rec.TonicRecord, rec.Amount)
	}
	switch {
	case effect == TonicDiscount && rec.Amount >= 100:
		return Tonic{}, fmt.Errorf("%s: %d%% off is not a discount the shop can give", rec.TonicRecord, rec.Amount)
	case effect == TonicPressure && rec.Amount < 2:
		return Tonic{}, fmt.Errorf("%s: a pressure tonic multiplying vitae by %d pays nothing for the round it takes",
			rec.TonicRecord, rec.Amount)
	}
	if rec.Price <= 0 {
		return Tonic{}, fmt.Errorf("%s: a price of %d is not something the shop can charge", rec.TonicRecord, rec.Price)
	}
	return Tonic{
		Record:   rec.TonicRecord,
		Name:     rec.Name,
		Family:   rec.Family,
		Art:      rec.ArtKey(),
		Effect:   effect,
		Amount:   rec.Amount,
		Requires: rec.Requires,
		Price:    rec.Price,
	}, nil
}

// String is the effect as tonics.json writes it.
func (e TonicEffect) String() string {
	switch e {
	case TonicDiscount:
		return "discount"
	case TonicCostCut:
		return "cost-cut"
	case TonicPressure:
		return "pressure"
	case TonicHandSize:
		return "hand-size"
	case TonicDiscards:
		return "discards"
	case TonicRelicSlots:
		return "relic-slots"
	default:
		return fmt.Sprintf("TonicEffect(%d)", int(e))
	}
}

// TonicEffects is every effect in the vocabulary, in declaration order, for a page listing which
// ones the catalog reaches for.
func TonicEffects() []TonicEffect {
	return []TonicEffect{TonicDiscount, TonicCostCut, TonicPressure, TonicHandSize, TonicDiscards,
		TonicRelicSlots}
}

func parseTonicEffect(name string) (TonicEffect, error) {
	switch name {
	case "discount":
		return TonicDiscount, nil
	case "cost-cut":
		return TonicCostCut, nil
	case "pressure":
		return TonicPressure, nil
	case "hand-size":
		return TonicHandSize, nil
	case "discards":
		return TonicDiscards, nil
	case "relic-slots":
		return TonicRelicSlots, nil
	default:
		return 0, fmt.Errorf("%q is not an effect a tonic has", name)
	}
}

// newTonicOrder is the order a run walks the catalog in.
//
// **One function so a resumed run and a new one cannot shuffle it differently.** The order is not
// saved — it is rebuilt from the run code, as the journey is. It shuffles the *sorted* keys, so
// reordering tonics.json rerolls nothing; adding or removing a record does.
func newTonicOrder(runSeed int64) []string {
	keys := make([]string, 0, len(tonics))
	for _, t := range tonics {
		keys = append(keys, t.Record)
	}
	sort.Strings(keys)
	rng := rand.New(rand.NewSource(seeds.For(runSeed, seeds.TonicOrder)))
	rng.Shuffle(len(keys), func(i, j int) { keys[i], keys[j] = keys[j], keys[i] })
	return keys
}

// shopRealm is the realm the shop the run is standing in belongs to. **The fight counter has
// already moved on** by the time a shop is up — WonFight advances it — so the shop after a realm's
// portal room is still that realm's, and the next realm starts once the portal has been walked.
func (s *Session) shopRealm() int { return journey.RealmOf(s.fight - 1) }

// OfferTonic settles which tonic the current realm offers, if it has not been settled yet, and
// returns it. **Called by the shop on entry**; a realm's offer is decided once, at its first shop,
// and every later shop in that realm shows the same one — or an empty seat, once it is drunk.
//
// It reports false for a realm with nothing to offer: every tonic already offered, or none the run
// is allowed yet.
func (s *Session) OfferTonic() (Tonic, bool) {
	realm := s.shopRealm()
	for len(s.tonicOffers) < realm {
		s.tonicOffers = append(s.tonicOffers, "")
	}
	if key := s.tonicOffers[realm-1]; key != "" {
		return s.tonicOnOffer(key)
	}

	// **A realm that found nothing is asked again at its next shop, and finds nothing again**: what
	// the walk depends on is what has been offered and drunk, and neither can change inside a realm
	// that had no offer to buy.

	offered := map[string]bool{}
	for _, k := range s.tonicOffers {
		offered[k] = true
	}
	for _, k := range s.drunkTonics {
		offered[k] = true
	}
	for _, key := range s.tonicOrder {
		if offered[key] {
			continue
		}
		t, ok := TonicByKey(key)
		if !ok || (t.Requires != "" && !s.Drank(t.Requires)) {
			continue
		}
		s.tonicOffers[realm-1] = key
		return s.tonicOnOffer(key)
	}
	return Tonic{}, false
}

// TonicOnOffer is the current realm's tonic, if it has been settled and not yet drunk. It never
// settles one; see OfferTonic.
func (s *Session) TonicOnOffer() (Tonic, bool) {
	realm := s.shopRealm()
	if realm < 1 || len(s.tonicOffers) < realm {
		return Tonic{}, false
	}
	return s.tonicOnOffer(s.tonicOffers[realm-1])
}

func (s *Session) tonicOnOffer(key string) (Tonic, bool) {
	if key == "" || s.Drank(key) {
		return Tonic{}, false
	}
	return TonicByKey(key)
}

// Drank reports whether the run has drunk that tonic.
func (s *Session) Drank(key string) bool {
	for _, k := range s.drunkTonics {
		if k == key {
			return true
		}
	}
	return false
}

// DrunkTonics is every tonic the run has drunk, in the order it drank them, as a copy.
func (s *Session) DrunkTonics() []string { return append([]string(nil), s.drunkTonics...) }

// CanDrinkTonic is whether the tonic is the one on offer and the purse covers its price. The
// question, not the guard — DrinkTonic checks both itself.
func (s *Session) CanDrinkTonic(key string) bool {
	t, ok := s.TonicOnOffer()
	return ok && t.Record == key && s.vitae >= s.Price(t.Price)
}

// DrinkTonic pays for the tonic on offer and drinks it, and reports whether it could. **The purse
// moves first**, as every purchase's does.
func (s *Session) DrinkTonic(key string) bool {
	t, ok := s.TonicOnOffer()
	if !ok || t.Record != key || !s.SpendVitae(s.Price(t.Price)) {
		return false
	}
	s.drunkTonics = append(s.drunkTonics, key)
	return true
}

// tonicTotal sums Amount over every drunk tonic with the given effect.
func (s *Session) tonicTotal(e TonicEffect) int {
	n := 0
	for _, key := range s.drunkTonics {
		if t, ok := TonicByKey(key); ok && t.Effect == e {
			n += t.Amount
		}
	}
	return n
}

// Price is what the shop charges for something listed at base, after every discount the run has
// drunk. **Every price in the shop goes through here** — relics, packs, potions, rerolls, tonics —
// and nothing a player sells does: a sale is a trade, not a purchase.
//
// Each discount is its percentage of the price, rounded up, so it is always worth at least one
// vitae; a price never falls below one. A free thing — base zero — stays free.
func (s *Session) Price(base int) int {
	if base <= 0 {
		return base
	}
	price := base
	for _, key := range s.drunkTonics {
		t, ok := TonicByKey(key)
		if !ok || t.Effect != TonicDiscount {
			continue
		}
		price -= DiscountCut(base, t.Amount)
	}
	if price < 1 {
		price = 1
	}
	return price
}

// DiscountCut is what pct percent off a price of base takes off it: the share rounded up, so any
// discount on any price is worth at least one vitae. Price applies the floor of one.
func DiscountCut(base, pct int) int {
	if base <= 0 || pct <= 0 {
		return 0
	}
	return (base*pct + 99) / 100
}

// CostCut is how many action points come off every card the run plays. See combat.CostWith.
func (s *Session) CostCut() int { return s.tonicTotal(TonicCostCut) }

// VitaeFactor is what every vitae the run earns is multiplied by: the product of every pressure
// tonic drunk, one when there are none.
func (s *Session) VitaeFactor() int {
	f := 1
	for _, key := range s.drunkTonics {
		if t, ok := TonicByKey(key); ok && t.Effect == TonicPressure {
			f *= t.Amount
		}
	}
	return f
}

// earn is AddVitae for vitae the run *earned* — a win, a rune — through VitaeFactor. A sale is not
// earned and goes through AddVitae at face value.
func (s *Session) earn(n int) { s.AddVitae(n * s.VitaeFactor()) }

// HandSize is how many cards a hand of this run is dealt to: base plus every hand-size tonic.
func (s *Session) HandSize(base int) int { return base + s.tonicTotal(TonicHandSize) }

// Discards is how many discards a round of this run allows: base plus every discards tonic.
func (s *Session) Discards(base int) int { return base + s.tonicTotal(TonicDiscards) }

// tonicRounds is what the drunk tonics do to the round limit.
func (s *Session) tonicRounds() int {
	n := 0
	for _, key := range s.drunkTonics {
		if t, ok := TonicByKey(key); ok && t.Effect == TonicPressure {
			n -= pressureRounds
		}
	}
	return n
}

// resumeTonics puts a saved run's tonics back. **A tonic the catalog no longer holds is refused**,
// on the terms a relic is: a run resumed without a rule it paid for is a run the player would have
// to work out had changed. An offered key the catalog has lost is forgotten, since what it cost was
// a seat rather than vitae.
func (s *Session) resumeTonics(drunk, offers []string) error {
	for _, key := range drunk {
		if _, ok := TonicByKey(key); !ok {
			return fmt.Errorf("tonic %q is not one this build has", key)
		}
		if s.Drank(key) {
			return fmt.Errorf("tonic %q was drunk twice", key)
		}
		s.drunkTonics = append(s.drunkTonics, key)
	}
	for _, key := range offers {
		if _, ok := TonicByKey(key); !ok {
			key = ""
		}
		s.tonicOffers = append(s.tonicOffers, key)
	}
	return nil
}
