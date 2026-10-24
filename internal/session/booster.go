package session

// The boosters: the run's side of what the shop sells sight-unseen.
//
// **A bag of rocks, a vial of essence and a sack of runes** *(owner's call, 2026-08-27)*. Each costs
// five vitae, each holds four of one catalog, and each gives the player exactly one of the four —
// the other three are gone. What is bought is the *choice*, which is what makes them different from
// a relic on the shelf: a relic is a thing you read and then pay for, and these are paid for and
// then read.
//
// **They became a catalog on 2026-09-14.** The name was a constant in `internal/screens`, the price
// and the size a pair of constants here, and the picture a switch — so three cards that are one
// kind of thing were described in three files, and none of the figures could be moved without a
// build. `data/boosters.json` is the record now and this file is where one is refused: the job
// `stone.go`, `essence.go` and `potion.go` each do for their own catalog, and here for their
// reason too — a booster is bought by a *run*, and the rules have never heard of a shop.
//
// **What is inside is still not here.** The four are drawn by the screen from a stream of their
// own when the booster is opened, so a purchase interrupted by a quit leaves nothing to snapshot —
// see `internal/screens/shop_boosters.go`. This file says what stands on the seat and what it costs.

import (
	"fmt"

	"github.com/curiousjc/ascend-duel/data"
)

// BoosterContents is which catalog is inside a booster. **A closed vocabulary**, in Go rather than
// in the file: a fourth catalog is a stream to seed, a dialog to draw and a thing to do with what is
// chosen, never something `boosters.json` can assert into existence. Same posture as a potion's effect
// and an essence's target.
type BoosterContents int

const (
	// ContentsStones is the bag. A stone is spent the moment it is chosen: the click that picks a
	// rock is the click that puts it on the ladder.
	ContentsStones BoosterContents = iota

	// ContentsEssences is the vial. An essence is aimed at a card of the run's deck in the dialog
	// itself, so it too is spent before the dialog closes.
	ContentsEssences

	// ContentsRunes is the sack, and the one whose contents leave the shop with the player rather
	// than being applied on the spot — a rune goes into the sack and is spent between the turns of
	// a fight.
	ContentsRunes

	// ContentsCantrips is the bundle of scrolls. **A cantrip is carried like a rune**, into the
	// scroll case, and cast between the turns of a fight onto the fighter for the rest of it.
	//
	// **The one pack whose draw repeats** *(owner's call, 2026-09-28)*: the catalog is shorter than
	// the smallest bundle, so a bundle is drawn with replacement rather than cut short.
	ContentsCantrips

	// ContentsCards is the action bundle: playing cards out of the duelist's own pool, any of which
	// may come carrying a card trait. **Spent on the spot** — the card taken goes into the run's deck
	// before the dialog closes. See TakeActionCard.
	ContentsCards
)

// Noun is the word the card's face and its tooltip write for what is inside — "4 stones, keep 1".
// **The same value the behavior switches on**, so a face and a dialog cannot name different things.
func (c BoosterContents) Noun() string {
	switch c {
	case ContentsStones:
		return "stones"
	case ContentsRunes:
		return "runes"
	case ContentsCantrips:
		return "cantrips"
	case ContentsCards:
		return "cards"
	default:
		return "essences"
	}
}

// Booster is one booster, resolved against the vocabulary above.
//
// **Not comparable**, unlike Stone, Essence and Potion: it carries its tooltip's lines. Nothing
// holds two to compare — a seat is named by Record, which is a string and is the key everything
// here takes.
type Booster struct {
	Record   string
	Name     string
	Family   string
	Art      string
	Contains BoosterContents
	Size     int
	Price    int

	// Title and Hint are the dialog's heading and the line under it, already resolved: an empty
	// Title in the file becomes the Name here, and an empty Hint becomes DefaultBoosterHint. **Resolved once, at load**, so the screen reads a
	// string rather than deciding what a blank field meant.
	Title string
	Hint  string

	// Tip is what resting on the shelf card says about what is inside. The count and the price are
	// not in it: the screen computes both from the fields above.
	Tip []string
}

// DefaultBoosterHint is the line under a booster's title when its record writes none: the cards on the
// table say what they are, and this says how many of them the player gets.
const DefaultBoosterHint = "Choose One"

// boosters is the validated catalog, in file order, built once at package init.
//
// **A bad record panics at init.** A booster naming a catalog this build has not got is a seat that
// takes five vitae and opens a dialog with nothing in it, which is the same failure an unearnable
// achievement is and takes the same exit.
var boosters = loadBoosters()

// Boosters is the whole catalog, in the order the file writes it. **Which two stand on a shelf is the
// screen's roll**, from a stream of its own — see internal/screens/shop_packs.go.
func Boosters() []Booster {
	out := make([]Booster, len(boosters))
	copy(out, boosters)
	return out
}

// BoosterKeys is every booster's record key, in file order, for anything that walks the catalog.
func BoosterKeys() []string {
	out := make([]string, 0, len(boosters))
	for _, g := range boosters {
		out = append(out, g.Record)
	}
	return out
}

// BoosterByKey finds one by its record key.
func BoosterByKey(key string) (Booster, bool) {
	for _, g := range boosters {
		if g.Record == key {
			return g, true
		}
	}
	return Booster{}, false
}

// BoosterHolding finds **the largest** booster holding one catalog, and whether there is one.
//
// **It returned the first until 2026-09-15**, when that was the same thing: one booster could hold a
// catalog. Now that a catalog can be held at three sizes, "the first" is whichever the file happens
// to list first, and every caller is really asking about the largest — the stone catalog asks how
// many rocks it has to be able to fill, which is a question about the biggest bag, and the two
// review sheets are showing a reader what a catalog is drawn into at its widest. A caller wanting a
// *particular* booster has its record key and should use BoosterByKey.
func BoosterHolding(c BoosterContents) (Booster, bool) {
	var best Booster
	found := false
	for _, g := range boosters {
		if g.Contains != c {
			continue
		}
		if !found || g.Size > best.Size {
			best, found = g, true
		}
	}
	return best, found
}

// CanAffordBooster reports whether the purse covers one. **The question, not the guard** — BuyBooster
// checks the purse itself, exactly as Buy sits beside CanBuy. It exists so a shelf can dim a card
// rather than swallow a click.
func (s *Session) CanAffordBooster(key string) bool {
	g, ok := BoosterByKey(key)
	if !ok {
		return false
	}
	return s.vitae >= s.Price(g.Price)
}

// BuyBooster pays for a booster and reports whether it could.
//
// **The purse moves and nothing else does.** What is inside is drawn by the screen and applied when
// the player picks one, so a purchase interrupted by a quit costs the vitae and hands back nothing —
// which is the same deal a shop makes anywhere. Rolling the contents here would put the offer in the
// run's state and mean snapshotting a bag nobody has opened yet.
func (s *Session) BuyBooster(key string) bool {
	g, ok := BoosterByKey(key)
	if !ok {
		return false
	}
	if !s.SpendVitae(s.Price(g.Price)) {
		return false
	}
	s.count(StatBoosterBought+key, 1)
	return true
}

func loadBoosters() []Booster {
	recs := data.LoadBoosters()
	if len(recs) == 0 {
		panic("boosters.json: the catalog is empty, and the shelf has a pane to fill")
	}

	seen := map[string]bool{}
	holds := map[string]string{}
	out := make([]Booster, 0, len(recs))
	for _, rec := range recs {
		g, err := resolveBooster(rec)
		if err != nil {
			panic("boosters.json: " + err.Error())
		}
		if seen[g.Record] {
			panic("boosters.json: two records keyed " + g.Record)
		}
		seen[g.Record] = true

		// **One booster per catalog *per size*** *(owner's call, 2026-09-15)*. It was one per catalog
		// outright, because the contents are dealt from a stream salted per catalog and two boosters
		// holding stones would have drawn the identical rocks — the second seat being the first one
		// again at a different price. That is fixed at the source rather than forbidden here:
		// `seeds.ForFightSeat` splits the catalog's stream by the booster's own record, so three bags
		// of three sizes deal three unrelated sets.
		//
		// What is still refused is two boosters holding the same catalog at the same size, which is a
		// genuine duplicate however it is priced: the same sealed object twice, and nothing in the
		// shop or the dialog could tell a player which one they were looking at.
		seat := fmt.Sprintf("%s/%d", g.Contains.Noun(), g.Size)
		if first, clash := holds[seat]; clash {
			panic(fmt.Sprintf("boosters.json: %s and %s are both %d %s",
				first, g.Record, g.Size, g.Contains.Noun()))
		}
		holds[seat] = g.Record

		out = append(out, g)
	}
	return out
}

// resolveBooster turns one record into a Booster, or says why it cannot.
func resolveBooster(rec data.BoosterData) (Booster, error) {
	if rec.BoosterRecord == "" {
		return Booster{}, fmt.Errorf("a record with no BoosterRecord")
	}
	if rec.Name == "" {
		return Booster{}, fmt.Errorf("%s: no Name, and the card is headed by one", rec.BoosterRecord)
	}

	contains, err := parseBoosterContents(rec.Contains)
	if err != nil {
		return Booster{}, fmt.Errorf("%s: %w", rec.BoosterRecord, err)
	}

	// **Fewer than two is refused**, because the whole of a booster is that it holds several and you
	// keep one: a booster offering nothing to choose between is a card the player pays for and
	// cannot tell apart from a bug.
	if rec.Size < 2 {
		return Booster{}, fmt.Errorf("%s: holds %d, and what is bought is the choice between them",
			rec.BoosterRecord, rec.Size)
	}
	if rec.Price < 0 {
		return Booster{}, fmt.Errorf("%s: a price of %d", rec.BoosterRecord, rec.Price)
	}

	title := rec.Title
	if title == "" {
		title = rec.Name
	}
	hint := rec.Hint
	if hint == "" {
		hint = DefaultBoosterHint
	}

	return Booster{
		Record:   rec.BoosterRecord,
		Name:     rec.Name,
		Family:   rec.Family,
		Art:      rec.Art,
		Contains: contains,
		Size:     rec.Size,
		Price:    rec.Price,
		Title:    title,
		Hint:     hint,
		Tip:      append([]string(nil), rec.Tip...),
	}, nil
}

// parseBoosterContents reads the one word the file is allowed to write for what is inside.
func parseBoosterContents(word string) (BoosterContents, error) {
	switch word {
	case "stones":
		return ContentsStones, nil
	case "essences":
		return ContentsEssences, nil
	case "runes":
		return ContentsRunes, nil
	case "cantrips":
		return ContentsCantrips, nil
	case "cards":
		return ContentsCards, nil
	case "":
		return 0, fmt.Errorf("no Contains, and a booster is what is inside it")
	default:
		return 0, fmt.Errorf("a Contains of %q, which is no catalog this build has", word)
	}
}
