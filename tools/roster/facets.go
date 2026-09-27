package roster

import (
	"sort"
	"strconv"
	"strings"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/tools/sheetfilter"
)

// The three axes the roster is narrowed on, and the attributes the page tags with.
//
// **Every value is counted off the entries rather than listed here.** A motif authored onto a
// ninth realm puts a ninth chip on the page with nothing edited, and an element no record can be
// dealt as is not offered — which is the same rule the coverage grid is under: the page reports
// the catalog, it does not describe it.
//
// **Realm is a fact about the motif and the other two are facts about the record**, which is what
// the group and item split in sheetfilter is for: the realm chips cut whole sections out, and tier
// and element cut plates inside whatever sections are left.

// facetsFor builds the chip bar's contents out of the catalog it is about to draw.
func facetsFor(entries []Entry) []sheetfilter.Facet {
	var out []sheetfilter.Facet
	if f, ok := realmFacet(entries); ok {
		out = append(out, f)
	}
	if f, ok := tierFacet(entries); ok {
		out = append(out, f)
	}
	if f, ok := elementFacet(entries); ok {
		out = append(out, f)
	}
	return out
}

// realmFacet offers every realm any motif can theme, ascending, with the number of records that
// can be met on it.
func realmFacet(entries []Entry) (sheetfilter.Facet, bool) {
	lo, hi := span(entries)
	counts := map[int]int{}
	for _, e := range entries {
		for _, f := range realmsOf(e.Band, lo, hi) {
			counts[f]++
		}
	}
	if len(counts) == 0 {
		return sheetfilter.Facet{}, false
	}

	realms := make([]int, 0, len(counts))
	for f := range counts {
		realms = append(realms, f)
	}
	sort.Ints(realms)

	f := sheetfilter.Facet{Key: "realm", Label: "realm"}
	for _, n := range realms {
		f.Values = append(f.Values, sheetfilter.Value{
			Value: strconv.Itoa(n), Label: strconv.Itoa(n), Count: counts[n],
		})
	}
	return f, true
}

// tierFacet offers the three rooms of a realm, in the order they are fought.
func tierFacet(entries []Entry) (sheetfilter.Facet, bool) {
	counts := map[string]int{}
	for _, e := range entries {
		if e.Tier != "" {
			counts[e.Tier]++
		}
	}
	f := sheetfilter.Facet{Key: "tier", Label: "room"}
	for _, tier := range data.TierOrder {
		if counts[tier] == 0 {
			continue
		}
		f.Values = append(f.Values, sheetfilter.Value{Value: tier, Label: tier, Count: counts[tier]})
	}
	return f, len(f.Values) > 0
}

// elementFacet offers every colour a record on this page can be dealt as.
//
// **A record counts once per element it can wear**, not once overall, so the figures add up to
// more than the catalog — which is the true statement about a roster where one creature is five
// cards.
func elementFacet(entries []Entry) (sheetfilter.Facet, bool) {
	counts := map[string]int{}
	for _, e := range entries {
		for _, el := range e.Elements {
			counts[el]++
		}
	}
	f := sheetfilter.Facet{Key: "element", Label: "element"}
	for _, el := range data.AffinityElements {
		if counts[el] == 0 {
			continue
		}
		f.Values = append(f.Values, sheetfilter.Value{Value: el, Label: el, Count: counts[el]})
	}
	return f, len(f.Values) > 0
}

// realmsOf writes a band out as the realms in it, against the page's own span.
//
// **A zero band is every realm the page knows about**, matching MotifData.AllowsRealm — a motif
// written without the field is fightable anywhere, so it has to answer every chip rather than
// none. The span is the widest any authored band reaches, which is the only definition of "every
// realm" a report on this catalog can have.
func realmsOf(band [2]int, lo, hi int) []int {
	if band != [2]int{} {
		lo, hi = band[0], band[1]
	}
	if lo == 0 || hi < lo {
		return nil
	}
	out := make([]int, 0, hi-lo+1)
	for f := lo; f <= hi; f++ {
		out = append(out, f)
	}
	return out
}

// span is the lowest and highest realm any authored band reaches.
func span(all []Entry) (lo, hi int) {
	for _, e := range all {
		if e.Band == [2]int{} {
			continue
		}
		if lo == 0 || e.Band[0] < lo {
			lo = e.Band[0]
		}
		if e.Band[1] > hi {
			hi = e.Band[1]
		}
	}
	return lo, hi
}

// realmTokens is a band as the attribute the chips match against.
func realmTokens(band [2]int, lo, hi int) string {
	realms := realmsOf(band, lo, hi)
	parts := make([]string, 0, len(realms))
	for _, f := range realms {
		parts = append(parts, strconv.Itoa(f))
	}
	return strings.Join(parts, " ")
}
