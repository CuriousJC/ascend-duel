package screens

import (
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

// ledgerRecords is the walk itself: one record per thing that happened, in the order the resolver
// produced them, kept for the length of the run.
//
// **Records rather than sentences.** See session/record.go for the argument and
// internal/ui/ledger_prose.go for the one place they become English — what this walk decides is
// which events are worth an entry and what each entry holds, never how it reads.
//
// **One record per thing that happened, not one per event.** A busy round is 25-30 events, so
// writing the log verbatim would be an account nobody could read. Which events fold into which
// record is presentation of events the engine already decided; it computes nothing, so what is
// written here still cannot disagree with what the round did.
//
// **The attack phase is one record, and it is the hand's.** The defenses write one each; the attack
// cards write none. A turn lands one blow, so five entries saying "Duelist attacks with an earth
// strike" would describe a round that does not happen, and the one that mattered — what the five
// cards came to — would be the sixth. **Every hand takes that record, the No Hand included**: a
// lone attack is the catalog's one-card hand and is announced like any other.
func (s *CombatScene) ledgerRecords(events []combat.Event) []session.LedgerRecord {
	var out []session.LedgerRecord

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

	add := func(r session.LedgerRecord) { out = append(out, r) }

	for _, e := range events {
		switch e.Kind {
		case combat.KindRoundStart, combat.KindRoundEnd:
			// The account holds one round per block, so saying which round it is would be an entry
			// spent on something the heading already carries — and a heading belongs to the caller,
			// which is the only one of the two that knows whether a round has anything before it.

		case combat.KindAction:
			// **A hand-forming side's attack card writes no record.** Its beat still passes — the
			// engine announces every card so the table can light it and playback can count slots —
			// but the entry for the whole phase is the KindHand below.
			//
			// **A solo attacker has no phase line, so the card's own record is the entry.** There
			// is no KindHand coming for it, and an attack reporting nothing but a damage figure
			// with no verb in front of it would be the one action in the round that never says
			// what it was.
			if combat.Plain(e.Action).Category() == combat.CategoryAttack && !s.soloAttacker(e.Side) {
				break
			}
			add(s.actRecord(e.Side, combat.Card{Concept: e.Action, Element: e.Element}))

		case combat.KindChilled:
			add(session.LedgerRecord{
				Kind: session.KindChilled, Side: sideWord(e.Side), Name: s.sideName(e.Side),
				Card: combat.ConceptOf(e.Action).Label,
			})

		case combat.KindMissed:
			// It belongs to the attacker's own entry rather than opening one, because the card
			// *was* played. Naming the shock is the whole point: a blow that simply missed would
			// look like a bug in a game with no dice in it.
			add(session.LedgerRecord{Kind: session.KindMissed, Side: sideWord(e.Side)})

		case combat.KindStatus:
			add(session.LedgerRecord{
				Kind: session.KindStatus, Side: sideWord(e.Side),
				Status: combat.StatusOf(e.Status).Key,
			})

		case combat.KindDrained:
			// **It belongs to the attacker's entry**, like a status does, because it is something
			// the blow did rather than an event of its own. **The relic names itself**, so a second
			// drain relic cannot narrate identically to the first.
			add(session.LedgerRecord{
				Kind: session.KindDrained, Side: sideWord(e.Side),
				Relic: combat.RelicOf(e.Relic).Name, Amount: e.Amount,
			})

		case combat.KindRegenerated:
			// **An entry of its own, where a drain attaches to one.** This happens at the top of a
			// turn with nothing before it, so there is nothing to attach to.
			add(session.LedgerRecord{
				Kind: session.KindRegenerated, Side: sideWord(e.Side), Name: s.sideName(e.Side),
				Amount: e.Amount, Relic: combat.RelicOf(e.Relic).Name,
			})

		case combat.KindBurned:
			// A tick belongs to nobody's card, so it opens its own entry, and it carries the
			// victim's side because it is a thing happening *to* them — which is also the only side
			// the event names. **The status names itself**, so a second damage-over-time status
			// cannot narrate as a burn.
			add(session.LedgerRecord{
				Kind: session.KindTicked, Target: sideWord(e.Target), Name: s.sideName(e.Target),
				Status: combat.StatusOf(e.Status).Key, Amount: e.Amount,
			})

		case combat.KindHand:
			add(session.LedgerRecord{
				Kind: session.KindBlow, Side: sideWord(e.Side), Hand: ui.HandTitle(e),
			})
			out = append(out, ui.HandTermRecords(e, s.wornBy(e.Side), played[e.Side])...)

		case combat.KindRaised:
			// **The count that is standing, not the count this card added.** Two Guards in a turn
			// is one duelist behind six shields, and an account saying "+3" twice makes the reader
			// do the arithmetic the readout has already done.
			add(session.LedgerRecord{Kind: session.KindRaised, Side: sideWord(e.Side), Amount: e.Life})

		case combat.KindVitae:
			// **Two riders pay vitae and they are different things.** A held card is paid for being
			// kept back; a played silver card gambled and came up. See combat.Event.Rider.
			kind := session.KindHeld
			if e.Rider == combat.RiderSilver {
				kind = session.KindSilver
			}
			add(session.LedgerRecord{Kind: kind, Side: sideWord(e.Side), Amount: e.Amount})

		case combat.KindExpired:
			// **Shields that were never spent are the player's own decision coming back**, and a
			// readout that simply went blank would read as a bug.
			add(session.LedgerRecord{Kind: session.KindLapsed, Side: sideWord(e.Side), Amount: e.Amount})

		case combat.KindBlocked:
			// **The only record that the attack happened at all**, since it landed nothing and
			// there is no damage entry coming.
			add(session.LedgerRecord{Kind: session.KindBlocked, Side: sideWord(e.Side), Amount: e.Amount})

		case combat.KindTimeUp:
			// **An entry of its own.** Nobody swung, so there is no attacker's sentence for this to
			// belong to — and the fall that follows would otherwise be the only account of the
			// biggest thing that can happen in a fight.
			add(session.LedgerRecord{
				Kind: session.KindTimeUp, Target: sideWord(e.Target), Name: s.sideName(e.Target),
				Amount: e.Amount,
			})

		case combat.KindDamage:
			add(session.LedgerRecord{Kind: session.KindDamage, Side: sideWord(e.Side), Amount: e.Amount})

		case combat.KindDefeated:
			add(session.LedgerRecord{
				Kind: session.KindDefeated, Target: sideWord(e.Target), Name: s.sideName(e.Target),
			})
		}
	}

	return out
}

// actRecord is one duelist playing one card.
//
// **The card is described rather than worded.** Its label, its element, which half of the turn it
// belongs to and whether it raises something are four facts; the clause built out of them —
// "attacks with a fire strike" — is the translator's, so a change to how a card reads reaches every
// account already saved.
func (s *CombatScene) actRecord(side combat.Side, c combat.Card) session.LedgerRecord {
	rec := session.LedgerRecord{
		Kind:    session.KindAct,
		Side:    sideWord(side),
		Name:    s.sideName(side),
		Card:    combat.ConceptOf(c.Concept).Label,
		Element: ui.ElementName(c.Element),
		Verb:    ui.CategoryInk(c.Category()),
		Raises:  combat.ConceptOf(c.Concept).Verb == combat.VerbShield,
	}
	if c.Category() == combat.CategoryAttack {
		rec.Weight = c.Amount()
	}
	return rec
}

// sideWord is which side a record belongs to, as the ledger names it. **A word rather than a
// combat.Side**, because a record is saved and an ordinal in a file eventually means something else.
func sideWord(side combat.Side) string {
	if side == combat.SideB {
		return session.SideFoe
	}
	return session.SideYou
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
	return ui.PaneRowsFor(ui.LedgerLines(s.ledgerRecords(events)))
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
