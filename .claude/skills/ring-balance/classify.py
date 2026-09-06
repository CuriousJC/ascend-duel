"""What every ring in the catalogue *is*, derived from its rules rather than authored.

    python .claude/skills/ring-balance/classify.py                 # the summary, every axis by rarity
    python .claude/skills/ring-balance/classify.py --by category element
    python .claude/skills/ring-balance/classify.py --list defense
    python .claude/skills/ring-balance/classify.py --ring "Fire Ring"
    python .claude/skills/ring-balance/classify.py --shelf           # counts against what a player actually sees
    python .claude/skills/ring-balance/classify.py --holes          # a filled family with an empty sibling

Run from the repo root. It reads data/rings.json and data/statuses.json and puts
each ring on several axes. Nothing is written down in rings.json: a ring's
category is a fact about its verbs, so the file cannot disagree with the rules.

The tables below are the whole taxonomy. A new verb in internal/combat/ring.go
must be added to VERBS or this refuses to run -- the same rule the game's own
loaders follow, and the reason it cannot go quietly stale.
"""
import json
import sys

# --- the taxonomy -----------------------------------------------------------
#
# categories are many-to-many: a ring may be offense and growth at once.
# payload, scope and breadth are one value per rule, so several per ring.

# verb -> (categories, payload)
VERBS = {
    # offense
    "scale-damage":            (("offense",), "multiplicative"),
    "scale-hand-damage":       (("offense",), "multiplicative"),
    "scale-damage-per-vitae":  (("offense",), "multiplicative"),
    "add-hand-damage":         (("offense",), "flat"),
    "add-damage-per-held":     (("offense",), "flat"),
    "add-damage-per-vitae":    (("offense",), "flat"),
    "add-dmg":                 (("offense",), "flat"),
    "repeat-card":             (("offense",), "repeat"),
    "echo-attack":             (("offense",), "repeat"),
    # defense
    "add-hp":                  (("defense",), "flat"),
    "scale-hp":                (("defense",), "multiplicative"),
    # tempo -- buys action points, never damage
    "adjust-cost":             (("tempo",), "flat"),
    "demote-card":             (("tempo",), "flat"),
    # economy -- the run's purse and its picks
    "scale-propagation":       (("economy",), "multiplicative"),
    "adjust-prize-vitae":      (("economy",), "flat"),
    "adjust-picks":            (("economy",), "flat"),
    # enabler -- changes what you are holding so something else can fire
    "set-element":             (("enabler",), "enabler"),
    # growth -- value is a function of time, and the accumulator is state
    "grow-on-hit":             (("offense", "growth"), "stateful"),
    "grow-on-win":             (("growth",), "stateful"),
    "grow-on-turn":            (("growth",), "stateful"),
    "grow-per-card":           (("growth",), "stateful"),
    "reset-growth":            (("growth",), "stateful"),
    # the split verb -- read off the status it applies
    "apply-status":            (None, "status"),
}

# status Effect kind -> category. CHILLED steals a card off their turn, which is
# damage you never take: denial counts as defense here (owner's call, 2026-09-06).
STATUS_EFFECTS = {
    "damage-over-time":      "offense",
    "damage-amplification":  "offense",
    "lose-actions":          "defense",
    "miss-chance":           "defense",
    "damage-reduction":      "defense",
}

# When -> scope. How often the rule gets to matter.
SCOPES = {
    "card-cost":    "per-card",
    "card-damage":  "per-card",
    "card-drawn":   "per-card",
    "attack-lands": "per-blow",
    "blow-formed":  "per-blow",
    "turn-taken":   "per-turn",
    "deck-built":   "per-fight",
    "fight-start":  "per-fight",
    "fight-won":    "per-run",
    "prizes-dealt": "per-run",
}

# predicate key -> breadth. How much of the deck a rule can see.
BREADTHS = {
    "Element": "element", "Form": "form", "Concept": "concept",
    "Tier": "tier", "Hand": "hand", "MinForms": "hand", "Lead": "positional",
}

AXES = ("category", "payload", "scope", "breadth", "rarity", "build",
        "when", "do")

# --- deriving ---------------------------------------------------------------

RINGS = json.load(open("data/rings.json"))
STATUSES = {s["StatusRecord"]: s for s in json.load(open("data/statuses.json"))}


def unknown(what, word):
    sys.exit("unknown %s: %r -- add it to the table in "
             ".claude/skills/ring-balance/classify.py" % (what, word))


def classify(ring):
    """Every axis of one ring, as sets of strings."""
    out = {a: set() for a in AXES}
    out["rarity"].add(ring["Rarity"])
    for rule in ring["Rules"]:
        when = rule["When"]
        if when not in SCOPES:
            unknown("moment", when)
        out["when"].add(when)
        out["scope"].add(SCOPES[when])

        cond = rule.get("If") or {}
        for k in cond:
            if k not in BREADTHS:
                unknown("predicate", k)
        keys = [k for k in cond if k != "MinForms"] or list(cond)
        out["breadth"].add(BREADTHS[keys[0]] if keys else "unconditional")
        for k in keys:
            out["build"].add("%s=%s" % (k, cond[k]))
        if not keys:
            out["build"].add("(any card)")

        for effect in rule["Then"]:
            do = effect["Do"]
            if do not in VERBS:
                unknown("verb", do)
            cats, payload = VERBS[do]
            out["do"].add(do)
            out["payload"].add(payload)
            if cats is None:                       # apply-status
                status = STATUSES.get(effect.get("Status"))
                if status is None:
                    unknown("status", effect.get("Status"))
                kind = status["Effect"]
                if kind not in STATUS_EFFECTS:
                    unknown("status effect", kind)
                out["category"].add(STATUS_EFFECTS[kind])
            else:
                out["category"].update(cats)
            # a scaling verb told to scale *down* is a drawback, whatever else it is
            if payload == "multiplicative" and effect.get("Amount", 100) < 100:
                out["category"].add("drawback")
    return out


TABLE = {r["Name"]: classify(r) for r in RINGS}

# --- reporting --------------------------------------------------------------

ORDER = {
    "category": ["offense", "defense", "tempo", "economy", "enabler", "growth",
                 "drawback"],
    "payload": ["flat", "multiplicative", "repeat", "status", "stateful",
                "enabler"],
    "scope": ["per-card", "per-blow", "per-turn", "per-fight", "per-run"],
    "breadth": ["unconditional", "element", "form", "concept", "tier", "hand",
                "positional"],
    "rarity": ["common", "uncommon", "rare"],
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
    print("\n%s x %s   (a ring is counted in every cell it occupies)\n"
          % (rows, cols))
    head = "".join("%10s" % str(c)[:9] for c in cv)
    print("%-*s%s%10s" % (width, "", head, "rings"))
    for r in rv:
        line = "%-*s" % (width, r)
        for c in cv:
            n = sum(1 for k in TABLE
                    if r in TABLE[k][rows] and c in TABLE[k][cols])
            line += "%10s" % (n or ".")
        total = sum(1 for k in TABLE if r in TABLE[k][rows])
        print(line + "%10d" % total)
    print("\n%d rings in the catalogue" % len(TABLE))


def summary():
    print("%d rings, classified from their rules alone" % len(TABLE))
    for axis in ("category", "payload", "scope", "breadth"):
        crosstab(axis, "rarity")


def listing(value):
    hits = sorted(k for k in TABLE
                  if any(value in TABLE[k][a] for a in AXES))
    if not hits:
        print("nothing matches %r. axis values:" % value)
        for a in AXES:
            print("  %-9s %s" % (a, ", ".join(str(v) for v in values(a))))
        sys.exit(1)
    print("%d rings matching %r:\n" % (len(hits), value))
    for name in hits:
        c = TABLE[name]
        print("  %-26s %-9s %-26s %s"
              % (name, "/".join(sorted(c["rarity"])),
                 "+".join(sorted(c["category"])),
                 "+".join(sorted(c["payload"]))))


def one(name):
    match = [k for k in TABLE if k.lower() == name.lower()]
    if not match:
        match = [k for k in TABLE if name.lower() in k.lower()]
    if not match:
        sys.exit("no ring named %r" % name)
    for k in match:
        print("\n%s" % k)
        for axis in AXES:
            print("  %-10s %s"
                  % (axis, ", ".join(sorted(str(v) for v in TABLE[k][axis]))))


SHELF_TICKETS = {"common": 10, "uncommon": 4, "rare": 1}


def shelf():
    """Counts against what a player actually sees. The two are not the same.

    The shelf draws on rarity tickets -- 10 / 4 / 1 -- so a category living at one
    tier has a share nothing about its record count would tell you.
    """
    total = sum(SHELF_TICKETS[r["Rarity"]] for r in RINGS)
    print("\n%d rings, %d shelf tickets\n" % (len(TABLE), total))
    for axis in ("category", "payload", "breadth", "rarity"):
        print("%-16s %6s %10s %10s" % (axis, "rings", "% of cat", "% of shelf"))
        for v in values(axis):
            hits = [k for k in TABLE if v in TABLE[k][axis]]
            tickets = sum(SHELF_TICKETS[list(TABLE[k]["rarity"])[0]] for k in hits)
            print("%-16s %6d %9.1f%% %9.1f%%"
                  % (v, len(hits), 100.0 * len(hits) / len(TABLE),
                     100.0 * tickets / total))
        print("")


def holes():
    """A build axis some category never reaches. A question, not a finding."""
    print("\ncoverage holes -- a filled family with an empty sibling:\n")
    for prefix in ("Element=", "Form="):
        builds = [b for b in values("build") if b.startswith(prefix)]
        for cat in values("category"):
            held = [b for b in builds
                    if any(cat in TABLE[k]["category"] and b in TABLE[k]["build"]
                           for k in TABLE)]
            # A category keyed on nothing at all is not a hole in this family --
            # economy rings read the purse and never look at a card.
            if not held:
                continue
            missing = [b for b in builds if b not in held]
            if missing:
                print("  %-10s never on: %s" % (cat, ", ".join(missing)))
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
elif args[0] == "--ring" and len(args) == 2:
    one(args[1])
elif args[0] == "--shelf":
    shelf()
elif args[0] == "--holes":
    holes()
else:
    sys.exit(__doc__)
