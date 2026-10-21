"""The unlock ledger: which achievements open which relics, and what is still unpaired on either side.

Run from the repository root:

    python .claude/skills/unlocks/unlocks.py            # the whole ledger
    python .claude/skills/unlocks/unlocks.py --unpaired # only the two lists still to match

Reads data/achievements.json and data/relics.json and nothing else. Prints; writes nothing.
"""

import json
import sys
from collections import defaultdict

RARITY_ORDER = ["common", "uncommon", "rare", "mythic"]


def load(path):
    with open(path, encoding="utf-8") as f:
        return json.load(f)


def main():
    unpaired_only = "--unpaired" in sys.argv[1:]
    achievements = load("data/achievements.json")
    relics = load("data/relics.json")

    granted_by = defaultdict(list)  # unlock key -> achievement keys
    for a in achievements:
        for key in a.get("Unlocks") or []:
            granted_by[key].append(a["APIName"])

    gated = defaultdict(list)  # unlock key -> relic records
    for r in relics:
        if r.get("Unlock"):
            gated[r["Unlock"]].append(r)

    if not unpaired_only:
        print("PAIRED: unlock key -> achievements that grant it -> relics behind it")
        for key in sorted(set(granted_by) | set(gated)):
            by = ", ".join(granted_by.get(key, [])) or "(NOTHING GRANTS IT - fails the launch)"
            behind = gated.get(key, [])
            print(f"\n  {key}")
            print(f"    granted by: {by}")
            if not behind:
                print("    opens:      (NOTHING READS IT - fails the launch)")
            for r in behind:
                print(f"    opens:      {r['RelicRecord']:<28} {r.get('Rarity', ''):<9} {r.get('Family', '')}")
        print()

    lone = [a for a in achievements if not a.get("Unlocks")]
    print(f"ACHIEVEMENTS THAT OPEN NOTHING ({len(lone)} of {len(achievements)})")
    for a in lone:
        trigger = a.get("Trigger", {})
        what = trigger.get("Moment") or trigger.get("Counter") or trigger.get("Kind", "")
        extra = trigger.get("Value") or trigger.get("N") or ""
        print(f"  {a['APIName']:<24} {what} {extra}".rstrip() + f"  - {a['Description']}")

    pool = [r for r in relics if not r.get("Unlock")]
    print(f"\nRELICS IN THE DEFAULT POOL ({len(pool)} of {len(relics)}), by family in file order")
    families = defaultdict(list)
    order = []
    for r in pool:
        fam = r.get("Family", "")
        if fam not in families:
            order.append(fam)
        families[fam].append(r)
    for fam in order:
        rs = families[fam]
        mix = defaultdict(int)
        for r in rs:
            mix[r.get("Rarity", "")] += 1
        shape = ", ".join(f"{mix[t]} {t}" for t in RARITY_ORDER if mix[t])
        print(f"\n  {fam or '(no family)'} - {shape}")
        for r in rs:
            print(f"    {r['RelicRecord']:<28} {r.get('Rarity', ''):<9} {r.get('Name', '')}")


if __name__ == "__main__":
    main()
