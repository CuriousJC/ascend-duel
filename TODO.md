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
      1 / 2 / 2 / 4 action points. A duel is a sequence of rounds: the player spends an
      AP budget on a set, side A's set resolves in full, then side B's, then control
      returns to re-plan. `Spd` buys budget (`AP = 4 + Spd/10`), not turn frequency.
      All integer arithmetic, so rounds are exactly reproducible.
      - Side A always resolves first, and the screen maps the player to A. A Guard the
        player places is therefore up for exactly the enemy's reply.
      - The enemy raises its guard *after* A has acted, so `Guarded` persists on the
        `Duelist` across rounds — otherwise the enemy's Guard could never protect
        anything. It clears at the start of its owner's next volley, which means it is
        still set in the state returned from the round it was raised in.
      - Superseded an earlier speed-timeline model where a fast duelist took whole
        extra turns. That is wrong for a game you re-plan every round: extra turns are
        invisible when you only ever plan one.
- [x] **DUEL! button.** `DuelButtonAction` resolves the whole duel up front; the screen
      only replays. A caption line reports what playback is showing.
- [~] **Decide Constitution → HP.** First real game rule. Wired: `NewCombatantFrom` sets
      `MaxLife = Con * entities.LifePerCon` and starts `CurrentLife` full, and
      `DrawHealthBar` now scales the red fill by `CurrentLife/MaxLife` over a darker
      track, so the bars are live rather than decorative.
      - [ ] `LifePerCon = 5` is a placeholder, not a decision. With current
            `combatants.json` it gives Fighter1 60 HP and Monster1 160 — the source Con
            values (12 vs 32) are lopsided, so tune the data before the constant.
      - [x] `CurrentLife` now moves, driven by playback of the resolved log.
- [ ] **Define `Action`.** Duel is: queue N actions → hit DUEL! → alternating turns,
      multiple actions per side per turn.
- [ ] **Drag-and-drop for the action box.** `DrawCombatButton` is the DUEL! button now;
      `DrawActionBox` / `DrawActions` are still empty stubs. The engine side is ready and
      waiting: `combat.AllActions` is the palette, `ActionKind.Cost()` the price,
      `Duelist.ActionPoints()` the budget and `CanAfford` the validity check — the UI
      just has to write `gs.FighterActions`. Until it exists, `defaultFighterPlan` in
      [combat.go](internal/screens/combat.go) stands in for the player's choices.
      This is the decision point on hand-rolled UI vs ebitenui — see "Open decisions".
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

- [ ] **`Scene` interface** to replace the two parallel switches in `game.go`. Deferred
      until per-screen combat state starts crowding `GlobalState` — that's the trigger.
      *(analysis §1, §2)*
- [ ] **Split `GlobalState`** into `Resources` (assets/fonts, read-only), `Layout`,
      `Session` (run progress), and per-screen state.
- [ ] **Ascend / tower loop.** `ascend.go` is a bare `package screens`; `Ascend` and
      `Credits` are empty cases in both switches.
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

- [ ] `data.Combatants` package var is declared and never assigned — dead, and a trap.
- [ ] `Update*Screen` returns `error`; callers discard it. Propagate or drop it.
- [x] `Life_max` / `Life_current` → `MaxLife` / `CurrentLife` (Go naming).
- [ ] `CreateRoundedRecMask` left edge is `radius+width` wide, should be `radius`. Works
      only because it's clipped at x=0.
- [ ] Quitting exits status 1 and logs `Closing` like a crash — `Update` returns an
      error, `main` hands it to `log.Fatal`. Wants a sentinel error and an `errors.Is`
      check so a clean quit exits 0.
- [ ] `.gitattributes` with `*.go text eol=lf` — every file currently fails `gofmt -l`
      on line endings alone.
- [ ] Embedded-asset drift: `thunder-ring.png` and the three `*-effect.png` files exist
      in `assets/` but aren't embedded.
- [ ] Local `main` branch is ~40 commits stale. `git merge --ff-only origin/main`.

## Open decisions

- [?] **Hand-rolled UI vs [ebitenui](https://github.com/ebitenui/ebitenui) (MIT).**
      Not part of Ebitengine — separate community project. Drag-and-drop for the action
      box is where hand-rolling stops being cheap, so this decision lands with the next
      feature.
- [?] **Title hue shift.** `title.go` calls `ChangeHSV(1, 1, 1)`; `hueTheta` is *radians*,
      so that's a ~57° rotation, not identity. Source PNG is warm gold/amber. If the
      title looks greener on screen than the file, it's unintended — identity is
      `ChangeHSV(0, 1, 1)`.
- [?] **Start screen.** `NewGlobalState()` opens on `Combat` as a dev shortcut. Flip back
      to `Title` when combat has enough to be worth navigating to.

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
