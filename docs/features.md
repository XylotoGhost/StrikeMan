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
game updates. Wingman-only maps (`de_brewery`, `de_dogtown`) are hidden when
you pick a competitive mode. Workshop maps come from a collection ID you set in
settings and load through `host_workshop_map`; if a workshop map's Steam tags
say it is not a wingman map, StrikeMan refuses before touching the server
rather than letting it silently fall back to competitive.

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

## Multiple servers and languages

Servers are configured in ⚙ with one marked as the default; a switcher appears
in the header once you have more than one. RCON passwords live in the
operating system's credential store (Windows Credential Manager, macOS
Keychain, Linux Secret Service), not in the config file.

The interface follows your operating system's language, with English and
German available and an override in settings.
