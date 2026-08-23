# TODO

**The work list.** [MECHANICS.md](MECHANICS.md) is what the game *is*; this is what to build.
When the two disagree, `MECHANICS.md` wins — say so rather than guessing. `ideas.md` is the
unfiltered inbox feeding both.

Completed work is not kept here. Git history has it.

Status: `[ ]` open · `[~]` in progress · `[?]` needs a decision

---

## Now — quick wins, independent of any design decision

- [ ] **The build band's ring row spreads to both edges, and two rings read as two unrelated
      things.** `ringSlotAt` puts the first ring flush left and the last flush right, which is
      right for five and is what closes them up into a hand. On the between-fights band the row
      runs to 99% with no enemy card ending it, so a run wearing two puts one beside the duelist
      card and the other in the far corner of an empty screen. **Visible now that the shop hangs a
      sell price under each one** *(2026-08-22)*. Either the band's row wants its own width, or the
      pitch wants a maximum. Pre-existing; the shop only made it obvious.
- [ ] **Two rounded-rectangle implementations exist.** Cards rasterise their corners in
      plain Go (`internal/cards/shape.go`) because `internal/cards` must render without a
      graphics context; health bars use `CreateRoundedRecMask` + `ebiten.BlendSourceIn`.
      Migrating health bars onto the plain-Go path would collapse the two, and is the only
      way to get back to one — the reverse is impossible, since the mask path needs a window.
      Low priority, but it is a real inconsistency.
- [~] **Rings are bought and sold; what is left is the row itself.** The grammar is built
      *(2026-08-17)* and the shop landed *(2026-08-21)*, so all seventeen are reachable in a run.
      What the shop leaves open:
      - **A run opens bare, so exactly the first duel has no statuses in it** *(owner's call,
        2026-08-21)*. 5 vitae against a base ring of 3 means the first shop can already afford a
        colour. **Nothing has played it**, and the two ends were set a conversation apart.
      - **Nothing re-orders the worn row, and worn order is the firing order.** Rings fire left to
        right and compound, a bought ring goes on at the right-hand end, and selling out of the
        middle shifts everything after it — so the one thing a player cannot choose is the order
        two rings apply in. Drag-to-reorder on the build band's ring row while standing in the
        shop is the obvious answer — that row *is* the shop's worn row as of 2026-08-22 — and it
        is the gesture the action box already has.
      - **Every price is a judgement and nothing measures one.** 2 to 7 off a base ring of 3,
        against an income of 5-10 a fight. `tools/balance` cannot see a ring at all, so what a
        damage ring is worth in vitae has never been checked against what it does to a duel — the
        entry below, on making a worn set a posture axis, is what would.
      - **Nothing is rare and nothing rerolls.** The shelf is three off a flat shuffle of what you
        are not wearing, so a run sees most of the catalogue and no ring is harder to find than
        another.
      - **`[?]` The purse stops binding once five rings are on** — around fight four or five at these
        prices, after which vitae buys nothing but swaps, and propagation is interest on money with
        no use. Either the shop needs a second thing to sell — cards, a reroll, a sixth finger — or
        the late run needs a sink. A design question, not a number to nudge.
      - **One authored name was invented and should be reviewed**: `bulwark-ring` (+25 HP;
        `heart-ring` is the skill's name for the growing one). The discount ring was named on
        2026-08-22 — `thrifty-ring` became `warm-ring`, one of four. Twenty-seven of the
        thirty-five also have **no art**, and draw as a ring card with a hole in it.
      - **`tools/balance` still cannot see a ring.** It wears the four elemental ones and
        nothing else, so a damage, discount or stat ring is unmeasurable — a worn set as a
        posture axis is what that needs.
      - **The two cost sorts read the card's printed cost, not the wearer's.** `costChainLess`
        in `combat_sort.go` and `sortPileEntries` in `combat_deck.go` are pure functions with
        tests pinning them, so a discount ring would order a hand by a number the cards no
        longer show. **The shop landed on 2026-08-21, so a discount ring can be equipped now
        and this is live** — and which of the two numbers a sort *should* use is a real question,
        not obviously the discounted one.
      - `Spec.Dragging` was added for the ring preview and is unused by both the row and the
        hand, which does have drag-and-drop and no visual for it.
- [ ] **The score's loop point is rounded, not authored.** `loopTicks` rounds the last
      note-off to the nearest bar, which for `ascending.mid` trims 60 ticks (about 62ms)
      of a drum tail past bar 13. That is inaudible and the tail is folded back over the
      start anyway, but the rounding is a *guess at intent*. If a future score wants a
      loop that is not its full length — an intro bar played once, say —
      `audio.NewInfiniteLoopWithIntro` already supports it and the loop point would need
      to come from the file (a marker meta-event) rather than from arithmetic.
- [ ] **No human has run a Linux build.** CI builds and tests on Linux under `xvfb-run`, so
      the code compiles and the tests pass there — but no released Linux binary has been
      downloaded and launched on a desktop, and the apt dependency list is the most likely
      thing to be wrong.
      - It **cannot be checked locally from Windows**: `GOOS=linux go vet` fails inside
        Ebitengine's OpenGL driver because cross-compiling disables cgo and the Linux driver
        is cgo-gated. That is the same reason the runner needs the X11/GL/ALSA headers.
      - Sherman has a GUI Linux box and is the intended tester; the owner's is headless.
- [ ] **Bevel the widgets, not just the glyphs.** Buttons, cards and the resolution panes all
      want the treatment the glyphs got — a palette with an outline, a lit edge and a shadowed
      one, rather than a single colour scaled up and down. The "name one colour and scale it"
      rule is really about how a surface responds to hover, press and disable, and it has been
      doing duty as a rule about what a surface may look like, which is further than it needs
      to go.
      - `systems.Palette` already exists and is the obvious thing to widen to.
      - Do the buttons first: they have three states to show, so the payoff is visible
        immediately and the state-versus-surface split gets tested by something real.
      - The panes are the least urgent and the largest areas, so a heavy bevel there will
        read as chrome. Worth doing last and lightly.
- [~] **Worms — the between-fights deck alteration.** *(built 2026-08-17; see MECHANICS.md)* Win a
      fight, choose one of two worms, choose the card it takes. `data/worms.json` holds six, the
      grammar is a target plus a value across seven targets, and `session.Apply` is the one place a
      deck is altered.
      What is left:
      - **The deck overlay no longer hides cards** *(owner's call, 2026-08-23)*. The row cap and
        the "+N more not shown" line are gone; a row that outgrows the comfortable pitch overlaps
        harder instead, per row. What is still open is what a heavily overlapped row is *worth*
        reading — at forty cards a row the visible strip is the form mark and little else, so the
        counts idea (attacks, plans, how many of each colour) is still the thing that would answer
        "what is my deck" better than pictures do. Owner's ideas, owner's call, deferred.
      - **Worms have no art.** They draw as `cards.Hand` with a zero cost and no form, so what
        shows is the name and the text. A style of their own is what this wants once there is a
        picture to put on one.
      - **No rarity and no weighting.** Every worm is equally likely to be offered.
      - **The catalogue is ten worms across seven targets**, of which four are the same recolour
        in four colours. It wants more *kinds*, not more colours.
      - **Vitae is awarded, propagates, and is spent** *(2026-08-21)*. The prize card pays into
        `session.Session`, the purse earns +1 per 5 held after every win *(2026-08-17)*, and the
        shop is what takes it out again. What is unchecked is the rate against the prices: both
        ends were set by judgement, a fight apart.

- [ ] **Brands need a data file and a way to be acquired.** The mechanic is already decided —
      see `MECHANICS.md`'s Brands section: they alter the container where rings alter the
      contents, they are permanent *for the run*, and nothing takes one off. What does not
      exist is any of it in code: no `brands.json`, no acquisition, no seat on the duelist.
      Blocked on `Session` like the rest of the run-level state.

- [ ] **The playback cursor is still seven flat fields on `CombatScene`.** `log`, `cursor`,
      `ticks`, `round`, `rounds`, `fighterAfter` and `enemyAfter` are one thing — the round being
      replayed — and could group the way the theatre did *(2026-08-21)*. **Left alone deliberately
      and it is the weaker case**: those fields already sit together with good names, so what
      grouping adds is tidiness rather than a guarantee. The theatre was worth doing because it
      made a *rule* structural; this would not. Revisit if a second screen ever replays anything.
## Next — where the game actually starts

- [ ] **A status can be authored and a *kind* of status cannot.** `statuses.json` holds four
      records against four closed effect kinds — `damage-over-time`, `lose-actions`,
      `miss-chance`, `damage-reduction` — and every kind has exactly one record using it. That is
      the point of the vocabulary being closed, but it means the decoupling is untested by
      anything real: no second fire status exists, so nothing in play proves a colour can carry
      two.
      - The cheapest proof is a ring, not a status: **Storm is authored** and applies shocked
        *and* chilled off one lightning card, which is the two-statuses-one-card case. It is
        unreachable until something equips it.
      - `cards.MaxEffects` is 4 and the badge row fits six at the current pitch, so a fifth
        status is a one-number layout change with a test already pointing at it.

- [ ] **Nothing reads card order any more, so drag-to-reorder decides nothing.** The one-blow
      rewrite on 2026-08-14 took the last two consumers of within-phase order at once: counted
      hands read a turn as a *set*, and every raised defend answers the single blow regardless of
      when it went up. Cross-phase ordering has meant nothing since phases landed. **The gesture
      is still there, still costs the player attention, and now buys them nothing.**
      - **This is the design hole to fill, not a bug to fix.** Either the row stops presenting
        itself as orderable, or something starts reading order again.
      - **Sequence hands are the obvious candidate** and the schema's `run` match kind was
        dropped in the same rewrite, so they would have to be rebuilt — and rebuilt differently,
        because a sequence is now a shape *within* the one hand rather than a second blow.
        `MECHANICS.md` §Sequences holds what was lost.
      - **Hand sorting widened it on 2026-08-16.** Three buttons now arrange the row and
        re-arrange it on every refill, so the order is machine-chosen as well as mechanically
        inert — a drag survives until the next deal and then goes. Taken deliberately: reading
        the hand is worth more today than a gesture nothing consumes. It does not answer the
        question, it raises the price of leaving it open.
      - The initiative entry below and the defence-targeting idea are both really this question
        arriving from other directions. Answer it once.
- [ ] **Revisit whether an initiative system makes sense for resolution.** There is no
      initiative in the rules: with one contiguous turn per side there is no exchange for a
      faster action to lead, so a number on every card was reporting a distinction the
      resolver had stopped making.
      - **What it would need to come back:** somewhere for it to bite. Ordering *within* a phase
        used to be the free candidate and is no longer — nothing reads it, so initiative would
        have to bring its own consumer. The other candidate is a partial interleave where one
        designated fast action jumps a phase, which reintroduces the legibility problem phases
        were adopted to solve.
      - **The card has no room.** The left column is a form mark over a stack of cost
        dashes, and everything right of it is the effect text; an initiative badge has nowhere
        to go that does not take space from one of the two.
- [?] **Defence targeting has lost most of its content.** The entry used to be "a defense is a
      pool rather than a choice — one card for the first attack and another for the second".
      **There is only one attack now**, so there is nothing to distribute defences across and
      every raised card answers the same blow.
      - What survives is a weaker question: whether a defence should be able to name something
        about the incoming hand — its element, whether it forms a hand — rather than a slot.
      - What is definitely gone is the ordering half. See the drag entry above; this is the same
        hole.
- [ ] **The eight statuses are four.** `MECHANICS.md` §Elements now designs a self-side mirror for
      each element — enflame, focus, charge, ward — and none is built.
      - **Blocked on the source rule having an enemy half**, which is affixes. A self-status needs
        attunement exactly as an opponent-side one does, and a status with no source is not a
        status.
      - It doubles the badge art: `assets/effect/` is keyed by element and would become element ×
        target. `TestEveryStatusElementHasABadge` is what holds that today.
      - **Lightning-on-self must not be a roll.** A shock is the only randomness in the engine and
        `CLAUDE.md` requires a second one to make its argument from scratch.
- [~] **Retune enemy life totals — the doubling overshot, and it is measured.** *(2026-08-16)*
      Every enemy's HP was doubled when the stats changed, at the owner's call, because fights were
      squishy. `tools/balance` before and after:

      | | walls, of 96 |
      |---|---|
      | before per-enemy decks | 12 |
      | per-enemy decks, HP left alone | 15 |
      | decks and doubled HP | 44 |
      | enemies stopped hand-forming *(2026-08-17)* | 45 |
      | the 10% ascent curve *(2026-08-17)* | 74 |
      | hands narrowed to damage alone *(2026-08-17)* | 84 |
      | **three matching axes — what ships** *(2026-08-19)* | **76** |

      - So **the decks cost three walls, the doubling twenty-nine, the hand removal one, the ascent
        curve another twenty-nine and the damage-only narrowing ten — and the three matching axes
        gave eight back.** Floors 1–2 are untouched — floor 1's outer room is the
        curve's baseline — and everything from floor 3 up is a wall.
      - **This is accepted rather than open** *(2026-08-16, owner's call)*. The deep tower is meant
        to need a build, and **rings are what gets the player over those walls** — so the number to
        object to is a wall on a *shallow* floor, not the count. The ascension is not expected to
        be winnable yet.
      - What is left here is the shallow end, and **it got worse rather than better on
        2026-08-17** — see the entry below.
      - **Do not tune anything until `tools/balance` reports a distribution** — see below. Every
        figure it prints is one draw, and it measures a fighter with no build at all.
- [ ] **`tools/balance` is a single sample and needs to be a distribution.** Combat rolls for
      lightning as of 2026-08-14, so a posture winning half its duels and one winning all of them
      print the same line. The tool seeds one fixed `balanceSeed` per run, which keeps a result
      reproducible and hides the variance entirely.
      - What it wants: N duels per matchup off successive seeds, and a win *rate* rather than a
        verdict. The enemy count is 96 and a duel resolves in microseconds, so cost is not the
        obstacle — the output format is.
      - **Every balance figure this repo has ever recorded was measured against the multi-blow
        model and is deleted rather than annotated.** `MECHANICS.md` says what has to be
        re-measured; nothing should be tuned off memory of the old table.
- [ ] **The ladder is priced against rarity and has never been checked against outcomes.**
      *(2026-08-19)* All sixteen entries come from `tools/handodds` through
      `100 + 58.6 × ln(1/P)`, which says a rung pays in proportion to how hard it is to build — and
      says nothing about whether the resulting fights are good ones. `tools/balance` still reports
      who won rather than a distribution, so that check cannot be made yet. See the entry above.
- [ ] **The three axes are a broad buff and nothing on the enemy side answers it.** *(2026-08-19)*
      `tools/balance` goes 84 walls → **76**, and clear speeds move everywhere: Clear Pod falls to
      `all-out` in 4 rounds where it took 6. Most of that is not the multipliers — it is that two
      mismatched attacks now *sum* where the bigger one used to land alone, and that four crush
      cards became a form Four of a Kind rather than a concept Three of a Kind.
      - **The lever is the roster or the curve, not the ladder**, if this turns out to be too much:
        the ladder is now priced against a model and moving it by feel would throw that away.
- [ ] **The best hand is chosen on multiplier, not on the damage it would deal.** *(predates the
      axes; easier to hit since)* `Jab Jab Jab Cut Cut` takes the card Three of a Kind at 255 over
      the card Two Pair at 230, and deals 382 instead of 460, because the trips uses three cards and
      the two pair four. The fix is to pick on the resulting blow and tie-break on the multiplier —
      knowable before resolution, so it breaks none of the matching rules.
- [ ] **`[?]` Two Pair is rarer than Three of a Kind on the form and element axes**, so the ladders
      are forced to climb against their own rarity at those two rungs — 31% paying less than 61% on
      form, 23% less than 42% on element. Either the inversion is accepted or those rungs are
      renamed; poker's ordering is what makes it look wrong.
- [ ] **Nine walls sit on floors 2–4, and those are a bug.** *(2026-08-17)* The total of
      seventy-four is accepted — see the retune entry above — but a wall on a *shallow* floor is
      the failure `tools/balance` was built for. An unwinnable enemy is invisible while playing,
      because losing slowly looks like losing to bad draws.
      - **It was one — Dire Wolf — until the ascent curve landed.** Removing the enemies' hands
        actually fixed Dire Wolf; the 10% per-room curve then put nine enemies in its place, at
        floor 2's outer room. Amber Slime, Android Mk I, Dire Wolf, Giant Spider, Green Slime II,
        Rot Hound, Sken, Specimen A, Yellow Pod.
      - **Eight of the nine survived the three matching axes** *(2026-08-19)*, and Yellow Pod falls
        in three rounds now. The other eight got closer rather than beaten — Rot Hound and Sken go
        from 30% of the enemy's life left to 7% — so this is still the open entry it was, on a
        smaller margin.
      - **Floors 1–2 are still clean**, which is the line that matters most: floor 1 is the curve's
        baseline and nothing in the 1–2 or 1–3 bands is a wall.
      - The deep floors are meant to need rings and brands. **Floor two is not**, because there is
        nothing the player could have bought by then.
      - Three ways out and they are not equivalent: retune those nine in `data/enemies.json`, flatten
        the ascent curve (MECHANICS.md carries the `[?]` asking whether the curve or the roster
        should carry the climb), or accept it until more rings exist. **Do not tune until
        `tools/balance` reports a distribution** — see below.
      - **The margins are readable now** *(2026-08-18)*: the roster table writes a wall as
        `NOTHING - a wall (closest trips, enemy 3% left)`, so a near-miss can be told from a
        hopeless one. `Blue Slime II` ends at 3%, `Green Pod` at 9%, `Blue Cube` and
        `Hunter Drone` at 15–17% — all filed beside `Bio-Titan Omega` at 97% until the column
        existed.
      - **`trips` is the closest posture on every wall in the roster**, which is its own
        finding: nothing else is near.
- [ ] **`[?]` Nothing measures what Plan is worth.** `tools/balance` deals no cards, so the
      `planning` posture holds a wider hand of nothing and the row reads 2 AP as pure loss.
      - This needs the sim to draw, which is the deckbuilder entry below and its seventh stream.
        Until then Plan's price is a guess, and it is the one new card whose value the tool is
        structurally unable to see.
- [ ] **`[?]` The three attack forms differ only in which cards pair with which.** Stab, slash
      and crush cost the same, hit the same and carry no riders, so a form is a choice of *which*
      pair to build and nothing else. That is deliberately where the rework stopped.
      - What would earn the third form its place is a rider that differs in **kind**: something
        stab does to a defence, something crush does to a status, something slash does across
        several cards. The concept grid's old rule applies — a form that is only a different
        word is three cards and one decision.
- [ ] **Two levers answer draw variance and neither is priced against the other.** A hand of eight
      from 48 cards is 17% of the deck; `discardsPerRound` throws cards away and Plan widens the
      next hand by two for 2 AP. They pull in opposite directions — one narrows what you hold, the
      other broadens it — and nothing has decided which is the primary answer.
- [ ] **`[?]` Pair fires on most turns, which makes the bottom rung a global buff to the AI.**
      Any two copies of one attack forms it, and the enemy planners repeat a card far more
      readily than a player assembling a mixed hand does. Measured as a real regression on the
      old model — the player's postures *lost* ground when Pair landed — and the rewrite changed
      the number without addressing the shape.
      - Options: drop Pair's multiplier and start the ladder at Two Pair, or give it a reward
        that does not scale with strength.
- [ ] **`[?]` The mix axis is a function of the hand axis, so most of the grid cannot be dealt.**
      A concept ships one card per colour, so every same-concept hand is forced to all-distinct
      elements: a pair is always duo, a flurry always trio, a barrage always rainbow, and **drab and
      mono are unreachable above one card**. Two axes collapse to one for exactly the hands a player
      is trying to build, which is the strongest argument for changing the deck's composition rather
      than its multipliers.
      - **Drab and Mono are reachable only by the opponent**, whose deck is twelve copies each of
        two drab cards. The mix axis is not dead content; the two sides reach it from opposite ends.
      - **Expected to resolve when the deck changes over a run** — the owner's call, and the
        deckbuilder entry below is where that lands. Recorded so nobody tunes the grid against a
        deck that cannot show most of it.
      - It also means the mix multiplier is not really optional today: every hand pays one.
- [ ] **Procedurally generated enemies.** 96 hand-written records in `data/enemies.json` with
      the combat screen walking a shuffled band per floor is scaffolding. An enemy should be
      **generated** from the floor, so the tower can be endless and a seed can reproduce it.
      - **Assembled from parts, not rolled from scratch.** The pieces that exist or are already
        decided: a **stat line** scaled by floor depth; a **deck** (`internal/decks` has the
        enemy pile); an **affix** that transforms that deck rather than adding to it; a
        **portrait**; and a **personality** — see the leanings note above.
      - **Needs its own randomness stream**, per the stream rules in `CLAUDE.md`. Enemy
        selection already reads `RunSeed ^ enemySelectSalt`; generation is what will draw on it
        next, and must salt its own source rather than share one.
      - **Blocked on nothing**, but the current records become **seeds for the generator or
        test fixtures**, not the roster.
- [?] **Ordering model — phases ship, three alternatives are kept.** `ResolutionOrder` is a
      single pure function, so swapping between any of these is one function body plus its tests.
      - **Phases, chosen and built.** Preparations, then *one* attack, then defenses, then the
        enemy. Defenses front-load because the enemy goes last. Chosen on legibility grounds:
        interleaving may not be graspable by players. **See `MECHANICS.md` for the full entry and
        its costs** — cross-phase reordering stops meaning anything, Guard persistence dissolves,
        and stagger's rarity has to come from elsewhere. The three below are what it was chosen
        over, kept because the experiment may not survive contact.
      - **All three alternatives assume a turn is several separate blows**, which stopped being
        true on 2026-08-14. Contested slots and wind-up time both pair *actions* against actions;
        with one blow per side there is nothing to pair. Reviving any of them now means reviving
        multi-blow attacks with it, which is a bigger change than swapping `ResolutionOrder`'s
        body — **the "one function and its tests" cost quoted above no longer holds for these.**
      - **Contested slots.** Queues alternate; initiative decides who leads each pairing. Every
        action of yours meets one of theirs — "every ask gets an answer". The cost: a fast action
        placed late still resolves late, because initiative never lets an action jump to an
        earlier exchange. Quick means "wins its exchange", not "happens early".
      - **Wind-up time.** Initiative is how long an action takes to come out, accumulating
        down the queue; actions resolve in time order across both sides. Gives the "land
        three fast hits before the slow enemy connects" feel, and reordering still bites
        because your queue is consumed in order. Costs the pairing symmetry entirely.
        Distinct from the rejected speed-timeline model: tempo is bought with card choice,
        not accrued from `Spd`, and AP still caps the action count.
      - **Initiate / respond.** The richest and the biggest change. An exchange is one
        initiator's action plus the opponent's response *if they queued one*; whoever's
        next card is faster initiates, and a card that cannot respond is not consumed —
        it waits. That waiting is what lets a fast plan land several blows against a slow
        one while keeping action-and-answer where both sides planned for it.
        - Needs a taxonomy that does not exist: **role** (can it initiate, respond, or
          both), **response timing** (does the answer land before or after the blow),
          **consumption** (does answering spend the card), **effect** (modify, negate,
          counter).
        - The payoff falls out of response timing for free: let a response resolve first
          only when its initiative is *lower*, and Guard at 2 blocks Strike (3) and Heavy
          (5) but is too slow against Quick (1). **Quick becomes the guard-breaker** from
          numbers already on the cards, and a defensive card's initiative becomes the
          single number saying what it is for.
        - It also dissolves the Guard-persistence oddity. A Guard consumed by the attack
          it answers has no "lasts until its owner acts again" rule to reason about, and
          `TestGuardHoldsWhileItsOwnerDoesNothing` stops describing anything.
        - **Deadlock is a real hazard.** If cards are spent only by initiating or
          responding, two sides holding nothing but responses have nobody to initiate and
          nothing to resolve — an unbounded round. Needs an explicit rule (discard
          unanswered responders at round end, or let a responder initiate as a no-op) and
          a test, because it will not show up until the AI learns to guard.
        - Open besides that: is a too-slow response wasted or does it carry to the next
          attack; do unanswerable cards wait forever or get skipped; should responses cost
          less AP than attacks.
- [ ] **Graded reveal of the enemy's actions.** A concealed action shows its category and not
      its name. That is one cut; the proposal worth building out is to reveal *grades* rather
      than identities — does it damage, does it apply a status, how fast is it. The player reads
      the shape of the opponent's round and plans against it without knowing the specific card.
      - Hidden but graded is a third thing from hidden and from random, and probably the most
        interesting: it rewards reading without punishing with pure guesswork.
      - Wants a reveal level per action rather than the current boolean, and something on
        the enemy side that decides how much leaks — an affix, a ring, a floor property.
      - **The table currently draws the opponent's cards face up**, which overrides concealment
        entirely. `cards.Spec.FaceDown` is the lever built for putting it back; this entry is
        what decides how far back.
- [ ] **Deckbuilder — nothing adds or removes cards.** The hand, draw, discard and reshuffle
      all work, and the deck is 48 cards built from `data/duelist_cards.json`. What is missing
      is the *building*: no card enters or leaves the deck, so there is no thinning reward and
      no acquisition. That needs the loot loop, which needs the tower.
      - **Open:** whether the deck is per-run or per-fight, and whether the discard reshuffles
        within a fight or only between them.
      - **Moving draw into `internal/combat` got cheaper.** `ResolveRound` already takes an
        injected `*rand.Rand` for the lightning roll, so the parameter this entry warned about
        is paid for. **It must not share that source** — a shuffle and a miss-roll are different
        concerns, and the stream rules say a stream is only ever advanced by its own. Doing this
        would let `tools/balance` measure anything that touches the hand.
      - Hand size is 8 and was sized against a 30-card deck, deliberately left alone when the
        deck grew to 48. **It is a base rather than a fixed size since 2026-08-15** — Plan adds two
        for one round, via `handTarget`. Worth re-deciding once thinning exists, and now also
        because the attack ladder's one-copy-per-colour shape is what makes half the hand grid
        undealable.
- [ ] **The demo has never shown a plan card resolving.** `demoSeedName` is `three-strikes` and
      `demoClickRun` is `Strike`, so the scripted round plays Strikes and nothing else. The
      narration written for Prepare, Plan and Defend has never appeared on screen, and neither
      has the table's attack/plan break — the row it splits has never had a plan card in it.
      The demo is the only thing that looks at the screen without a person sitting there.
      - **The one-blow rewrite made this more urgent, not less.** Nobody has yet *looked* at a
        round where five attack cards are announced and one figure lands, or at a hand line
        naming a hand and a mix together, or at an attack card that resolved and contributed
        nothing. Those are all screen questions and `go test` cannot answer any of them.
      - Point it at a hand that plays a Defend into a heavy enemy turn. `all-plans` (seed 3) deals
        eight distinct concepts including all three plan cards — Prepare, Plan and Defend together
        are 6 AP, so the whole plan vocabulary fits one round — and `both-verbs` (seed 1) is the
        cheapest hand that puts a plan and an attack in the same round.
      - `demoClickRun` has to agree with the seed or the click phase silently selects fewer
        cards and nothing forms. That pairing is what `tools/seeds` exists to keep honest.
- [ ] **The fight log keeps a fight, not a run, and cannot be scrolled.** *(the log landed and the
      Resolution feed was removed, 2026-08-18 — `combat_log.go`)* Both limits are real and neither
      is fixable inside the panel.
      - **`Init` clears `s.rounds`**, so the account of a fight is gone the moment the next one
        starts and the post-battle screen has nothing to show. A run log means moving the rounds
        onto `session.Session`, where run-level state belongs — and then deciding what a log of
        forty fights is *for*, because it is a different thing from a fight's.
      - **A long fight loses its oldest rounds.** The panel keeps the newest rows that fit and
        reports the rest as `... N earlier`, which is honest and is the wrong shape for a thing
        built to be read back. The input vocabulary has no wheel and no keyboard; **a drag on the
        panel is the gesture that exists and is unclaimed**.
- [ ] **Nothing on screen says what to press, or why DUEL! is dark.** `(press DUEL!)` was a line in
      the Resolution feed and went with it *(2026-08-18)*; there has been no caption box since
      2026-08-11. The button's own face says DUEL!, which is probably enough for the prompt — the
      second half is not covered at all: a dark DUEL! button is explained only by the AP bar going
      red, which says something is wrong rather than what to do about it.
      - Worth watching for rather than fixing blind. The first version of a fix is a line under the
        bar, and the reason there was never one is that the band it would sit in is the one the sum
        is written across.
- [ ] **The written account cannot show two things the engine produces.** Both are presentation
      gaps rather than rules problems, they arrived together on 2026-08-14, and they moved with the
      prose from the Resolution feed into the fight log *(2026-08-18)* — the walk is the same
      `logRows`, so the gaps are the same gaps. A third — which cards formed the hand — was
      answered on the table instead, which rings them in `attentionYellow`, and by the mathbox,
      which names each one.
      - **An attack card that was announced and contributed nothing.** `Strike, Jab, Strike`
        reads as three actions with no sign that the Jab was outside the hand.
      - **A slot deleted by a stagger**, which the log still writes as though it happened.
- [ ] **Preview the hand while the player is still planning.** `combat.BlowFor` is exported and
      is the same function the resolver calls, so a previewed hand would be the hand that fires
      by construction rather than by two pieces of code agreeing. Nothing calls it from the
      screen. This is what makes *building toward a shape* legible before DUEL! is pressed rather
      than after.

### Cards and piles — presentation

- [ ] **Long press pulls a card forward.** The hand overlaps, so most of a card can be covered by
      the one in front of it, and long press is the gesture that lifts one clear to be read.
      - **Hover took the other half of this on 2026-08-21.** MECHANICS.md §Hover and long press
        now reads *hover explains, long press is the touch equivalent* — the reversal of what was
        recorded when hover was first rejected. What is left here is **un-occluding**, which is a
        separate want from explaining: a tooltip says what a card does and still does not let you
        see the card.
      - **Long press itself is unbuilt**, and it is what a touchscreen or a controller would use to
        ask the question hover asks. It needs the three-way press decision — past
        `dragThreshold` is a drag, held past a tick count without moving is a long press, released
        before either is a click that toggles selection. **The distance and time thresholds must
        not fight each other.**
      - The card's own text is 18pt in a ~100px column and `TestEveryCardTextFitsItsBand` fails
        rather than letting a line off the bottom. Anything else the card wants to say goes in the
        tooltip now rather than needing a bigger card.

- [ ] **The math band should wrap, not shrink** *(2026-08-22)*. `layOutMath` lays the sum out as one
      centred line and, since Echo landed, **shrinks every item by a common factor when the line is
      wider than the band** — floored at `minMathShrink`, 0.6. That is a stopgap: seven terms is
      reachable now (five cards in a legal turn plus the two extra landings an echo seats behind the
      first), and the answer to a line that will not fit is a second line, not smaller type.
      - **Why it is not done yet**: every figure *flies* from the card that paid it into its resting
        place, so a wrap is not a text-layout change — it is a second row of destinations, and the
        `x` and `=` have to land somewhere that still reads as one sum.
      - **What would say it is needed**: a real game showing a shrunk line. `TestTheWidestSumFitsItsBand`
        proves the deliberately-absurd case fits *after* shrinking; it says nothing about whether the
        result is readable at 0.6.
      - It is also the first thing to revisit if `MaxEchoLandings` ever rises above 5.
      - **The arithmetic behind it is already wide enough** *(owner's call, 2026-08-22)*. The event's
        term arrays hold 25 landings — every card of a legal turn, each landing up to
        `MaxEchoLandings` times — so a long repeat-and-echo chain is fully *resolved* today and only
        the drawing of it is short. Wrapping is what lets the screen show what the rules already
        compute.

- [ ] **The tooltip covers four surfaces and not every card on screen** *(2026-08-21)*. Hand cards,
      the deck overlay, worn rings, the shop's two rows, both fighter cards, the reward screen's
      prizes and its offered cards all explain themselves. What does not:
      - **The table's two rows during playback** — the cards actually being resolved. They are the
        one place a player is watching rather than deciding, which is the argument for leaving them
        out, but it is also where "why did that hit for 96" is asked.
      - **Individual status badges.** Hovering the enemy card lists every status on it; hovering one
        badge does nothing, because `internal/cards` draws the row and no badge rectangle reaches
        the screen. A per-badge tooltip needs a geometry accessor from that package.
      - **The AP bar, the discard count, the tower place** and the other figures written straight
        onto the table. Each is a number with no legend anywhere.

## Later

- [ ] **Game speed setting.** User-facing options: *very slow · slow · normal · fast ·
      very fast*, scaling how quickly the duel event log plays back. Ship "normal" only
      to begin with, but route it through a setting rather than a constant so the other
      four are a data change later.
      - Today the pacing is `beatTicks` in [clock.go](internal/screens/clock.go) — one
        constant, one caller, so this is cheap right now and gets steadily more expensive as
        animation and sound land and each grows its own timing constant.
      - Speed must scale *presentation only*. `combat.ResolveRound` already decided the
        whole round before playback starts, so speed can never change an outcome — worth
        protecting, since "fast mode plays differently" is a classic bug in this shape of
        game.
      - Belongs to whatever settings screen eventually backs `SettingsButtonAction`,
        which currently only prints.
- [ ] **Show the run seed, and allow entering one.** `GlobalState.RunSeed` is set once by
      `main` and logged; enemy selection is the first stream reading it. Without a way to see a
      seed and type one back, replayable runs are invisible to the player.
      - **This is the one typed-text field in the whole game**, per the input vocabulary.
      - Both card shuffles read `RunSeed` now, salted per side and per fight by
        `CombatScene.shuffleSeeds`, so a typed seed already reaches the cards. `deckSeed` is
        the debugging pin over the top of it.
- [ ] **Don't pre-roll into a fixed array — keep a seeded stream per concern.** A
      `*rand.Rand` seeded once *is* an infinite deterministic list; a pre-generated slice
      is just the first N entries of it, and N has to be guessed. The endless tower has
      no worst case to size against, so any N is eventually wrong.
      - **Rerolls advance the cursor**, which is exactly the intended behaviour: reroll
        and you get the next offer down the list. No separate reroll stream needed.
      - Replay stays exact because the *list* is fixed by the seed. Identical choices
        consume identical draws; different choices land at a different position in the
        same list. That is the property worth having, and it survives rerolls.
      - Materialize a window of a stream into a slice only when something needs to
        *inspect* it — a balance sim or a test — not as the storage model.
      - The one discipline this needs: a stream is only ever advanced by its own
        concern. Never borrow the loot stream to pick an enemy.
- [ ] **Split the rest of `GlobalState`** into `Resources` (assets/fonts/data,
      read-only), `Layout`, and `Session` (run progress). Deferred: `Session` has nothing
      to hold until the tower loop exists, and the remaining fields are not crowding
      anything. The seed streams and the worn rings live in `Session` when it lands.
- [ ] **Profile — persistent unlocks across runs.** The standard roguelike shape where the
      tower is the run and the profile is what survives it.
      - **A different lifetime from `Session`**, the per-run state in the split above.
        `Session` holds the current floor and the rings collected this climb, and dies with
        the run. A profile spans every run and is the only thing that does.
      - **It is the first thing in the game that must be serialized as real state.** The save
        format decided below is a seed plus a choice log, which works precisely because a run
        is replayable from its inputs. Accumulated unlocks are not derivable from anything —
        they are the residue of runs already thrown away — so a profile needs an actual file
        format with a version stamp and does not get the replay trick for free.
      - **One profile, implicit, for now.** No slot picker and no naming, because naming
        needs a text field and the one text field in the game is spoken for by the seed.
      - What actually unlocks is undecided: cards for the starting deck, enemies in the pool,
        floors, whole alternate decks. Worth answering alongside the loot loop, since an
        unlock and a reward are the same object with different lifetimes.
- [ ] **Several profiles, and the second text screen.** Explicitly a later problem, split out
      so that "one profile for now" does not quietly become "one profile forever". Multiple
      profiles need naming, naming needs typing, and typing makes the one-text-field rule in
      `CLAUDE.md` into two.
      - That is a rule change rather than a feature. Revisit the hand-rolled-UI decision
        under "Open decisions" at the same time — its `[?]` trigger is precisely "the seed
        text field turns out to be painful", and a second field doubles the exposure.
      - Numbered slots picked from a list would dodge the text field entirely, at the cost of
        "Profile 2" meaning nothing to the player.
- [ ] **Ascend / tower loop.** `ascend.go` is a bare `package screens`; `Ascend` and
      `Credits` are empty cases in the scene registry. Structure decided:
      - **8 floors, 3 fights each — 24 fights to the top.** The layout is fixed, not
        generated. Only the enemies and the offers are random.
      - **A binary loot choice after every fight.** Two options, pick one.
      - **A binary floor choice after the last fight on a floor**, on top of that fight's
        loot choice.
      - **Floor 8 ends the run** for the first version — 7 floor choices, no offer at the
        top.
      - Floor choices steer **enemy affixes and behaviour** — "this is a cold floor",
        "this is a fire floor" — plus whatever other levers exist by then. The specific
        options are undecided; the mechanism is the part that matters.
      - Run progress (current floor, current fight, collected rings/brands/pets) is the
        `Session` state in the `GlobalState` split above — this is the feature that
        forces that refactor.
- [ ] **Save format: seed plus choice log, not serialized state.** Falls out of seeding
      for free, and only stays free if nobody builds save/load the other way first.
      - A run is fully described by its seed and the ordered list of **every player
        input**, which is more than the loot and floor picks:
        - **The action set queued each round.** ~5 rounds x 24 fights, so this is the
          bulk of the log, not a footnote.
        - Which of the two loot offers was taken, per fight.
        - Which of the two floor offers was taken, per floor.
        - Every reroll — it is a decision *and* it advances a stream, so omitting it
          desyncs everything after it.
      - A few KB rather than a few dozen bytes, and it grows with duel length rather
        than being fixed size. Still trivial. It survives every change to the shape of
        in-memory state, and doubles as a replay file and a reproducible bug report.
      - Recording action plans is what makes hand-editing a save interesting: loot picks
        only answer "what if I took the other ring", where plans answer "what if I had
        guarded on round 3". It is also what makes a "this seed is winnable" claim
        checkable — a proof is just a choice log that replays to a win.
      - **Serialize card *keys*, not `ConceptID`s.** This got sharper on 2026-08-16: an ID is an
        index into a registry built by walking `duelist_cards.json` and then every enemy's deck,
        so it is stable for one build of one data set and for nothing else. Adding an enemy
        renumbers every concept after it. `Element` carries the same `iota` hazard it always did.
      - **`[?]` The combat roll has to be settled before this ships.** `MECHANICS.md` requires
        rolling on every attack phase and discarding the irrelevant result, precisely so a
        balance tweak does not shift every later roll in a run. `shockMisses` short-circuits when
        the attacker carries no shock, so the stream only advances when lightning is in play —
        which is exactly the drift the rule forbids. **It got narrower on 2026-08-16**: a shock
        now needs a thunder ring on the attacker to exist at all, so an unringed duel advances
        the stream never. Nothing depends on stored seeds yet, so it
        is cheap to fix now and expensive to fix after a save format exists.
      - Serializing live state instead means a migration every time state changes — the
        refactor this whole set of decisions exists to avoid.
      - Cost: loading replays the run to reach the current point. Trivial here, since
        a whole duel resolves in microseconds.
      - Caveat: this only holds while the rules are stable. A balance change invalidates
        old saves, so the format needs a rules-version stamp and a plan for what happens
        when it does not match.
- [ ] **Watch: "is this seed winnable?" as a solvable question.** Not a feature to build
      now — a *property to avoid destroying*. Deterministic combat plus deterministic
      streams plus a bounded choice space means a run is in principle searchable.
      - What it would give: guaranteeing a daily seed is beatable, difficulty grading a
        seed by how narrow its winning lines are, and finding degenerate loot hands
        without playing thousands of runs.
      - Feasibility is unclear and worth being honest about. 24 binary loot choices and 7
        binary floor choices is only ~2^31 paths, which pruning handles easily — but the
        *combat plan* each fight is also a choice, and the number of ways to spend an AP
        budget multiplies that out fast. Fixing a plan policy makes it tractable and
        answers a weaker question: winnable *by this policy*.
      - What would kill it, and therefore what to weigh decisions against:
        - Hidden information the solver cannot see.
        - Randomness resolved *during* a fight from an unseeded source. A *seeded* shuffle
          is not this: the deck order is fixed by the seed, so a solver exploring a line of
          play knows its draws exactly and replay stays exact — the branching factor grows
          but the question stays well-posed. What would genuinely kill it is randomness
          drawn from a global or a clock, which the determinism rules already forbid.
        - Unbounded state — anything that makes a position not comparable to another.
        - Real-time or reflex elements.
      - **The lightning roll is the first thing to test this against, and it survives.** It is
        seeded and injected rather than global, so a solver exploring a line still knows the
        outcome exactly — the branching factor is unchanged because the roll is a function of
        the seed and the position, not a fresh coin. What it *does* cost is that the conditional
        advance above makes two positions with the same visible state consume different amounts
        of stream, so fixing that is worth something here too.
      - None of the rest are on the roadmap, so the property is currently mostly free. Re-read
        this before adding anything that breaks one.
- [ ] **Endless tower (after the 8-floor version works).** Keep climbing until the curve
      stops you, rather than a fixed summit. Scaling probably exponential.
      - Design the floor loop so 8 is a *configured stop*, not a baked-in constant, or
        this becomes a rewrite instead of a setting.
      - Exponential scaling wants a sanity check on integer range and on the health bar:
        the bar scales by `CurrentLife/MaxLife` so it copes, but a four-digit damage number
        will not fit the fighter cards.
      - The interesting design question is what actually stops you. Enemy stats
        outrunning yours, or a resource that runs down?
- [ ] **Enemy model: one archetype, scaled and affixed.** Enemies as the same creature with
      main stats growing by depth, plus affixes that may stack. This contradicts how
      `data/enemies.json` is shaped — 96 fully-specified records — so the data wants to become a
      base statline, a scaling rule, and a pool of affixes to draw from.
      - `AvailableAffixes` already anticipates this and is still unread.
      - Affixes must compose. Two on one enemy is the normal case, not an edge case.
      - Floor choices feed this directly: "a cold floor" biases which affixes appear.
- [ ] **Headless balance sim for the difficulty curve.** `tools/balance` plays one posture per
      enemy; the curve wants thousands of duels across floor depths, plotted against where the
      player loses.
      - `internal/combat` has no Ebitengine dependency precisely so this is possible.
      - Needs the enemy scaling rule and a plausible player-plan model first, so it is
        downstream of the two items above — but it is the reason to keep combat pure.
- [ ] **Affixes.** `cold` / `hot` / `charged` / `undying` are in `data/enemies.json` and
      never read. Ring and effect art is partly present.
- [ ] **Sign the Windows binary so Defender stops saying "Unknown publisher."** Not urgent and
      **not cheap** — this entry exists mainly so the cost is written down before someone
      assumes it is a workflow tweak.
      - **Signing changes the name in the dialog, not the dialog.** SmartScreen gates on
        *reputation*, which a brand-new certificate does not have. A signed build shows
        "Justin Crosby" instead of "Unknown publisher" and still warns until enough clean
        downloads accrue. Only an EV certificate has historically carried instant
        reputation, and even that is reported to reset on renewal.
      - **A `.pfx` in GitHub secrets is not an option.** CA/Browser Forum rules since
        June 2023 require the private key on FIPS 140-2 Level 2 hardware or a cloud HSM,
        so CI signing means a hosted signing service, not a file.
      - **Azure Artifact Signing** (renamed from Trusted Signing) is the cheap path at
        ~$9.99/month and is built for CI. Eligibility is the catch: it was restricted to
        US/Canada organisations with three years of verifiable history, with individual
        signup coming and going through preview, and there are reports of the role
        assignment needing a paid Entra ID tier on top. **Check current eligibility before
        costing this** — the answer has moved repeatedly.
      - **It would put signing credentials in the release workflow**, which is the one job
        already holding `contents: write`. Weigh that against the repo's rule that the
        publish job is the last place to widen access.
      - `[?]` **This may be the wrong problem to spend money on.** The game is intended to
        sell through Steam, which delivers its own signed client and does not put the
        binary through this path. If the GitHub releases stay a developer-and-playtester
        convenience, "Unknown publisher" may be acceptable indefinitely. Decide whether
        the download page is ever a *player-facing* distribution channel first; that
        answer, not the price, is what settles this.

## Open decisions

- [?] **Title hue shift.** `title.go` calls `ChangeHSV(1, 1, 1)`; `hueTheta` is *radians*,
      so that's a ~57° rotation, not identity. Source PNG is warm gold/amber. If the
      title looks greener on screen than the file, it's unintended — identity is
      `ChangeHSV(0, 1, 1)`.
- [?] **Hand-rolled UI versus [ebitenui](https://github.com/ebitenui/ebitenui).** Hand-rolled
      wins today and the trigger for revisiting is written down: **if the seed text field turns
      out to be painful.** A text input with a caret, selection and clipboard is the one widget
      genuinely cheaper to take than to build. Everything else the game needs — buttons, cards,
      drag-to-reorder with live action-point validation — is a *game* widget, which is exactly
      where general-purpose toolkits are weakest, and a toolkit would also be a dependency to
      licence-check against a product that will be sold.

## Licensing (for an eventual Steam release)

Model: source stays public under PolyForm Noncommercial 1.0.0, nobody else may commercialise
it. Justin and Sherman have a signed agreement covering the relicense. The Apache 2.0 grant on
everything published before the relicense is irrevocable and accepted — no history rewrite.

- [ ] **Put Sherman's legal name in `LICENSE`.** The Required Notice currently names the
      GitHub handle `KingSherman1820`. Deliberately deferred — a written partnership
      agreement covers the two of them — but a copyright notice naming only a handle is
      weak if it ever has to be enforced.
- [ ] **`THIRD-PARTY-NOTICES` file.** Apache-2.0 and BSD deps may sit inside a
      restricted-licence product, but only if their notices and attributions travel with
      the binary. Needed for a Steam build, not for the repo.
- [ ] Contact address for licensing enquiries. Deferred deliberately; anonymous is fine
      for now, and `CONTRIBUTING.md` points people at issues instead. Use a purpose-made
      address rather than a personal one when it happens.
- [ ] Get thirty minutes of actual legal review before relying on any of this. The
      licence is standard and well drafted; the contributor grant in `CONTRIBUTING.md`
      is a reasonable draft written by a non-lawyer.
- [ ] Confirm FiraSans and RobotoFlex (expected OFL / Apache — low risk).

**Cleared, and the register to check a new dependency against.** No GPL anywhere — it cannot
go into a product licensed this way.

| What | Licence |
|---|---|
| Ebitengine | Apache 2.0 |
| `github.com/ebitengine/oto/v3` | Apache 2.0, first-party to Ebitengine |
| `golang.org/x/*`, incl. `golang.org/x/image` | BSD-3-Clause |
| `Kubasta.ttf` | CC0, per the author's own FontStruct page |
| Enemy portraits — PVGames, Humble *Isometric Assets Galore* | permits shipping inside a game |
