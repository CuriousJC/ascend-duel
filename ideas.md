# Ideas

**This is the inbox.** Entries here are unfiltered and not decided. Once one is settled it is
promoted into [MECHANICS.md](MECHANICS.md) if it is a rule, or [TODO.md](TODO.md) if it is a
task, and struck from here.

- we could do something for holidays
- **Reveal the enemy's queued actions during selection.** They were concealed in the queued-actions
  pane unless `DebugGameplay` is on — `CombatScene.concealEnemy` is the one predicate. Turning
  that off for real makes the duel perfect information, so the choice stops being "guess what
  they are doing" and becomes "answer what they are doing". That is a game-design call, not a
  presentation one, and it collides with the hidden-vs-random question already open in
  `TODO.md`.
  - **The table row already draws them face up**, on the owner's call, and deliberately ignores
    `concealEnemy`. So the reveal is half-made in practice; `cards.Spec.FaceDown` is the lever
    for putting it back.
