// Package entities is the game-world actors: a Combatant is a combat.Duelist that can be drawn.
//
// The stats live in the embedded rules type so internal/combat can take them without ever seeing
// a picture, and what this package adds is the handful of things a screen needs and the engine
// must not have — the record key, the portrait key, the name, the card back.
//
// There is no sprite here, and no Ebitengine import at all. Both duelists are cards now, so a
// portrait travels as an assets key rather than as an image, and the card is drawn by
// internal/cards from raw bytes. That is one of the two things standing between this package and
// needing a window, and it is worth keeping.
//
// # Hydration is where the ascent is applied
//
// NewEnemyFrom takes a fight index and grows the record's HP and DMG through pyramid.ScaleToFight,
// so an unscaled opponent cannot be built by accident: every caller has to say where in the climb
// this one stands. Actions is deliberately left alone — it is the budget a deck is spent out of
// rather than a measure of how hard one hits.
//
// The curve itself is not here. It moved to internal/pyramid, which is tower generation rather
// than hydration, and which tools/balance reads without needing this package at all.
package entities
