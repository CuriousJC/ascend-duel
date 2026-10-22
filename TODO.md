# TODO

**The work list.** [MECHANICS.md](MECHANICS.md) is what the game *is*; this is what to build.
When the two disagree, `MECHANICS.md` wins — say so rather than guessing. `ideas.md` is the
unfiltered inbox feeding both.

Completed work is not kept here. Git history has it.

Status: `[ ]` open · `[~]` in progress · `[?]` needs a decision

**Nothing is added here unless the owner asks for it to be tracked.**

---

## Steam

- [ ] **Rich presence.** What the friends list shows beside the player — the realm, the room, the
  creature being fought. `ISteamFriends.SetRichPresence` is already in go-steamworks; the strings
  are localization tokens defined on the Steamworks partner site. Set from `internal/steam`, read
  off the run where a station changes, and nothing from a scene directly.
- [ ] **Steam stats.** Lifetime figures on the Steam profile, and progress bars on the count
  achievements ("37/100"). go-steamworks has no `SetStat`/`GetStat`, so this is a small binding of
  our own or an upstream PR. The figures already exist — `profile.Counters` and the run stats — so
  the work is mirroring them from `internal/screens/achieve.go`, the same door achievements use.
