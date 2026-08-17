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
- [~] **Rings do one thing, and cannot be acquired.** The four elemental rings confer their
      element's status and nothing else does *(2026-08-16)* — `combat.Duelist.Rings` is what
      `resolveAttackPhase` reads. What is missing is the loop around it: **no buying, no
      equipping, no unequipping**, and the worn set is the `startingRings` constant in
      `internal/screens` (fire, ice, lightning; earth deliberately left off so the gate is
      testable in play).
      - **Blocked on `Session`**, and only that: a ring has to survive a fight and
        `CombatScene` does not. Vitae actually being spent comes after.
      - **The discount and the flip are still unwritten.** `Card.Cost()` is the seat the
        discount sits in and `Duelist.Rings` is now the thing it would read, so both are
        writable — the screen's AP bar and over-budget check are the other half.
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
      - **The deck overlay hides cards as you build.** Rows cap at `deckMaxPerRow = 12` and an
        `element` worm migrates cards between colour rows, so building toward a colour pushes that
        row past the cap. **The replacement is not a bigger cap**: the panel wants counts —
        attacks, plans, how many of each colour — rather than every card drawn at once. Owner's
        ideas, owner's call, deferred.
      - **Worms have no art.** They draw as `cards.Hand` with a zero cost and no family, so what
        shows is the name and the text. A style of their own is what this wants once there is a
        picture to put on one.
      - **No rarity and no weighting.** Every worm is equally likely to be offered.
      - **The catalogue is ten worms across seven targets**, of which four are the same recolour
        in four colours. It wants more *kinds*, not more colours.
      - **Vitae is awarded and never spent.** The +5 card pays into `session.Session`, the combat
        screen shows the purse, and there is no shop to spend it in.

- [ ] **Brands need a data file and a way to be acquired.** The mechanic is already decided —
      see `MECHANICS.md`'s Brands section: they alter the container where rings alter the
      contents, they are permanent *for the run*, and nothing takes one off. What does not
      exist is any of it in code: no `brands.json`, no acquisition, no seat on the duelist.
      Blocked on `Session` like the rest of the run-level state.

## Next — where the game actually starts

- [ ] **Build the ring grammar.** Designed in full on 2026-08-17 and **nothing is built**. Read
      [.claude/skills/rings/SKILL.md](.claude/skills/rings/SKILL.md) first — it holds the
      `When`/`If`/`Then` shape, the three closed vocabularies, the moment→code-seat table and
      what a ring may never do. `MECHANICS.md` under *Rings* holds the argument for the shape;
      *Vitae* holds the propagation rule Banker scales.
      - **Two decisions are still open and both block a specific ring**: whether propagation's
        +5 cap binds Banker as well (`MECHANICS.md` → *Vitae*), and which effect a growing
        ring's accumulator feeds when it holds more than one (the skill). Everything else is
        settled.
      - **Build order, because each step unblocks the next:**
        1. **`statuses.json` and the decoupling.** Nothing else can be expressed until a status
           is a thing in its own right. Re-index `Duelist.Statuses` from element to status,
           move `effectKeys` with it, and decide `cards.MaxEffects` — it is 4 *because* there
           are four elements.
        2. **`Duelist.Rings` becomes a slice** of rules-level rings, and `WearsRing` becomes a
           query over it. This is what unblocks every non-elemental ring.
        3. **Parse `rings.json` in `internal/session`**, beside the worms and for the same
           reason, handing `combat` the rules type. `data` stays ignorant.
        4. **The four combat moments** — `card-cost`, `card-damage`, `attack-lands`,
           `deck-built`. `Card.Cost()`, `Card.Damage()` and `Card.Amount()` are already methods
           on the card, so three of these are a line each rather than a rewrite.
        5. **Vitae propagation as a rule of the run**, then the three moments outside combat:
           `fight-start`, `fight-won`, `prizes-dealt`.
        6. **Growing rings last** — they are the only ones holding state, and the first ring
           thing that will have to be serialized.
      - **`tools/balance` cannot see any of this** and a damage ring is unmeasurable until it
        can. Say so rather than guessing at a multiplier.
      - Ring *appearance* is explicitly not in scope; the row looks fine as it is.

- [ ] **Nothing reads card order any more, so drag-to-reorder decides nothing.** The one-blow
      rewrite on 2026-08-14 took the last two consumers of within-phase order at once: counted
      hands read a turn as a *set*, and every raised defend answers the single blow regardless of
      when it went up. Cross-phase ordering has meant nothing since phases landed. **The gesture
      is still there, still costs the player attention, and now buys them nothing.**
      - **This is the design hole to fill, not a bug to fix.** Either the row stops presenting
        itself as orderable, or something starts reading order again.
      - **Sequence combos are the obvious candidate** and the schema's `run` match kind was
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
      - **The card has no room.** The left column is a family mark over a stack of cost
        dashes, and everything right of it is the effect text; an initiative badge has nowhere
        to go that does not take space from one of the two.
- [?] **Defence targeting has lost most of its content.** The entry used to be "a defense is a
      pool rather than a choice — one card for the first attack and another for the second".
      **There is only one attack now**, so there is nothing to distribute defences across and
      every raised card answers the same blow.
      - What survives is a weaker question: whether a defence should be able to name something
        about the incoming hand — its element, whether it forms a combo — rather than a slot.
      - What is definitely gone is the ordering half. See the drag entry above; this is the same
        hole.
- [ ] **`[?]` Nothing reads recoil, so no enemy deck should hold one yet.** An attack aimed at
      `self` is built and resolves — plain self-damage, before the blow, forming no hand — and the
      planner will never queue one, because it costs life and buys nothing. That is correct as far
      as it goes and it means recoil is currently unauthored content.
      - **What would make it worth playing is a rider**: a self-status (see below), a discount, or
        a hand that pays more for having been bought with blood.
      - `TestThePlannerNeverSpendsAnAttackOnItself` pins that the search does not stumble into one.
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
      | enemies stopped comboing *(2026-08-17)* | 45 |
      | **the 10% ascent curve — what ships** *(2026-08-17)* | **74** |

      - So **the decks cost three walls, the doubling twenty-nine, the combo removal one, and the
        ascent curve another twenty-nine.** Floors 1–2 are untouched — floor 1's outer room is the
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
- [ ] **Nine walls sit on floors 2–4, and those are a bug.** *(2026-08-17)* The total of
      seventy-four is accepted — see the retune entry above — but a wall on a *shallow* floor is
      the failure `tools/balance` was built for. An unwinnable enemy is invisible while playing,
      because losing slowly looks like losing to bad draws.
      - **It was one — Dire Wolf — until the ascent curve landed.** Removing the enemies' combos
        actually fixed Dire Wolf; the 10% per-room curve then put nine enemies in its place, at
        floor 2's outer room. Amber Slime, Android Mk I, Dire Wolf, Giant Spider, Green Slime II,
        Rot Hound, Sken, Specimen A, Yellow Pod.
      - **Floors 1–2 are still clean**, which is the line that matters most: floor 1 is the curve's
        baseline and nothing in the 1–2 or 1–3 bands is a wall.
      - The deep floors are meant to need rings and brands. **Floor two is not**, because there is
        nothing the player could have bought by then.
      - Three ways out and they are not equivalent: retune those nine in `data/enemies.json`, flatten
        the ascent curve (MECHANICS.md carries the `[?]` asking whether the curve or the roster
        should carry the climb), or accept it until more rings exist. **Do not tune until
        `tools/balance` reports a distribution** — see below.
- [ ] **`[?]` Nothing measures what Plan is worth.** `tools/balance` deals no cards, so the
      `planning` posture holds a wider hand of nothing and the row reads 2 AP as pure loss.
      - This needs the sim to draw, which is the deckbuilder entry below and its seventh stream.
        Until then Plan's price is a guess, and it is the one new card whose value the tool is
        structurally unable to see.
- [ ] **`[?]` The three attack families differ only in which cards pair with which.** Stab, slash
      and crush cost the same, hit the same and carry no riders, so a family is a choice of *which*
      pair to build and nothing else. That is deliberately where the rework stopped.
      - What would earn the third family its place is a rider that differs in **kind**: something
        stab does to a defence, something crush does to a status, something slash does across
        several cards. The concept grid's old rule applies — a family that is only a different
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
        because the attack ladder's one-copy-per-colour shape is what makes half the combo grid
        undealable.
- [ ] **The demo has never shown a plan card resolving.** `demoSeedName` is `three-strikes` and
      `demoClickRun` is `Strike`, so the scripted round plays Strikes and nothing else. The
      narration written for Prepare, Plan and Defend has never appeared on screen, and neither
      has the table's attack/plan break — the row it splits has never had a plan card in it.
      The demo is the only thing that looks at the screen without a person sitting there.
      - **The one-blow rewrite made this more urgent, not less.** Nobody has yet *looked* at a
        round where five attack cards are announced and one figure lands, or at a combo line
        naming a hand and a mix together, or at an attack card that resolved and contributed
        nothing. Those are all screen questions and `go test` cannot answer any of them.
      - Point it at a hand that plays a Defend into a heavy enemy turn. `all-plans` (seed 3) deals
        eight distinct concepts including all three plan cards — Prepare, Plan and Defend together
        are 6 AP, so the whole plan vocabulary fits one round — and `both-verbs` (seed 1) is the
        cheapest hand that puts a plan and an attack in the same round.
      - `demoClickRun` has to agree with the seed or the click phase silently selects fewer
        cards and nothing forms. That pairing is what `tools/seeds` exists to keep honest.
- [ ] **The Resolution pane cannot show three things the engine now produces.** All three are
      presentation gaps rather than rules problems, and they arrived together on 2026-08-14:
      - **An attack card that was announced and contributed nothing.** `Strike, Jab, Strike`
        reads as three actions with no sign that the Jab was outside the hand.
      - **Which cards formed the hand.** The event carries the list — a counted hand is not
        contiguous, so it is a list rather than a span — and nothing draws the bracket.
      - **A slot deleted by a stagger**, which the pane still draws as though it happened.
- [ ] **Preview the hand while the player is still planning.** `combat.BlowFor` is exported and
      is the same function the resolver calls, so a previewed combo would be the combo that fires
      by construction rather than by two pieces of code agreeing. Nothing calls it from the
      screen. This is what makes *building toward a shape* legible before DUEL! is pressed rather
      than after.

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
