# TODO

**The work list.** [MECHANICS.md](MECHANICS.md) is what the game *is*; this is what to build.
When the two disagree, `MECHANICS.md` wins — say so rather than guessing. `ideas.md` is the
unfiltered inbox feeding both.

Completed work is not kept here. Git history has it.

Status: `[ ]` open · `[~]` in progress · `[?]` needs a decision

---

## Now — quick wins, independent of any design decision

- [ ] **Two rounded-rectangle implementations exist.** Cards rasterise their corners in
      plain Go (`internal/cards/shape.go`) because `internal/cards` must render without a
      graphics context; health bars use `CreateRoundedRecMask` + `ebiten.BlendSourceIn`.
      Migrating health bars onto the plain-Go path would collapse the two, and is the only
      way to get back to one — the reverse is impossible, since the mask path needs a window.
      Low priority, but it is a real inconsistency.
- [~] **Rings are drawn and nothing else.** `cards.RingStyle` and `cards.Ring` draw one —
      same format, pink border, artwork instead of glyphs, no cost or category — and the
      combat screen has a **ring row** drawing the entries in `data/rings.json` at full size,
      with `worn/5` under it. That is the layout only: no buying, no equipping, no
      unequipping, and no rule reads a ring.
      - **Blocked on `Session`**, and only that. Elements are in `internal/combat` and
        `Card.Cost()` is the seat a discount sits in, so the discount and the flip are
        writable now — but a ring has to survive a fight and `CombatScene` does not. Vitae
        actually being spent comes after.
      - The sketch equips everything defined up to the cap on every `Init`, standing in for
        equipment until `Session` exists.
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

## Next — where the game actually starts

- [ ] **Revisit whether an initiative system makes sense for resolution.** There is no
      initiative in the rules: with one contiguous turn per side there is no exchange for a
      faster action to lead, so a number on every card was reporting a distinction the
      resolver had stopped making.
      - **What it would need to come back:** somewhere for it to bite. The obvious
        candidates are ordering *within* a phase (currently queue order, which is free and
        legible), or a partial interleave where one designated fast action jumps a phase.
        Both reintroduce the legibility problem phases were adopted to solve, so the bar is
        that it buys something combos and category ordering do not.
      - **The card has no room.** The left column is a category glyph over a stack of cost
        dashes, and everything right of it is the effect text; an initiative badge has nowhere
        to go that does not take space from one of the two.
- [ ] **Defenses target a specific incoming attack.** The resolution order is right; what is
      missing is that a defense is a pool rather than a choice. **Guard stays untargeted** — it
      covers you entirely, which is exactly why it is a prepare. **Dodge and Riposte should each
      name the enemy attack they answer**: dodge the first and riposte the second, or riposte the
      first and dodge the second, and those are different rounds.
      - **This is the ordering lever initiative was meant to be, and a better one.** Initiative
        decided *when* your action happened; this decides *what it happens to*, which is a
        decision the player makes rather than a number they read off a card.
      - **Half of this landed on 2026-08-14.** `Duelist.Defends` is an ordered queue and the
        front of it answers the next attack, so *which of your defences meets which blow* is
        already the player's decision, made by dragging within the defend phase. What is still
        missing is naming a **specific enemy action** rather than a position in the sequence —
        which is what makes it robust to the opponent queueing more or fewer attacks than you
        expected.
      - **The hard part is the UI, not the rules.** Engine-side this is a target field on
        the queued action and a lookup at negation time. Screen-side the player has to point
        at an enemy attack that is **concealed while planning** — you know the enemy has two
        actions and their categories, not what they are. Targeting "their second attack" is
        therefore targeting a slot, not a card, which may actually be the cleaner design.
      - Input vocabulary allows exactly click and drag. Dragging a Dodge onto an enemy row
        in the Resolution pane is the obvious gesture and would give that pane its first
        interactive job. No right click, no keyboard.
      - **Open:** what happens when the targeted attack does not arrive — the enemy queued
        fewer actions than you predicted, or died first. Wasted, or does it fall back to the
        next attack? Wasted is more honest and makes prediction matter; falling back makes
        targeting a free upgrade over the current pool.
- [ ] **Enemies plan from a hand but not from a personality.** `combat.PlanFor(style, duelist,
      hand)` picks between four planners named by a string on the data record, so behaviour is
      data and the 96-enemy roster in `data/enemies.json` is tunable without touching Go.
      - **brute** biggest attack affordable · **swarm** as many attacks as the round allows ·
        **warden** Guard then attacks · **tactician** banks with Gather then unloads.
      - What it wants to become is a set of **leanings** — how readily it defends, whether it
        banks, whether it plays toward a combo — that a generator can dial, rather than one of
        four functions chosen by a string. That is the piece procedural enemies need.
      - `AvailableAffixes` is in `data/enemies.json` and read by nothing.
      - `[?]` Earth has no affix; whether it can be a floor theme is still open.
- [ ] **Shocking beats the whole roster, and dodging nearly does.** `tools/balance` over all 96
      enemies, 2026-08-14: **shocking 96, dodging 93, retreating 86** against `all-out`'s 50. A
      posture strong against everything is a pricing bug, not a good card.
      - **Shock is the clearest case and the lever is one constant.** `shockPerHit` in
        [status.go](internal/combat/status.go): a lightning attack deals full damage *and*
        cancels an enemy attack, so two of them a round negate most enemy turns for free. A
        shock that took two hits to apply, or that only stopped the victim's first attack, are
        the two candidates.
      - **Negation is priced against how many blows arrive**, so widening enemy budgets makes
        dodging and retreating *better*, not worse. Raising enemy AP would lift Guard and Brace
        with them, though, which could narrow the gap without touching the negations — a data
        change in `data/enemies.json` plus a re-run.
      - Read the balance table as a **best case**: the fighter repeats one posture every round
        and always draws it. An enemy that beats a posture there beats it always; the reverse
        does not hold.
- [ ] **`[?]` Retreat is a bigger Dodge, and the concept grid says every tier should differ in
      kind.** Three negations for four points is priceable — which is why it replaced a 4-cost
      defend that negated a whole turn and reflected it — but it is the one cell in the grid
      that is only a number.
      - What would earn the cell: a rider that makes *volume* mean something. Clearing a status
        as you give ground, or the third charge doing something the first two do not.
      - `retreatCharges` is the lever if it only needs tuning.
- [ ] **Guard versus Dodge.** The *guarding* posture (Guard + Strike, 5 of 6 AP) loses across
      the roster where *dodging* wins everywhere. Guard costing 3 eats most of a round's budget
      and leaves one Strike behind it, so a broad halving does not pay for itself.
      - Candidates: Guard to 2, or the budget grows, or halving becomes something stronger.
      - A defensive card's worth is set by the shapes it faces, so tune it against the whole
        roster rather than against a handful of enemies.
- [ ] **Two floor-1 enemies are free.** `tools/balance` has **Clear Pod** and **Clear Slime**
      losing to every posture — its own "free" condition. Both are wardens on floors 1-2, so a
      pushover opener may be intentional; decide that rather than leaving it as an accident of
      the stat line. The tool was written to catch this, and it is currently the only thing it
      is flagging.
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
      - **Phases, chosen and built.** Preparations, then attacks, then defenses, then the enemy.
        Defenses front-load because the enemy goes last. Chosen on legibility grounds:
        interleaving may not be graspable by players. **See `MECHANICS.md` for the full entry and
        its costs** — cross-phase reordering stops meaning anything, Guard persistence dissolves,
        and stagger's rarity has to come from elsewhere. The three below are what it was chosen
        over, kept because the experiment may not survive contact.
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
      all work, and the deck is 60 cards built from `data/duelist_cards.json`. What is missing
      is the *building*: no card enters or leaves the deck, so there is no thinning reward and
      no acquisition. That needs the loot loop, which needs the tower.
      - **Open:** whether the deck is per-run or per-fight, and whether the discard reshuffles
        within a fight or only between them.
      - **Moving draw into `internal/combat` is the change to weigh carefully.** It would let
        `tools/balance` measure anything that touches the hand — Sift especially. The cost is
        that `ResolveRound` grows an injected source parameter and `TestRoundIsDeterministic`
        changes shape to seed it explicitly. Never a global; see the determinism rules.
      - Hand size is 8 and was sized against a 30-card deck, deliberately left alone when the
        deck doubled to 60. Worth re-deciding once thinning exists, since a thinned deck changes
        what a hand of 8 means.
- [?] **What a Feint does when a second defence sits behind the one it strips.** The strip takes
      one charge off the front of the defend queue and the blow then meets whatever is now at
      the front, so **two Dodges beat a Feint and one does not** — and a Retreat, at three
      charges, beats it twice over. That is coherent and nobody chose it; it is a consequence of
      where the strip sits.
      - The alternative reading is that a Feint's damage bypasses the defend layer entirely once
        it has stripped something, which is stronger and makes stacked defences worthless
        against it.
      - `TestFeintStripIsUnconditional` pins the current behaviour against a Retreat, so the
        rule is at least written down now. What is still open is whether it is the right one.
- [?] **Does Sift stack?** It does today: `siftsResolved() * siftExtraDiscards`, so two Sifts
      throw four extra cards away. Gather's within-round stacking is a documented deliberate
      choice; Sift's is just what the loop does. Two Sifts is 4 AP of a 6 AP round to churn
      six cards, which may be fine — but it should be a decision.
- [ ] **Price Sift, and build something that can measure it.** 2 AP replacing seven of eight
      cards largely erases the consistency cost of a 60-card deck, which is a lot for the
      cheapest prepare after Gather.
      - **`tools/balance` structurally cannot see it.** Sift's effect is on the hand, the hand
        is on the scene, and the tool has no deck — so this is the one concept in the game
        with no instrument pointed at it. Either the tool grows a deck (see the deckbuilder
        entry) or something else measures draw variance. The second is probably a small harness
        over `screens.OpeningHand`, which already deals real hands headlessly.
- [ ] **Move two interaction rules from code comments into `MECHANICS.md`.** Both are decided,
      implemented and tested; both are recorded in the wrong file, and MECHANICS is supposed to
      be what the game *is*. A designer reading it today would not learn either:
      - Brace and Guard **both** apply, quartering a blow — a Brace answers the attack as the
        front of the defend queue, and the Guard behind it halves what is left.
      - Feint's strip is **unconditional** and fires even when the blow is about to be stopped
        anyway, deliberately, so the card has no hidden interaction with something the player
        cannot see.
- [ ] **The demo has never shown a new card resolving.** `demoSeedName` is `strike-flurry` and
      `demoClickRun` is `Strike`, so the scripted round plays Strikes and a Riposte. The
      narration written for Feint, Retreat, Brace and Sift — the "stopped by a retreat" line,
      the "strips their riposte" line, "braced" — has never appeared on screen, and the demo is
      the only thing that looks at the screen without a person sitting there.
      - Point it at a hand that plays a Retreat into a multi-attack enemy turn.
        `all-categories` (seed 6) deals eight distinct concepts including Retreat and Brace, and
        a swarm throws five attacks — enough to spend all three charges and show the fourth blow
        landing, which is the whole card in one round.
      - `demoClickRun` has to agree with the seed or the click phase silently selects fewer
        cards and nothing forms. That pairing is what `tools/seeds` exists to keep honest.

### Cards and piles — presentation

- [ ] **Long press pulls a card forward.** Every card now carries its effect text — see
      `cardEffects` in [combat_panes.go](internal/screens/combat_panes.go) — filling the card
      beside the cost column. The hand overlaps, so most of a card can be covered by the one in
      front of it, and long press is the gesture that lifts one clear to be read.
      - **This is long press, not hover.** `MECHANICS.md` §Long press assigns "explains" to long
        press and records that hover was considered and rejected; CLAUDE.md's input vocabulary
        has no hover in it. The split to preserve if hover ever returns is **hover un-occludes,
        long press explains** — printing the text on the face has merged the two, and this task
        is what is left of both.
      - The gesture has a designed shape: a press is a three-way decision — past
        `dragThreshold` is a drag, held past a tick count without moving is a long press,
        released before either is a click that toggles selection. **The distance and time
        thresholds must not fight each other.**
      - The text is 18pt in a ~100px column, centred in the space the cost column leaves, and
        `TestEveryCardTextFitsItsBand` fails rather than letting a line off the bottom.
        Anything else the card wants to say needs a bigger card.

## Later

- [ ] **Game speed setting.** User-facing options: *very slow · slow · normal · fast ·
      very fast*, scaling how quickly the duel event log plays back. Ship "normal" only
      to begin with, but route it through a setting rather than a constant so the other
      four are a data change later.
      - Today the pacing is `eventDwellTicks` in [combat.go](internal/screens/combat.go) — one
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
      - The two card shuffles still read fixed constants — `deckSeed` on the scene and
        `decks.EnemySeed` — and both become `RunSeed`-derived when `Session` lands. Each salts
        its own source; never share one.
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
      - **Serialize action names, not `iota` ordinals.** `ActionKind` and `Element` are both
        `iota`-based and append-only, so inserting a new one anywhere but the end silently
        reinterprets every existing log — a saved `Guard` becomes whatever now sits at 1, with
        no error. Same applies to any other enum that reaches the save file.
      - Serializing live state instead means a migration every time state changes — the
        refactor this whole set of decisions exists to avoid.
      - Cost: loading replays the run to reach the current point. Trivial here, since
        combat is pure integer arithmetic and a whole duel resolves in microseconds.
      - Caveat: this only holds while the rules are stable. A balance change invalidates
        old saves, so the format needs a rules-version stamp and a plan for what happens
        when it does not match.
- [ ] **Watch: "is this seed winnable?" as a solvable question.** Not a feature to build
      now — a *property to avoid destroying*. Deterministic combat plus deterministic
      streams plus a bounded choice space means a run is in principle searchable.
      - What it would give: guaranteeing a daily seed is beatable, difficulty grading a
        seed by how narrow its winning lines are, and finding degenerate loot combos
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
      - None of those are on the roadmap, so the property is currently free. Re-read this
        before adding anything that breaks one.
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
