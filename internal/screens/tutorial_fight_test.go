package screens

import (
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/curiousjc/ascend-duel/assets"
	"github.com/curiousjc/ascend-duel/data"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/decks"
	"github.com/curiousjc/ascend-duel/internal/entities"
	"github.com/curiousjc/ascend-duel/internal/seeds"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/state"
	"github.com/curiousjc/ascend-duel/internal/tutorial"
	"github.com/curiousjc/ascend-duel/internal/ui"
)

// The taught fight, played headlessly the way the combat screen plays it: the same deal, the same
// refill, the same enemy planner and the same resolver.
//
//   - Round one is the three Slices the lesson names, none of them in the creature's element.
//   - Round two is the creature's-element Block, plus whatever cards the script names beside it,
//     the Block eating both of the creature's two attacks and banking a surge.
//   - Round three is whatever kills it, on the surged budget.
//
// taughtFight is everything a step or a test needs to know about that fight.
type taughtFight struct {
	element combat.Element
	record  string

	hands [3][]combat.Card // what the player holds at the top of each round
	plans [3][]combat.Card // what the creature commits to each round

	slices    []combat.Card // round one's three
	block     combat.Card   // round two's shield
	second    []combat.Card // everything round two plays, the Block included
	fireHit   combat.Card   // an attack in the creature's element, in round two's hand
	finisher  []combat.Card // a round-three play that kills
	needSurge bool          // no round-three play inside the unsurged budget kills

	afterRound [3]struct{ fighter, creature combat.Duelist }
}

// playTaughtFight plays the lesson's three rounds on a run seed against one creature, and says why
// it does not hold if it does not.
//
// alongside is the cards round two plays beside the Block, written as the script writes them —
// "arcane Thump".
func playTaughtFight(runSeed int64, record, element string, shape data.JourneyData,
	motifs map[string]data.MotifRecord, me data.DuelistData, alongside []string) (taughtFight, error) {

	var f taughtFight
	f.record = record
	el, ok := combat.ParseElement(element)
	if !ok || el == combat.Basic {
		return f, fmt.Errorf("realm element %q", element)
	}
	f.element = el

	var s CombatScene
	s.rng = rand.New(rand.NewSource(seeds.ForFight(runSeed, seeds.PlayerDeck, 0)))
	s.resetDeck(nil)

	fighter := combat.Duelist{
		DMG: me.DMG, MaxLife: me.HP, CurrentLife: me.HP, Actions: me.Actions,
		RoundLimit: combat.DefaultRoundLimit,
	}
	rec, found := motifs[record]
	if !found {
		return f, fmt.Errorf("no record %q", record)
	}
	creature := entities.NewEnemyFrom(rec, element, 0, shape).Duelist
	pile := decks.NewEnemyPile(record, element, seeds.ForFight(runSeed, seeds.EnemyDeck, 0), decks.EnemyHandSize)

	hold := func() []combat.Card {
		out := make([]combat.Card, 0, len(s.hand))
		for _, c := range s.hand {
			out = append(out, c.Card)
		}
		return out
	}
	spend := func(played []combat.Card) {
		// One copy per concept per element, so the pair is the card.
		type key struct {
			c combat.ConceptID
			e combat.Element
		}
		gone := map[key]bool{}
		for _, c := range played {
			gone[key{c.Concept, c.Element}] = true
		}
		kept := s.hand[:0]
		for _, c := range s.hand {
			if gone[key{c.Concept, c.Element}] {
				s.discard = append(s.discard, c.Card)
				continue
			}
			kept = append(kept, c)
		}
		s.hand = kept
		s.drawHand()
	}
	label := func(c combat.Card) string { return combat.ConceptOf(c.Concept).Label }
	isAttack := func(c combat.Card) bool { return combat.ConceptOf(c.Concept).Verb == combat.VerbAttack }

	// Round one: exactly the three Slices, none in the creature's element, and no other concept
	// holding three — so the lit set is the only Card Three of a Kind in the hand.
	f.hands[0] = hold()
	counts := map[combat.ConceptID]int{}
	for _, c := range f.hands[0] {
		counts[c.Concept]++
		if c.Concept == combat.Slice {
			if c.Element == el {
				return f, fmt.Errorf("a Slice in the creature's element is dealt")
			}
			f.slices = append(f.slices, c)
		}
	}
	if len(f.slices) != 3 {
		return f, fmt.Errorf("%d Slices dealt, want 3", len(f.slices))
	}
	for id, n := range counts {
		if id != combat.Slice && n >= 3 {
			return f, fmt.Errorf("a second set of three")
		}
	}
	// The Block the lesson points at in round one and plays in round two is already in hand.
	haveBlock := false
	for _, c := range f.hands[0] {
		if label(c) == "Block" && c.Element == el {
			f.block, haveBlock = c, true
		}
	}
	if !haveBlock {
		return f, fmt.Errorf("no Block in the creature's element in the opening hand")
	}

	f.plans[0] = pile.Plan(creature)
	_, fighter, creature = combat.ResolveRound(fighter, creature, f.slices, f.plans[0], 1, combat.Sources{})
	f.afterRound[0].fighter, f.afterRound[0].creature = fighter, creature
	if !creature.Alive() {
		return f, fmt.Errorf("the three Slices kill it in round one")
	}
	if !fighter.Alive() {
		return f, fmt.Errorf("round one kills the player")
	}

	// Round two: the Block and whatever the script names beside it, against exactly two attacks.
	spend(f.slices)
	f.hands[1] = hold()
	haveHit := false
	for _, c := range f.hands[1] {
		if isAttack(c) && c.Element == el {
			f.fireHit, haveHit = c, true
		}
	}
	if !haveHit {
		return f, fmt.Errorf("no attack in the creature's element in round two's hand")
	}
	f.plans[1] = pile.Plan(creature)
	attacks := 0
	for _, c := range f.plans[1] {
		if isAttack(c) {
			attacks++
		}
	}
	if attacks != 2 {
		return f, fmt.Errorf("the creature throws %d attacks in round two, want 2", attacks)
	}
	f.second = []combat.Card{f.block}
	for _, name := range alongside {
		c, ok := cardNamed(f.hands[1], name)
		if !ok {
			return f, fmt.Errorf("no %s in round two's hand", name)
		}
		f.second = append(f.second, c)
	}
	if fighter.CostOf(f.second) > fighter.ActionPoints() || len(f.second) > fighter.MaxActions() {
		return f, fmt.Errorf("round two's play costs more than the turn has")
	}
	_, fighter, creature = combat.ResolveRound(fighter, creature, f.second, f.plans[1], 2, combat.Sources{})
	f.afterRound[1].fighter, f.afterRound[1].creature = fighter, creature
	if fighter.Surge != 2 {
		return f, fmt.Errorf("round two banks a surge of %d, want 2", fighter.Surge)
	}
	if !fighter.Alive() || !creature.Alive() {
		return f, fmt.Errorf("someone dies in round two")
	}

	// Round three: something in the hand kills it on the surged budget.
	spend(f.second)
	f.hands[2] = hold()
	f.plans[2] = pile.Plan(creature)
	budget := fighter.ActionPoints()
	hand := f.hands[2]
	killsInside := func(limit int) []combat.Card {
		for mask := 1; mask < 1<<len(hand); mask++ {
			var play []combat.Card
			for i := range hand {
				if mask&(1<<i) != 0 {
					play = append(play, hand[i])
				}
			}
			if len(play) > fighter.MaxActions() || fighter.CostOf(play) > limit {
				continue
			}
			if _, _, after := combat.ResolveRound(fighter, creature, play, f.plans[2], 3, combat.Sources{}); !after.Alive() {
				return play
			}
		}
		return nil
	}
	f.finisher = killsInside(budget)
	if f.finisher == nil {
		return f, fmt.Errorf("nothing in round three's hand kills it on %d AP", budget)
	}
	f.needSurge = killsInside(budget-fighter.Surge) == nil
	return f, nil
}

// TestTheTutorialsFightPlaysAsTaught holds the shipped script to what its steps say: the three
// Slices wound without killing, round two's play — the Block and the cards named beside it — leaves
// the creature standing while the Block eats two hits in its element and surges, and round three can
// kill it on the surged budget. **Whether the kill needs the surge is logged, not required**
// *(owner's call)*. **If it goes red the fix is a new seed** — TestFindATaughtFight finds
// one — not a weaker check.
func TestTheTutorialsFightPlaysAsTaught(t *testing.T) {
	parkTutorial(t)
	script := tutorial.Load()
	runSeed, err := seeds.Parse(script.Seed)
	if err != nil {
		t.Fatalf("tutorial.json seed %q: %v", script.Seed, err)
	}
	motifs, shape := data.LoadMotifs(), data.LoadJourney()
	run := session.Start(motifs, shape, runSeed)

	f, err := playTaughtFight(runSeed, script.Enemy, run.Element(), shape,
		data.MotifRecords(motifs), data.LoadDuelists()["Fighter1"], alongsideTheBlock(script))
	if err != nil {
		t.Fatalf("seed %s against %s: %v", script.Seed, script.Enemy, err)
	}

	// **The steps name the element**, so the realm has to be the one they name.
	for _, step := range script.Steps {
		if strings.Contains(step.Text, "dual") && !strings.Contains(step.Text, run.Element()) {
			t.Errorf("step %q names a shield that is not the realm's %s", step.Key, run.Element())
		}
	}
	t.Logf("round two hand %s", describeCards(f.hands[1]))
	t.Logf("%s against %s (%s): slices %s; %d → %d → %d, finished by %s (surge needed: %v)", script.Seed, script.Enemy,
		run.Element(), describeCards(f.slices), f.afterRound[0].creature.MaxLife, f.afterRound[0].creature.CurrentLife,
		f.afterRound[1].creature.CurrentLife, describeCards(f.finisher), f.needSurge)
}

// TestTheCardStepsNameTheCardsTheyLight: a step that describes a card describes the one its
// anchor lights — its element and its cost, and a defend's shields — in the real opening hand under
// the real sort, against the realm the seed deals. A new seed or a new sort that lights a different
// card fails here rather than on screen.
func TestTheCardStepsNameTheCardsTheyLight(t *testing.T) {
	parkTutorial(t)
	gs := &state.GlobalState{ScreenWidth: state.ScreenWidth, ScreenHeight: state.ScreenHeight}
	s := newTaughtScene(t, gs)

	script := tutorial.Load()
	runSeed, _ := seeds.Parse(script.Seed)
	realmName := session.Start(data.LoadMotifs(), data.LoadJourney(), runSeed).Element()
	realm, _ := combat.ParseElement(realmName)
	s.enemy.Duelist.Element = realm

	lit := map[tutorial.Anchor]combat.Card{}
	if match := s.matchingCards(gs); len(match) > 0 {
		lit[tutorial.AnchorMatchingCard] = s.hand[match[0]].Card
	}
	if i, ok := s.elementCard(true); ok {
		lit[tutorial.AnchorElementBlock] = s.hand[i].Card
	}

	// A named card is read in the hand of the round its step comes up in: the opening hand before
	// the first DUEL, round two's after it.
	motifs := data.LoadMotifs()
	f, err := playTaughtFight(runSeed, script.Enemy, realmName, data.LoadJourney(),
		data.MotifRecords(motifs), data.LoadDuelists()["Fighter1"], alongsideTheBlock(script))
	if err != nil {
		t.Fatalf("the taught fight does not play: %v", err)
	}
	named := func(step tutorial.Step) (combat.Card, bool) {
		round := 0
		for _, earlier := range script.Steps {
			if earlier.Key == step.Key {
				break
			}
			if earlier.Until == tutorial.CondDuelPressed {
				round++
			}
		}
		if len(step.Cards) != 1 {
			return combat.Card{}, false
		}
		return cardNamed(f.hands[round], step.Cards[0])
	}

	described := 0
	for _, step := range script.Steps {
		if step.Until != tutorial.CondNext || !step.Anchor.NamesCards() || !strings.Contains(step.Text, "This ") {
			continue
		}
		described++
		card, ok := lit[step.Anchor]
		if step.Anchor == tutorial.AnchorNamedCards {
			card, ok = named(step)
		}
		if !ok {
			t.Errorf("step %q describes a card and its anchor %q lights none in that round's hand",
				step.Key, step.Anchor)
			continue
		}
		text := strings.ToLower(step.Text)
		element := strings.ToLower(ui.ElementName(card.Element))
		wants := [][]string{
			{element},
			{fmt.Sprintf("costs %s ap", numberWord(card.Cost())), fmt.Sprintf("costs %d ap", card.Cost())},
		}
		if combat.ConceptOf(card.Concept).Verb != combat.VerbAttack {
			wants = append(wants, []string{fmt.Sprintf("%s %s shields", numberWord(card.Amount()), element)})
		} else if strings.Contains(text, "x your") {
			wants = append(wants, []string{fmt.Sprintf("%dx your", card.Amount()/100)})
		}
		for _, any := range wants {
			found := false
			for _, w := range any {
				found = found || strings.Contains(text, w)
			}
			if !found {
				t.Errorf("step %q lights a %s/%v costing %d and does not say %q",
					step.Key, combat.ConceptOf(card.Concept).Label, card.Element, card.Cost(), any[0])
			}
		}
	}
	if described == 0 {
		t.Error("no step describes a card, so this test checks nothing")
	}
}

// cardNamed is the first card in a hand answering to a name as the script writes one.
func cardNamed(hand []combat.Card, name string) (combat.Card, bool) {
	for _, c := range hand {
		if cardIsNamed(c, name) {
			return c, true
		}
	}
	return combat.Card{}, false
}

// alongsideTheBlock is what the shipped script has round two play beside the Block: the cards the
// step asking for the shield names, the Block itself left out.
func alongsideTheBlock(script tutorial.Script) []string {
	for _, step := range script.Steps {
		if step.Anchor != tutorial.AnchorNamedCards || step.Until != tutorial.CondCardsQueued {
			continue
		}
		var out []string
		for _, name := range step.Cards {
			if f := strings.Fields(name); f[len(f)-1] != "Block" {
				out = append(out, name)
			}
		}
		return out
	}
	return nil
}

func describeCards(cs []combat.Card) string {
	var out []string
	for _, c := range cs {
		out = append(out, fmt.Sprintf("%s/%v", combat.ConceptOf(c.Concept).Label, c.Element))
	}
	return strings.Join(out, " ")
}

// TestFindATaughtFight searches run codes for one whose first room is a fire realm and whose deal
// plays the lesson's three rounds. A search, not a check.
//
//	TUTORIAL_TESTS=1 SEEDSEARCH=1 go test ./internal/screens -run TestFindATaughtFight -v -timeout 900s
func TestFindATaughtFight(t *testing.T) {
	parkTutorial(t)
	if os.Getenv("SEEDSEARCH") == "" {
		t.Skip("a search, not a check: set SEEDSEARCH=1 to run it")
	}
	want := os.Getenv("TAUGHT_ELEMENT")
	if want == "" {
		want = "fire"
	}

	motifs, shape := data.LoadMotifs(), data.LoadJourney()
	records := data.MotifRecords(motifs)
	me := data.LoadDuelists()["Fighter1"]
	art := assets.LoadImageData()

	// Any creature with a picture in the element may stand in the taught room — the script names it.
	// **Round two is played as the Block alone**: the cards a script names beside it depend on the
	// hand a candidate deals, so TestTheTutorialsFightPlaysAsTaught is what checks the script's own.
	var candidates []string
	keys := make([]string, 0, len(records))
	for key := range records {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if _, ok := art[records[key].ArtKey(want)]; ok {
			candidates = append(candidates, key)
		}
	}

	reasons := map[string]int{}
	found := 0
	for v := int64(0); v < seeds.Space && found < 25; v++ {
		run := session.Start(motifs, shape, v)
		if run.Element() != want {
			continue
		}
		native := run.Enemy()
		motif := run.Motif()
		for _, record := range candidates {
			// The realm's own motif, so the room, the backdrop and the creature agree.
			if !strings.HasPrefix(record, motif+"-") {
				continue
			}
			f, err := playTaughtFight(v, record, want, shape, records, me, nil)
			if err != nil {
				reasons[err.Error()]++
				if err.Error() == "3 Slices dealt, want 3" {
					break
				}
				if strings.Contains(err.Error(), "Slices dealt") || strings.Contains(err.Error(), "Block in") ||
					strings.Contains(err.Error(), "second set") || strings.Contains(err.Error(), "Slice in the") {
					break // the deal fails for every creature alike
				}
				continue
			}
			found++
			t.Logf("%s  %-30s native %-28s surge-needed %-5v hp %d→%d→%d  finish %s",
				seeds.Code(v), record, native, f.needSurge,
				f.afterRound[0].creature.MaxLife, f.afterRound[0].creature.CurrentLife,
				f.afterRound[1].creature.CurrentLife, describeCards(f.finisher))
		}
	}
	if found == 0 {
		t.Logf("why candidates failed: %v", reasons)
		t.Fatal("no run code plays the lesson")
	}
}

// TestTheShieldStepLightsAllThreeAtOnce: the step asking for round two's play lights every card it
// names together, in the real round-two hand, and says how many to take.
func TestTheShieldStepLightsAllThreeAtOnce(t *testing.T) {
	parkTutorial(t)
	script := tutorial.Load()
	runSeed, _ := seeds.Parse(script.Seed)
	motifs, shape := data.LoadMotifs(), data.LoadJourney()
	run := session.Start(motifs, shape, runSeed)
	f, err := playTaughtFight(runSeed, script.Enemy, run.Element(), shape,
		data.MotifRecords(motifs), data.LoadDuelists()["Fighter1"], alongsideTheBlock(script))
	if err != nil {
		t.Fatal(err)
	}

	for _, step := range script.Steps {
		if step.Anchor != tutorial.AnchorNamedCards || step.Until != tutorial.CondCardsQueued {
			continue
		}
		gs := &state.GlobalState{ScreenWidth: state.ScreenWidth, ScreenHeight: state.ScreenHeight}
		gs.Run = session.New(nil)
		s := stubCombat()
		s.hand = nil
		for _, c := range f.hands[1] {
			s.hand = append(s.hand, paletteCard{Card: c})
		}
		s.sortHand()
		onStep(gs, step)

		rects, ok := s.tutorialRects(gs, step.Anchor)
		if !ok || len(rects) != len(step.Cards) {
			t.Errorf("step %q names %d cards and lights %d", step.Key, len(step.Cards), len(rects))
		}
		if want := "select all " + numberWord(step.Count); waitingHint(step) != want {
			t.Errorf("step %q says %q, want %q", step.Key, waitingHint(step), want)
		}
	}
}
