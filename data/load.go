package data

// The four things every catalog loader does, written once.
//
// **Fifteen `Load*` functions were fifteen copies of the same eight lines** — unmarshal the
// embedded bytes, panic on failure, walk the list into a map keyed by the record's own key field —
// and six `*Order` functions were the same walk-and-sort. That is not a tidiness complaint: a body
// copied fifteen times is a place where fourteen of them silently do not get a fix. Two of the
// fifteen checked for a duplicate key and thirteen did not, so a catalog authored with the same key
// twice lost a record with nothing failing, and which catalogs those were was an accident of which
// one somebody happened to be looking at when the check was written.
//
// **Every catalog is checked now**, which is the one behavior this consolidation changed. It can
// only fire on a file that is already broken, and no shipped catalog trips it.
//
// **A loader still lives beside its own records.** What moved here is the plumbing; the struct, the
// field tags, the validation that is about *this* catalog's vocabulary and the doc comment
// explaining what the file is all stay in `<name>_data.go`, because that is what somebody reads when
// they are authoring a record.

import (
	"encoding/json"
	"maps"
	"slices"
)

// parse unmarshals an embedded catalog, naming the file in the panic.
//
// **A panic rather than an error, like everything else in this package.** These bytes are
// `//go:embed`ed, so a failure here is a malformed file that shipped inside the binary — there is
// no recovery and no caller who could do anything useful with an error. It is the same call
// `internal/combat` makes when a card names a verb the rules do not have: refuse the launch rather
// than run a game missing a catalog.
//
// The file name is the whole of the message because that is what the author needs; the previous
// wording drifted between naming the file and naming the Go type, so half the panics told you which
// file to open and half told you which struct to go and find.
func parse[T any](raw []byte, file string) []T {
	var list []T
	if err := json.Unmarshal(raw, &list); err != nil {
		panic(file + ": " + err.Error())
	}
	return list
}

// parseOne unmarshals a catalog that is a single JSON object rather than a list of records.
//
// Two files are shaped this way and both for a reason: hands.json wraps its ladder so the file can
// carry something about the ladder as a whole, and tutorial.json is one script rather than a
// catalog of anything.
func parseOne[T any](raw []byte, file string) T {
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		panic(file + ": " + err.Error())
	}
	return out
}

// mustBeUniquelyKeyed is keyed's check for a catalog that stays a slice.
//
// **Order is what these keep** — an achievement is drawn in the order the file writes it — so they
// cannot go through a map, and the uniqueness a map gets for free has to be asked for.
func mustBeUniquelyKeyed[T any](list []T, file string, key func(T) string) []T {
	seen := make(map[string]bool, len(list))
	for _, rec := range list {
		k := key(rec)
		if k == "" {
			panic(file + ": a record has no key")
		}
		if seen[k] {
			panic(file + ": two records keyed " + k)
		}
		seen[k] = true
	}
	return list
}

// keyed parses a catalog into a map by each record's own key, refusing a duplicate.
//
// **A duplicate is refused rather than resolved**, because either resolution is wrong: last-wins
// silently deletes the record somebody wrote first, and first-wins silently ignores the edit they
// just made. Neither fails a test — the catalog is simply one record short, in a game where nobody
// counts the relics — so the only honest answer is to not start.
//
// An empty key is the same failure wearing a different hat: every record in `data/` is reached by
// its key, so one without a key is a record nothing can ever ask for.
func keyed[T any](raw []byte, file string, key func(T) string) map[string]T {
	list := mustBeUniquelyKeyed(parse[T](raw, file), file, key)
	out := make(map[string]T, len(list))
	for _, rec := range list {
		out[key(rec)] = rec
	}
	return out
}

// fileOrder is every record's key in the order the file writes them.
//
// **This is the motif order a catalog is authored in** — the relics' flips together, the ring
// families in ladder order, the weapons along the concept ladder — which is information a sorted
// walk throws away. The review sheets group by `Family` and walk this, so a page reads as the file
// does.
//
// It re-parses rather than being an ordering stored on the map, because a map has no order to store
// one on and every other caller wants the sorted keys.
func fileOrder[T any](raw []byte, file string, key func(T) string) []string {
	list := parse[T](raw, file)
	out := make([]string, 0, len(list))
	for _, rec := range list {
		out = append(out, key(rec))
	}
	return out
}

// sortedKeys is every key in a catalog, sorted.
//
// **Sorted because the loaders return maps and Go randomizes that order deliberately.** For the
// sheets that is a layout that would deal a different row every run; for the shop it is worse than
// that, because what a sack or a shelf holds is a shuffle of one of these lists — an unsorted walk
// would make a purchase depend on map iteration and take the run's reproducibility with it. See
// the `randomness` skill.
//
// **By key rather than by name or element**: the key is the one field guaranteed unique, and a sort
// on something that can tie is a sort that can still shuffle. A catalog wanting a different order —
// the two rosters sort by floor — writes its own comparison and falls back to the key for the same
// reason.
func sortedKeys[T any](m map[string]T) []string {
	return slices.Sorted(maps.Keys(m))
}
