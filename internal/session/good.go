package session

// The sealed goods: the run's side of what the shop sells sight-unseen.
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
// build. `data/goods.json` is the record now and this file is where one is refused: the job
// `stone.go`, `essence.go` and `potion.go` each do for their own catalog, and here for their
// reason too — a good is bought by a *run*, and the rules have never heard of a shop.
//
// **What is inside is still not here.** The four are drawn by the screen from a stream of their
// own when the good is opened, so a purchase interrupted by a quit leaves nothing to snapshot —
// see `internal/screens/shop_goods.go`. This file says what stands on the seat and what it costs.

import (
	"fmt"

	"github.com/curiousjc/ascend-duel/data"
)

// GoodContents is which catalog is inside a sealed good. **A closed vocabulary**, in Go rather than
// in the file: a fourth catalog is a stream to seed, a dialog to draw and a thing to do with what is
// chosen, never something `goods.json` can assert into existence. Same posture as a potion's effect
// and an essence's target.
type GoodContents int

const (
	// ContentsStones is the bag. A stone is spent the moment it is chosen: the click that picks a
	// rock is the click that puts it on the ladder.
	ContentsStones GoodContents = iota

	// ContentsEssences is the vial. An essence is aimed at a card of the run's deck in the dialog
	// itself, so it too is spent before the dialog closes.
	ContentsEssences

	// ContentsRunes is the sack, and the one whose contents leave the shop with the player rather
	// than being applied on the spot — a rune goes into the sack and is spent between the turns of
	// a fight.
	ContentsRunes
)

// Noun is the word the card's face and its tooltip write for what is inside — "4 stones, keep 1".
// **The same value the behavior switches on**, so a face and a dialog cannot name different things.
func (c GoodContents) Noun() string {
	switch c {
	case ContentsStones:
		return "stones"
	case ContentsRunes:
		return "runes"
	default:
		return "essences"
	}
}

// Good is one sealed good, resolved against the vocabulary above.
//
// **Not comparable**, unlike Stone, Essence and Potion: it carries its tooltip's lines. Nothing
// holds two to compare — a seat is named by Record, which is a string and is the key everything
// here takes.
type Good struct {
	Record   string
	Name     string
	Family   string
	Art      string
	Contains GoodContents
	Size     int
	Price    int

	// Title and Hint are the dialog's heading and the line under it, already resolved: an empty
	// Title in the file becomes the Name here, and an empty Hint becomes the computed
	// "take one of the four, the rest are gone". **Resolved once, at load**, so the screen reads a
	// string rather than deciding what a blank field meant.
	Title string
	Hint  string

	// Tip is what resting on the shelf card says about what is inside. The count and the price are
	// not in it: the screen computes both from the fields above.
	Tip []string
}

// goods is the validated catalog, in file order, built once at package init.
//
// **A bad record panics at init.** A good naming a catalog this build has not got is a seat that
// takes five vitae and opens a dialog with nothing in it, which is the same failure an unearnable
// achievement is and takes the same exit.
var goods = loadGoods()

// Goods is the whole catalog, in the order the file writes it. **Which two stand on a shelf is the
// screen's roll**, from a stream of its own — see internal/screens/shop_packs.go.
func Goods() []Good {
	out := make([]Good, len(goods))
	copy(out, goods)
	return out
}

// GoodKeys is every good's record key, in file order, for anything that walks the catalog.
func GoodKeys() []string {
	out := make([]string, 0, len(goods))
	for _, g := range goods {
		out = append(out, g.Record)
	}
	return out
}

// GoodByKey finds one by its record key.
func GoodByKey(key string) (Good, bool) {
	for _, g := range goods {
		if g.Record == key {
			return g, true
		}
	}
	return Good{}, false
}

// GoodHolding finds **the largest** good holding one catalog, and whether there is one.
//
// **It returned the first until 2026-09-15**, when that was the same thing: one good could hold a
// catalog. Now that a catalog can be held at three sizes, "the first" is whichever the file happens
// to list first, and every caller is really asking about the largest — the stone catalog asks how
// many rocks it has to be able to fill, which is a question about the biggest bag, and the two
// review sheets are showing a reader what a catalog is drawn into at its widest. A caller wanting a
// *particular* good has its record key and should use GoodByKey.
func GoodHolding(c GoodContents) (Good, bool) {
	var best Good
	found := false
	for _, g := range goods {
		if g.Contains != c {
			continue
		}
		if !found || g.Size > best.Size {
			best, found = g, true
		}
	}
	return best, found
}

// CanAffordGood reports whether the purse covers one. **The question, not the guard** — BuyGood
// checks the purse itself, exactly as Buy sits beside CanBuy. It exists so a shelf can dim a card
// rather than swallow a click.
func (s *Session) CanAffordGood(key string) bool {
	g, ok := GoodByKey(key)
	if !ok {
		return false
	}
	return s.vitae >= g.Price
}

// BuyGood pays for a sealed good and reports whether it could.
//
// **The purse moves and nothing else does.** What is inside is drawn by the screen and applied when
// the player picks one, so a purchase interrupted by a quit costs the vitae and hands back nothing —
// which is the same deal a shop makes anywhere. Rolling the contents here would put the offer in the
// run's state and mean snapshotting a bag nobody has opened yet.
func (s *Session) BuyGood(key string) bool {
	g, ok := GoodByKey(key)
	if !ok {
		return false
	}
	return s.SpendVitae(g.Price)
}

func loadGoods() []Good {
	recs := data.LoadGoods()
	if len(recs) == 0 {
		panic("goods.json: the catalog is empty, and the shelf has a pane to fill")
	}

	seen := map[string]bool{}
	holds := map[string]string{}
	out := make([]Good, 0, len(recs))
	for _, rec := range recs {
		g, err := resolveGood(rec)
		if err != nil {
			panic("goods.json: " + err.Error())
		}
		if seen[g.Record] {
			panic("goods.json: two records keyed " + g.Record)
		}
		seen[g.Record] = true

		// **One good per catalog *per size*** *(owner's call, 2026-09-15)*. It was one per catalog
		// outright, because the contents are dealt from a stream salted per catalog and two goods
		// holding stones would have drawn the identical rocks — the second seat being the first one
		// again at a different price. That is fixed at the source rather than forbidden here:
		// `seeds.ForFightSeat` splits the catalog's stream by the good's own record, so three bags
		// of three sizes deal three unrelated sets.
		//
		// What is still refused is two goods holding the same catalog at the same size, which is a
		// genuine duplicate however it is priced: the same sealed object twice, and nothing in the
		// shop or the dialog could tell a player which one they were looking at.
		seat := fmt.Sprintf("%s/%d", g.Contains.Noun(), g.Size)
		if first, clash := holds[seat]; clash {
			panic(fmt.Sprintf("goods.json: %s and %s are both %d %s",
				first, g.Record, g.Size, g.Contains.Noun()))
		}
		holds[seat] = g.Record

		out = append(out, g)
	}
	return out
}

// resolveGood turns one record into a Good, or says why it cannot.
func resolveGood(rec data.GoodData) (Good, error) {
	if rec.GoodRecord == "" {
		return Good{}, fmt.Errorf("a record with no GoodRecord")
	}
	if rec.Name == "" {
		return Good{}, fmt.Errorf("%s: no Name, and the card is headed by one", rec.GoodRecord)
	}

	contains, err := parseGoodContents(rec.Contains)
	if err != nil {
		return Good{}, fmt.Errorf("%s: %w", rec.GoodRecord, err)
	}

	// **Fewer than two is refused**, because the whole of a good is that it holds several and you
	// keep one: a sealed good offering nothing to choose between is a card the player pays for and
	// cannot tell apart from a bug.
	if rec.Size < 2 {
		return Good{}, fmt.Errorf("%s: holds %d, and what is bought is the choice between them",
			rec.GoodRecord, rec.Size)
	}
	if rec.Price < 0 {
		return Good{}, fmt.Errorf("%s: a price of %d", rec.GoodRecord, rec.Price)
	}

	title := rec.Title
	if title == "" {
		title = rec.Name
	}
	hint := rec.Hint
	if hint == "" {
		hint = fmt.Sprintf("take one of the %d, the rest are gone", rec.Size)
	}

	return Good{
		Record:   rec.GoodRecord,
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

// parseGoodContents reads the one word the file is allowed to write for what is inside.
func parseGoodContents(word string) (GoodContents, error) {
	switch word {
	case "stones":
		return ContentsStones, nil
	case "essences":
		return ContentsEssences, nil
	case "runes":
		return ContentsRunes, nil
	case "":
		return 0, fmt.Errorf("no Contains, and a good is what is inside it")
	default:
		return 0, fmt.Errorf("a Contains of %q, which is no catalog this build has", word)
	}
}
