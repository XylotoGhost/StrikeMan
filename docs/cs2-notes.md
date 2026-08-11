# CS2 and RCON notes

Findings from probing a live CS2 dedicated server (build 1.41.7.4/14174).
They explain why parts of StrikeMan work the way they do, and they are worth
knowing before assuming a feature is simply missing.

## Bots

`bot_quota` is the only precise control, and in `bot_quota_mode normal` it is
not a maximum you may add up to — it is **the number of bots the server keeps**:

- Setting it to 4 produces exactly 4 bots, setting it to 2 drops back to 2, and
  0 clears them and keeps them away.
- It is also a hard cap: with the quota full, `bot_add` does nothing at all.

`bot_add_ct` / `bot_add_t` do **not** add one bot. Measured from a clean state
of one bot, a single `bot_add_ct` took the server to **three** bots and
rewrote `bot_quota` to match — it fills out a whole team layout (one human plus
three bots is a full 2v2). That is what made a single "+ Bot" click fill the
server to 4v4.

So StrikeMan sets the count directly and lets the server place the bots. The
cost is that CS2 offers no reliable way to add one bot to a chosen side in this
mode, which is why there is a single "+ Bot" rather than per-team buttons.

Two consequences worth remembering:

- Kicking bots without zeroing the quota just makes the server refill them.
  "Kick bots" sets `bot_quota 0` first.
- Kicking a single bot from the player list lowers the quota by one, otherwise
  the slot is refilled within seconds.

## No live scoreboard

CS2 exposes no score, round number or match phase over RCON. Verified:
`mp_teamscore_*` stays at 0 even after forcing round ends, `status` and
`status_json` carry no score, and the server pushes no console events to RCON
clients. A real scoreboard would need Game State Integration (the server POSTing
match state to a listener) or a plugin — more moving parts than this tool is
meant to have, so it is deliberately absent.

## No warmup or pause state

Nothing reports whether a warmup or a pause is running; `mp_warmup_pausetimer`
is a setting, not a state. StrikeMan therefore tracks what it started itself.

To keep the countdown honest, starting a warmup also sets
`mp_warmuptime_all_players_connected 0`. Left at its default of 15, CS2 cuts
the warmup short once everyone has connected and the displayed countdown would
be a lie. The trade-off is that warmup now runs its full length even when
everyone is ready — "go live" ends it.

## Bans cannot be listed or lifted

`banid`, `listid` and `removeid` all exist and execute without error, but they
record nothing usable: a ban made through `banid` never appears in `listid`,
in any SteamID format, including for a connected bot. There is no ban list to
show and nothing to unban from.

What players actually hit after an auto-kick is a timed lockout controlled by
`sv_kick_ban_duration` (15 minutes by default) and `sv_vote_kick_ban_duration`.
Turning the "kicked players banned 15 min" switch off is the real fix.

## Which maps play which mode

There is no way to ask the server which maps support wingman. `find wingman`
returns nothing, and the only map-group commands are `mapgroup` and
`print_mapgroup_sv`, which report whatever group the server is configured with
— on a workshop server, the workshop collection. Reading Valve's pool would
mean *changing* the server's map group first, so it is not something a manager
tool should do behind your back.

StrikeMan used to carry a hand-written list instead, and it went stale exactly
as you would expect: `de_poseidon` and `de_eldorado` play wingman, were missing
from it, and were therefore hidden from the wingman map list *and* refused if
picked — while the server was happily running wingman on one of them.

So it asks the server the only way that actually works, by watching:

- the server is running `game_mode 2` on a map and has not fallen back, so that
  map plays wingman;
- a preset asked for a mode and `game_mode` came back as something else, so
  that map does not play it.

Both are conclusive, and the second is free — the mode was already verified
after every preset. What is learned is stored in `mapModes` in the config file
and gets more accurate the more you play. Anything not yet seen stays *unknown*
and is offered rather than hidden, because hiding a map that plays fine is the
worse mistake and trying it is what settles the question.

Two maps cannot be learned by watching, because they would be offered for
competitive until somebody tried them: `de_brewery` and `de_dogtown` load only
the wingman intro prefabs in `status`, with no team select, making them
wingman-only. They are seeded, and any observation overrides the seed.

Workshop tags are a weaker signal in the same space — the uploader writes them,
so a map tagged "Wingman" may not play it. They are used only while nothing has
been observed, and lose to the first observation.

The wingman game mode config also enables overtime, while standard wingman ends
8:8 as a draw, so both wingman presets turn it back off explicitly.

## Modes and rounds

| Mode | `game_type` / `game_mode` | `mp_maxrounds` |
| --- | --- | --- |
| Competitive | 0 / 1 | 24 (MR12) |
| Wingman | 0 / 2 | 16 (MR8) |

A map load re-runs the matching game mode config, which resets convars such as
`mp_autokick` — the reason sticky admin toggles exist. Setting `game_mode` and
changing map in one go occasionally does not take on the first attempt; the
mode is verified after the map loads and a warning is shown if the server fell
back.

## Protocol quirks

- CS2 answers RCON asynchronously and the classic empty-`RESPONSE_VALUE` end
  marker does not work, so the client reads until a short quiet period.
- Responses **do** echo the id of the command that caused them (verified: a
  `status` sent as id 2 comes back as id 2, `maps *` as id 3). That is what
  makes reading-until-quiet safe: output still trickling in from an earlier
  command carries the earlier id and is dropped, instead of being returned as
  the current command's answer. Without that check, one slow reply silently
  became another command's result.
- Waiting for an answer to *start* and reading a packet that has already
  started must be bounded separately. A packet whose header is in the socket is
  on its way, so tying its body to the short between-packets wait truncates
  long answers on a slow link — with no error, since a timeout looks exactly
  like "the response ended".
- A command that produces no answer at all must be an error, not an empty
  string. `maps *` takes far longer than a status poll, and over the internet
  (≈420 ms per round trip to this server, against ≈40 ms on the LAN) a 2 s
  budget was not the comfortable margin it looked like on a local network. An
  empty string was indistinguishable from an empty map list, which is how the
  map dropdown came up blank with nothing reported anywhere.
- A wrong password makes the server close the connection rather than reply with
  id -1, which surfaced as a bare "EOF" until it was translated into a real
  message.
- `find <substring>` lists convars and commands, but it does **not** index the
  ban commands — test those by executing them, since unknown commands answer
  `Unknown command 'x'!`.
- Semicolons work: `status; game_mode; mp_maxrounds` returns in one round trip,
  which is how the status poll stays cheap.
- With no humans connected the server hibernates and rounds do not simulate, so
  bot-only matches never progress.
