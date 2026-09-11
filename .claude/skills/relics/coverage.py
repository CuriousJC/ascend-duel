"""The ring coverage grid, derived from data/ rather than remembered.

    python .claude/skills/rings/coverage.py          # the grid, then the gaps
    python .claude/skills/rings/coverage.py --gaps   # gaps only

Run from the repo root. It reads data/rings.json and data/duelist_cards.json and
prints, for every (When, Do, predicate) cell, which rings occupy it -- and then
which cells on a symmetry axis are empty while their siblings are filled.

A gap is a question, never a finding. Defend takes no scale-damage because a
defend card deals nothing; that empty cell is the grammar being right.
"""
import collections
import json
import sys

R = json.load(open("data/rings.json"))
C = json.load(open("data/duelist_cards.json"))
H = json.load(open("data/hands.json"))["hands"]

ELEMENTS = sorted({e for c in C for e in c.get("Elements", [])})
FORMS = sorted({c["Form"] for c in C})
CONCEPTS = sorted({c["Label"] for c in C})
TIERS = sorted({c["Cost"] for c in C})
HANDS = [h["key"] for h in H]
AXES = {"Element": ELEMENTS, "Form": FORMS, "Concept": CONCEPTS, "Tier": TIERS,
        "Hand": HANDS}
PREDICATES = ("Element", "Form", "Concept", "Tier", "Hand", "Lead")

grid = collections.defaultdict(list)
for ring in R:
    for rule in ring["Rules"]:
        cond = rule.get("If") or {}
        keyed = tuple((p, cond[p]) for p in PREDICATES if p in cond)
        for effect in rule["Then"]:
            grid[(rule["When"], effect["Do"]) + keyed].append(ring["Name"])


def show_grid():
    print(f"axes: {len(ELEMENTS)} elements, {len(FORMS)} forms, "
          f"{len(CONCEPTS)} concepts, tiers {TIERS}\n")
    for cell in sorted(grid, key=str):
        when, do = cell[0], cell[1]
        cond = " ".join(f"{p}={v}" for p, v in cell[2:]) or "(no If)"
        names = ", ".join(sorted(set(grid[cell])))
        print(f"{when:<14} {do:<20} {cond:<22} {names}")


def show_gaps():
    # For each (When, Do) that is keyed on exactly one axis, which values are missing?
    families = collections.defaultdict(set)
    for cell in grid:
        if len(cell) == 3 and cell[2][0] in AXES:
            families[(cell[0], cell[1], cell[2][0])].add(cell[2][1])
    print("\ngaps -- a filled family with an empty sibling:\n")
    for (when, do, axis), have in sorted(families.items()):
        missing = [v for v in AXES[axis] if v not in have]
        if missing:
            print(f"{when} / {do} keyed on {axis}: "
                  f"has {sorted(have)}, missing {missing}")
    print("\nunused vocabulary:")
    used_when = {c[0] for c in grid}
    used_do = {c[1] for c in grid}
    print(f"  moments in use: {sorted(used_when)}")
    print(f"  verbs in use:   {sorted(used_do)}")
    print("  (compare against the two tables in SKILL.md; a verb with no ring "
          "behind it is a row to question, not to fill)")


if "--gaps" not in sys.argv:
    show_grid()
show_gaps()
