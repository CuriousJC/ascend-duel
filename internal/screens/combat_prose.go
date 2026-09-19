package screens

import (
	"fmt"
	"log"

	"github.com/curiousjc/ascend-duel/internal/cards"
	"github.com/curiousjc/ascend-duel/internal/combat"
	"github.com/curiousjc/ascend-duel/internal/session"
	"github.com/curiousjc/ascend-duel/internal/ui"
)

// The combat scene's half of the prose and the card faces.
//
// **These are methods on CombatScene, which is what keeps them out of internal/ui.** Everything
// else in prose.go, prose_terms.go and card_art.go is a free function over an event or a spec —
// shared vocabulary three screens and the ledger draw through — and a method on a scene is the one
// shape that cannot cross a package line, because a type's methods must be declared beside it.
//
// The split is not an inconvenience to work around: it is the boundary stating which of these
// decide something about *this duel* and which describe a card or an event to anybody who asks.

// backSpec is a face-down card of this duelist's deck.
//
// **A duelist and a card back go together** *(2026-08-11)*: the plan is to offer different
// duelists as different decks, and the mark on the back is how you tell at a glance whose
// deck is on the table. The name comes from `data/duelists.json` and is parsed here rather
// than at load, because `internal/entities` must not import the drawing package — the same
// separation the element mapping below exists for.
//
// An unrecognized name falls back to the triangle and says so once. A back is cosmetic;
// refusing to draw the draw pile over one would be a worse outcome than the wrong shape.
func (s *CombatScene) backSpec() cards.Spec {
	mark, ok := cards.ParseBackMark(s.fighter.CardBack)
	if !ok && s.fighter.CardBack != "" && !ui.WarnedBack {
		ui.WarnedBack = true
		log.Printf("cards: duelist card back %q is not a mark; using %v",
			s.fighter.CardBack, mark)
	}
	return cards.Spec{FaceDown: true, Back: mark}
}

// ledgerLines is the walk itself: one line per thing that happened, worded once, kept for the
// length of the run. See session/ledger.go for why the run stores these rather than the events
// they were written from.
func (s *CombatScene) ledgerLines(events []combat.Event) []session.LedgerLine {
	var rows []session.LedgerLine

	// **Which card is which, for the arithmetic underneath a blow.** A hand event names its terms
	// as indices into its own side's resolved actions, and those actions are events that have not
	// arrived yet when it does — so the turn is indexed up front. See combat.Event.HandCards.
	played := map[combat.Side][]combat.Card{}
	for _, e := range events {
		if e.Kind == combat.KindAction {
			played[e.Side] = append(played[e.Side],
				combat.Card{Concept: e.Action, Element: e.Element})
		}
	}

	// cur is the line the next outcome attaches to, or -1 when the last thing appended was
	// an announcement rather than an action. curSide is whose line it is.
	//
	// **The side is tracked rather than read back off the row's swatch**, because the hand line
	// wears amber and takes outcomes: a damage event compared against that swatch would read
	// every hit as belonging to the wrong duelist.
	cur := -1
	curSide := combat.SideA
	outcomes := 0

	// Outcomes are appended to the tail of the sentence, after the verb, so the colored verb
	// never moves as a line grows.
	attach := func(what string) {
		if cur < 0 {
			return
		}
		sep := " - "
		if outcomes > 0 {
			sep = ", "
		}
		rows[cur].Spans = append(rows[cur].Spans, session.LedgerSpan{Text: sep + what})
		outcomes++
	}

	// act opens a line in the form "<who> <verb> <what>", with the verb carrying its
	// category's color. See cardPhrase.
	act := func(side combat.Side, c combat.Card) {
		rows = append(rows, session.LedgerLine{
			Voice: ui.VoiceFor(side),
			Spans: append([]session.LedgerSpan{
				{Text: s.sideName(side) + " "},
				{Text: ui.VerbFor(c.Category()), Ink: ui.CategoryInk(c.Category()), Mark: true},
			}, ui.ElementSpans(" "+ui.CardPhrase(c)+ui.CardWeight(c))...),
		})
		cur, curSide = len(rows)-1, side
		outcomes = 0
	}

	announce := func(label string, voice string) {
		rows = append(rows, session.Line(voice, label))
		cur = -1
	}

	// blow opens the attack phase's one line: what the hand formed and what it adds up to.
	//
	// **It is an announcement that takes outcomes**, which no other line here is. The blow's
	// damage, a shocked miss and any status it lands all belong to it, because the cards that
	// would otherwise have carried them no longer write lines of their own.
	blow := func(e combat.Event) {
		rows = append(rows, session.LedgerLine{
			Voice: session.VoiceHand,
			// **The hand's name is the whole line** *(owner's call, 2026-09-02)*. It read
			// "HAND!  Duelist lands Three of a Kind (Card)", and every word before the name was
			// already said by something on the row: the amber swatch says a hand formed, and in a
			// player's ledger the duelist is who forms them. What is left is the rung and what it
			// came to.
			//
			// **And it is not marked.** Bold is the whole panel and an underline under a name that
			// is already alone on its line reads as a mistake rather than as emphasis — see the
			// multiplier in the sum, which lost its underline for the same reason.
			Spans: []session.LedgerSpan{{Text: ui.HandTitle(e), Ink: session.InkHand}},
		})
		cur, curSide = len(rows)-1, e.Side
		outcomes = 0
	}

	for _, e := range events {
		switch e.Kind {
		case combat.KindRoundStart, combat.KindRoundEnd:
			// The feed holds one round, so saying which round it is would be a line spent on
			// something the caption and the character block both already carry. **The fight log
			// does need it and writes its own heading**, from the position of the round in the
			// fight rather than from these events — a heading belongs to the caller, which is
			// the only one of the two that knows whether a round has anything before it.

		case combat.KindAction:
			// **A hand-forming side's attack card writes no line.** Its beat still passes — the engine
			// announces every card so the table can light it and playback can count slots — but the
			// sentence for the whole phase is the KindHand below.
			//
			// **A solo attacker has no phase line, so the card's own sentence is the line**
			// *(2026-08-17)*. There is no KindHand coming for it, and an attack that reported
			// nothing but a damage figure with no verb in front of it would be the one kind of
			// action in the round that never says what it was.
			if combat.Plain(e.Action).Category() == combat.CategoryAttack && !s.soloAttacker(e.Side) {
				break
			}
			act(e.Side, combat.Card{Concept: e.Action, Element: e.Element})

		case combat.KindChilled:
			announce(fmt.Sprintf("%s is chilled - %v is lost", s.sideName(e.Side), combat.ConceptOf(e.Action).Label),
				ui.VoiceFor(e.Side))

		case combat.KindMissed:
			// It attaches to the attacker's own line rather than announcing, because the card
			// *was* played — the line above it is real and this is what became of it. Naming
			// the shock is the whole point: a blow that simply missed would look like a bug in
			// a game with no dice in it.
			attach("misses - shocked")

		case combat.KindStatus:
			attach(ui.StatusPhrase(e.Status))

		case combat.KindDrained:
			// **It attaches to the attacker's own line**, like a status does, because it is
			// something the blow did rather than an event of its own: the hit is the line above and
			// this is the rest of what that hit was worth.
			//
			// **The relic names itself.** A second drain relic would otherwise narrate identically
			// to the first, which is the argument KindBurned already makes for naming its status.
			attach(fmt.Sprintf("%s drains %d", combat.RelicOf(e.Relic).Name, e.Amount))

		case combat.KindRegenerated:
			// **A line of its own, where a drain attaches to one.** A drain is part of what the
			// blow was worth and has an attacker's sentence above it to hang on; this happens at
			// the top of a turn with nothing before it, so there is nothing to attach to.
			announce(fmt.Sprintf("%s restores %d - %s", s.sideName(e.Side), e.Amount,
				combat.RelicOf(e.Relic).Name), ui.VoiceFor(e.Side))

		case combat.KindBurned:
			// A tick belongs to nobody's card, so it opens its own line. It carries the
			// victim's swatch because it is a thing happening *to* them, which is also the
			// only side the event names.
			//
			// **The status names itself** *(2026-08-17)*: with statuses decoupled from the colors,
			// a second damage-over-time status would otherwise narrate identically to the first.
			announce(fmt.Sprintf("%s %s %d",
				s.sideName(e.Target), ui.TickVerb(e.Status), e.Amount), ui.VoiceFor(e.Target))

		case combat.KindHand:
			// **This is the attack phase's line, and every hand takes it — the No Hand
			// included** *(2026-08-19)*. There used to be a branch here writing an ordinary attack
			// sentence when `e.Hand` was `HandNone`, on the argument that announcing "HAND!" over
			// a single Bash empties the word. **It had been unreachable for some time**:
			// `blowFor` falls back to the catalog's `no-hand` entry, so a turn with an attack
			// in it always names a hand and the branch could not fire. What the log actually
			// printed was the hand line, correctly, while the code beside it said otherwise.
			//
			// The No Hand is an equal citizen throughout now, on the owner's call, so this is
			// deliberate rather than merely true.
			blow(e)
			rows = append(rows, s.handTermLines(e, played[e.Side])...)

		case combat.KindRaised:
			// **The count that is standing, not the count this card added.** Two Guards in a turn
			// is one duelist behind six shields, and a feed saying "+3" twice makes the reader do
			// the arithmetic the readout has already done.
			attach(fmt.Sprintf("%s up", ui.ShieldCount(e.Life)))

		case combat.KindVitae:
			// **Two riders pay vitae and they are different sentences** *(2026-09-10)*. A held card
			// is paid for being kept back; a played silver card gambled and came up. This said
			// "kept back for 8 vitae" over both until `Event.Rider` existed, which was wrong about
			// the one thing the player had just decided. See combat.Event.Rider.
			if e.Rider == combat.RiderSilver {
				attach(fmt.Sprintf("silver pays %d vitae", e.Amount))
			} else {
				attach(fmt.Sprintf("kept back for %d vitae", e.Amount))
			}

		case combat.KindExpired:
			// **A line of its own, because the row emptying needs a reason beside it.** Shields
			// that were never spent are the player's own decision coming back, and a readout that
			// simply went blank would read as a bug.
			attach(fmt.Sprintf("%s lapse", ui.ShieldCount(e.Amount)))

		case combat.KindBlocked:
			// **A sentence of its own rather than a clause on the damage line**, because there is
			// no damage line: the attack landed nothing, so the feed's only record that it happened
			// at all is this.
			attach(fmt.Sprintf("blocked - %s left", ui.ShieldCount(e.Amount)))

		case combat.KindTimeUp:
			// **A line of its own, and it opens one.** Nobody swung, so there is no attacker's
			// sentence for this to attach to — and the fall that follows it on the next event would
			// otherwise be the only record of the biggest thing that has ever happened in the feed.
			announce(fmt.Sprintf("%s is out of time - the duel takes %d",
				s.sideName(e.Target), e.Amount), ui.VoiceFor(e.Target))

		case combat.KindDamage:
			// **Damage whose side does not match the line it is attaching to is damage running the
			// other way**, which reads as something done back rather than as a hit of its own.
			// Nothing in the game produces it as of 2026-08-15 — the counter-attacking card that
			// did was cut — and it is kept because the test is a side comparison rather than a
			// card name, so it costs one branch and catches the case rather than mis-narrating it.
			switch {
			case cur >= 0 && curSide != e.Side:
				attach(fmt.Sprintf("hits back for %d", e.Amount))
			default:
				attach(fmt.Sprintf("%d damage", e.Amount))
			}

		case combat.KindDefeated:
			announce(fmt.Sprintf("%s falls", s.sideName(e.Target)), ui.VoiceFor(e.Target))
		}
	}

	return rows
}

// sideName is who a Resolution line belongs to, written out beside the swatch that already
// says it in color. **Saying it twice is deliberate**: the colors carry the pattern at a
// glance, but a line that begins "Bash" reads as an instruction rather than a report, and
// with both sides' actions in one list the reader has to hold which color is which. The name
// makes each line stand on its own.
//
// **It reads the combatant rather than the roster** *(2026-08-11)*. It used to index the
// fight order and print the record key, which is why the four records were named
// Monster1..Tactician1 — style names standing in for creature names because there was
// nowhere else to put one. Records carry a Name now, so a line says "Ogre Warlord attacks"
// rather than "OgreWarlord attacks".
// soloAttacker reports whether a side's attack cards resolve one at a time rather than as a hand.
//
// **It reads the duelist the round was resolved for, not the side** — see
// `combat.Duelist.SoloAttacks`. Two things on this screen change with it and both would otherwise
// have to guess: the feed writes a sentence per attack card because no hand line is coming, and
// the table lights one card at a time because no single blow is being assembled.
//
// A missing combatant answers false, which is the hand-forming case: this is asked while drawing, and
// a half-built scene should read as the ordinary round rather than as an enemy's.
func (s *CombatScene) soloAttacker(side combat.Side) bool {
	c := s.fighter
	if side == combat.SideB {
		c = s.enemy
	}
	return c != nil && c.SoloAttacks
}

func (s *CombatScene) sideName(side combat.Side) string {
	c := s.fighter
	if side == combat.SideB {
		c = s.enemy
	}
	if c != nil && c.Name != "" {
		return c.Name
	}
	return ui.DuelistName
}

// **Only the scripted demo calls this**, so a bare `staticcheck ./...` reports it unused and is
// wrong — that run does not compile combat_demo_on.go. A symbol is dead only when it is dead under
// every build tag; see the audit skill, which has the intersection as a one-liner. It was deleted
// once on that mistake.
//
// logRows writes the sentences for a run of events: one line per thing that happened, in the
// order the resolver produced them.
//
// **One line per slot, not one per event.** A busy round is 25-30 events, so writing the log
// verbatim would be a panel nobody could read. Merging an action with its outcome is
// presentation of events the engine already decided; it computes nothing, so what is written
// here still cannot disagree with what the round did. **Hands and chills get lines of their
// own**, because they are not something a card did â€” folding a hand into the line of the card
// that happened to start it would bury the one thing worth reading.
//
// **The attack phase is one line, and it is the hand's** *(2026-08-14)*. The defenses write a
// line each; the attack cards write none. A turn lands one blow, so five sentences
// saying "Duelist attacks with an earth strike" described a round that does not happen, and the
// line that mattered â€” what the five cards came to â€” was the sixth. **Every hand takes that line,
// the No Hand included** *(2026-08-19)* â€” a lone attack is the catalog's one-card hand and is
// announced like any other.
//
// **It takes the events rather than reading the round off the scene** *(2026-08-18)*. It was the
// Resolution feed's walk, over `s.log[:cursor+1]`; the feed is gone and the fight log is what
// draws these rows now, over every round of the fight. Passing the slice is what let one walk
// serve both while both existed, and it is what keeps this function free of any opinion about
// where the rows are going.
//
// It knows nothing about capacity, overflow or which row is live. Those are properties of the
// panel the rows are poured into, not of the round, and they stay with the caller.
func (s *CombatScene) logRows(events []combat.Event) []ui.PaneRow {
	return ui.PaneRowsFor(s.ledgerLines(events))
}

// handTermLines is one line per landing of a hand, in the order the sum counts them.
//
// **A term is a landing, not a card** — an echoed card seats the same index two or three times
// with a figure each — which is why the card is named on every line rather than only the first: a
// player reading back a Three of a Kind wants to see Cut three times, not one Cut and two orphan
// numbers.
//
// `played` is the side's resolved actions in order, which is what `HandCards` indexes. A hand
// naming an action the walk did not see writes no line rather than guessing at one; that cannot
// happen from a resolved round and is checked because the alternative is a panic in a panel.
func (s *CombatScene) handTermLines(e combat.Event, played []combat.Card) []session.LedgerLine {
	if e.HandCardCount <= 0 {
		return nil
	}

	relics := s.wornBy(e.Side)
	out := make([]session.LedgerLine, 0, e.HandCardCount+1)

	// **What raised the DMG comes before the cards it raised**, because that is the order the
	// arithmetic happens in: the rung is read, the duelist swings bigger, and only then is there a
	// term to write. See handDMGLines.
	out = append(out, ui.HandDMGLines(e, relics)...)

	for i := 0; i < e.HandCardCount && i < len(e.HandAmounts); i++ {
		idx := e.HandCards[i]
		if idx < 0 || idx >= len(played) {
			continue
		}
		card := played[idx]
		ink := ui.ElementInk(card.Element)

		spans := []session.LedgerSpan{
			{Text: fmt.Sprintf("%-14s", ui.TermCardName(card)), Ink: ink},
			{Text: fmt.Sprintf("%4d", ui.TermBase(e, i)), Ink: ink},
		}
		spans = append(spans, ui.TermNotes(e, i, relics)...)

		out = append(out, session.LedgerLine{Voice: session.VoiceTerm, Spans: spans})
	}

	// **Then the flat terms, in the order the sum adds them.** They are the same three the hand
	// dialog draws after the cards, and leaving them out is what made this panel print a sum that
	// did not come to its own total. See flatTermLines.
	out = append(out, ui.FlatTermLines(e, relics)...)

	// **The sum, under the terms it adds up, in the figures the hand dialog flew into place.** It
	// is the last line rather than the first because that is the order the arithmetic happens in
	// and the order the dialog acts it out in: the cards, then what they came to.
	out = append(out, session.LedgerLine{Voice: session.VoiceTerm, Spans: ui.HandMathSpans(e, played)})
	return out
}

// wornBy is what a side is wearing, in worn order. The opponent wears nothing today — creatures
// have no fingers — so this is the player's row in every case that matters, and it is asked by
// side rather than assumed so that the day one does, the account says which relic.
func (s *CombatScene) wornBy(side combat.Side) []combat.WornRelic {
	c := s.fighter
	if side == combat.SideB {
		c = s.enemy
	}
	if c == nil {
		return nil
	}
	return c.Duelist.WornRelics()
}
