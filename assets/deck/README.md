Full-bleed card backs, one per deck.

Keyed `deck-<stem>`: `deck/rainbow-gem.png` is the key `deck-rainbow-gem`, and the stem is
what `data/decks.json` writes in its `Art` field. Committed at the card's own 200x280.

`docs/art/deck_art_prompt.MD` is what they are generated from, with the record's `Draw` pasted
after it. There is no default back here on purpose — a deck with no picture draws the plain dark
back `internal/cards` paints in code.
