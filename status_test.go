package main

import (
	"strings"
	"testing"
)

// Captured from the live CS2 server, including the 65535 filler rows the
// server prints for free slots and the appended convar batch.
const sampleStatus = `Server:  Running [172.17.0.13:27016]
Client:  Disconnected
----- Status -----
@ Current  :  game
source   : console
hostname : Mylo Gaming | CS2 Server
spawn    : 12
version  : 1.41.7.4/14174 10847 secure  public
steamid  : [G:1:13064182] (85568392933103606)
udp/ip   : 172.17.0.13:27016 (public 85.216.99.177:27016)
os/type  : Linux dedicated
players  : 1 humans, 2 bots (0 max) (not hibernating) (unreserved)
---------spawngroups----
loaded spawngroup(  1)  : SV:  [1: de_nuke | main lump | mapload]
loaded spawngroup(  2)  : SV:  [2: prefabs/misc/team_select | main lump | mapload | point_prefab]
---------players--------
  id     time ping loss      state   rate adr name
65535 [NoChan]    0    0 challenging      0unknown ''
   0      BOT    0    0     active      0 'Squad'
   2    00:43   24    0     active 786432 85.216.99.177:62135 'XylotoGhost'
#end
game_type = 0
game_mode = 2
mp_limitteams = 0
mp_maxrounds = 16
mp_warmuptime = 120
mp_warmup_online_enabled = true
mp_friendlyfire = true
mp_overtime_enable = false
mp_autokick = false
sv_kick_ban_duration = 0`

func TestParseStatus(t *testing.T) {
	st := parseStatus(sampleStatus)

	if st.Hostname != "Mylo Gaming | CS2 Server" {
		t.Errorf("hostname = %q", st.Hostname)
	}
	if st.Map != "de_nuke" {
		t.Errorf("map = %q, want de_nuke", st.Map)
	}
	if st.Humans != 1 || st.Bots != 2 {
		t.Errorf("humans/bots = %d/%d, want 1/2", st.Humans, st.Bots)
	}
	if st.GameType != 0 || st.GameMode != 2 {
		t.Errorf("game_type/game_mode = %d/%d, want 0/2", st.GameType, st.GameMode)
	}
	if st.MaxRounds != 16 {
		t.Errorf("maxRounds = %d, want 16 (wingman is MR8)", st.MaxRounds)
	}
	if st.LimitTeams != 0 {
		t.Errorf("limitTeams = %d, want 0", st.LimitTeams)
	}
	if st.WarmupTime != 120 || st.WarmupOnline != 1 {
		t.Errorf("warmup = %d/%d, want 120/1", st.WarmupTime, st.WarmupOnline)
	}
}

func TestParseStatusPlayers(t *testing.T) {
	st := parseStatus(sampleStatus)

	if len(st.Players) != 2 {
		t.Fatalf("got %d players, want 2 (the 65535 slot row must be ignored)", len(st.Players))
	}
	bot, human := st.Players[0], st.Players[1]
	if bot.Name != "Squad" || !bot.Bot {
		t.Errorf("first player = %+v, want bot Squad", bot)
	}
	if human.Name != "XylotoGhost" || human.Bot {
		t.Errorf("second player = %+v, want human XylotoGhost", human)
	}
	if human.UserID != "2" || human.Ping != "24" {
		t.Errorf("human userID/ping = %s/%s, want 2/24", human.UserID, human.Ping)
	}
}

func TestParseBuild(t *testing.T) {
	version, build := parseBuild(sampleStatus)
	if version != "1.41.7.4" || build != 14174 {
		t.Errorf("parseBuild = %q/%d, want 1.41.7.4/14174", version, build)
	}
	if _, build := parseBuild("no version here"); build != 0 {
		t.Errorf("missing version should give build 0, got %d", build)
	}
}

func TestReadToggle(t *testing.T) {
	cases := []struct {
		out, cvar string
		want      int
	}{
		{"mp_friendlyfire = true", "mp_friendlyfire", 1},
		{"mp_overtime_enable = false", "mp_overtime_enable", 0},
		// sv_kick_ban_duration is a duration in minutes: any value above
		// zero means kicked players are banned.
		{"sv_kick_ban_duration = 15", "sv_kick_ban_duration", 1},
		{"sv_kick_ban_duration = 0", "sv_kick_ban_duration", 0},
		{"something else entirely", "mp_autokick", -1},
	}
	for _, c := range cases {
		if got := readToggle(c.out, c.cvar); got != c.want {
			t.Errorf("readToggle(%q, %q) = %d, want %d", c.out, c.cvar, got, c.want)
		}
	}
}

func TestStatusTogglesCoverEveryToggle(t *testing.T) {
	st := parseStatus(sampleStatus)
	for _, toggle := range Toggles {
		if _, ok := st.Toggles[toggle.ID]; !ok {
			t.Errorf("toggle %q missing from parsed status", toggle.ID)
		}
	}
	if st.Toggles["autokick"] != 0 {
		t.Errorf("autokick = %d, want 0 (off)", st.Toggles["autokick"])
	}
	if st.Toggles["friendlyfire"] != 1 {
		t.Errorf("friendlyfire = %d, want 1 (on)", st.Toggles["friendlyfire"])
	}
}

// Parsing runs from both the status poll and the map-load goroutine. It used
// to fill a shared regex cache on demand, and Go aborts the whole process on a
// concurrent map write. Run with -race (and without) to keep it honest.
func TestParseStatusIsConcurrencySafe(t *testing.T) {
	const goroutines = 8
	done := make(chan struct{})
	for i := 0; i < goroutines; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for n := 0; n < 200; n++ {
				st := parseStatus(sampleStatus)
				if st.Map != "de_nuke" {
					t.Errorf("map = %q", st.Map)
					return
				}
				readToggle(sampleStatus, "sv_kick_ban_duration")
				// A convar with no precompiled pattern takes the fallback
				// path, which must not touch shared state either.
				readToggle(sampleStatus, "mp_some_unknown_convar")
			}
		}()
	}
	for i := 0; i < goroutines; i++ {
		<-done
	}
}

// The convar batch must ask for everything the parser then reads back.
func TestStatusConvarsCoverToggles(t *testing.T) {
	batch := statusConvars()
	for _, toggle := range Toggles {
		if !strings.Contains(batch, toggle.Cvar) {
			t.Errorf("statusConvars() does not query %q", toggle.Cvar)
		}
	}
	if !strings.Contains(batch, warmupOnlineCvar) {
		t.Errorf("statusConvars() does not query %q", warmupOnlineCvar)
	}
}
