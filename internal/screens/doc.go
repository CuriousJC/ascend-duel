// Package screens is one Scene per screen of the game: the title, the duel, the between-fight
// scenes, the credits. Each owns its own state and its own widgets and draws itself; the game
// loop above it does nothing but pick which one is active.
//
// # Where to start
//
// Scene is the contract — Init, Update, Draw — and scene.go explains the lifecycle, including the
// one rule that catches people out: Init may run more than once, because a screen can be
// re-entered, and re-entering the combat screen is how the next fight starts.
//
// flow.go is the loop. A scene that has finished calls advanceRun; where that leads is the run's
// business, not the scene's. Adding a screen to the game is a phase in internal/session, an entry
// in this package's phaseScreens table, and an entry in the registry in internal/game — and
// nothing else, because no scene names its successor.
//
// # The rules this package works under
//
//   - internal/combat decides rounds; a screen only replays them. Never change a rule to make a
//     screen look right. Say which of the two is wrong and let the owner decide.
//   - Presentation may never change an outcome. A whole round is resolved before playback begins,
//     so playback speed, a flight, a dialog that pauses the cursor and every debug view may alter
//     pacing and must not alter results.
//   - A screen's working state lives on its scene, never on GlobalState. If exactly one screen
//     reads a field, it belongs to that screen. If it has to outlive a fight, it belongs to
//     internal/session.
//   - Clicks and drags only. No right click, no hotkeys, and one typed-text field in the whole
//     game — the seed. Anything that wants a keyboard needs a different design.
//   - Cards fly; they never appear. Anything that changes where it is on screen travels there.
//
// # Testing
//
// This package links Ebitengine, so most of it cannot be tested without a window. The tests that
// exist are a deliberate narrow exception: they compare constants and walk switch statements,
// create no images, and guard cross-package invariants a compiler cannot see. They are not licence
// to test the rest of the screen, and nothing here should reach for a window to keep one alive.
// What cannot be unit-tested gets a tool instead — see the demoplay build tag.
//
// # The files
//
// These are one package and Go does not care where a declaration sits, so the boundaries below are
// *reading* boundaries: the point is that an edit does not start by finding your place in two
// thousand lines. A symbol named in the combat-screen skill may sit in any of them, and a grep
// over internal/screens is the way to find it.
//
// A file boundary is not a reason to change what a function does. Moving something between these
// files is a move, not a rewrite.
//
// Shared by every scene, and none of it is the combat screen's:
//
//   - ground.go — the table everything is painted on (screenGround) and the ink for what is
//     written straight onto it (groundInk). Anything drawn on a surface of its own takes that
//     surface's colours instead.
//   - travel.go — how anything gets from one place to another: travel, the delay-age-duration
//     clock every mover shares, plus easeOut, easeIn, lerpPoint and flyingTo. Ease out, so a
//     thing leaves quickly and lands gently. A flight is raised after the model has already
//     moved, so it is a ghost of something that has happened.
//   - card_draw.go — putting a card's picture on the screen. Four near-identical functions in
//     four files until 2026-08-21; wrappers over blitCard now, which is what stops a fifth
//     screen writing a fifth.
//   - card_art.go — the bridge to internal/cards. This side *specs* and card_draw.go *blits*:
//     everything here answers "what does a card of this kind look like", and the cache lives
//     here because rendering a spec writes every pixel in Go and is far too slow per frame.
//   - pane.go — a titled box with a list of rows in it, knowing nothing about combat. A row
//     arrives as three strings and two colours.
//   - prose.go — turning an event the engine has already decided into a sentence: logRows and the
//     vocabulary it draws on. It lives here and not in internal/combat on purpose — the rules
//     package names actions, it does not describe them. It computes nothing, which is what makes
//     it impossible for a panel to disagree with the round it reports.
//   - clock.go — the one speed every movement in the game is a fraction of. beatTicks is the
//     number, beat(num, den) is how anything else is written, and there is no second clock
//     anywhere. It lived on the combat screen until 2026-08-21, which left the between-fight
//     screens outside the setting the duel is paced by. clock_test.go parses this package and
//     fails on a raw duration, because a new screen inventing its own is invisible otherwise.
//   - flow.go — the phase-to-scene table and advanceRun.
//   - theatre.go — everything a scene has moving on it, as a contract rather than a struct: the
//     three rules that apply to all of it, the `theatre` interface a scene's own stage
//     implements, and the advance/running helpers that were four near-identical loops. It is
//     used by a scene rather than owned by one, which is what lets a between-fight screen move
//     things without reinventing the vocabulary.
//
// The between-fight scenes:
//
//   - title.go, ascend.go — the front screen, and two stubs.
//   - postbattle.go — pick a worm, then pick the card it eats. The first of the between-fight
//     scenes; a shop and a room choice come after it, and each is an ordinary scene rather than a
//     mode of the combat screen. It is also the prizes-dealt ring moment.
//   - postbattle_prose.go, postbattle_payout.go — the typewriter and the payout it types. The
//     typewriter is general: a block of lines, a character clock, a pause between sentences, and
//     an optional `pays` per line. The shop reuses it with no claims on any line.
//   - shop.go, shop_prose.go — three rings on a shelf, and the hooded creature who greets you.
//     **The shop has no worn row of its own** (2026-08-22): the build band's ring row is the row
//     that is clicked to sell, so a ring is in the same place on every screen that shows one —
//     and that is why selling asks twice. A click arms a crimson "Sell for N?" tab under the ring
//     and the tab commits it, because the seat a tooltip is asked for and the seat a sale is
//     committed in are now the same pixels. Buying still has no confirm and should not: it is
//     refused when it cannot be afforded, and it is not the click that costs you something you
//     already had.
//   - buildband.go — the duelist card and the worn rings, for a screen that is not a fight.
//     drawBuildBand is both halves; drawBuildCard and drawBuildRings are the halves, split because
//     the shop draws its own rings with a price under them. hoverBuildRings is the row's tooltip
//     for every screen that draws it — the reward screen had none for a day, which is a row a
//     player reads their build off going silent on the screen where they change that build.
//   - deckpanel.go — the deck overlay, as a widget over a `deckContents` rather than a method on
//     the combat scene (2026-08-22, TODO.md). Three screens put it up: a fight through its draw
//     pile, and the reward screen and the shop through `deckToggle` — a 44px `D` in the
//     bottom-right corner, the Log button's shape and rules. The panel shows every card you own,
//     in four colour rows plus a row of plans, at cards.Mini overlapped. **It never hides a card**
//     (owner's call, 2026-08-23): a row that has outgrown the comfortable pitch overlaps harder,
//     per row, rather than dropping the extras under a "+N more not shown" line — see rowPitchFor.
//     The rule that survives is that a card does not move when it is played, it only dims, so the
//     hand is included rather than excluded and availability is the last sort key, never the
//     first. Rows sort stab, slash, crush, plan,
//     cheapest first; formRank is written out rather than read off the enum, because the enum's
//     order is what an expanded hand ID is derived from and that is a rule. It sorts on form
//     rather than category because category has two values now, and sorting by it would put nine
//     cards in one undifferentiated block. Identity is the last key of all (2026-08-24): a card
//     carries an ID now, so two entries equal on every visible key would otherwise be left in
//     whatever order the piles handed them over.
//     The panel carries no words at the top at all (2026-08-24, owner's call): the title, the
//     counts line and the legend all went, and the grid starts at modalBareBodyTop, which clears
//     the close button and nothing else.
//   - deckpanel_view.go — how the panel is being *read*: two latched buttons along its bottom edge
//     and the three tallies under the grid (2026-08-24, owner's call). ALTERATIONS / AS OWNED picks
//     which face every card is drawn in — what the rings will deal, or what the run owns — and is
//     on by default, because a run wearing a flip ring never draws the deck it owns. FULL / PLAYED
//     picks which half the tallies count and which half is lit; it inverts the dimming and moves
//     nothing, which is the panel's governing idea applied to a second toggle, and it is not drawn
//     between fights, where there is one pile. The tallies are by form, by form and AP, and by
//     element, all three at once rather than behind a third toggle, and they count the laid-out
//     grid rather than the piles so both toggles reach them for free. Neither button can change
//     anything: this is a reading preference, the same standing as the hand's sort column.
//
// The combat screen, grouped by what a change is usually about:
//
//   - combat.go — the scene: CombatScene, Init, Update, Draw, startRound, how a round *spends*
//     the game's speed (eventDwells is a multiplier per event kind, all 1 today, with an entry
//     per kind and a test rather than a switch with a default — which is what silently gave new
//     kinds a quarter-second flash the first time this screen had per-kind pacing; the speed
//     itself is clock.go's), playback (advancePlayback, applyEvent, currentSlot), the caption
//     text, nextFight, and the trace layout dump.
//   - combat_deck.go — the cards and the piles: actionCard, the deck seed, the shuffle and draw,
//     spendSelected, and fightContents. actionCard is an alias for combat.Card — elements are
//     rules, so the hand, the queue and the round are one type and a card is never converted
//     between them. The panel that draws the deck left for deckpanel.go on 2026-08-22, because
//     three screens want it; what stayed is the piles, which are a fact about a fight, and
//     fightContents, which is the one place saying how three piles map onto the panel's two.
//   - combat_hud.go — everything around the round: the two fighter cards, drawBox, and the
//     discards badge. Both duelists are cards, in opposite top corners, each holding
//     name / DMG / AP / Vitae over a health bar and a fraction. duelistCardRect and enemyCardRect
//     are the one place each geometry is written, and the ring row takes both of its edges from
//     them.
//   - combat_rings.go — the ring row: full-size cards.RingStyle cards from data/rings.json, a rule
//     under them running the row's width, and the cap written as worn/5 on that rule's right end.
//     It draws what the run is wearing and decides nothing (2026-08-17): session.Session holds the
//     worn keys in worn order and session.Equip puts them on the duelist, so this file is a lookup
//     from key to record for the art and the name. maxRings reads combat.MaxWornRings rather than
//     declaring a second five. Nothing buys or unequips a ring yet. It holds the 12–46% band,
//     which is what pays for full-size ring cards. Its width is what the two fighter cards leave —
//     ringPaneRect reads duelistCardRect and enemyCardRect rather than a percentage, so the right
//     edge cannot go stale when a card moves. Two things it does deliberately: a fill, never a
//     frame — a plain grey backing one step lighter than the screen, no border, no title, no hue,
//     because a framed row reads as cards trapped in a panel while a bare row leaves nothing
//     saying where the middle begins; and the row drops 10px below the two cards so the three do
//     not share a top line and read as one wide object. The backing must never reach either card.
//     And the pitch is a function of how many rings are worn, first card flush left and last flush
//     right, so three stand apart and five close up and overlap by ~26px. Overlap rather than
//     shrink, because a card cannot be scaled and there is no ring style below this one.
//   - combat_actionbox.go — the hand and its drag-to-reorder, over the shared controller in
//     carddrag.go. Reordering a queued hand re-prices it; see combat_sort.go below.
//   - carddrag.go — the press-and-drag lifecycle every reorderable row of cards shares: the hand,
//     and the worn ring row on all three screens that draw it. dragRow is what a row supplies.
//   - combat_sort.go — how the hand is arranged, and the three square buttons that choose it:
//     $ cost, T type, E element, stacked in a column against the band's right edge and centred on
//     the cards. Cost is the default and every mode ends with it — each is the deck overlay's own
//     key chain with one key promoted to the front, so a row of cards means the same thing in the
//     hand as in the panel. Three things about it: the sort re-applies on every refill, so a drawn
//     card lands where it belongs rather than on the right-hand end and a drag survives only until
//     the next deal; sortMode is the one field Init does not reset, because it is a reading
//     preference rather than a fact about a duel; and it *can* change an outcome as of 2026-08-26,
//     since a growing ring steps between the cards of one blow and the queue's order therefore
//     prices them. The buttons stay live anyway — owner's call. sortHand returns the permutation it applied —
//     it sorts a slice of indices and rebuilds rather than sorting in place — because a card
//     sliding to its new seat has to know where it set off from and two identical cards cannot be
//     told apart after the fact. elementRank and categoryRank are written out rather than read off
//     the enums, for the reason formRank is: combat.Basic leads its enum as the zero value and
//     trails on screen, where the colours are what the statuses are counted on.
//   - combat_mathbox.go — the hand dialog: the blow's arithmetic acted out across the band above
//     the hand on the beat a hand fires — each card's own figure flying down into a line, then the
//     multiplier, then the answer, all of it at double the type it was drawn at before 2026-08-19,
//     the landing damage figure included since hitFigureSize is mathTotalSize. It also owns the
//     hand's name, which is one word with two homes (2026-08-19): handBanner sits it in the middle
//     of the table, centred on the screen and over the opponent's cards, while the round is
//     planned, and flies it down into the hand row at DUEL! — at one size, 80 points, mathNameSize
//     — as the cards fly up to the table. It rests there until the sum takes its figure,
//     breathing. A second line under it says what the hand is worth — 1.15x DMG, formatted through
//     the sum's own handMultiplierText and travelling with the name as one object, so the
//     multiplier is known while the hand is being chosen rather than met when it flies out of the
//     word. That line is the figure the sum then flies in: it is written at mathTermSize and does
//     not grow with the name, and the whole banner is taken down on exactly the frame its copy
//     sets off for the line — nothing is left lit over the hand while the sum finishes and the
//     opponent swings back. The name never leaves the screen and is never drawn twice: the box
//     suppresses its own shout while the banner is carrying the same word. Both states are bold,
//     faux, the pane's own two-pass idiom. It exists because a bare (20 x 1.5 = 30) was a record
//     and not an explanation: the total was visible and which card paid for which part of it was
//     not. It says nothing the KindHand event carries and computes nothing — a second drawing of
//     one event, never a second arithmetic — and it is the one thing on this screen that can stop
//     the playback cursor, because a sum revealed a figure at a time does not fit inside one
//     event's dwell. It still cannot change an outcome. mathScript is the half with no geometry in
//     it and is what the tests pin. It also holds mathBandHeight and mathBandGapAboveCards, the
//     depth the sum was laid out against.
//   - combat_shields.go — the pip row on the duelist card, one shield each, kept in step with
//     playback rather than with the model (2026-08-31). Built on combat_hits.go's pattern: the
//     engine has already raised or spent them, shownShields is the view that lets the row move on
//     the beat the event is drawn rather than when the round's end state is adopted. There is no
//     flight — a pip appears in a row on the card the player is already looking at, so the arrival
//     is the drawing.
//   - combat_hits.go — the damage figure landing, and the health bar that waits for it. The -N
//     travels out of wherever the blow was last seen and into the card whose bar it empties, and
//     the bar holds its old figure until the number arrives — so the drop and the arrival are one
//     event rather than two. The model has already moved, exactly as it has for a card in flight:
//     applyEvent writes the new life the instant the event is reached and the figure is raised
//     afterwards, so a figure in the air is a ghost of something that has happened. What lags is
//     the drawing, through shownLife, which is a view over the combatant and never a second copy
//     of it. It is the second thing that can stop the playback cursor, for the mathbox's reason: a
//     figure crossing half the screen does not fit inside one event's dwell, and the alternative
//     is the bar dropping before the number reaches it. Where the figure sets off from is
//     anchorBlow — the sum line when the turn scored a hand, the acting card's own seat when it
//     did not, because a solo attacker emits no KindHand and so has no sum. Its size and colour
//     are the sum's total's, on purpose: advancePlayback clears the box on the frame this
//     launches, so one number appears to set off rather than two to swap.
//   - combat_theatre.go — two things. combatTheatre is this screen's own stage: the cards in the
//     air, the two rows on the table, the damage figures, the banked points, the hand's name and
//     the sum it flies into — eleven flat fields on CombatScene until 2026-08-21, and one field
//     with one tick, one running and one clear since. choreography is the map of what travels,
//     out of what, into what. One entry per
//     EventKind, naming a source anchor, a target anchor, a gesture and the reason — and the
//     drawings themselves stay with the code that already owns each region, because a generic
//     renderer over nine genuinely different gestures would be more machinery than the nine
//     gestures. The rule it holds is one sentence: everything travels from the thing that caused
//     it to the thing it happened to, which is the card-flight argument applied to every event.
//     The checkable part is completeness — TestEveryEventKindIsChoreographed fails when a new kind
//     arrives without an entry, and a kind that genuinely has no picture says so with anchorNone
//     and a reason, because an absent entry and a deliberate silence otherwise read identically.
//     Anchors are named by role, never by side (anchorActorSeat, not a player's row), for the
//     reason SoloAttacks is a flag on a duelist rather than a rule about SideB. It is deliberately
//     not JSON (owner asked): every anchor is a geometry function that takes arguments, and a file
//     can only name one by string, so the Go table a file would need underneath it is this table.
//     If the timings turn out to be what gets edited over and over, lift those out and leave the
//     anchors — timings are data, anchors are code.
//   - combat_flight.go — every card that moves. Three things, all presentation-only, all on their
//     own clock, and none of which may change an outcome. The deck stack and its yellow modal
//     ring. cardFlight — the discard flying off left and the deal flying back in, turning face up
//     on the way; a flight is raised only after spendSelected has already moved the card, which is
//     what keeps planning(), the budget and the row's layout ignorant of it. handSlide — a card
//     moving from one slot in the row to another, a sort or the row closing up after cards were
//     spent; the only mover whose journey begins and ends in the hand, which is why it carries a
//     row size at each end, and flat and full size because it crosses inches rather than the
//     screen. resolvedCard — one of the player's cards flying out of the hand to its seat on the
//     table; the whole queue is dealt there when the round starts, not a card at a time as each
//     fires, so playback drives which card is lit rather than which cards exist. Because
//     ResolutionOrder decides the row, it is laid out in phase order without this file knowing
//     what a phase is. What says which cards earned the hand is which cards are still raised
//     (2026-08-19): noteHand narrows the lifted set to the ones the engine names.
//   - combat_table.go — the two hands facing each other: the player's played cards left-aligned,
//     the opponent's queued cards right-aligned, both full size in the band between the ring row
//     and the strip above the hand. It is what shows a round as a confrontation rather than as a
//     list. Each row breaks between its attacks and its plans (tableGroupGap), and the split is
//     read off combat.ResolutionOrder rather than counted here — the gap is spent out of the same
//     half-width, so it costs overlap rather than width and the two hands still cannot meet. Both
//     rows come from combat.ResolutionOrder, so both say what will happen rather than what was
//     planned. The opponent's cards are drawn face up and that is temporary — concealEnemy is the
//     screen's concealment predicate and this row deliberately ignores it, on the owner's call,
//     with cards.Spec.FaceDown already built as the lever for putting it back.
//   - combat_parasite.go — the bucket: the board piece a parasite is spent from, behind the P
//     button above the fight log. Two stages — pick a parasite, pick the cards out of the hand —
//     and live only while planning(), because a round is resolved before it is drawn.
//   - combat_pileslot.go — the square beside the draw pile, and modalUp. The fight log's button
//     used to stand there; what is left is the geometry, which four other controls are placed
//     against.
//   - ledger.go — the run's account of itself, and the panel that reads it back (2026-09-02).
//     Every fight, folded to a line each, with the fight in progress opened out; a dragged
//     scrollbar, because the input vocabulary has no wheel; and the arithmetic under every blow,
//     term by term, with the ring that priced each one named beside it. It replaced the fight log,
//     which held one fight, could not be scrolled, and dropped its oldest rows. **It is chrome
//     rather than a scene** — internal/game holds it — for two reasons: it is wanted on every
//     screen, and a screen could not be one, since leaving the combat screen and coming back
//     re-runs Init and deals a fresh duel. See session/ledger.go for why the run keeps worded
//     lines rather than the events they came from, and prose_terms.go for the working.
//   - combat_ledger.go — the three call sites that put a duel into that account: a fight opening,
//     a round finishing, a duel settling.
//   - prose_terms.go — a blow's working: one line per landing, what the card was worth, and which
//     ring bought or priced it. Every figure comes off the event, exactly as the hand dialog's do.
//   - seeds.go — the named opening-hand catalogue.
package screens
