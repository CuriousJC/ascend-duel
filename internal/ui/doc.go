// Package ui is the drawing vocabulary every scene shares: the table, the clock, the movers, the
// card faces, and the panels that belong to no screen.
//
// **It came out of internal/screens on 2026-09-17**, which was two thirds of the Go in the repo and
// one package. What moved is everything a scene *draws through* rather than anything a scene
// decides; what stayed is the scenes themselves. The cut was chosen by measuring rather than by
// taste — see the `audit` skill's pkgsplit, which reports every unexported name that would have to
// cross a proposed line, in both directions. This line has none going the wrong way.
//
// # The rule the boundary states
//
// **Nothing here knows what screen it is on, and nothing here decides anything about a run.** A
// caller hands over a rectangle, a state and a card; this package hands back a picture. That was
// already the intent — drawPane took a rect rather than a percentage, a widget took 0..1 and knew
// nothing about its range — and being a package is what makes it checkable instead of a habit.
//
// The boundary is also where sizes and placement part company. A control's *measurements* are the
// frame's and live here; *where* it goes is often measured from the hand or the action-point bar,
// and that stayed on the combat screen. See frame.go, which holds the line and says why.
//
// # What is in it
//
// **A grouping, not an inventory.** Each file's own header comment is the detail, and a list here
// that repeated them would be a second copy to keep in step — which is exactly how the package doc
// this was cut from came to describe less than half its own package.
//
//   - The table and the clock — ground.go, clock.go. screenGround is what everything is painted on
//     and everything dims toward; Beat is the game's one speed and there is no second one.
//     clock_test.go parses this package *and* internal/screens and fails on a raw duration.
//   - Movement — travel.go, cardslide.go, cardmorph.go, theater.go. A Travel is the
//     delay-age-duration clock every mover shares; a Morph is two finished faces and a clock; a
//     theater is everything a scene has moving on it, as a contract rather than a struct.
//   - Card faces — card_art.go specs a card and caches the render, card_draw.go blits one. The
//     cache is bounded: see maxCardCache, and the comment there about the bound it used to claim.
//   - Panels and dialogs — modal.go, pane.go, confirm.go, toast.go, deckpanel.go,
//     deckpanel_view.go, deckfilter.go, handspanel.go. The three dialog shapes are the near-full
//     screen panel, the small centered question, and the tutorial's bubble, which is a scene's.
//   - Rows of cards — carddrag.go is the press-and-drag lifecycle every reorderable row shares,
//     handsort.go the half of sorting that belongs to no screen, shield_row.go the pips.
//   - Words — prose.go and prose_terms.go turn an event the engine has already decided into a
//     sentence, elementink.go says which colored word takes which ink, tips.go builds the
//     tooltips. They compute nothing, which is what makes it impossible for a panel to disagree
//     with the round it reports.
//   - The frame — frame.go, uikit.go. The corner cards, the control strip, and the handful of
//     helpers that had come to live inside one screen.
//
// # It still links Ebitengine
//
// So the split buys readability and a real boundary, **not** testability: this package needs a
// window on Linux exactly as internal/screens does, and CI runs its tests under xvfb for the same
// reason. The packages that can be tested without a display are internal/combat, internal/session
// and the rest below them, and nothing here changes that.
//
// The tests that do exist are the same narrow exception internal/screens takes: they compare
// constants, walk switch statements and guard cross-package invariants a compiler cannot see. They
// create no images and are not license to test the rest of the drawing.
package ui
