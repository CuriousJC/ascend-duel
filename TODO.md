# TODO

**The work list.** [MECHANICS.md](MECHANICS.md) is what the game *is*; this is what to build.
When the two disagree, `MECHANICS.md` wins — say so rather than guessing. `ideas.md` is the
unfiltered inbox feeding both.

Completed work is not kept here. Git history has it.

Status: `[ ]` open · `[~]` in progress · `[?]` needs a decision

**Nothing is added here unless the owner asks for it to be tracked.**

---

## Now — quick wins, independent of any design decision

- [ ] **A card's face does not say what riders it carries** *(owner asked for this to be tracked,
      2026-09-02)*. Sixteen parasites now attach seven kinds of rider, and a ridden card looks
      exactly like an unridden one — the only place a rider is visible is the tooltip prose. That
      was tolerable at one rider on one parasite and is not at seven: a hand's worth of held-back
      cards paying vitae, doubling DMG or raising shields is a turn the player cannot read.
      `combat.MaxCardRiders` is 3 **because the face has room for three badges**, so the room was
      reserved and never used. What it needs: a badge per rider kind (the `assets/effect` route, or
      generated glyphs), a row on the card face that does not collide with the cost column or the
      text band, and something in the hand row that marks a card as worth *not* playing — the four
      in-hand riders are the first mechanic in the game that rewards leaving a card alone, and
      nothing on screen says so.

- [?] **Three parasites from the owner's list still need a design decision before they can be
      written as data** *(owner asked for this to be tracked, 2026-08-27; trimmed 2026-09-02 as
      the rest landed)*. **Lucky card**; **chance to increase a ring** (which ring, and increase
      what?); **wild card** (matches any axis, or any one axis you name?). Two of them are random,
      which needs its own stream and its own argument in `MECHANICS.md` per the `randomness` skill.

- [ ] **The score's loop point is rounded, not authored.** `loopTicks` rounds the last
      note-off to the nearest bar, which for `ascending.mid` trims 60 ticks (about 62ms)
      of a drum tail past bar 13. That is inaudible and the tail is folded back over the
      start anyway, but the rounding is a *guess at intent*. If a future score wants a
      loop that is not its full length — an intro bar played once, say —
      `audio.NewInfiniteLoopWithIntro` already supports it and the loop point would need
      to come from the file (a marker meta-event) rather than from arithmetic.
- [ ] **Brands need a data file and a way to be acquired.** The mechanic is already decided —
      see `MECHANICS.md`'s Brands section: they alter the container where rings alter the
      contents, they are permanent *for the run*, and nothing takes one off. What does not
      exist is any of it in code: no `brands.json`, no acquisition, no seat on the duelist.
      `session.Session` is where a worn brand would live, beside the worn rings.

## Next — where the game actually starts

- [ ] **Procedurally generated enemies.** 96 hand-written records in `data/enemies.json` with
      the combat screen walking a shuffled band per floor is scaffolding. An enemy should be
      **generated** from the floor, so the tower can be endless and a seed can reproduce it.
      - **Assembled from parts, not rolled from scratch.** The pieces that exist or are already
        decided: a **stat line** scaled by floor depth; a **deck** (`internal/decks` has the
        enemy pile); an **affix** that transforms that deck rather than adding to it; a
        **portrait**; and a **personality** — which plan it reaches for first.
      - **Needs its own randomness stream**, per the stream rules in `CLAUDE.md`. Enemy
        selection already reads `RunSeed ^ enemySelectSalt`; generation is what will draw on it
        next, and must salt its own source rather than share one.
      - **Blocked on nothing**, but the current records become **seeds for the generator or
        test fixtures**, not the roster.
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

### Cards and piles — presentation

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

- [ ] **The tooltip does not reach every card on screen** *(2026-08-21)*. Hand cards, the deck
      overlay, worn rings, the shop's two rows, both fighter cards, the reward screen's prizes and
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

- [ ] **Show the run seed, and allow entering one.** `GlobalState.RunSeed` is set once by
      `main` and logged; enemy selection is the first stream reading it. Without a way to see a
      seed and type one back, replayable runs are invisible to the player.
      - **This is the one typed-text field in the whole game**, per the input vocabulary.
      - **The spelling is done** *(2026-08-25)*: a seed is a six-character Crockford base32
        code, `seeds.Code` / `seeds.Parse` — case-insensitive in, upper case out, with `O`/`I`/`L`
        folded to the digits they look like — and `main` prints it every launch. What is left is
        somewhere to show it and a field to type it into.
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
        *inspect* it — a test, say — not as the storage model.
      - The one discipline this needs: a stream is only ever advanced by its own
        concern. Never borrow the loot stream to pick an enemy.
- [ ] **Split the rest of `GlobalState`** into `Resources` (assets/fonts/data, read-only) and
      `Layout`. **The `Session` third of this already landed** — `internal/session` holds the deck,
      the fight index, the purse, the worn rings in worn order, and the run's phase — so what is
      left is the read-only half. Deferred: the remaining fields are not crowding anything.
- [ ] **What actually unlocks.** The profile exists and holds an `unlocks` set — `internal/profile`
      — and nothing writes to it. Undecided: cards for the starting deck, enemies in the pool,
      floors, whole alternate decks. Worth answering alongside the loot loop, since an unlock and a
      reward are the same object with different lifetimes.
      - **Hand discovery is the one already specified** — MECHANICS.md has hands discovered rather
        than given, and `profile.Profile.HandsDiscovered` is the field waiting for it. Gating the
        table is a balance change and belongs in a commit where its effect can be seen, not in the
        one that added the file.
- [ ] **An achievement the player can see.** `first-steps` is awarded on a won duel and lands in
      `profile.json` silently: there is no toast and no achievements screen. `Profile.Award` already
      reports whether an award was new, which is what a toast would hang off.
      - A toast is a new widget and the frame in `internal/game/chrome.go` is the only thing that
        outlives a scene, so it is the natural home and the bar for joining it is high — see
        CLAUDE.md. Worth deciding whether this is a toast at all or a line on a between-fights
        screen.
      - Steam's overlay draws its own popup when that lands, so this may turn out to be a thing
        only the non-Steam build needs.
- [ ] **A run that ends.** Delete-on-death is decided and has nowhere to fire from: a defeat
      currently offers `Retry` and puts the same opponent back up, so no run ever ends.
      `profile.DeleteRun` is written and uncalled, waiting for whatever ends one.
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
        `PhaseReward` — the worm and the card it eats.
      - **A binary floor choice after the last fight on a floor**, on top of that fight's
        loot choice. **`PhaseChoice` is the station and it has no screen**, so `advanceRun`
        walks past it.
      - **Floor 8 ends the run** for the first version — 7 floor choices, no offer at the
        top.
      - Floor choices steer **enemy affixes and behaviour** — "this is a cold floor",
        "this is a fire floor" — plus whatever other levers exist by then. The specific
        options are undecided; the mechanism is the part that matters.
      - Run progress lives in `session.Session`, which already carries the fight index, the
        purse and the worn rings. **The floor is what it does not have**: `fight` is a room count
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
        only answer "what if I took the other ring", where plans answer "what if I had
        guarded on round 3".
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
