// Package pyramid is the climb: how many rooms a floor holds, how much harder each room is
// than the one below it, and which opponent stands in each one.
//
// **It is tower generation, not a rule about resolving a round**, which is why it is not in
// internal/combat, and it is not a screen's either — the combat screen used to own the fight
// order on its own scene, which meant the shape of the run was decided by the screen you fight
// on and rebuilt every time you entered it. `tools/balance` needs the same arithmetic and
// cannot import a screen, which is the other half of why this is a package.
//
// **No Ebitengine, ever**, for that reason, and no randomness of its own: New takes the source
// it shuffles with, so the caller owns which stream is being advanced. See the randomness skill.
//
// **This is where the room choice will bite** — the scene after the shop that shapes what comes
// next. A Pyramid is a run's fight order, held by the session, so shaping the next room is a
// method here rather than a screen writing into another screen's state.
package pyramid
