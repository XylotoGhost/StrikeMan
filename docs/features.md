# Features in detail

## Game: mode and map together

Pick the mode you *intend* to play first — the map list immediately narrows to
maps that support it, whatever the server is running at the moment. That means
wingman maps are reachable while a competitive match is still up, which is the
whole point. The mode the server is actually running is marked **● running**,
so intent and reality never get confused.

"Apply mode & load map" sets the mode and reloads the map; "Change map only"
leaves the mode alone.

### Presets

| Preset | Rules |
| --- | --- |
| Competitive 5v5 | MR12, a 12:12 ends in a draw |
| Premier 5v5 | MR12, but a 12:12 is played out in overtime (MR3, no limit) |
| Wingman 2v2 | MR8 on wingman maps, an 8:8 ends in a draw |
| Wingman 3v3 | Wingman maps and rules with 3 players per team |

Each preset states its own match rules rather than trusting whatever the
previous mode left behind, so applying one always lands on a known state.

**Wingman 3v3** is the reason this app exists: after the map loads it sets
`mp_limitteams 0` and `mp_autoteambalance 0`, which lifts the two-per-team cap
that normally makes wingman a 2v2. Verified on a live server by putting three
bots on each side.

**Premier** here means the ruleset, not the matchmaking. Valve's Premier also
involves map veto and CS Rating, which are matchmaking-only and cannot exist on
a private server. The one server-side difference is overtime.

### Maps

Official maps come from the server's own `maps *` listing, so the list follows
game updates. Workshop maps come from a collection ID you set in settings and
load through `host_workshop_map`.

Which of them play which mode is **learned from the server**, not kept in a
list that goes stale every time CS2 ships a map:

- **Plays this mode** — the server has actually run the map in it, either
  because you applied a preset that stuck, or simply because it was running
  that way when StrikeMan looked.
- **Not tried yet** — nobody knows. Offered anyway: hiding a map that plays
  fine is the worse mistake, and trying it is what settles the question.
- Hidden — the server has been asked for this mode on that map and quietly
  fell back to another, so it cannot play it.

The lists therefore get more accurate the more you play, and a map added by a
game update needs no update to StrikeMan. What it has worked out is kept in
`mapModes` in the config file. Workshop tags are used only while nothing has
been observed — the uploader writes them, so a map tagged "Wingman" that turns
out not to play it is dropped after the first attempt.

## Match

**Warmup** is one button: it starts a warmup of exactly the length you set and
turns into "end warmup · go live" with a countdown. Applying a preset or
changing the map starts the same warmup afterwards, so the countdown means the
same thing everywhere. It is skipped on an empty server, which is when CS2
skips warmup too. The length is remembered per server.

**Pause / resume** and **restart match** do what they say. Restarting always
asks first, because it resets the score.

### Switches

Friendly fire and overtime are match rules: they follow whichever preset you
apply.

Auto-kick and kick bans are admin choices and can be *kept* per server. A map
load re-runs the game mode config and turns them back on, so StrikeMan
re-applies your choice afterwards and marks the switch with 📌. Turn that off
per server in settings if you would rather the server decide.

- **Auto-kick (idle / team damage)** — `mp_autokick` plus
  `mp_spawnprotectiontime`. Off means nobody is kicked for team damage.
- **Kicked players banned 15 min** — `sv_kick_ban_duration` and
  `sv_vote_kick_ban_duration`. Off lets a kicked player rejoin at once.

Together these are the fix for someone getting locked out for a quarter of an
hour after some joke team damage.

## Teams and players

Swap sides, scramble, set team names, kick a player, and add or remove bots one
at a time. Bot counts are driven by `bot_quota` — see
[CS2 and RCON notes](cs2-notes.md#bots) for why that matters and why bots
cannot be added to a chosen side.

Vanilla CS2 has no RCON command to force a specific player onto a team, so
team assignment is swap, scramble, kick and names. Anything more would need a
server plugin such as CounterStrikeSharp.

## Confirmations

Anything disruptive asks first *while players are on the server*: applying a
mode, changing the map, going live, swapping sides, scrambling. Kicking a
player and restarting the match always ask, empty server or not.

## Server card

Version and build, uptime, and an update check against Steam's own
"is this build current" endpoint — so an outdated server is visible before
match night rather than during it.

## Servers

### First launch

With nothing configured yet there is nothing to manage, so StrikeMan opens on a
setup screen instead of an empty app: pick a language, type in the server, and
press **Test connection** before committing to it. The test only runs `status`,
which reads, so pressing it during a live match is harmless. **Import from a
file…** fills the same form from a server file (below).

### Switching and adding

The server the whole window acts on is picked from the dropdown at the top of
the app, with **+** next to it to add another. Everything that is not "which
server" lives behind **⚙ Settings**: the server list, per-server details, and
the language.

One server is marked as the default (★) and is the one selected at start-up.

RCON passwords live in the operating system's credential store (Windows
Credential Manager, macOS Keychain, Linux Secret Service), not in the config
file.

### Import and export

**Export…** writes the selected server to a small JSON file, and **Import…**
reads one back. It carries the name, address, port, workshop collection, warmup
length and admin-toggle preferences — **never the password**, which the person
importing types in themselves. That is what makes the file safe to send to
whoever else is administering the server.

```json
{
  "format": 1,
  "app": "StrikeMan",
  "exported": "2026-08-09T18:12:00Z",
  "servers": [
    {
      "name": "Mylo Gaming | CS2 Server",
      "host": "192.168.178.66",
      "port": 27016,
      "collectionId": "3070284539"
    }
  ]
}
```

The password is not omitted at write time — the exported type has no field for
one, so it cannot leak even if the stored settings grow. A file may hold several
servers, and StrikeMan's own `config.json` imports too (its passwords are
dropped on the way in). Names that already exist are suffixed rather than
overwritten, so an import never destroys a configured server.

## Languages

The interface follows your operating system's language, with English and
German available and an override in settings.
