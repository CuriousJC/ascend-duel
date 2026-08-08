# Ideas

**This is the inbox.** Entries here are unfiltered and not decided. Once one is settled it is
promoted into [MECHANICS.md](MECHANICS.md) if it is a rule, or [TODO.md](TODO.md) if it is a
task, and struck from here.

Bosses, attributes, floors and rings lived here until 2026-08-05 and have all been promoted —
bosses and floors into the tower section, attributes into its own section with the armour and
scaling conflicts recorded, rings into the ring section. The originals are in git history if
the wording is ever wanted.

- we could do something for holidays
- **Show the enemy's queued actions during selection, and play the round as cards flying up
  rather than as sentences.** Raised 2026-08-08. Two separate things that arrived together:
  - **Reveal.** The opponent's queued actions are concealed in the Action Flow pane unless
    `DebugGameplay` is on — `CombatScene.concealEnemy` is the one predicate. Turning that off
    for real makes the duel perfect information, so the choice stops being "guess what they
    are doing" and becomes "answer what they are doing". That is a game-design call, not a
    presentation one, and it collides with the hidden-vs-random question already open in
    `TODO.md`. **Decide the reveal on its own merits before building the animation**, or the
    animation will settle it by default.
  - **Playback as cards.** Replace the Resolution pane's sentences with the actual cards:
    mine and theirs, flowing up out of the two queues into a shared duel space in
    `ResolutionOrder` order. Mostly cosmetic for now — it says the same thing the prose says,
    in the vocabulary the rest of the screen already uses. Cheap version first: animate the
    existing cards up and leave the prose beside them, so the sentences stay available while
    the motion is being judged.
  - Constraint either way: **presentation may never change outcomes.** `ResolveRound` still
    decides the whole round before playback starts; cards flying up is the log being replayed,
    exactly as the prose is now.
