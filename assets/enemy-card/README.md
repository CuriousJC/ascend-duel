Pictures for the creature cards, one per attack flavor per element.

Keyed by filename stem: `enemy-card/stab-bite-fire.png` is the key `stab-bite-fire`, built from
the `Art` stem `data/enemy_card_art.json` writes on the flavor and the element the card was dealt
as. Committed at the card's own 200x280, the figure fitted inside `data.EnemyCardArtField` by
`go run ./tools/relicart -kind enemy-card`.

`docs/art/enemy_card_art_prompt.MD` is what they are generated from. There is no default face
here on purpose: a flavor with no picture draws its cards as they always looked.
