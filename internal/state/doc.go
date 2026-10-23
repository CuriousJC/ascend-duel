// Package state is what is genuinely shared between every scene and system: input, timing,
// layout, the loaded resources, which screen is active, and the run in progress. It is threaded by
// pointer into everything.
//
// The test for belonging here is not "is it important" but "does more than one screen need it".
// GlobalState used to carry the combat screen's duel log, playback cursor and combatants, which
// meant every screen could see them and none of them were anyone else's business. A screen's own
// working data lives on its scene; what has to outlive a fight lives on the run.
//
// # Layering
//
// It sits near the bottom of the dependency graph and must stay there. It imports data,
// internal/session and Ebitengine, and nothing else — no entities, no models, no screens. If it
// starts importing those again, screen state has leaked back into it.
//
// The one documented bend is Run, which makes this package import internal/combat transitively. A
// run is not screen state: it belongs beside ActiveScreen rather than on whichever screen happened
// to need it first.
//
// # Placing things
//
// PctX and PctY are the intended way to position anything. They replaced a dozen cached fields for
// halves, thirds and quarters, which could not express 40% and could not be extended to without
// inventing a field name per fraction. Percentages anchor a group; offsets within a group stay in
// pixels, and sizes are never percentages.
//
// # The dev tools flag
//
// DevTools is one switch for every dev tool, read off the profile once in main — nothing in the
// game sets it, and the input vocabulary has no keyboard to flip it with. No tool behind it may
// ever change an outcome.
package state
