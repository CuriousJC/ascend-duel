"""What every relic in the catalogue *is*, derived from its rules rather than authored.

    python .claude/skills/relic-balance/classify.py                 # the summary, every axis by rarity
    python .claude/skills/relic-balance/classify.py --by aspect breadth
    python .claude/skills/relic-balance/classify.py --list defense
    python .claude/skills/relic-balance/classify.py --relic "Fire Ring"
    python .claude/skills/relic-balance/classify.py --shelf           # counts against what a player actually sees
    python .claude/skills/relic-balance/classify.py --holes           # a filled family with an empty sibling

Run from the repo root. It reads data/relics.json and puts each relic on several axes. Nothing is
written down in relics.json: a relic's aspects are a fact about its verbs and its predicates, so the
file cannot disagree with the rules.

The aspects are the nine in docs/design/build-aspects.md, and that document is where they are
defined. The tables below are only the sorting of the grammar into them. A new verb, moment or
predicate in internal/combat/relic.go must be added here or this refuses to run -- the same rule the
game's own loaders follow, and the reason it cannot go quietly stale.
"""
import json
import sys

# --- the taxonomy -----------------------------------------------------------
#
# A relic carries a primary aspect -- the first aspect of its first effect -- and every aspect any of
# its effects or predicates reaches. payload, scope and breadth are one value per rule, so several per
# relic.

ASPECTS = ["economy", "composition", "hands", "action-capacity", "tempo", "consistency",
           "offense", "defense", "engine-capacity"]

# verb -> (aspects, primary first; payload)
VERBS = {
    # offense
    "scale-damage":            (("offense",), "multiplicative"),
    "add-dmg":                 (("offense",), "flat"),
    "add-dmg-per-held":        (("offense",), "flat"),
    "repeat-card":             (("offense",), "repeat"),
    "echo-attack":             (("offense",), "repeat"),
    "awaken":                  (("offense",), "flat"),
    # a hit landing where it would have fizzled is damage that was not thrown before
    "pierce-element":          (("offense",), "enabler"),
    # luck: widens the paying band on every roll
    "scale-rolls":             (("offense",), "multiplicative"),
    # damage read off the purse: offense, priced by how much economy the run kept
    "add-dmg-per-vitae":       (("offense", "economy"), "flat"),
    "scale-damage-per-vitae":  (("offense", "economy"), "multiplicative"),
    # hands
    "add-hand-dmg":            (("hands", "offense"), "flat"),
    "scale-hand-damage":       (("hands", "offense"), "multiplicative"),
    # defense
    "add-hp":                  (("defense",), "flat"),
    "scale-hp":                (("defense",), "multiplicative"),
    "heal-share":              (("defense",), "flat"),
    "heal-on-block":           (("defense",), "flat"),
    # sustain off a blow: the share scales with what the blow was worth
    "drain-damage":            (("defense", "offense"), "multiplicative"),
    "raise-shield":            (("defense",), "flat"),
    "shield-per-card":         (("defense",), "flat"),
    "keep-shields":            (("defense", "tempo"), "flat"),
    "match-foe-shields":       (("defense",), "enabler"),
    # thorns: a block that hits back
    "reflect-damage":          (("defense", "offense"), "multiplicative"),
    # action capacity -- what a turn can commit
    "adjust-cost":             (("action-capacity",), "flat"),
    "demote-card":             (("action-capacity",), "flat"),
    "surge-per-card":          (("action-capacity",), "flat"),
    # a block that banks AP: the design document's own example of the overlap
    "surge-on-block":          (("action-capacity", "defense"), "flat"),
    # tempo -- the fight's deadline
    "adjust-round-limit":      (("tempo",), "flat"),
    # economy -- the run's purse
    "scale-propagation":       (("economy",), "multiplicative"),
    "adjust-prize-vitae":      (("economy",), "flat"),
    "vitae-on-block":          (("economy", "defense"), "flat"),
    # consistency -- what the deck can be made to deliver
    "adjust-picks":            (("consistency", "composition"), "flat"),
    "adjust-essence-targets":  (("consistency", "composition"), "flat"),
    # composition -- changes what you are holding so something else can fire
    "set-element":             (("composition",), "enabler"),
    "set-form":                (("composition",), "enabler"),
    # engine capacity -- more resources carried at once
    "copy-consumable":         (("engine-capacity",), "flat"),
    # growth: value is a function of time, which is tempo -- a build that is strong later. The
    # accumulator is read by the relic's other effects, which carry their own aspects.
    "grow-on-hit":             (("offense", "tempo"), "stateful"),
    "grow-on-streak":          (("offense", "tempo"), "stateful"),
    "grow-on-win":             (("tempo",), "stateful"),
    "grow-on-turn":            (("tempo",), "stateful"),
    "grow-per-card":           (("tempo",), "stateful"),
    "reset-growth":            (("tempo",), "stateful"),
}

# When -> scope. How often the rule gets to matter.
SCOPES = {
    "card-cost":     "per-card",
    "card-damage":   "per-card",
    "card-drawn":    "per-card",
    "attack-lands":  "per-blow",
    # one per incoming hit a shield eats, which is the enemy's blow landing on nothing
    "hit-blocked":   "per-blow",
    "blow-formed":   "per-blow",
    "turn-taken":    "per-turn",
    "turn-start":    "per-turn",
    "equipped":      "per-fight",
    "fight-begun":   "per-fight",
    "fight-won":     "per-run",
    "prizes-dealt":  "per-run",
    # an essence edits the deck the whole climb is played with, so its reach is the run's
    "essence-spent": "per-run",
}

# predicate key -> (breadth, the aspect a rule narrowed by it reaches, or None)
#
# A rule keyed on an element, a form or a concept rewards a deck built toward it, which is
# composition; one keyed on a rung rewards assembling it, which is hands.
BREADTHS = {
    "Element":   ("element", "composition"),
    "Form":      ("form", "composition"),
    "Concept":   ("concept", "composition"),
    "Tier":      ("tier", "composition"),
    "Favored":   ("favored", "composition"),
    "Hand":      ("hand", "hands"),
    "Hands":     ("hand", "hands"),
    "MinForms":  ("hand", "hands"),
    "Lead":      ("positional", None),
    "Preceding": ("positional", None),
}

AXES = ("primary", "aspect", "payload", "scope", "breadth", "rarity", "build",
        "drawback", "when", "do")

# --- deriving ---------------------------------------------------------------

RELICS = json.load(open("data/relics.json", encoding="utf-8"))


def unknown(what, word):
    sys.exit("unknown %s: %r -- add it to the table in "
             ".claude/skills/relic-balance/classify.py" % (what, word))


def classify(relic):
    """Every axis of one relic, as sets of strings."""
    out = {a: set() for a in AXES}
    out["rarity"].add(relic["Rarity"])
    for rule in relic["Rules"]:
        when = rule["When"]
        if when not in SCOPES:
            unknown("moment", when)
        out["when"].add(when)
        out["scope"].add(SCOPES[when])

        cond = rule.get("If") or {}
        for k in cond:
            if k not in BREADTHS:
                unknown("predicate", k)
            if BREADTHS[k][1]:
                out["aspect"].add(BREADTHS[k][1])
        keys = [k for k in cond if k != "MinForms"] or list(cond)
        out["breadth"].add(BREADTHS[keys[0]][0] if keys else "unconditional")
        for k in keys:
            out["build"].add("%s=%s" % (k, cond[k]))
        if not keys:
            out["build"].add("(any card)")

        for effect in rule["Then"]:
            do = effect["Do"]
            if do not in VERBS:
                unknown("verb", do)
            aspects, payload = VERBS[do]
            if not out["primary"]:
                out["primary"].add(aspects[0])
            out["do"].add(do)
            out["payload"].add(payload)
            out["aspect"].update(aspects)
            # a scaling verb told to scale *down*, or a clock told to shorten, takes something away
            down = (payload == "multiplicative" and effect.get("Amount", 100) < 100) or \
                (do == "adjust-round-limit" and effect.get("Amount", 0) < 0)
            if down:
                out["drawback"].add("drawback")
    return out


TABLE = {r["Name"]: classify(r) for r in RELICS}

# --- reporting --------------------------------------------------------------

ORDER = {
    "primary": ASPECTS,
    "aspect": ASPECTS,
    "payload": ["flat", "multiplicative", "repeat", "stateful", "enabler"],
    "scope": ["per-card", "per-blow", "per-turn", "per-fight", "per-run"],
    "breadth": ["unconditional", "element", "form", "concept", "tier", "favored", "hand",
                "positional"],
    "rarity": ["common", "uncommon", "rare", "mythic"],
}


def values(axis):
    if axis not in AXES:
        sys.exit("no axis %r. axes: %s" % (axis, ", ".join(AXES)))
    seen = {v for c in TABLE.values() for v in c[axis]}
    known = ORDER.get(axis)
    if known:
        return [v for v in known if v in seen] + sorted(seen - set(known))
    return sorted(seen, key=str)


def crosstab(rows, cols):
    rv, cv = values(rows), values(cols)
    width = max([len(str(v)) for v in rv] + [len(rows)]) + 2
    print("\n%s x %s   (a relic is counted in every cell it occupies)\n"
          % (rows, cols))
    head = "".join("%10s" % str(c)[:9] for c in cv)
    print("%-*s%s%10s" % (width, "", head, "relics"))
    for r in rv:
        line = "%-*s" % (width, r)
        for c in cv:
            n = sum(1 for k in TABLE
                    if r in TABLE[k][rows] and c in TABLE[k][cols])
            line += "%10s" % (n or ".")
        total = sum(1 for k in TABLE if r in TABLE[k][rows])
        print(line + "%10d" % total)
    print("\n%d relics in the catalogue" % len(TABLE))


def summary():
    print("%d relics, classified from their rules alone" % len(TABLE))
    for axis in ("primary", "aspect", "payload", "scope", "breadth"):
        crosstab(axis, "rarity")


def listing(value):
    hits = sorted(k for k in TABLE
                  if any(value in TABLE[k][a] for a in AXES))
    if not hits:
        print("nothing matches %r. axis values:" % value)
        for a in AXES:
            print("  %-9s %s" % (a, ", ".join(str(v) for v in values(a))))
        sys.exit(1)
    print("%d relics matching %r:\n" % (len(hits), value))
    for name in hits:
        c = TABLE[name]
        print("  %-30s %-9s %-16s %-40s %s"
              % (name, "/".join(sorted(c["rarity"])), "".join(c["primary"]),
                 "+".join(a for a in ASPECTS if a in c["aspect"]),
                 "+".join(sorted(c["payload"]))))


def one(name):
    match = [k for k in TABLE if k.lower() == name.lower()]
    if not match:
        match = [k for k in TABLE if name.lower() in k.lower()]
    if not match:
        sys.exit("no relic named %r" % name)
    for k in match:
        print("\n%s" % k)
        for axis in AXES:
            print("  %-10s %s"
                  % (axis, ", ".join(sorted(str(v) for v in TABLE[k][axis]))))


# The shelf's tickets per tier. A mythic is not drawn on tickets -- each seat rolls for one on its
# own stream at data.MythicOdds -- so it holds no share of the ticket draw.
SHELF_TICKETS = {"common": 10, "uncommon": 4, "rare": 1, "mythic": 0}


def shelf():
    """Counts against what a player actually sees. The two are not the same.

    The shelf draws on rarity tickets -- 10 / 4 / 1 -- so an aspect living at one tier has a share
    nothing about its record count would tell you.
    """
    total = sum(SHELF_TICKETS[r["Rarity"]] for r in RELICS)
    print("\n%d relics, %d shelf tickets (mythics roll apart and hold none)\n"
          % (len(TABLE), total))
    for axis in ("primary", "aspect", "payload", "breadth", "rarity"):
        print("%-16s %6s %10s %10s" % (axis, "relics", "% of cat", "% of shelf"))
        for v in values(axis):
            hits = [k for k in TABLE if v in TABLE[k][axis]]
            tickets = sum(SHELF_TICKETS[list(TABLE[k]["rarity"])[0]] for k in hits)
            print("%-16s %6d %9.1f%% %9.1f%%"
                  % (v, len(hits), 100.0 * len(hits) / len(TABLE),
                     100.0 * tickets / total))
        print("")


def holes():
    """A build axis some aspect never reaches. A question, not a finding."""
    print("\ncoverage holes -- a filled family with an empty sibling:\n")
    for prefix in ("Element=", "Form="):
        builds = [b for b in values("build") if b.startswith(prefix)]
        for aspect in values("aspect"):
            held = [b for b in builds
                    if any(aspect in TABLE[k]["aspect"] and b in TABLE[k]["build"]
                           for k in TABLE)]
            # An aspect keyed on nothing at all is not a hole in this family --
            # economy relics read the purse and never look at a card.
            if not held:
                continue
            missing = [b for b in builds if b not in held]
            if missing:
                print("  %-16s never on: %s" % (aspect, ", ".join(missing)))
    print("\n  A hole is a question. Defend takes no scale-damage because a "
          "defend card\n  deals nothing; that emptiness is the grammar being "
          "right.")


args = sys.argv[1:]
if not args:
    summary()
elif args[0] == "--by" and len(args) == 3:
    crosstab(args[1], args[2])
elif args[0] == "--list" and len(args) == 2:
    listing(args[1])
elif args[0] == "--relic" and len(args) == 2:
    one(args[1])
elif args[0] == "--shelf":
    shelf()
elif args[0] == "--holes":
    holes()
else:
    sys.exit(__doc__)
