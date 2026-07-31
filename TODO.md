# TODO

Working notes for ascend-duel. Findings and reasoning live in `analysis.md`;
this file is just the running list.

Status: `[ ]` open · `[x]` done · `[~]` in progress · `[?]` needs a decision

---

## Now — quick wins, independent of any design decision

- [x] **Edge-triggered clicks.** `UpdateButton` used `ebiten.IsMouseButtonPressed`
      (level-triggered), firing `OnClick` every tick the mouse was held. Now uses
      `inpututil.IsMouseButtonJustPressed` / `JustReleased`, arming on press and firing
      on release inside bounds so dragging off cancels. *(analysis §5)*
      - [x] Verified in the running game after flipping the start screen to `Title`:
            one `Settings Button Clicked!` per click in stdout, where the old
            level-triggered code printed once per tick held.
- [x] **Extract `newCombatantFrom(gs, record)`.** Two near-identical init blocks in
      `InitCombatScreen`; the `SpriteRect` bug came from exactly this duplication.
      *(analysis §7)* Landed as `entities.NewCombatantFrom(data, sheet)` plus a small
      `combatantFromRecord(gs, key)` helper in `combat.go` — `entities` can't take a
      `*GlobalState` because `state` already imports `entities`, so the screen does the
      map lookups and the entity does the construction. `NewCombatant()` removed: it
      built an invalid combatant and had nothing left calling it.
- [x] **Move button positioning out of `Draw`.** `DrawTitleScreen` writes `ScreenX/Y`
      that `Update` then hit-tests against — stale on frame 1 and whenever Ebiten skips
      a `Draw`. *(analysis §3)* Moved to a new `InitTitleScreen`, run once from
      `UpdateTitleScreen` off the `NewScreen` flag. Safe to compute once now that the
      internal resolution is fixed.
- [x] **Screen-transition nil deref.** Latent bug surfaced by flipping the start screen
      to `Title`: an action sets `ActiveScreen`+`NewScreen` mid-`Update`, then `Draw`
      runs on the *new* screen before its `Init` ever has — `DrawCombatScreen` hit a nil
      `gs.Fighter`. `Game.Draw` now returns early while `NewScreen` is set, costing one
      skipped frame per transition. Was invisible while the game booted into Combat,
      because Ebiten's first `Update` always beat the first `Draw`.
- [x] `SettingsButtonAction` sets `NewScreen` without changing screen — removed; with the
      `Draw` guard it would now cost a skipped frame and a pointless re-`Init`.
- [x] **Fixed internal resolution.** `Layout` returns a constant 1280x960 and lets
      Ebiten letterbox, making every absolute coordinate safe. *(analysis §8, decided)*
      `game.ScreenWidth`/`ScreenHeight` are now the single source — `main.go`'s private
      duplicate pair is gone and `SetWindowSize` reads the same constants.
      Resizing enabled in `main.go` via `SetWindowResizingMode`, so the letterbox path
      actually runs.
      - [x] Verified on screen: the scene scales as a unit and letterboxes, and the
            layout no longer reflows on resize.
- [ ] **Stop allocating in `Draw`.** `DrawButton` makes a new `ebiten.Image` per button
      per frame (180/sec); `DrawHealthBar` makes two per bar per frame (240/sec).
      `Button.Image` already exists and is only used for bounds — render into it on
      state change instead. *(analysis §4)*

## Next — where the game actually starts

- [x] **`package combat` as a pure core.** No ebiten import.
      `ResolveRound(a, b Duelist, aActions, bActions []ActionKind, round int)` returns an
      ordered event log for **one round** plus the state both sides end in; the combat
      screen holds the log, a playback cursor and a tick timer, applies one event at a
      time, then adopts the returned duelists when the cursor catches up. 19 tests, the
      first in the repo. `entities.Combatant` embeds `combat.Duelist`, which keeps the
      sprite out of the rules and leaves `gs.Fighter.Str` reading the same as before.
      - Rewriting the turn model from the timeline version touched this file, its tests,
        and ~20 lines of screen glue. No drawing code moved. That is the payoff of the
        split, and it is worth remembering the next time the rules change.
- [x] **Define `Action`.** `ActionKind` is Strike / Guard / Heavy / Quick, costing
      1 / 2 / 2 / 4 action points. A duel is a sequence of rounds: both sides spend an
      AP budget on a set, the two sets resolve **alternately** — one of A's, one of B's —
      then control returns to re-plan. `Spd` buys budget (`AP = 4 + Spd/10`), not turn
      frequency. All integer arithmetic, so rounds are exactly reproducible.
      - Side A takes the first turn, and the screen maps the player to A. A Guard the
        player places is therefore up for the enemy's reply.
      - Whichever queue is longer keeps acting alone once the other empties, so a speed
        advantage is spent at the tail of the round.
      - **Alternating replaced volley-per-side on 2026-07-31**, when the resolution pane
        made the old order visible and it read wrong. The point is that resolution order
        is a thing the player can see and, later, manipulate — speed effects and other
        tweaks need somewhere to bite, and two monolithic volleys gave them none.
        `KindVolleyStart` went with it; there are no volleys to start.
      - `Guarded` persists on the `Duelist` and clears at the start of its owner's next
        *action*, so it covers every opposing action in between, across a round boundary
        if the guard was queued last. A duelist who queues nothing therefore keeps its
        guard — deliberate, pinned by `TestGuardHoldsWhileItsOwnerDoesNothing`, and
        harmless because standing still deals no damage.
      - Superseded an earlier speed-timeline model where a fast duelist took whole
        extra turns. That is wrong for a game you re-plan every round: extra turns are
        invisible when you only ever plan one.
- [x] **DUEL! button.** `DuelButtonAction` resolves **one round** and hands the screen a
      log to replay; control returns to the player to re-plan. It ignores presses during
      playback and once someone is down. A caption line reports what playback is showing,
      and between rounds shows the queued plan and its AP cost.
- [x] **Decide Constitution → HP.** `NewCombatantFrom` sets
      `MaxLife = Con * entities.LifePerCon` and starts `CurrentLife` full;
      `DrawHealthBar` scales the red fill by `CurrentLife/MaxLife` over a darker track,
      and playback of the resolved log drives it. `LifePerCon = 5` survived the rebalance
      and now gives Fighter 60 and Monster 100 — the lopsided 60-vs-160 spread was a data
      problem, not a constant problem, and was fixed in `combatants.json`.
- [ ] **Drag-and-drop for the action box.** `DrawCombatButton` is the DUEL! button now;
      `DrawActionBox` / `DrawActions` are still empty stubs. The engine side is ready and
      waiting: `combat.AllActions` is the palette, `ActionKind.Cost()` the price,
      `Duelist.ActionPoints()` the budget and `CanAfford` the validity check — the UI
      just has to write `s.fighterActions`. Until it exists, `defaultFighterPlan` in
      [combat.go](internal/screens/combat.go) stands in for the player's choices.
      - Hand-rolled, no toolkit — decided, see "Open decisions". Drag state lives on
        `CombatScene` alongside everything else it owns.
      - **Flow, decided:** build a plan → DUEL! → watch playback → build a *fresh* plan →
        DUEL! again. The queue empties every round; nothing carries over. Makes each
        round a real decision rather than a default nobody revisits.
      - `defaultFighterPlan` is deleted once this lands — it only ever stood in for the
        player.
      - The planning/playback phase stays derived from `cursor >= len(log)` rather than
        becoming its own field, so there is one source of truth.
      - DUEL! is disabled while the queue is empty — first real use of the existing
        `models.ButtonStateDisabled`. An empty plan is mechanically legal and
        `ResolveRound` handles it, but it means standing still while being hit.
      - Under-spending the budget is allowed; forbidding it would add a rule for no gain.
      - **No "repeat last plan" shortcut.** Explicitly not wanted: every round being a
        real decision is where the balance and design work lives, and a repeat button
        undercuts that. Revisit only if playtesting proves it tedious — do not add it
        pre-emptively.
- [ ] **Real opponent AI.** `combat.PlanGreedy` buys the biggest attack it can afford and
      never guards. Deterministic and fine for testing, but not a fight.
- [x] **Wire `Str` / `Spd` / `Con`.** `Con` feeds `MaxLife`, `Str` sets action damage,
      `Spd` sets turn frequency. All three are live.
- [x] **Rebalance `combatants.json` — measured, not guessed.** Was Fighter
      Str10/Spd11/Con12 vs Monster Str1/Spd31/Con32, which gave 16 rounds against a
      monster that removed 15 life total. Now Fighter **Str10/Spd20/Con12** (60 life,
      6 AP) vs Monster **Str10/Spd15/Con20** (100 life, 5 AP). Verified headlessly:
      - `Guard + Strike + Strike` **wins** on round 5, fighter ends on 12/60.
      - `Heavy + Strike` **loses** on round 3, monster still on 10/100.
      - The greedy plan deals more damage per round and still loses the race, so the
        plan choice decides the fight. That is the balance point worth protecting when
        these numbers change again.
      - ~1.8s of playback per round.
      - Raising Monster Str to 10 also removes the degenerate case where Guard halved a
        1-damage hit to nothing.
      - [?] Still open: should Guard subtract flat armour rather than halving? `ideas.md`
            lists armour as a core attribute alongside speed and damage, so halving may
            be the wrong long-term shape even though it now behaves.

## Later

- [ ] **Game speed setting.** User-facing options: *very slow · slow · normal · fast ·
      very fast*, scaling how quickly the duel event log plays back. Ship "normal" only
      to begin with, but route it through a setting rather than a constant so the other
      four are a data change later.
      - Today the pacing is `duelTicksPerEvent = 8` in
        [combat.go](internal/screens/combat.go) — one constant, one caller, so this is
        cheap right now and gets steadily more expensive as animation and sound land and
        each grows its own timing constant.
      - Speed must scale *presentation only*. `combat.ResolveRound` already decided the
        whole round before playback starts, so speed can never change an outcome — worth
        protecting, since "fast mode plays differently" is a classic bug in this shape of
        game.
      - Belongs to whatever settings screen eventually backs `SettingsButtonAction`,
        which currently only prints.
- [ ] **Seeded randomness for replayable runs.** Goal: record a seed with a run so the
      same enemies and offers can be replayed while the player makes different choices.
      Nothing is stochastic yet, so there is no work to do — but see the determinism
      rules in `CLAUDE.md`, which exist to keep this cheap.
      - **Three separate streams, decided:** enemy selection, loot offers, floor offers.
        Tower layout is *not* a stream — it is fixed at 8 floors of 3 fights.
      - Separate streams mean adding a random call in one system cannot shift another
        system's results. Without that, tweaking loot generation silently rerolls every
        enemy in the tower, and balance testing becomes impossible to reason about.
      - Show the seed somewhere and allow entering one, or the feature is invisible.
      - `internal/combat` is currently pure integer arithmetic with no randomness at all,
        and `TestRoundIsDeterministic` pins that. Keep it that way — if randomness enters
        combat, it arrives as an injected source, not a global.
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
- [x] **`Scene` interface** replacing the two parallel switches in `game.go`, plus the
      per-screen half of the `GlobalState` split — the same trigger fired both, and they
      were one piece of work. *(analysis §1, §2)*
      - `Scene` is `Init`/`Update`/`Draw`; scenes are registered once in a map, so the
        `Update` and `Draw` paths can no longer drift apart.
      - Scenes own their state *and* build their own widgets, wiring them to their own
        methods. `main.go` no longer constructs buttons for screens it knows nothing
        about, and `DuelButtonAction` became `CombatScene.startRound`.
      - `NewScreen` is consumed centrally in `game.Update`; scenes never touch it.
      - `GlobalState` went from 34 fields to 20 and no longer imports `combat`,
        `entities` or `models`.
- [ ] **Split the rest of `GlobalState`** into `Resources` (assets/fonts/data,
      read-only), `Layout`, and `Session` (run progress). Deferred: `Session` has nothing
      to hold until the tower loop exists, and the remaining fields are not crowding
      anything. The seed streams will live in `Session` when it lands.
- [ ] **Ascend / tower loop.** `ascend.go` is a bare `package screens`; `Ascend` and
      `Credits` are empty cases in both switches. Structure decided:
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
        `Session` state in the `GlobalState` split below — this is the feature that
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
      - **Serialize action names, not `iota` ordinals.** `ActionKind` is `iota`-based, so
        inserting a new action anywhere but the end silently reinterprets every existing
        log — a saved `Guard` becomes whatever now sits at 1, with no error. Same applies
        to any other enum that reaches the save file.
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
        - Randomness resolved *during* a fight. `internal/combat` has none today, and
          this is another reason to keep it that way.
        - Unbounded state — anything that makes a position not comparable to another.
        - Real-time or reflex elements.
      - None of those are on the roadmap, so the property is currently free. Re-read this
        before adding anything that breaks one.
- [ ] **Endless tower (after the 8-floor version works).** Keep climbing until the curve
      stops you, rather than a fixed summit. Scaling probably exponential.
      - Design the floor loop so 8 is a *configured stop*, not a baked-in constant, or
        this becomes a rewrite instead of a setting.
      - Exponential scaling wants a sanity check on integer range and on the health bar:
        `DrawHealthBar` scales by `CurrentLife/MaxLife` so it copes, but a four-digit
        damage number will not fit the current caption.
      - The interesting design question is what actually stops you. Enemy stats
        outrunning yours, or a resource that runs down?
- [ ] **Enemy model: one archetype, scaled and affixed.** Enemies are essentially the
      same creature with main stats growing by depth, plus affixes that may stack. This
      contradicts how `combatants.json` is shaped today — a flat list of fully-specified
      records — so the data wants to become a base statline, a scaling rule, and a pool
      of affixes to draw from.
      - `AvailableAffixes` already anticipates this and is still unread.
      - Affixes must compose. Two on one enemy is the normal case, not an edge case.
      - Floor choices feed this directly: "a cold floor" biases which affixes appear.
- [ ] **Headless balance sim for the difficulty curve.** The curve gets playtested a lot,
      and most of that testing should not require playing.
      - `internal/combat` has no Ebitengine dependency precisely so this is possible:
        run thousands of duels across floor depths and plot where the player loses.
      - Needs the enemy scaling rule and a plausible player-plan model first, so it is
        downstream of the two items above — but it is the reason to keep combat pure.
- [ ] **Affixes.** `cold` / `hot` / `charged` / `undying` are in `combatants.json` and
      never read. Ring and effect art is partly present.
- [ ] **Replace Tyrian placeholder art — now a release blocker, not a cosmetic swap.**
      Technically still easy: the `SpriteSheet` + `SpriteRect` indirection means it's a
      data change. Keep it that way — no identifiers or packages named for it, no
      assumptions baked in about its palette or rect conventions.
      - The reason it got promoted: the Tyrian set has no formal license, only "use and
        abuse them as desired" on a 2007 Lost Garden blog post. That is fine for a public
        hobby repo and thin for a paid release. **This has to be resolved before the game
        is sold anywhere.**
      - Applies to everything under `assets/tyrian_graphics/` plus the two consolidated
        sheets actually embedded (`tyrian_monster_sprites.png`, `tyrian_ship_sprites.png`).

## Housekeeping

- [x] `data.Combatants` package var is declared and never assigned — dead, and a trap.
      Removed; `LoadCombatants()` returns the map and `GlobalState.Combatants` holds it.
- [x] `Update*Screen` returns `error`; callers discard it. Now propagated out of
      `Game.Update`. Ebitengine stops the loop on any non-nil error from `Update`, so a
      screen error is fatal by design — the only sensible reading of an error a screen
      cannot handle itself.
- [x] `Life_max` / `Life_current` → `MaxLife` / `CurrentLife` (Go naming).
- [x] `CreateRoundedRecMask` left edge is `radius+width` wide, should be `radius`. Works
      only because it's clipped at x=0. Fixed — no visual change, since the overdraw was
      always clipped away; it would have corrupted the mask the moment the function was
      called with a non-zero x.
- [x] Quitting exits status 1 and logs `Closing` like a crash. Now `game.ErrClosing`, a
      sentinel, with an `errors.Is` check in `main` — a deliberate quit exits 0 silently
      and anything else still goes to `log.Fatal`.
- [x] `.gitattributes` with `* text=auto eol=lf` — every file used to fail `gofmt -l` on
      line endings alone, making the check useless. PNG and TTF marked binary so they are
      never converted. The existing working tree was normalized with `gofmt -w`; the file
      only governs what git writes on checkout, so it would not have fixed files already
      on disk. `gofmt -l .` is now clean and worth running.
- [x] Embedded-asset drift: `thunder-ring.png` and the three `*-effect.png` files existed
      in `assets/` but were not embedded. All four now embedded and mapped
      (`thunderring_png`, `fireeffect_png`, `frozeneffect_png`, `thundereffect_png`),
      ready for the affix work.
- [x] Local `main` branch was 47 commits stale — fast-forwarded, and the merged
      `dad-work` / feature branches deleted locally and on the remote. `origin` now
      carries `main` only.

## Open decisions

- [x] **Hand-rolled UI vs [ebitenui](https://github.com/ebitenui/ebitenui) — decided:
      hand-roll everything.** Not part of Ebitengine; a separate community project.
      - **The design decision that settles it:** every interaction in this game is a
        click or a drag-and-drop, and there will be exactly **one** text input in the
        whole game — the seed field. Recorded as a firm rule in `CLAUDE.md`.
      - Drag-and-drop was the wrong trigger for this decision. The action box is a *game*
        widget — draggable cards with live AP validation — and general-purpose toolkits
        are weakest at bespoke game widgets. Roughly 200–300 lines hand-rolled, using hit
        testing and `inpututil` handling the codebase already has.
      - Repo data as of 2026-07-31, via the GitHub API: created 2020, last push
        2026-04-22, 928 stars, 74 forks, **73 open issues**, MIT, not archived. Releases
        v0.7.3 (Mar 2026), v0.7.2 (Sep 2025), v0.7.0 (Aug 2025). Contributors:
        mcarpenter622 322 commits, blizzy78 151, mat007 42, then a long tail.
      - What that means: two people are effectively the project, it is still pre-1.0
        after six years (so API churn is expected, not just abandonment risk), and a
        non-critical issue would likely sit. Fine for a hobby dependency, poor for
        something load-bearing in a product being sold.
      - It is also retained-mode — it owns a widget tree and its own event dispatch —
        which is a second UI architecture running alongside the `Scene` design rather
        than a layer that slides out cleanly.
      - [?] Revisit **only** if the seed text field turns out to be genuinely painful.
            Even then it is the easiest possible text input: short, ASCII, fixed
            charset, no selection or clipboard or IME needed. `ebiten.AppendInputChars`
            plus backspace handling is plausibly ~60 lines, so this trigger may never
            actually fire. If it does, confine any toolkit to whole screens and keep its
            types out of scene fields and signatures.
- [?] **Title hue shift.** `title.go` calls `ChangeHSV(1, 1, 1)`; `hueTheta` is *radians*,
      so that's a ~57° rotation, not identity. Source PNG is warm gold/amber. If the
      title looks greener on screen than the file, it's unintended — identity is
      `ChangeHSV(0, 1, 1)`.
- [x] **Start screen.** Flipped from the `Combat` dev shortcut back to `Title`. Combat is
      one click away and is now worth navigating to. Flipping it is also what surfaced
      the screen-transition nil deref above.

## Licensing (for an eventual Steam release)

Model: source stays public, nobody else may commercialise it. Justin and Sherman
have a signed agreement covering the relicense.

- [x] **Relicensed from Apache 2.0 to PolyForm Noncommercial 1.0.0.** Source-available,
      not open source. Read / build / modify / share for noncommercial purposes; selling
      is reserved to the copyright holders. README states the terms in plain English.
- [ ] **Put Sherman's legal name in `LICENSE`.** The Required Notice currently names the
      GitHub handle `KingSherman1820`. Deliberately deferred — a written partnership
      agreement covers the two of them — but a copyright notice naming only a handle is
      weak if it ever has to be enforced.
- [x] **Contributor grant — `CONTRIBUTING.md`.** Under a noncommercial licence an outside
      contributor keeps copyright and has *not* granted the right to sell their work, so
      merging unlicensed contributions would leave the game unsellable. Contributors now
      grant a perpetual, royalty-free, commercial-use-and-relicense licence, signalled by
      `git commit -s` (DCO convention). Landed before the repo has outside PRs, which is
      the only cheap time to do it.
- [x] **Streaming and video explicitly permitted, including monetized.** Additional
      Permissions section at the top of `LICENSE`, restated in the README. Keeps the
      standard PolyForm text unmodified rather than editing the licence body.
- [ ] **`THIRD-PARTY-NOTICES` file.** Apache-2.0 and BSD deps may sit inside a
      restricted-licence product, but only if their notices and attributions travel with
      the binary. Needed for a Steam build, not for the repo.
- [ ] Contact address for licensing enquiries. Deferred deliberately; anonymous is fine
      for now, and `CONTRIBUTING.md` points people at issues instead. Use a purpose-made
      address rather than a personal one when it happens.
- [ ] Get thirty minutes of actual legal review before relying on any of this. The
      licence is standard and well drafted; the contributor grant in `CONTRIBUTING.md`
      is a reasonable draft written by a non-lawyer.
- [x] Ebitengine — Apache 2.0. Permissive, so it can be included in a
      noncommercially-licensed product; notices must be retained.
- [x] All transitive deps — BSD (`golang.org/x/*`) or Apache. **No GPL anywhere** — this
      is what makes the relicense possible at all.
- [x] `Kubasta.ttf` — CC0, per the author's own FontStruct page.
- [ ] Confirm FiraSans and RobotoFlex (expected OFL / Apache — low risk).
- [ ] Tyrian art — see the release blocker under "Later". No formal licence, and that
      must be resolved before the game is sold.

Note: the Apache 2.0 grant on everything published before this change is irrevocable.
Anyone who already had the code keeps commercial rights to those snapshots. Accepted
deliberately — there was not enough there to matter — so no history rewrite. The new
licence governs from here forward.
