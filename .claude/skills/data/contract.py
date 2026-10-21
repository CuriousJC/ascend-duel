"""The catalog contract, checked against the repo rather than remembered.

    python .claude/skills/data/contract.py            # every catalog, every expectation
    python .claude/skills/data/contract.py --gaps     # only what is missing
    python .claude/skills/data/contract.py relics     # one catalog

Run from the repo root. Every catalog the player meets is held to the same concept shape -- see
"The catalog contract" in SKILL.md -- and this prints, per catalog, which parts of it the repo
actually has, read off the files themselves: the record's fields, the sheet registered in
tools/sheets, the Compendium book, the stat family, the scenario field, the MECHANICS heading.

**A dash is a question, not a finding.** Some catalogs have a reason to go without a part -- a hand
has no picture of its own, a potion is drunk on the spot and never carried -- and that reason
belongs in the `why` column of CATALOGS below, where it stops being reported as missing. Anything
reported as a gap without one is either work or a reason still to write down.

**CATALOGS is configuration, not state.** It says where to look for each part of a catalog; whether
the part is there is read off the repo every run. A new catalog is a row here, exactly as a new
skill is a row in CLAUDE.md's index.
"""
import json
import os
import re
import sys

# Each catalog: the file, the field that keys a record, and where each part of the contract lives.
#   sheet   - the tools/<dir> registered in tools/sheets/main.go
#   book    - the Compendium label in internal/screens/compendium.go
#   stat    - the Stat... constant in internal/session/stats.go
#   scen    - the scenario field in internal/scenario/scenario_on.go
#   mech    - a MECHANICS.md heading that has to exist
#   why     - parts deliberately absent, and the one-line reason
CATALOGS = {
    "relics": dict(file="data/relics.json", key="RelicRecord", sheet="relicsheet", book="RELICS",
                   stat="StatRelicBought", scen="Relics", mech="## Relics"),
    "relic-traits": dict(file="data/relic_traits.json", key="RelicTrait", sheet="traitsheet",
                         book="TRAITS", stat="StatRelicTrait", scen="RelicTraits", mech="### Traits",
                         why={"art": "a trait is drawn on the relic carrying it, never a picture of its own"}),
    "card-traits": dict(file="data/card_traits.json", key="CardTrait", sheet="traitsheet",
                        book="TRAITS", stat="StatCardTrait", scen="Traits", mech="### Traits",
                        why={"art": "a trait is drawn on the card carrying it, never a picture of its own"}),
    "essences": dict(file="data/essences.json", key="EssenceRecord", sheet="essencesheet",
                     book="ESSENCES", stat="StatEssenceUsed", scen="Essences", mech="## Essences"),
    "runes": dict(file="data/runes.json", key="RuneRecord", sheet="runesheet", book="RUNES",
                  stat="StatRuneUsed", scen="Runes", mech="## Runes"),
    "stones": dict(file="data/stones.json", key="StoneRecord", sheet="stonesheet", book="STONES",
                   stat="StatStoneUsed", scen="Stones", mech="## Stones"),
    "cantrips": dict(file="data/cantrips.json", key="CantripRecord", sheet="cantripsheet",
                     book="CANTRIPS", stat="StatCantripCast", scen="Cantrips", mech="## Cantrips"),
    "potions": dict(file="data/potions.json", key="PotionRecord", sheet="potionsheet",
                    book="POTIONS", stat="StatPotionDrunk", scen="Potions", mech="### The shop",
                    why={"scen": "drunk on the spot, so nothing is carried to put on",
                         "words": "the tooltip is computed from Effect and Amount"}),
    "tonics": dict(file="data/tonics.json", key="TonicRecord", sheet="tonicsheet", book="TONICS",
                   stat="StatTonicDrunk", scen="Tonics", mech="## Tonics",
                   why={"words": "the tooltip is computed from Effect and Amount"}),
    "goods": dict(file="data/goods.json", key="GoodRecord", sheet="goodsheet", book="SEALED GOODS",
                  stat="StatGoodBought", scen="Goods", mech="### A sealed good is a screen",
                  why={"scen": "opened in the shop, never carried"}),
    "playing-cards": dict(file="data/duelist_cards.json", key="Label", sheet="cardsheet",
                          book="CARDS", stat="StatCardTaken", scen="Deck", mech="### The card language",
                          why={"art": "the picture is card_art.json's, per card and element",
                               "words": "the card language's verb writes its line (carddesc)",
                               "family": "a concept stands on its form's ladder, which is its family"}),
    "hands": dict(file="data/hands.json", list="hands", key="key", sheet="handsheet", book="HANDS",
                  stat="StatHandFormed", scen="Hands", mech="## Hands",
                  why={"art": "a hand is drawn as real cards, never a picture of its own",
                       "words": "the rung's name is the whole of what it says",
                       "family": "a rung's axis is its family"}),
    "creatures": dict(file="data/motifs", key="Record", sheet="motifsheet", book="CREATURES",
                      stat="StatCreatureDefeated", scen="Enemy", mech="## Enemies",
                      why={"family": "the file it is in is its motif",
                           "words": "a creature says its name and title; its motif carries the Text"}),
    "achievements": dict(file="data/achievements.json", key="APIName", sheet="achievementsheet",
                         book="ACHIEVEMENTS", stat=None, scen=None, mech="## Achievements",
                         why={"stat": "the profile's own unlock list is the tally",
                              "family": "Steamworks' own fields, no family to group by",
                              "scen": "earned, never put on"}),
    "music": dict(file="data/music.json", key="File", sheet=None, book=None,
                  stat=None, scen=None, mech=None,
                  why={"words": "a track says nothing; its pack's Name is its credit line",
                       "art": "heard, never drawn",
                       "family": "its Pack is where it came from, and nothing groups by more",
                       "sheet": "the audio is not in git, so a page could show nothing; tools/privateassets is the report",
                       "book": "heard, not collected",
                       "stat": "nothing about a track is earned or spent",
                       "scen": "a fight's track is rolled off the run seed, which a scenario's Seed already pins",
                       "mech": "presentation, not a rule; privateassets/README.md holds its design"}),
    "sounds": dict(file="data/sounds.json", key="File", sheet=None, book=None,
                   stat=None, scen=None, mech=None,
                   why={"words": "an effect says nothing; its pack's Name is its credit line",
                        "art": "heard, never drawn",
                        "family": "its Pack is where it came from, and its Cues are what it is for",
                        "sheet": "the audio is not in git, so a page could show nothing; tools/privateassets is the report",
                        "book": "heard, not collected",
                        "stat": "nothing about an effect is earned or spent",
                        "scen": "a cue is raised by playback, which a scenario reaches by playing",
                        "mech": "presentation, not a rule; privateassets/README.md holds its design"}),
}

PARTS = ("words", "art", "family", "sheet", "book", "stat", "scen", "mech")
LABELS = {
    "words": "Name+Text", "art": "Art+Draw", "family": "Family", "sheet": "sheet",
    "book": "Compendium", "stat": "stat", "scen": "scenario", "mech": "MECHANICS",
}


def read(path):
    try:
        return open(path, encoding="utf-8").read()
    except OSError:
        return ""


SHEETS = read("tools/sheets/main.go")
BOOKS = read("internal/screens/compendium.go")
STATS = read("internal/session/stats.go")
SCENARIO = read("internal/scenario/scenario_on.go")
MECHANICS = read("MECHANICS.md")


def first_record(c):
    path = c["file"]
    if os.path.isdir(path):
        for root, _, files in sorted(os.walk(path)):
            if "motif.json" in files:
                d = json.load(open(os.path.join(root, "motif.json"), encoding="utf-8"))
                recs = d.get("Records") or d.get("Creatures") or []
                return recs[0] if recs else {}
        return {}
    d = json.load(open(path, encoding="utf-8"))
    if "list" in c:
        d = d[c["list"]]
    return d[0] if d else {}


def check(c):
    rec = first_record(c)
    keys = set(rec)
    name = bool(keys & {"Name", "name", "Label", "DisplayName"})
    text = bool(keys & {"Text", "Tip", "Description"})
    return {
        "words": name and text,
        "art": bool(keys & {"Art", "AchievedIcon"}) and "Draw" in keys,
        "family": "Family" in keys,
        "sheet": f'"./tools/{c["sheet"]}"' in SHEETS,
        "book": f'label: "{c["book"]}"' in BOOKS,
        "stat": bool(c["stat"]) and re.search(rf"\b{c['stat']}\s*=", STATS) is not None,
        "scen": bool(c["scen"]) and f'`json:"{c["scen"]}"`' in SCENARIO,
        "mech": bool(c["mech"]) and c["mech"] in MECHANICS,
    }


def main(argv):
    gaps_only = "--gaps" in argv
    names = [a for a in argv if not a.startswith("--")] or list(CATALOGS)
    width = max(len(n) for n in names)
    if not gaps_only:
        print(" " * width + "  " + "  ".join(f"{LABELS[p]:<10}" for p in PARTS))
    for n in names:
        c = CATALOGS[n]
        got = check(c)
        why = c.get("why", {})
        if gaps_only:
            missing = [p for p in PARTS if not got[p] and p not in why]
            if missing:
                print(f"{n:<{width}}  missing: " + ", ".join(LABELS[p] for p in missing))
            continue
        cells = []
        for p in PARTS:
            mark = "yes" if got[p] else ("n/a" if p in why else "-")
            cells.append(f"{mark:<10}")
        print(f"{n:<{width}}  " + "  ".join(cells))
    if not gaps_only:
        print()
        for n in names:
            for p, reason in CATALOGS[n].get("why", {}).items():
                print(f"  {n} has no {LABELS[p]}: {reason}")


if __name__ == "__main__":
    main(sys.argv[1:])
