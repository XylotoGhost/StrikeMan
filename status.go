package main

// Reading the server's state: parsing `status` output and the convars that
// back the UI switches, plus keeping sticky admin toggles applied.

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Player struct {
	UserID string `json:"userId"`
	Name   string `json:"name"`
	Ping   string `json:"ping"`
	Bot    bool   `json:"bot"`
}

type Status struct {
	Connected  bool   `json:"connected"`
	Error      string `json:"error"`
	Hostname   string `json:"hostname"`
	Map        string `json:"map"`
	Humans     int    `json:"humans"`
	Bots       int    `json:"bots"`
	GameType   int    `json:"gameType"`
	GameMode   int    `json:"gameMode"`
	LimitTeams int    `json:"limitTeams"`
	MaxRounds  int    `json:"maxRounds"`
	Version    string `json:"version"`
	// Toggle states by toggle ID: -1 unknown, 0 off, 1 on.
	Toggles      map[string]int `json:"toggles"`
	WarmupTime   int            `json:"warmupTime"`
	WarmupOnline int            `json:"warmupOnline"`
	Players      []Player       `json:"players"`
}

var (
	reHostname   = regexp.MustCompile(`(?m)^hostname\s*:\s*(.+)$`)
	reMap        = regexp.MustCompile(`loaded spawngroup\(\s*1\)\s*:\s*SV:\s*\[1:\s*([^|\]]+?)\s*\|`)
	rePlayerLine = regexp.MustCompile(`^\s*(\d+)\s+(\S+)\s+(\d+)\s+\d+\s+(\w+)\s+\d+\s*(\S*)\s+'(.+)'\s*$`)
	rePlayerCnt  = regexp.MustCompile(`(?m)^players\s*:\s*(\d+) humans, (\d+) bots`)
	reGameType   = regexp.MustCompile(`game_type = (\d+)`)
	reGameMode   = regexp.MustCompile(`game_mode = (\d+)`)
	reLimitTeams = regexp.MustCompile(`mp_limitteams = (\d+)`)
	reMaxRounds  = regexp.MustCompile(`mp_maxrounds = (\d+)`)
	reWarmupTime = regexp.MustCompile(`mp_warmuptime = (\d+)`)
	reVersion    = regexp.MustCompile(`(?m)^version\s*:\s*(\S+?)/(\d+)`)
)

const warmupOnlineCvar = "mp_warmup_online_enabled"

// cvarRe holds one regex per convar we read back. Built once at start-up:
// compiling them on demand into a shared map raced between the status poll
// and the map-load goroutine, and Go aborts the process on a concurrent map
// write.
var cvarRe = func() map[string]*regexp.Regexp {
	m := map[string]*regexp.Regexp{}
	add := func(cvar string) {
		m[cvar] = regexp.MustCompile(regexp.QuoteMeta(cvar) + ` = (\S+)`)
	}
	for _, t := range Toggles {
		add(t.Cvar)
	}
	add(warmupOnlineCvar)
	return m
}()

// statusConvars is the batch queried alongside `status` on every poll.
func statusConvars() string {
	parts := []string{"game_type", "game_mode", "mp_limitteams", "mp_maxrounds",
		"mp_warmuptime", warmupOnlineCvar}
	for _, t := range Toggles {
		parts = append(parts, t.Cvar)
	}
	return strings.Join(parts, "; ")
}

// readToggle finds "<cvar> = value" in RCON output. Booleans and numbers both
// count: sv_kick_ban_duration is minutes, where anything above 0 means "on".
// Returns -1 when the convar isn't in the output.
func readToggle(out, cvar string) int {
	re, ok := cvarRe[cvar]
	if !ok {
		re = regexp.MustCompile(regexp.QuoteMeta(cvar) + ` = (\S+)`)
	}
	m := re.FindStringSubmatch(out)
	if m == nil {
		return -1
	}
	switch strings.ToLower(m[1]) {
	case "true":
		return 1
	case "false":
		return 0
	}
	if n, err := strconv.Atoi(m[1]); err == nil {
		if n > 0 {
			return 1
		}
		return 0
	}
	return -1
}

// parseStatus turns the raw output of `status` (plus the convar batch) into a
// Status. Split out from GetStatus so it can be tested against captured
// server output without a network.
func parseStatus(out string) Status {
	st := Status{
		Connected:  true,
		GameType:   -1,
		GameMode:   -1,
		LimitTeams: -1,
		Players:    []Player{},
		Toggles:    map[string]int{},
	}
	if m := reHostname.FindStringSubmatch(out); m != nil {
		st.Hostname = strings.TrimSpace(m[1])
	}
	if m := reMap.FindStringSubmatch(out); m != nil {
		st.Map = strings.TrimSpace(m[1])
	}
	if m := rePlayerCnt.FindStringSubmatch(out); m != nil {
		st.Humans, _ = strconv.Atoi(m[1])
		st.Bots, _ = strconv.Atoi(m[2])
	}
	if m := reGameType.FindStringSubmatch(out); m != nil {
		st.GameType, _ = strconv.Atoi(m[1])
	}
	if m := reGameMode.FindStringSubmatch(out); m != nil {
		st.GameMode, _ = strconv.Atoi(m[1])
	}
	if m := reLimitTeams.FindStringSubmatch(out); m != nil {
		st.LimitTeams, _ = strconv.Atoi(m[1])
	}
	if m := reMaxRounds.FindStringSubmatch(out); m != nil {
		st.MaxRounds, _ = strconv.Atoi(m[1])
	}
	if m := reWarmupTime.FindStringSubmatch(out); m != nil {
		st.WarmupTime, _ = strconv.Atoi(m[1])
	}
	st.WarmupOnline = readToggle(out, warmupOnlineCvar)
	for _, t := range Toggles {
		st.Toggles[t.ID] = readToggle(out, t.Cvar)
	}
	for _, line := range strings.Split(out, "\n") {
		m := rePlayerLine.FindStringSubmatch(line)
		// 65535 rows are connection slots the server prints while empty.
		if m == nil || m[1] == "65535" || m[4] != "active" {
			continue
		}
		st.Players = append(st.Players, Player{
			UserID: m[1],
			Name:   m[6],
			Ping:   m[3],
			Bot:    m[2] == "BOT" || m[5] == "BOT",
		})
	}
	return st
}

// parseBuild extracts the build number from the `status` version line, e.g.
// "version : 1.41.7.4/14174 10847 secure" -> "1.41.7.4", 14174.
func parseBuild(out string) (string, int) {
	m := reVersion.FindStringSubmatch(out)
	if m == nil {
		return "", 0
	}
	build, _ := strconv.Atoi(m[2])
	return m[1], build
}

// GetStatus polls `status` plus the convar batch. Batching them into one round
// trip is attempted once; if the server doesn't split on semicolons, the
// convars are queried separately from then on.
func (a *App) GetStatus() Status {
	convars := statusConvars()

	a.mu.Lock()
	batch := a.canBatch == nil || *a.canBatch
	firstTry := a.canBatch == nil
	a.mu.Unlock()

	cmd := "status"
	if batch {
		cmd = "status; " + convars
	}
	out, err := a.exec(cmd)
	if err != nil {
		return Status{Error: err.Error(), Players: []Player{}, Toggles: map[string]int{}}
	}

	if firstTry {
		works := reGameMode.MatchString(out)
		a.mu.Lock()
		a.canBatch = &works
		a.mu.Unlock()
		batch = works
	}
	if !batch {
		if extra, err := a.exec(convars); err == nil {
			out += "\n" + extra
		}
	}

	st := parseStatus(out)
	if version, build := parseBuild(out); build > 0 {
		st.Version = version
		a.mu.Lock()
		a.build = build
		a.mu.Unlock()
	}

	a.enforceSticky(st.Toggles)
	return st
}

// ---- Toggles ----

func (a *App) GetToggles() []Toggle { return Toggles }

// SetToggle flips a switch. Admin toggles are remembered per server (when
// sticky is enabled) so a map load or preset cannot quietly undo them.
func (a *App) SetToggle(id string, on bool) error {
	t := toggleByID(id)
	if t == nil {
		return tErr("error.unknownToggle", id)
	}
	cmds := t.Off
	if on {
		cmds = t.On
	}
	if err := a.execAll(cmds...); err != nil {
		return err
	}
	if !t.Admin {
		return nil
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	s := a.config.serverByName(a.active)
	if s == nil || !s.StickyEnabled() {
		return nil
	}
	if s.Sticky == nil {
		s.Sticky = map[string]bool{}
	}
	s.Sticky[id] = on
	return a.config.Save()
}

// enforceSticky re-applies remembered admin toggles when the server has
// drifted from them — after a map load, a preset, or anything else that
// re-runs the gamemode config. Throttled so a convar that refuses to stick
// cannot spam the server or the log.
func (a *App) enforceSticky(observed map[string]int) {
	type fix struct {
		toggle *Toggle
		want   bool
	}
	var todo []fix

	a.mu.Lock()
	s := a.config.serverByName(a.active)
	if s != nil && s.StickyEnabled() {
		for id, want := range s.Sticky {
			t := toggleByID(id)
			if t == nil {
				continue
			}
			is, known := observed[id]
			if !known || is < 0 || (is == 1) == want {
				continue
			}
			if time.Since(a.lastEnforce[id]) < stickyThrottle {
				continue
			}
			a.lastEnforce[id] = time.Now()
			todo = append(todo, fix{toggle: t, want: want})
		}
	}
	a.mu.Unlock()

	// Run the commands outside the lock: they are network round trips.
	for _, f := range todo {
		cmds, state := f.toggle.Off, "state.off"
		if f.want {
			cmds, state = f.toggle.On, "state.on"
		}
		if err := a.execAll(cmds...); err == nil {
			a.logKey("log.keptToggle", f.toggle.LabelKey, state)
		}
	}
}
