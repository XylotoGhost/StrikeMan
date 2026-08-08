# StrikeMan

A small native desktop app (Windows / macOS / Linux) to manage a private
Counter-Strike 2 server over RCON — without typing RCON commands.

Built with [Wails](https://wails.io): a Go backend (self-written Source RCON
client, no heavy dependencies) and a plain HTML/CSS/JS frontend in a native
window. No npm, no build pipeline, ~10 source files.

## Features

- **Live status** — hostname, current map, player list (auto-refresh).
- **Map switcher** — the dropdown is built from the server's own installed map
  list (`maps *`), plus the maps of a Steam Workshop **collection** you
  configure. Workshop maps load via `host_workshop_map`.
- **Presets** — one click sets the mode convars, reloads the map and applies
  follow-up settings:
  - *Competitive 5v5* — `game_type 0`, `game_mode 1`
  - *Wingman 2v2* — `game_type 0`, `game_mode 2`
  - *Wingman 3v3* — wingman rules and maps with 3 players per team:
    after the map reload it sets `mp_limitteams 0` and `mp_autoteambalance 0`,
    which lifts the 2-per-team limit (verified against a live CS2 server by
    putting 3+ bots on each team).
- **Match control** — warmup timer (start/end), pause / unpause, restart round.
- **Teams** — swap sides, scramble, set team names, add/kick bots,
  kick individual players.
  (Vanilla CS2 has no RCON command to force a specific player onto a team —
  that would need a server plugin such as CounterStrikeSharp.)
- **Console** — free-text RCON console with output, as an escape hatch.

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

## Code layout

| File | Purpose |
| --- | --- |
| `main.go` | Wails app entry, embeds `frontend/` |
| `app.go` | All backend methods callable from the UI |
| `rcon.go` | Minimal Source RCON protocol client |
| `presets.go` | Game mode presets as data |
| `steam.go` | Workshop collection lookup (no API key needed) |
| `config.go` | Config load/save |
| `frontend/` | The UI: one HTML page, one stylesheet, one script |

## License

MIT
