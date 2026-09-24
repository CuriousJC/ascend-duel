# TODO

**The work list.** [MECHANICS.md](MECHANICS.md) is what the game *is*; this is what to build.
When the two disagree, `MECHANICS.md` wins — say so rather than guessing. `ideas.md` is the
unfiltered inbox feeding both.

Completed work is not kept here. Git history has it.

Status: `[ ]` open · `[~]` in progress · `[?]` needs a decision

**Nothing is added here unless the owner asks for it to be tracked.**

---

## Now — quick wins, independent of any design decision

- [ ] **Stand up the private-asset bucket and arm the release guard** *(owner asked for this to
      be tracked)*. **Nothing here is a repository change** — the workflow, the sync script and
      the guard are written, reviewed and turned off. Both release steps, the OIDC sync and
      `go run ./tools/privateassets -require`, are gated on the `SOUNDS_BUCKET` repository
      variable, so a release cut in this state ships the synthesized score everywhere and passes.
      What is left is AWS account work and three variable settings, in this order.

      **1. AWS, once.**
      - An OIDC identity provider for `token.actions.githubusercontent.com`, if the account does
        not already have one.
      - A private bucket with public access blocked, holding the bundle under the prefix the
        manifest names: `aws s3 sync privateassets/audio/ s3://<bucket>/sounds/v1/ --exclude "*"
        --include "*.wav"`. **`v1` has to match the `Bundle` field** in
        `privateassets/audio/manifest.json`, which is where the script reads the prefix from — a
        prefix typed into the workflow instead is the edit nobody reviews.
      - A role with a read-only policy: `s3:GetObject` on `arn:aws:s3:::<bucket>/sounds/*`, plus
        `s3:ListBucket` on the bucket with a prefix condition. **The list permission is not
        optional** — `aws s3 sync` enumerates before it fetches and fails without it.
      - A trust policy with the audience `sts.amazonaws.com`, which is the audience the script
        asks GitHub for, and a `sub` condition. **The two release entrances produce different
        subs**: a manual run is `repo:CuriousJC/ascend-duel:ref:refs/heads/main` and a pushed tag
        is `repo:CuriousJC/ascend-duel:ref:refs/tags/v*`. One `StringLike` on
        `repo:CuriousJC/ascend-duel:*` covers both and is the loosest worth having; two patterns
        is the tighter version. **This repository is public**, so the trust policy is what stands
        between a merged pull request and the bucket — read-only on one prefix is what keeps the
        worst case boring.

      **2. Confirm the runners have `jq` and the `aws` CLI**, including under bash on
      `windows-latest`. Both are believed preinstalled and neither has been checked; the script
      needs both, nothing tests for them, and the failure lands in the job holding the token.

      **3. Set the three repository variables** — `SOUNDS_BUCKET`, `SOUNDS_ROLE_ARN`,
      `SOUNDS_REGION`. Setting `SOUNDS_BUCKET` is the switch: it turns the sync and the guard on
      together, which is why both steps carry the one condition.

      **4. Cut a release and read the log.** `require the private assets` should print `every
      manifested file is present, matching and attributed`. `manifested but not present` means
      the sync fetched nothing — wrong prefix, or credentials that did not assume — and the build
      fails rather than shipping quiet, which is the behaviour the whole arrangement is for.

      Until all of this is done a release ships without the loops. That is a correct build and a
      quieter game than the one on the owner's machine, not a broken one.

- [?] **Three runes from the owner's list still need a design decision before they can be
      written as data** *(owner asked for this to be tracked)*. **Lucky card**; **chance to
      increase a relic** (which relic, and increase what?); **wild card** (matches any axis, or
      any one axis you name?). Two of them are random, which needs its own stream and its own
      argument in `MECHANICS.md` per the `randomness` skill.

- [ ] **Re-check the tutorial's taught shop after the relic key renames** *(owner asked for this
      to be tracked)*. The shop shelf is a weighted draw over `relics.json`'s *sorted* keys, so
      any record added, removed or renamed moves what seed `0009D4` lands on. Nothing fails,
      because no test asserts which relics the taught shelf offers; the shop step is the part of
      the lesson a catalog edit can break silently.
      `SEEDSEARCH=1 go test ./internal/screens -run TestFindATutorialSeed` is the search if the
      shelf now reads badly. See the tutorial section of `CLAUDE.md`, which carries the
      constraints a replacement seed has to satisfy.

- [ ] **A curve tool: plug in bases, pick a motif, read the whole tower** *(owner asked for this
      to be tracked)*. The ascent curve is two compounding growth rates in `data/tower.json` and a
      per-record `HP`/`DMG` base in a motif file, and the only way to see what a number does today
      is to play to the floor it lands on. What is wanted is a page that takes the bases and the
      two rates and prints every fight in the tower — floors down the side, outer / inner / boss
      across, `HP / DMG` in each cell — plus a few rows past floor 8 so the geometric wall is
      visible.
      - **The arithmetic is `pyramid.ScaleToFight`**, which is fixed-point integer on purpose (see
        `ascent.go`), so the tool must call it rather than reimplement it in floating point — two
        answers to one question is the stale-sheet failure.
      - **The step is the fight, not the floor**: `step = (floor-1)*FightsPerFloor + room`, which
        is what makes floor 2's outer room harder than floor 1's boss.
      - **Picking a motif means reading its records' bases** and drawing one column per record, so
        the page answers "what does a Goblin Bomber actually hit for on floor 5" rather than
        "what does base 100 do".
      - **Rows past floor 8 are the point, not a flourish** — the endless tower is where the two
        rates have to be felt, and a table stopping at the summit says nothing about them.
      - It belongs under `docs/sheets/` with the rest, built by a tool under `tools/`, and it is
        the one page there that is interactive: the bases and the rates are inputs, because the
        question is "what would happen if" rather than "what is".

- [ ] **The score's loop point is rounded, not authored.** `loopTicks` rounds the last
      note-off to the nearest bar, which for `ascending.mid` trims 60 ticks (about 62ms)
      of a drum tail past bar 13. That is inaudible and the tail is folded back over the
      start anyway, but the rounding is a *guess at intent*. If a future score wants a
      loop that is not its full length — an intro bar played once, say —
      `audio.NewInfiniteLoopWithIntro` already supports it and the loop point would need
      to come from the file (a marker meta-event) rather than from arithmetic.
- [ ] **Brands need a data file and a way to be acquired.** The mechanic is already decided —
      see `MECHANICS.md`'s Brands section: they alter the container where relics alter the
      contents, they are permanent *for the run*, and nothing takes one off. What does not
      exist is any of it in code: no `brands.json`, no acquisition, no seat on the duelist.
      `session.Session` is where a worn brand would live, beside the worn relics.

## Next — where the game actually starts

- [ ] **Boss advantages** *(owner asked for this to be tracked)*. Every boss is the same boss
      today: the tier puts it further up the ascent curve and nothing else separates it from the
      creatures on its own floor. What is wanted is **one advantage per boss, drawn from a pool
      the record carries**, out of a closed vocabulary checked at package init — the shape
      `internal/achieve` is under, and for its reason: an advantage a file can assert into
      existence is a boss rule nothing implements.
      - **Different bosses carry different pools**, overlapping where two bosses deserve the same
        trick.
      - **Rolled off its own salted stream**, so a replayed run code meets the same boss with the
        same advantage. See the `randomness` skill before adding the salt.
      - The vocabulary itself is undecided. Candidates that need no new rules are a heavier deck
        and a single enormous card; the rest — taking vitae, applying a status, healing on a
        kill, an extra action — are new verbs, and a shield-raising boss contradicts the rule in
        `CLAUDE.md` that creatures raise none.

- [ ] **The tutorial is switched off and has to be re-taught** *(owner asked for this to be
      tracked)*. `main.teachThisRun` returns false, so no run starts the lesson and the script in
      `data/tutorial.json` is unreachable. What broke it is that the lesson pinned a creature and
      a run code together: the taught hand, the taught blow and the creature's answering turn were
      one tuned set, and the roster it named no longer exists.
      - **Re-teaching it means choosing a motif, an element and a fresh run code** that together
        deal a hand the script can describe, land a blow that wounds without killing, and leave
        the creature alive to swing back — the constraints in the tutorial section of `CLAUDE.md`
        still hold, and `SEEDSEARCH=1 go test ./internal/screens -run TestFindATutorialSeed` is
        the search.
      - **The three tests that guarded the promise are skipped, not deleted**, so they come back
        with the seed rather than being rediscovered.

- [ ] **What an element does to a creature** *(owner asked for this to be tracked)*. A creature is
      instantiated as one element, and today that element picks the art and marks the attack cards
      and nothing else — a fire goblin and an ice goblin resolve identically. What it could carry:
      a status applied on hit, a resistance, a weakness, or something the floor's element does to
      the **player** rather than to the creature.
      - **It needs its own argument in `MECHANICS.md`** before it is written, and a status applied
        by a creature is the first thing in the game to put one on the player.

- [ ] **What makes an inner-chamber creature different from an outer one** *(owner asked for this
      to be tracked)*, beyond its place on the ascent curve. The tier is a position today. Whether
      it should also be a shape — a deck rule, a budget, a behaviour — is open.

### Cards and piles — presentation

- [ ] **The math band should wrap, not shrink.** `layOutMath` lays the sum out as one
      centered line and **shrinks every item by a common factor when the line is wider than
      the band** — floored at `minMathShrink`, 0.6. That is a stopgap: seven terms is
      reachable now (five cards in a legal turn plus the two extra landings an echo seats behind the
      first), and the answer to a line that will not fit is a second line, not smaller type.
      - **Why it is not done yet**: every figure *flies* from the card that paid it into its resting
        place, so a wrap is not a text-layout change — it is a second row of destinations, and the
        `x` and `=` have to land somewhere that still reads as one sum.
      - **What would say it is needed**: a real game showing a shrunk line.
        `TestTheWidestSumFitsItsBand` proves the deliberately-absurd case fits *after*
        shrinking; it says nothing about whether the result is readable at 0.6.
      - It is also the first thing to revisit if `MaxEchoLandings` ever rises above 5.
      - **The arithmetic behind it is already wide enough.** The event's
        term arrays hold 25 landings — every card of a legal turn, each landing up to
        `MaxEchoLandings` times — so a long repeat-and-echo chain is fully *resolved* today and only
        the drawing of it is short. Wrapping is what lets the screen show what the rules already
        compute.

- [ ] **The tooltip does not reach every card on screen.** Hand cards, the deck
      overlay, worn relics, the shop's two rows, both fighter cards, the reward screen's prizes and
      its offered cards all explain themselves. What does not:
      - **The table's two rows during playback** — the cards actually being resolved. They are the
        one place a player is watching rather than deciding, which is the argument for leaving them
        out, but it is also where "why did that hit for 96" is asked.
      - **Individual status badges.** Hovering the enemy card lists every status on it; hovering one
        badge does nothing, because `internal/cards` draws the row and no badge rectangle reaches
        the screen. A per-badge tooltip needs a geometry accessor from that package.
      - **The AP bar, the discard count, the tower place** and the other figures written straight
        onto the table. Each is a number with no legend anywhere.

## Later

- [ ] **Let a run seed be typed in.** A player can *read* a code — the run-over splash and the
      settings screen both draw it — but cannot hand one back, so a run is reproducible and not
      replayable.
      - **This is the one typed-text field in the whole game**, per the input vocabulary, and it
        is the trigger `internal/models/doc.go` names for revisiting the hand-rolled-UI decision:
        a caret, a selection and a clipboard are the one widget cheaper to take than to build.
      - **Everything under it is done.** `seeds.Parse` reads a code, `state.SeedPinned` says a
        seed was chosen rather than rolled, and both card shuffles derive from `RunSeed` — so a
        typed seed already reaches the cards. What is missing is the field and where New Run
        offers it.
- [ ] **Don't pre-roll into a fixed array — keep a seeded stream per concern.** A
      `*rand.Rand` seeded once *is* an infinite deterministic list; a pre-generated slice
      is just the first N entries of it, and N has to be guessed. The endless tower has
      no worst case to size against, so any N is eventually wrong.
      - **Rerolls advance the cursor**, which is exactly the intended behavior: reroll
        and you get the next offer down the list. No separate reroll stream needed.
      - Replay stays exact because the *list* is fixed by the seed. Identical choices
        consume identical draws; different choices land at a different position in the
        same list. That is the property worth having, and it survives rerolls.
      - Materialize a window of a stream into a slice only when something needs to
        *inspect* it — a test, say — not as the storage model.
      - The one discipline this needs: a stream is only ever advanced by its own
        concern. Never borrow the loot stream to pick an enemy.
- [ ] **Split the rest of `GlobalState`** into `Resources` (assets/fonts/data, read-only) and
      `Layout`. **The `Session` third of this already landed** — `internal/session` holds the deck,
      the fight index, the purse, the worn relics in worn order, and the run's phase — so what is
      left is the read-only half. Deferred: the remaining fields are not crowding anything.
- [ ] **What actually unlocks.** The profile exists and holds an `unlocks` set — `internal/profile`
      — and nothing writes to it. Undecided: cards for the starting deck, enemies in the pool,
      floors, whole alternate decks. Worth answering alongside the loot loop, since an unlock and a
      reward are the same object with different lifetimes.
      - **Hand discovery is the one already specified** — MECHANICS.md has hands discovered rather
        than given, and `profile.Profile.HandsDiscovered` is the field waiting for it. Gating the
        table is a balance change and belongs in a commit where its effect can be seen, not in the
        one that added the file.
- [ ] **Several profiles, and the second text screen.** Explicitly a later problem, split out
      so that "one profile for now" does not quietly become "one profile forever". Multiple
      profiles need naming, naming needs typing, and typing makes the one-text-field rule in
      `CLAUDE.md` into two.
      - That is a rule change rather than a feature. Revisit the hand-rolled-UI decision at the
        same time — see `internal/models/doc.go`, whose trigger for reaching for a toolkit is
        precisely "the seed text field turns out to be painful", and a second field doubles the
        exposure.
      - Numbered slots picked from a list would dodge the text field entirely, at the cost of
        "Profile 2" meaning nothing to the player.
- [ ] **Ascend / tower loop.** `ascend.go` is a bare `package screens`; `Ascend` and
      `Credits` are empty cases in the scene registry. Structure decided:
      - **8 floors, 3 fights each — 24 fights to the top.** The layout is fixed, not
        generated. Only the enemies and the offers are random.
      - **A binary loot choice after every fight.** Two options, pick one. **Built** as
        `PhaseReward` — the essence and the card it eats.
      - **A binary floor choice after the last fight on a floor**, on top of that fight's
        loot choice. **`PhaseChoice` is the station and it has no screen**, so `advanceRun`
        walks past it.
      - **Floor 8 ends the run** for the first version — 7 floor choices, no offer at the
        top.
      - Floor choices steer **enemy affixes and behavior** — "this is a cold floor",
        "this is a fire floor" — plus whatever other levers exist by then. The specific
        options are undecided; the mechanism is the part that matters.
      - Run progress lives in `session.Session`, which already carries the fight index, the
        purse and the worn relics. **The floor is what it does not have**: `fight` is a room count
        and `pyramid` derives the floor from it, so a run that chooses its own floors needs one
        stored rather than computed.
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
        only answer "what if I took the other relic", where plans answer "what if I had
        guarded on round 3".
      - **Serialize card *keys*, not `ConceptID`s.** An ID is an index into a registry built by
        walking `duelist_cards.json` and then every enemy's deck,
        so it is stable for one build of one data set and for nothing else. Adding an enemy
        renumbers every concept after it. `Element` carries the same `iota` hazard it always did.
      - **`[?]` The combat roll has to be settled before this ships.** `MECHANICS.md` requires
        rolling on every attack phase and discarding the irrelevant result, precisely so a
        balance tweak does not shift every later roll in a run. `shockMisses` short-circuits when
        the attacker carries no shock, so the stream only advances when lightning is in play —
        which is exactly the drift the rule forbids. It is narrow: a shock needs a lightning
        relic on the attacker to exist at all, so a bare duel never advances the stream. Nothing
        depends on stored seeds yet, so it is cheap to fix now and expensive to fix after a save
        format exists.
      - Serializing live state instead means a migration every time state changes — the
        refactor this whole set of decisions exists to avoid.
      - Cost: loading replays the run to reach the current point. Trivial here, since
        a whole duel resolves in microseconds.
      - Caveat: this only holds while the rules are stable. A balance change invalidates
        old saves, so the format needs a rules-version stamp and a plan for what happens
        when it does not match.
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
      `data/enemies.json` is shaped — fully-specified records, one per creature — so the data
      wants to become a base statline, a scaling rule, and a pool of affixes to draw from.
      - `AvailableAffixes` already anticipates this and is still unread.
      - Affixes must compose. Two on one enemy is the normal case, not an edge case.
      - Floor choices feed this directly: "a cold floor" biases which affixes appear.
## Art still to generate

*(owner asked for this to be tracked)*. **Two batches outstanding.** Every other catalog is
complete — relics, essences, runes, stones, potions and the sealed goods all carry an `Art` key
and a `Draw` brief, with no key naming a file that is not on disk. Each catalog's own sheet
counts the gaps and marks them in pink, so that claim is checkable rather than a figure written
here.

- [ ] **The playing cards — 95 pictures, `data/card_art.json`.** One per card per element, drawn
      full bleed under the card's own type. **Every brief is written**; what is missing is the art,
      so every record has a `Draw` and an empty `Art`. The scheme is three axes in one picture: a
      vapor humanoid whose *weapon* says the form, whose *weapon scale and pose commitment* say the
      rung, and whose *color* says the element. `docs/art/card_art_prompt.MD` is the prompt, and
      files go into `assets/card/` keyed by filename stem — `jab-fire.png` is `jab-fire`.
      - **Nothing breaks while this is empty.** `data.DefaultCardArt` is deliberately blank, so an
        unauthored record draws the card exactly as it looked before the catalog existed. This is
        the one catalog in `data/` with no fallback face, and that is the design.
      - **The hard constraint is value, and nothing enforces it.** `cards.Hand` sets `ArtUnder`
        rather than `ArtBleed`, so the card keeps its near-black ink and there is **no scrim** — a
        dark picture makes a card unreadable and no test goes red. The prompt carries the measured
        map of where type lands; check a delivered batch against it before filing.
      - **It can be filed a few at a time.** 95 is a lot to commission at once and each record is
        independent, so a form or an element can be done as a block.

- [ ] **The numbered damage badges — 66 pictures, `assets/damage/`.** Eleven multipliers
      (`quarter`, `half`, `1`..`9`) in five elements plus a neutral, in the diamond outline.
      `docs/art/damage_art_prompt.MD` is the prompt; `go run ./tools/badgesheet` is the page they
      are reviewed on, and it currently renders 66 gaps.
      - **The numeral is drawn into the art**, which is the whole reason this is 66 files rather
        than 6 — type and art do not reduce the same way, and the badge is drawn at 32 and at 16.
      - **Nothing breaks while this is empty.** A value with no drawn badge falls back to the blank
        badge with its figure printed on top, which is what shipped before — so a partial batch
        degrades one value at a time rather than emptying the corner.
      - **Review by value, not by color.** The failure this set has is a numeral drifting in weight
        or position between the six colors of one value, and that is invisible one file at a time.
        The badge sheet is laid out eleven rows of six for exactly that.
      - **`quarter` is the one at risk.** It carries the most ink of any numeral into the smallest
        space; if it cannot be read at 16 pixels the rung needs a different answer rather than a
        smaller font.

- [?] **The relic catalog is pixel art and nothing else is.** The pictures in `assets/relic/`
      came from a prompt asking for chunky blocks and sixteen flat colors; the essences, runes,
      stones and goods came from the smooth block every prompt carries today, so a relic card
      and a stone card do not look like one game. **Closing it means regenerating one side or
      the other** — the whole relic catalog, or everything else — and it is the owner's call
      which. `docs/art/relic_art_prompt_pixel_archived.MD` is kept live-shaped and clearly
      marked not-live so the direction can be reversed by regenerating rather than by
      reconstructing a prompt from git.

- [?] **Which outline the damage badge takes.** All three shapes were drawn
      unnumbered — circle, starburst, diamond — and `cards.DefaultBadgeShape` is the one line that
      picks. **The numbered batch is diamond only**, so switching after it lands means commissioning
      66 more rather than changing a constant. The three blanks are on the badge sheet to be
      compared before that batch is ordered.

## Platform readiness — the Steam Deck refactor, and the seams a port would need

*(owner asked for these to be tracked)*. Three tickets that add controller support and isolate the
desktop assumptions left in the game. **None of them changes combat rules, run rules, save
semantics or the visual design.** Take them in the order below: the input one is the biggest, and
the other two are independent of it and of each other.

**This is a pre-release refactor, not the next thing to build.** Steam Deck is a shipping target,
so all three land before release; ordinary development carries on with the mouse until then. What
each new screen owes ticket 1 in the meantime is the design check in `CLAUDE.md`'s input section —
could a focus ring walk this — so the refactor is a refactor rather than a redesign.

The posture they all share is the repo's own — rules stay below presentation, a failure in an
optional platform facility never stops play, an outcome never depends on presentation or on which
physical device is in the player's hands, and a test states the contract before the old path is
removed.

- [ ] **1. Semantic controls, and a controller that can play the whole game.** A run is playable
      start to finish on a standard gamepad with no mouse and no keyboard, and no screen or
      ordinary widget reads Ebitengine keys or mouse buttons directly. Mouse behavior is unchanged.
      - **The design record already allows it.** `CLAUDE.md`'s input section carries the two
        rules this ticket may not bend — no virtual cursor, and a semantic action reaches only
        controls that are on the screen — plus the question a new screen answers to stay
        refactorable. Read it before designing the focus model.
      - **The seam is an engine-neutral package** — suggested `internal/controls`, no Ebitengine
        import — holding an `Action` vocabulary that names intent rather than a button: confirm,
        cancel, the four navigations, previous and next region, secondary, inspect, menu. It hands
        out a per-tick frame answering `JustPressed` / `JustReleased` / `Pressed`, the pointer, and
        which device was last used. **The Ebitengine adapter sits at the top of the graph** —
        `internal/game`, or a narrow `internal/platform/ebiteninput` — sampled once near the start
        of `Game.Update`. Keyboard mappings ship beside the gamepad ones so navigation can be
        tested without a pad plugged in.
      - **No virtual cursor.** Controller focus is a first-class mode, not a hidden mouse being
        driven around. The two may be alternated at any moment, and switching device changes
        nothing but which treatment is drawn.
      - **A physical edge is sampled once a tick** and a held button does not repeat unless the
        control it is over asks for repeat.
      - **Focus is semantic, not derived from draw order.** A target has a stable identity for the
        screen it is on, a rectangle, an enabled state, an activation, and either explicit
        neighbors or a deterministic row and column order. Each scene builds its own focus graph in
        `Update`; **do not build a retained UI framework** — that is the decision
        `internal/models/doc.go` is under. Every screen picks a sensible default target, a disabled
        or hidden target is skipped, and a target that disappears hands focus to the nearest valid
        one deterministically.
      - **The focus treatment has to be legible against every state a card and a button already
        have**, and must not read as selection, unaffordability, hover raising or a latched sort.
      - **An overlay traps focus.** Nothing behind a modal, a toast, the ledger or a tutorial gate
        may be reached or fired. `Cancel` unwinds exactly one level: reorder mode, then the top
        overlay, then whatever the scene does with Back.
      - **The mouse contract survives intact** — a click fires on release over the same control,
        and dragging off cancels it. Confirm fires once on a defined edge. Both lifecycles get
        tests.
      - **Escape keeps meaning "press the settings cog"**, and the controller gets its own `Menu`
        action for it rather than `Cancel` being overloaded into a settings key.
      - **Controller reorder calls the same row operations mouse dragging calls** —
        `RowLift`/`RowReturn` — so worn-relic order and hand order cannot grow a second
        implementation. Entering reorder remembers the origin, Confirm commits, Cancel restores.
      - **Card focus follows the card's identity, not its seat**, through a sort, a refill, an
        insertion and a removal.
      - **Everything on the combat screen is reachable**: the hand, the sort tabs, the deck, hands
        and ledger panels, the consumables, Discard, DUEL! and the cog. Region cycling is a fixed
        order — the hand, the action buttons, the right-hand control column, the consumables and
        relics, back to the hand.
      - **A shop or reward screen opens on the first affordable offer**, or on the first offer if
        none is. An unaffordable offer may be inspected; it may not be activated.
      - **The tutorial gate grows a semantic half rather than being bypassed.** An anchor resolves
        to target IDs as well as to rectangles, focus cannot leave the allowed set, and a step
        naming a set of cards allows exactly those card identities — the same rule the rectangles
        are already under. `TestTheMatchingCardsGateLightsOnlyTheTaughtCards` is the pointer-side
        tripwire and wants a focus-side twin.
      - **Glyphs are looked up by semantic action and controller family**, never a letter typed
        into rule or screen text; text labels are an acceptable first version. Steam Input maps
        onto the semantic layer and is never consulted by a rule.
      - **A disconnect with no usable pad left pauses, or shows a non-destructive reconnect
        notice**, dismissible by mouse or keyboard on PC.
      - **The migration is ten steps and the last one is the guard**: the action types and their
        edge tests, the adapter populated in `Game.Update` with the old mouse fields still
        standing, `systems.UpdateButton`, the confirm dialog and the chrome plus the focus drawing,
        the sliders and scrollbars, the direct-click call sites, scene focus scopes and modal
        trapping, card rows, tutorial targets, device presentation and disconnect — then the
        transitional fields go and an architectural check holds the line.
      - **The direct readers today** are `internal/game/game.go` and `chrome.go`,
        `internal/systems/button_sys.go`, `slider_sys.go` and `scrollbar_sys.go`,
        `internal/ui/carddrag.go`, and the screens `ledger.go`, `postbattle.go`, `shop.go`,
        `shop_goods.go`, `shop_pouch.go`, `combat_flight.go` and `tutorial.go`. A grep for
        `inpututil`, `IsMouseButton` and `CursorPosition` is the list that stays current.
      - **The acceptance test is a controller-only smoke path** — title, tutorial or new run,
        combat, reward, shop, the next combat, settings, the end of the run — plus a
        controller-only completion of the tutorial with nothing outside its gate accepted. **That
        half waits on the tutorial being re-taught**, which is its own entry above; the rest of the
        ticket does not.
      - **Not in this ticket**: a console SDK, local multiplayer, any redesign of the combat
        screen, any change to combat timing, and full key rebinding — though the mapping boundary
        has to be able to carry rebinding later.

- [ ] **2. Profile and run persistence behind an injectable storage backend.** The JSON schemas,
      the version handling, the unknown-field preservation and the corruption policy are all
      exactly what they are today; what changes is that nothing above the backend knows a save is a
      file in a directory.
      - **The boundary is raw bytes**: read by name, write by name, delete, and say whether it is
        writable. Encoding and version policy stay above it, storage mechanics below. **No
        `Save(any)`** — the backend must not learn what a profile, a run, a version or an unknown
        field is.
      - **`LoadProfile`, `SaveProfile`, `LoadRun`, `SaveRun` and `DeleteRun` take the interface**,
        marshal and unmarshal exactly as they do now, and hand over complete bytes. Atomicity is an
        implementation detail of the filesystem backend, which keeps `ASCEND_DUEL_PROFILE`,
        `os.UserConfigDir`, lazy directory creation, filename validation, permissions, temp-file
        cleanup and the atomic rename.
      - **`main` constructs the desktop backend**; a screen never chooses storage. `GlobalState`
        carries the interface, and a bare `GlobalState` in a test gets an inert implementation
        through one helper rather than a nil check at every call site.
      - **`Store.Dir() == ""` is what the abstraction costs.** Two call sites —
        `internal/screens/run.go` and `save.go` — read an empty directory as "an inert store, so
        suppress the save error". That becomes an explicit capability query.
      - **"The backend cannot write" and `ProfileWritable == false` are different facts** and must
        stay different: the second means the loaded data is corrupt or from a future build and must
        not be overwritten, and no backend's writability may override it.
      - **Ledger export is an optional second capability, not part of saving.** A platform without
        exports disables the control rather than writing a ledger into the save container under a
        made-up name, and failing to export is not failing to save. The desktop exporter keeps the
        safe filename and the path-traversal protection.
      - **A memory fake with injectable read, write and delete failures** is what makes the
        nonfatal behavior testable without filesystem permission tricks, and the core contract
        tests run against both it and the real one.
      - **The tripwire is that no screen and no session code imports `os` or `filepath`** or
        assumes a config path, and that the desktop files stay byte-compatible.
      - **Not in this ticket**: Steam Cloud, which syncs the desktop directory from outside;
        console storage SDKs; mid-duel serialization; encryption; compression; and any change to
        saving only at phase boundaries.

- [ ] **3. Thin platform services for achievements and application lifecycle.** Game code publishes
      an achievement and reacts to suspend, resume and quit without importing Steam, Nintendo,
      PlayStation or Ebitengine platform APIs. **A no-op implementation is a legitimate shipping
      implementation** for a DRM-free desktop build.
      - **Suggested `internal/platform`**, engine-neutral, holding narrow capabilities and their
        no-op and recording-fake implementations; SDK adapters sit above it or behind build tags.
        Two capabilities only — unlock an achievement key, poll lifecycle events — independently
        replaceable. **No leaderboards, presence, networking, commerce, telemetry or DLC** until
        something implemented needs them.
      - **`combat`, `session`, `data` and achievement rule evaluation may never import it**, and a
        platform call may never touch a random stream, a resolution, a reward, a price, an unlock
        or a save.
      - **The local profile stays authoritative.** An outage cannot revoke or block an award. The
        toast is queued and the profile saved whether publication succeeded or not, and
        `internal/screens/achieve.go` is the one seam the publication call joins, after
        `Profile.Award` reports a genuinely new award.
      - **Reconciliation replaces a durable retry queue**: every key already in the profile is
        offered to the provider at startup, so offline play and transient failures heal themselves.
        `Unlock` is therefore idempotent by contract.
      - **A platform ID is mapped at the provider boundary if it has to be.** A stored profile key
        is never renamed to satisfy a provider.
      - **Lifecycle is processed near `Game.Update`, before any scene updates.** Suspended stops
        input, cancels in-flight pointer and controller gestures, pauses the clocks and the audio,
        and flushes the profile if it is writable. Resumed restores audio to the saved settings,
        clears stale edges so the button that woke the application cannot press a control,
        re-enumerates pads, and resumes the clocks **without applying elapsed wall time**.
        QuitRequested goes through the existing `ShouldClose` / `ErrClosing` path.
      - **Suspending does not invent a new save point.** The last phase-boundary snapshot is the
        resumable state, so a process killed mid-duel resumes that room from its start, exactly as
        quitting mid-duel does.
      - **The desktop window's close and focus become lifecycle events at the adapter**, rather
        than `ebiten.IsWindowBeingClosed` being consulted around the codebase.
      - **Fullscreen stays where it is.** It is already isolated in the settings screen and a
        console can make it a no-op; broadening this ticket to abstract every Ebitengine call is
        explicitly not wanted.
      - **Not in this ticket**: any SDK integration, mid-duel saves, any change to achievement
        keys, conditions or the toast, and any abstraction of rendering or of the game loop.

## Licensing (for an eventual Steam release)

Model: source stays public under PolyForm Noncommercial 1.0.0, nobody else may commercialise
it. Justin and Sherman have a signed agreement covering the relicense. The Apache 2.0 grant on
everything published before the relicense is irrevocable and accepted — no history rewrite.

- [ ] **Put Sherman's legal name in `LICENSE`.** The Required Notice currently names the
      GitHub handle `KingSherman1820`. Deliberately deferred — a written partnership
      agreement covers the two of them — but a copyright notice naming only a handle is
      weak if it ever has to be enforced.
- [ ] **`THIRD-PARTY-NOTICES` file.** Apache-2.0 and BSD deps may sit inside a
      restricted-license product, but only if their notices and attributions travel with
      the binary. Needed for a Steam build, not for the repo.
- [ ] Contact address for licensing enquiries. Deferred deliberately; anonymous is fine
      for now, and `CONTRIBUTING.md` points people at issues instead. Use a purpose-made
      address rather than a personal one when it happens.
- [ ] Get thirty minutes of actual legal review before relying on any of this. The
      license is standard and well drafted; the contributor grant in `CONTRIBUTING.md`
      is a reasonable draft written by a non-lawyer.
- [ ] Confirm FiraSans and RobotoFlex (expected OFL / Apache — low risk).

**Cleared, and the register to check a new dependency against.** No GPL anywhere — it cannot
go into a product licensed this way.

| What | License |
|---|---|
| Ebitengine | Apache 2.0 |
| `github.com/ebitengine/oto/v3` | Apache 2.0, first-party to Ebitengine |
| `golang.org/x/*`, incl. `golang.org/x/image` | BSD-3-Clause |
| `Kubasta.ttf` | CC0, per the author's own FontStruct page |
| Everything under `assets/` | first-party: generated from the prompts in `docs/art/`, or generated at runtime |
