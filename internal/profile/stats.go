package profile

// **A player's lifetime statistics: what every finished run came to, added together.**
//
// A Tally is plain counts by name, and it is the shape both halves of the record share: a run keeps
// one while it is under way, saved with it in run.json, and the profile keeps two — one for rolled
// runs and one for runs started on a chosen seed — which a finished run is added into once, when it
// ends. See internal/session/stats.go for the names and internal/screens/achieve.go for the add.

import "sort"

// Tally is a set of named figures, read three ways.
//
// **Totals add, Most keeps the highest and Least keeps the lowest**, which is the whole difference
// between "damage dealt", "biggest hit" and "fastest win". Three maps rather than one map and a rule
// per name, so a figure's arithmetic is said by where it is filed and a name nobody has met yet still
// adds up the right way when two tallies are put together.
//
// **A name, never an ordinal**, on the rule the whole package is under: `relic:fire-ring` and
// `dealt:element:fire` rather than a RelicID and an Element.
type Tally struct {
	Totals map[string]int `json:"totals,omitempty"`
	Most   map[string]int `json:"most,omitempty"`
	Least  map[string]int `json:"least,omitempty"`
}

// Add puts n on a total. A zero or an empty name is nothing, and writes no key.
func (t *Tally) Add(name string, n int) {
	if name == "" || n == 0 {
		return
	}
	if t.Totals == nil {
		t.Totals = map[string]int{}
	}
	t.Totals[name] += n
}

// Max records v against a high-water mark, if it is higher than what is there.
//
// **A zero is not a record.** A mark that has never been set reads as zero, so a zero offered to it
// would write a key saying nothing.
func (t *Tally) Max(name string, v int) {
	if name == "" || v <= 0 {
		return
	}
	if old, ok := t.Most[name]; ok && old >= v {
		return
	}
	if t.Most == nil {
		t.Most = map[string]int{}
	}
	t.Most[name] = v
}

// Min records v against a low-water mark, if it is lower than what is there or nothing is.
//
// **Absent is not zero here**, which is why Least is read with LeastOf's second return: the fewest
// rounds a duel has been won in is not "none" on a profile that has never won one.
func (t *Tally) Min(name string, v int) {
	if name == "" || v <= 0 {
		return
	}
	if old, ok := t.Least[name]; ok && old <= v {
		return
	}
	if t.Least == nil {
		t.Least = map[string]int{}
	}
	t.Least[name] = v
}

// Total, MostOf and LeastOf read one figure. A name nothing has written reads as zero, and LeastOf
// also says whether it was written at all.
func (t Tally) Total(name string) int  { return t.Totals[name] }
func (t Tally) MostOf(name string) int { return t.Most[name] }
func (t Tally) LeastOf(name string) (int, bool) {
	v, ok := t.Least[name]
	return v, ok
}

// Merge adds another tally into this one: totals summed, marks kept at their extremes.
func (t *Tally) Merge(o Tally) {
	for _, k := range sortedKeys(o.Totals) {
		t.Add(k, o.Totals[k])
	}
	for _, k := range sortedKeys(o.Most) {
		t.Max(k, o.Most[k])
	}
	for _, k := range sortedKeys(o.Least) {
		t.Min(k, o.Least[k])
	}
}

// Plus is the two tallies merged, leaving both as they were. **A copy rather than a merge in
// place**, because the reader that wants it — a page showing the profile with the run in progress
// added on — must not move either.
func (t Tally) Plus(o Tally) Tally {
	var out Tally
	out.Merge(t)
	out.Merge(o)
	return out
}

// Empty reports whether nothing has been written.
func (t Tally) Empty() bool { return len(t.Totals) == 0 && len(t.Most) == 0 && len(t.Least) == 0 }

// sortedKeys walks a map in a fixed order. Nothing in the merge depends on the order, but map
// iteration is never allowed to decide anything in this repo, and a merge that happened not to is
// one edit away from one that does.
func sortedKeys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// RecordRun adds a finished run's tally to the profile, into the rolled or the chosen-seed record.
//
// **The two are kept apart rather than one being dropped** *(owner's call, 2026-10-11)*. A chosen
// seed progresses nothing — no achievement, no unlock, no counter an achievement reads — because a
// journey that could have been looked up in advance is not the player's to be rewarded for. A
// statistic is not a reward, so a chosen-seed run is still written down; it is written down
// separately, so the figures beside the achievements stay ones the achievements could have been
// earned against.
func (p *Profile) RecordRun(chosen bool, run Tally) {
	if chosen {
		p.ChosenStats.Merge(run)
		return
	}
	p.Stats.Merge(run)
}
