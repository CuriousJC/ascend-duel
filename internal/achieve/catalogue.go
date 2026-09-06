package achieve

// The catalogue: **the JSON records, parsed into something that can be asked a question.**
//
// Loaded at package init, so a malformed record fails the launch rather than mid-climb — the same
// exit `combat.RegisterConcept` and `session.loadStones` take, and for the same reason.

import (
	"fmt"

	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
)

// The moments the code pushes. **Closed, and a record naming one that is not here is refused at
// load** — which is what stops the push model failing silently. Adding one is a constant here plus
// the one call site that raises it, and never something a file can assert into existence.
const (
	// MomentDuelWon is a fight won. It fires on every win, so an achievement hanging off it means
	// "the first one you ever win" only because the profile keeps one award.
	MomentDuelWon = "duel-won"

	// MomentTutorialFinished is the teaching run ending, dismissed or played through. Skipping
	// counts, on the same terms markTutorialSeen is written under.
	MomentTutorialFinished = "tutorial-finished"

	// MomentFloorReached is the climb arriving on a floor, carrying that floor in N. **A threshold
	// rather than an equality** — a record asking for five is earned by six, so a player who
	// somehow skipped the floor is not left with a hole in the page.
	MomentFloorReached = "floor-reached"

	// MomentCardAltered is a worm having changed a card, carrying the resulting card's label in
	// Value. **The label rather than the worm**, because what the player did is "made a Flinch" and
	// several worms can arrive at one — a demote from a Ward today, something else tomorrow.
	MomentCardAltered = "card-altered"
)

// moments is the same list, for validation and for the error message.
var moments = []string{
	MomentDuelWon, MomentTutorialFinished, MomentFloorReached, MomentCardAltered,
}

// The two counter families. **A prefix rather than a bare name**, so a counter is self-describing
// on disk and two axes cannot collide — `slash` is both a form and nothing like the concept
// `Slash`, and a player who plays five hundred slashing cards has not played five hundred Slashes.
//
// **Per concept, never per concept-and-element** *(owner's call, 2026-09-06)*: five colours of
// twelve concepts is sixty tallies to say what twelve say, and no achievement has wanted the
// distinction.
const (
	counterForm    = "form:"
	counterConcept = "concept:"
)

// Achievement is one record with its trigger resolved.
type Achievement struct {
	Key  string
	Name string
	How  string

	// Said is every line shown when it lands, in order, all at once.
	Said []string

	// Unlocks is what earning it opens up. Empty on every record today — see the field's own note
	// in data/achievements_data.go for why the two key spaces stay apart.
	Unlocks []string

	trigger trigger
}

// trigger is a record's condition, with its strings turned into the things this package compares.
type trigger struct {
	kind string

	// turn
	patterns []pattern

	// count
	counter string

	// moment
	moment string
	value  string

	// count's threshold and floor-reached's floor.
	n int
}

// pattern is a set of clauses that must all hold for one turn.
type pattern struct{ clauses []clause }

// clause is one condition over the cards of one category.
type clause struct {
	// of is the category filter. `anyCategory` means the whole turn.
	of category

	axis combat.Axis
	mode string
	n    int
}

// category is a clause's filter: one of the two real categories, or neither.
type category int

const (
	anyCategory category = iota
	attackOnly
	defendOnly
)

// Catalogue is every achievement this build knows, in file order.
type Catalogue struct {
	list []Achievement
	by   map[string]int
}

// All is every achievement, in the order the file wrote them — which is the order the page lists
// them in. See data.LoadAchievements for why that is not sorted.
func (c *Catalogue) All() []Achievement { return c.list }

// Find is one achievement by key.
func (c *Catalogue) Find(key string) (Achievement, bool) {
	i, ok := c.by[key]
	if !ok {
		return Achievement{}, false
	}
	return c.list[i], true
}

// catalogue is the one loaded at init. **Package state, like combat's concept registry**, and for
// the same reason: it is read once from embedded data and never changes, so threading it through
// every caller would describe a variable that does not vary.
var catalogue = load()

// Loaded is the catalogue this build shipped.
func Loaded() *Catalogue { return catalogue }

// load parses and validates the file. **It panics**, because a catalogue this build cannot read is
// a page of achievements that quietly cannot be earned, and a game that will not start says so
// where a game that starts does not.
func load() *Catalogue {
	records := data.LoadAchievements()

	out := &Catalogue{
		list: make([]Achievement, 0, len(records)),
		by:   make(map[string]int, len(records)),
	}

	for _, r := range records {
		t, err := parseTrigger(r.Trigger)
		if err != nil {
			panic(fmt.Sprintf("achievements.json: %s: %v", r.AchievementRecord, err))
		}
		if r.Name == "" || r.How == "" {
			panic("achievements.json: " + r.AchievementRecord + " needs a Name and a How")
		}
		out.by[r.AchievementRecord] = len(out.list)
		out.list = append(out.list, Achievement{
			Key:     r.AchievementRecord,
			Name:    r.Name,
			How:     r.How,
			Said:    r.Said,
			Unlocks: r.Unlocks,
			trigger: t,
		})
	}
	return out
}

// parseTrigger turns one record's trigger into something comparable, refusing everything it does
// not recognise.
//
// **A field belonging to another kind is an error rather than something ignored.** A record that
// wrote a `Counter` on a moment trigger meant something by it, and quietly dropping the field is
// how an achievement comes to fire on the wrong thing.
func parseTrigger(t data.TriggerData) (trigger, error) {
	switch t.Kind {
	case data.TriggerTurn:
		if len(t.Patterns) == 0 {
			return trigger{}, fmt.Errorf("a turn trigger needs at least one pattern")
		}
		if t.Counter != "" || t.Moment != "" || t.Value != "" || t.N != 0 {
			return trigger{}, fmt.Errorf("a turn trigger reads only Patterns")
		}
		out := trigger{kind: t.Kind}
		for _, p := range t.Patterns {
			parsed, err := parsePattern(p)
			if err != nil {
				return trigger{}, err
			}
			out.patterns = append(out.patterns, parsed)
		}
		return out, nil

	case data.TriggerCount:
		if t.N < 1 {
			return trigger{}, fmt.Errorf("a count trigger needs an N of at least 1")
		}
		if len(t.Patterns) != 0 || t.Moment != "" || t.Value != "" {
			return trigger{}, fmt.Errorf("a count trigger reads only Counter and N")
		}
		if err := checkCounter(t.Counter); err != nil {
			return trigger{}, err
		}
		return trigger{kind: t.Kind, counter: t.Counter, n: t.N}, nil

	case data.TriggerMoment:
		if len(t.Patterns) != 0 || t.Counter != "" {
			return trigger{}, fmt.Errorf("a moment trigger reads only Moment, Value and N")
		}
		if !known(moments, t.Moment) {
			return trigger{}, fmt.Errorf("no moment named %q; the game raises %v", t.Moment, moments)
		}
		// **N belongs to floor-reached and to nothing else**, and Value to card-altered. A moment
		// carrying a field its raiser never sets is a condition that can never be met.
		if t.N != 0 && t.Moment != MomentFloorReached {
			return trigger{}, fmt.Errorf("moment %q carries no N", t.Moment)
		}
		if t.Moment == MomentFloorReached && t.N < 1 {
			return trigger{}, fmt.Errorf("floor-reached needs the floor in N")
		}
		if t.Value != "" && t.Moment != MomentCardAltered {
			return trigger{}, fmt.Errorf("moment %q carries no Value", t.Moment)
		}
		if t.Moment == MomentCardAltered && t.Value == "" {
			return trigger{}, fmt.Errorf("card-altered needs the card's label in Value")
		}
		return trigger{kind: t.Kind, moment: t.Moment, value: t.Value, n: t.N}, nil

	default:
		return trigger{}, fmt.Errorf("no trigger kind %q; the three are %q, %q and %q",
			t.Kind, data.TriggerTurn, data.TriggerCount, data.TriggerMoment)
	}
}

func parsePattern(p data.PatternData) (pattern, error) {
	if len(p.Clauses) == 0 {
		return pattern{}, fmt.Errorf("a pattern needs at least one clause")
	}
	out := pattern{}
	for _, c := range p.Clauses {
		parsed, err := parseClause(c)
		if err != nil {
			return pattern{}, err
		}
		out.clauses = append(out.clauses, parsed)
	}
	return out, nil
}

// parseClause resolves one clause, and is where the axis names `hands.json` already uses are read a
// second time — through `combat.ParseAxis`, so the two files cannot disagree about what `form`
// means.
func parseClause(c data.ClauseData) (clause, error) {
	out := clause{mode: c.Mode, n: c.N}

	switch c.Of {
	case "":
		out.of = anyCategory
	case data.OfAttack:
		out.of = attackOnly
	case data.OfDefend:
		out.of = defendOnly
	default:
		return clause{}, fmt.Errorf("no category %q; a clause filters on %q, %q or nothing",
			c.Of, data.OfAttack, data.OfDefend)
	}

	switch c.Mode {
	case data.ModeCount:
		if c.Axis != "" {
			return clause{}, fmt.Errorf("a count clause counts cards, not values on an axis")
		}
		if c.N < 1 {
			return clause{}, fmt.Errorf("a count clause needs an N of at least 1")
		}
		return out, nil

	case data.ModeDistinct, data.ModeSame:
		axis, ok := combat.ParseAxis(c.Axis)
		if !ok {
			return clause{}, fmt.Errorf("no axis %q; the three are %v", c.Axis, combat.AllAxes)
		}
		out.axis = axis
		if c.Mode == data.ModeDistinct && c.N < 2 {
			return clause{}, fmt.Errorf("a distinct clause needs an N of at least 2")
		}
		if c.Mode == data.ModeSame && c.N != 0 {
			return clause{}, fmt.Errorf("a same clause reads no N")
		}
		return out, nil

	default:
		return clause{}, fmt.Errorf("no clause mode %q; the three are %q, %q and %q",
			c.Mode, data.ModeDistinct, data.ModeSame, data.ModeCount)
	}
}

// checkCounter refuses a tally name nothing will ever bump.
//
// **This is the check the count family most needs.** A misspelled counter is not a crash and not a
// wrong answer — it is an achievement that sits locked forever while the game happily plays on,
// which is indistinguishable from one nobody has got round to earning.
func checkCounter(name string) error {
	switch {
	case name == "":
		return fmt.Errorf("a count trigger needs a Counter")

	case len(name) > len(counterForm) && name[:len(counterForm)] == counterForm:
		if _, ok := combat.ParseForm(name[len(counterForm):]); !ok {
			return fmt.Errorf("counter %q names no form", name)
		}
		return nil

	case len(name) > len(counterConcept) && name[:len(counterConcept)] == counterConcept:
		// Player concepts are registered under their bare label at combat's package init, which
		// has run by the time this one does. An enemy's card is scoped to its record and could
		// never be played by the player, so it cannot be counted here either.
		if _, ok := combat.ConceptByKey(name[len(counterConcept):]); !ok {
			return fmt.Errorf("counter %q names no card the player can play", name)
		}
		return nil

	default:
		return fmt.Errorf("counter %q must start %q or %q", name, counterForm, counterConcept)
	}
}

func known(set []string, name string) bool {
	for _, s := range set {
		if s == name {
			return true
		}
	}
	return false
}
