# StrikeMan

A small native desktop app (Windows / macOS / Linux) to manage a private
Counter-Strike 2 server over RCON — without typing RCON commands.

Built with [Wails](https://wails.io): a Go backend (self-written Source RCON
client, no heavy dependencies) and a plain HTML/CSS/JS frontend in a native
window. No npm, no build pipeline, ~10 source files.

## Features

- **Live status** — hostname, current map, player list (auto-refresh).
- **Game card** — pick the mode you *want to play*, and the map dropdown
  immediately filters to maps supporting it, regardless of what the server is
  currently running. One button applies the mode and loads the map.
- **Map switcher** — the dropdown is built from the server's own installed map
  list (`maps *`), plus the maps of a Steam Workshop **collection** you
  configure. Workshop maps load via `host_workshop_map`.
- **Server info** — version plus an "is a server update available?" check
  against Steam, and uptime.
- **Warmup** — one button that starts a warmup of exactly the length you set and
  turns into "end warmup · go live" with a countdown. Starting it while players
  are on the server asks first. Applying a preset or changing the map starts the
  same warmup afterwards, so a map change and the button behave identically
  (skipped when nobody is on the server, which is when CS2 skips warmup too).
  The length is remembered per server.
- **Toggles** — friendly fire and overtime follow the preset you apply; the
  admin toggles (auto-kick, kick bans) can be kept across presets and map
  changes per server, marked with a 📌.
- **No accidental team-damage bans** — turning off *Auto-kick* stops kicks for
  team damage/idling (`mp_autokick`, `mp_spawnprotectiontime`), and turning off
  *Kicked players banned 15 min* lets a kicked player rejoin at once
  (`sv_kick_ban_duration`, `sv_vote_kick_ban_duration`). Because loading a map
  re-runs the game mode config and turns these back on, StrikeMan re-applies
  them afterwards when the sticky setting is enabled.
- **Announcements** — broadcast a chat message to everyone on the server.
- **Confirmations** — disruptive actions (map/mode change, swap, scramble,
  going live) ask first while players are on the server; kicking and
  restarting the match always ask.
- **Presets** — one click sets the mode convars, reloads the map and applies
  follow-up settings:
  - *Competitive 5v5* — `game_type 0`, `game_mode 1`, overtime off (12:12 draws)
  - *Premier 5v5* — same ruleset, but overtime on (`mp_overtime_maxrounds 6`,
    no overtime limit) so a 12:12 is played out. Note that Valve's Premier
    also means map veto and CS Rating, which are matchmaking-only and cannot
    exist on a private server — this is the ruleset, not the matchmaking.
  - *Wingman 2v2* — `game_type 0`, `game_mode 2`. MR8 with overtime on, which
    is what the server's own wingman config sets (checked over RCON).
  - *Wingman 3v3* — wingman rules and maps with 3 players per team:
    after the map reload it sets `mp_limitteams 0` and `mp_autoteambalance 0`,
    which lifts the 2-per-team limit (verified against a live CS2 server by
    putting 3+ bots on each team).
- **Match control** — warmup toggle, pause / resume, restart match.
- **Teams** — swap sides, scramble, set team names, add/kick bots,
  kick individual players.
  (Vanilla CS2 has no RCON command to force a specific player onto a team —
  that would need a server plugin such as CounterStrikeSharp.)
- **Console** — free-text RCON console with output, as an escape hatch.

## What CS2 does not expose over RCON

Two things StrikeMan deliberately does not pretend to know, both checked
against a live server rather than assumed:

- **Warmup and pause state.** No convar reports them, so StrikeMan tracks what
  it started itself and shows a countdown. To keep that countdown honest,
  starting a warmup also sets `mp_warmuptime_all_players_connected 0`, which
  would otherwise cut the warmup short once everyone connects — the trade-off
  is that warmup runs its full length even when everyone is ready, and "go
  live" ends it. Because StrikeMan also starts the warmup after its own map
  changes, the countdown is exact there too; only a map change made outside
  StrikeMan falls back to an estimate.
- **Bans.** `banid` and `listid` exist but record nothing usable — a ban made
  through them never appears in `listid`, so there is no ban list to show or
  unban from. The kick-ban players actually run into is a timed lockout
  controlled by `sv_kick_ban_duration`; turn that toggle off instead.

## No live scoreboard

There is deliberately no score display: CS2 exposes no score, round number or
match phase over RCON at all. Verified against a live server —
`mp_teamscore_*` stays at 0, `status`/`status_json` carry no score, and the
server pushes no console events to RCON clients. Showing it would require
Game State Integration or a server plugin, which is more moving parts than
this tool is meant to have.

## Configuration

Set the server host, RCON port, RCON password and (optionally) a workshop
collection ID via the ⚙ settings dialog. The config is stored outside the
repository in your OS user config directory
(e.g. `%AppData%\StrikeMan\config.json` on Windows) with the password kept
local only.

## Building

Requires [Go](https://go.dev) and the Wails v2 CLI
(`go install github.com/wailsapp/wails/v2/cmd/wails@latest`).

```sh
wails dev     # run with hot reload
wails build   # produces build/bin/StrikeMan(.exe/.app)
```

On Linux: `wails build -tags webkit2_41` (needs `libgtk-3-dev`,
`libwebkit2gtk-4.1-dev`).

Tagged pushes (`v*`) build Windows/macOS/Linux binaries via GitHub Actions and
attach them to the release.

## Languages

English and German. StrikeMan follows your operating system's language on
first run; the ⚙ settings dialog has a language selector to override it.
Adding a language means adding one block of keys to `frontend/i18n.js` —
presets, switches and log messages travel from the backend as keys, so they
translate with everything else.

## Development

```sh
go test ./...     # parsing, RCON packets and preset rules
go vet ./...
wails dev         # hot reload
```

CI runs `go vet`, `gofmt -l` and `go test -race` on every push. The race
detector needs cgo, which a bare Windows checkout usually lacks — that is why
it runs on Linux in CI, and it matters here because the status poll and the
map-load goroutine touch the same state.

## Code layout

| File | Purpose |
| --- | --- |
| `main.go` | Wails app entry, embeds `frontend/`, fits the window to the screen |
| `app.go` | App state and lifecycle, config/server bindings, server info |
| `status.go` | Parsing `status`, reading convars, sticky admin toggles |
| `game.go` | Maps, presets, warmup, match and team commands |
| `rcon.go` | Minimal Source RCON protocol client |
| `presets.go` | Presets, switches and timings as data |
| `steam.go` | Workshop lookup + server update check (no API key needed) |
| `config.go` | Config load/save (passwords go to the OS credential store) |
| `frontend/app.js` | Wiring: controls to backend, polling loop |
| `frontend/state.js` | State and backend calls, no DOM |
| `frontend/views.js` | Rendering and UI primitives |
| `frontend/i18n.js` | Translations (EN/DE) |

The frontend uses native ES modules — there is still no npm, bundler or build
step for it.

## License

MIT
