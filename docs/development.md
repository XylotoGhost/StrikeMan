# Development

Go backend, plain HTML/CSS/JS frontend, [Wails](https://wails.io) to put them
in a native window. No npm, no bundler, no frontend build step — the UI uses
native ES modules.

## Building

Requires [Go](https://go.dev) and the Wails v2 CLI:

```sh
go install github.com/wailsapp/wails/v2/cmd/wails@latest

wails dev     # hot reload
wails build   # -> build/bin/StrikeMan(.exe/.app)
```

On Linux: `wails build -tags webkit2_41` (needs `libgtk-3-dev` and
`libwebkit2gtk-4.1-dev`).

Tagged pushes (`v*`) build Windows, macOS and Linux binaries in GitHub Actions
and attach them to the release, so a macOS build does not require a Mac. Each
release carries both an installer (NSIS on Windows, `.dmg` on macOS) and a
portable archive, plus a `.sha256` for every file.

The tag becomes the version the app reports, stamped in with
`-ldflags "-X main.version=$TAG"`. A build without that stamp reports `dev` and
is never offered an update, so a local build cannot overwrite itself.

The Windows installer is configured for a per-user install
(`build/windows/installer/project.nsi` sets `WAILS_INSTALL_SCOPE "user"` and
`REQUEST_EXECUTION_LEVEL "user"`), which is what keeps both installation and
self-update free of UAC prompts.

## Tests

```sh
go vet ./...
go test ./...
```

The unit tests cover the parts that break quietly: parsing real captured
`status` output, reading convars back (including numeric ones such as
`sv_kick_ban_duration`), RCON packet encoding, the wrong-password path, and
the preset rules — for instance that the wingman presets never re-acquire
overtime.

CI additionally runs `gofmt -l` and `go test -race`. The race detector needs
cgo, which a bare Windows checkout usually lacks, so that is where the
concurrency guarantees are actually checked — it matters because the status
poll and the map-load goroutine touch the same state.

### Live tests

`integration_test.go` drives the real thing against a running server: applying
a preset, sticky toggles surviving a map load, the warmup that follows, and bot
counts. It changes the map and match settings, so it only runs when pointed at
a server deliberately:

```sh
STRIKEMAN_TEST_HOST=192.168.178.66 \
STRIKEMAN_TEST_PORT=27016 \
STRIKEMAN_TEST_PASSWORD=... \
go test -run TestLive -v
```

It skips everywhere else, including CI. The tests first push the server into a
*wrong* state — overtime on, auto-kick on, CS2's warmup shortcut restored — so
the assertions cannot pass on state that happened to be correct already.

## Layout

| File | Purpose |
| --- | --- |
| `main.go` | Wails entry, embeds `frontend/`, fits the window to the screen |
| `app.go` | App state and lifecycle, config/server bindings, server info |
| `status.go` | Parsing `status`, reading convars, sticky admin toggles |
| `game.go` | Maps, presets, warmup, match, teams and bots |
| `rcon.go` | Source RCON protocol client |
| `presets.go` | Presets, switches and timings as data |
| `steam.go` | Workshop lookup and server update check (no API key needed) |
| `update.go` | Self-update: GitHub releases, checksum, binary swap, restart |
| `config.go` | Config load/save; passwords go to the OS credential store |
| `frontend/app.js` | Wiring: controls to backend, polling loop |
| `frontend/state.js` | State and backend calls, no DOM |
| `frontend/views.js` | Rendering and UI primitives |
| `frontend/i18n.js` | Translations |

Every exported method on `App` is callable from the frontend as
`window.go.main.App.<Method>`.

### Concurrency

Wails calls the bound methods from its own goroutines, and `runAfterMapLoad`
runs in one of ours, so every field of `App` is guarded by a mutex. The lock is
never held across an RCON round trip or an HTTP request: read state under the
lock, do the I/O, write results back under the lock.

## Adding a language

Add a block of keys to `frontend/i18n.js` and an entry to `languages`.
Presets, switches, log lines and error messages all travel from the backend as
keys, so nothing stays half-translated. `missingKeys("de")` lists anything not
yet translated. Errors cross the Wails boundary as strings, so translatable
ones are tagged `i18n:key` with unit-separator arguments — see `tErr` in
`app.go` and `translateError` in `i18n.js`.
