package main

// App holds the backend state. Every exported method on it is callable
// from the frontend as window.go.main.App.<Method>.

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx      context.Context
	config   Config
	active   string // name of the currently selected server
	rcon     *Rcon
	canBatch *bool         // whether the server accepts semicolon-batched commands
	workshop []WorkshopMap // last fetched workshop list, for tag lookups
	build    int           // server build number from `status`, for the update check
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.config = LoadConfig()
	a.active = a.config.Default
	if a.config.serverByName(a.active) == nil && len(a.config.Servers) > 0 {
		a.active = a.config.Servers[0].Name
	}
	a.connectActive()
}

func (a *App) connectActive() {
	if a.rcon != nil {
		a.rcon.Close()
		a.rcon = nil
	}
	a.canBatch = nil
	if s := a.config.serverByName(a.active); s != nil {
		a.rcon = NewRcon(s.Host, s.Port, s.Password)
	}
}

// log sends a line to the frontend console card.
func (a *App) log(format string, args ...any) {
	runtime.EventsEmit(a.ctx, "log", fmt.Sprintf(format, args...))
}

// warn additionally pops up as an error toast in the frontend.
func (a *App) warn(format string, args ...any) {
	runtime.EventsEmit(a.ctx, "warn", fmt.Sprintf(format, args...))
}

func (a *App) exec(cmd string) (string, error) {
	if a.rcon == nil {
		return "", fmt.Errorf("no server configured")
	}
	return a.rcon.Exec(cmd)
}

// ---- Config & server selection ----

func (a *App) GetConfig() Config {
	return a.config
}

func (a *App) SaveConfig(cfg Config) error {
	a.config = cfg
	if err := cfg.Save(); err != nil {
		return err
	}
	if a.config.serverByName(a.active) == nil {
		a.active = a.config.Default
		if a.config.serverByName(a.active) == nil && len(a.config.Servers) > 0 {
			a.active = a.config.Servers[0].Name
		}
	}
	a.connectActive()
	return nil
}

func (a *App) GetActiveServer() string {
	return a.active
}

func (a *App) SelectServer(name string) error {
	if a.config.serverByName(name) == nil {
		return fmt.Errorf("unknown server %q", name)
	}
	a.active = name
	a.connectActive()
	a.log("Switched to server: %s", name)
	return nil
}

// ---- Raw console ----

func (a *App) RunCommand(cmd string) (string, error) {
	return a.exec(cmd)
}

// ---- Status ----

type Player struct {
	UserID string `json:"userId"`
	Name   string `json:"name"`
	Ping   string `json:"ping"`
	Bot    bool   `json:"bot"`
}

type Status struct {
	Connected  bool     `json:"connected"`
	Error      string   `json:"error"`
	Hostname   string   `json:"hostname"`
	Map        string   `json:"map"`
	Humans     int      `json:"humans"`
	Bots       int      `json:"bots"`
	GameType   int      `json:"gameType"`
	GameMode   int      `json:"gameMode"`
	LimitTeams int      `json:"limitTeams"`
	MaxRounds  int      `json:"maxRounds"`
	Version    string   `json:"version"`
	// Toggles: -1 unknown, 0 off, 1 on.
	FriendlyFire int      `json:"friendlyFire"`
	Overtime     int      `json:"overtime"`
	Autokick     int      `json:"autokick"`
	Players      []Player `json:"players"`
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
	reVersion    = regexp.MustCompile(`(?m)^version\s*:\s*(\S+?)/(\d+)`)
)

const modeConvars = "game_type; game_mode; mp_limitteams; mp_maxrounds; " + toggleConvars

// Toggles exposed as switches in the UI; also the allowlist for SetToggle.
var toggleCvars = []string{"mp_friendlyfire", "mp_overtime_enable", "mp_autokick"}

const toggleConvars = "mp_friendlyfire; mp_overtime_enable; mp_autokick"

// readToggle finds "<cvar> = true|false|1|0" in RCON output.
// Returns -1 when the convar isn't present.
func readToggle(out, cvar string) int {
	m := regexp.MustCompile(cvar + ` = (\w+)`).FindStringSubmatch(out)
	if m == nil {
		return -1
	}
	switch strings.ToLower(m[1]) {
	case "true", "1":
		return 1
	case "false", "0":
		return 0
	}
	return -1
}

// GetStatus polls `status` plus the mode convars. Batching them into one
// round trip is attempted once; if the server doesn't split on semicolons,
// the convars are queried separately from then on.
func (a *App) GetStatus() Status {
	st := Status{GameType: -1, GameMode: -1, LimitTeams: -1, Players: []Player{}}
	cmd := "status"
	if a.canBatch == nil || *a.canBatch {
		cmd = "status; " + modeConvars
	}
	out, err := a.exec(cmd)
	if err != nil {
		st.Error = err.Error()
		return st
	}
	if a.canBatch == nil {
		ok := reGameMode.MatchString(out)
		a.canBatch = &ok
	}
	if !*a.canBatch {
		if extra, err := a.exec(modeConvars); err == nil {
			out += "\n" + extra
		}
	}

	st.Connected = true
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
	if m := reVersion.FindStringSubmatch(out); m != nil {
		st.Version = m[1]
		a.build, _ = strconv.Atoi(m[2])
	}
	st.FriendlyFire = readToggle(out, "mp_friendlyfire")
	st.Overtime = readToggle(out, "mp_overtime_enable")
	st.Autokick = readToggle(out, "mp_autokick")
	for _, line := range strings.Split(out, "\n") {
		m := rePlayerLine.FindStringSubmatch(line)
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

// ---- Maps ----

type MapList struct {
	Official    []string      `json:"official"`
	Wingman     []string      `json:"wingman"`
	WingmanOnly []string      `json:"wingmanOnly"`
	Workshop    []WorkshopMap `json:"workshop"`
}

var reMapName = regexp.MustCompile(`^(ar|cs|de)_[a-z0-9_]+$`)

// GetMaps asks the server which official maps it has and pairs that with the
// configured workshop collection.
func (a *App) GetMaps() (MapList, error) {
	list := MapList{Official: []string{}, Wingman: WingmanMaps, WingmanOnly: WingmanOnlyMaps, Workshop: []WorkshopMap{}}
	out, err := a.exec("maps *")
	if err != nil {
		return list, err
	}
	for _, line := range strings.Split(out, "\n") {
		name := strings.TrimSpace(line)
		if reMapName.MatchString(name) && !strings.HasSuffix(name, "_vanity") {
			list.Official = append(list.Official, name)
		}
	}
	sort.Strings(list.Official)

	collectionID := ""
	if s := a.config.serverByName(a.active); s != nil {
		collectionID = s.CollectionID
	}
	if ws, err := FetchWorkshopMaps(collectionID); err != nil {
		a.log("Workshop collection: %v", err)
	} else {
		sort.Slice(ws, func(i, j int) bool {
			return strings.ToLower(ws[i].Title) < strings.ToLower(ws[j].Title)
		})
		list.Workshop = ws
		a.workshop = ws
	}
	return list, nil
}

// checkWingmanMap reports an error when a map is known not to support
// wingman: official maps by the WingmanMaps list, workshop maps by their
// Steam tags. Untagged workshop maps pass (the post-load check covers them).
func (a *App) checkWingmanMap(mapRef string, workshop bool) error {
	if workshop {
		for _, m := range a.workshop {
			if m.ID == mapRef && len(m.Tags) > 0 && !m.HasTag("wingman") {
				return fmt.Errorf("%q is not tagged as a Wingman map on the workshop — the server would fall back to Competitive", m.Title)
			}
		}
		return nil
	}
	if contains(WingmanMaps, mapRef) {
		return nil
	}
	return fmt.Errorf("%s does not support Wingman — pick one of the wingman maps", mapRef)
}

// ChangeMap loads an official map name or a workshop file ID.
func (a *App) ChangeMap(ref string, workshop bool) error {
	cmd := "changelevel " + ref
	if workshop {
		cmd = "host_workshop_map " + ref
	}
	a.log("> %s", cmd)
	_, err := a.exec(cmd)
	return err
}

// ---- Presets ----

func (a *App) GetPresets() []Preset {
	return Presets
}

// ApplyPreset sets the mode convars, reloads the map (mode changes only apply
// on map load) and, once the server is back, runs the preset's post commands.
func (a *App) ApplyPreset(id, mapRef string, workshop bool) error {
	p := presetByID(id)
	if p == nil {
		return fmt.Errorf("unknown preset %q", id)
	}
	if mapRef == "" {
		st := a.GetStatus()
		if st.Map == "" {
			return fmt.Errorf("no map selected and current map unknown")
		}
		mapRef = st.Map
	}
	if p.Wingman {
		if err := a.checkWingmanMap(mapRef, workshop); err != nil {
			return err
		}
	}
	a.log("Applying preset: %s on %s", p.Name, mapRef)
	for _, cmd := range p.Commands {
		if _, err := a.exec(cmd); err != nil {
			return err
		}
	}
	if err := a.ChangeMap(mapRef, workshop); err != nil {
		return err
	}
	go a.runAfterMapLoad(p)
	return nil
}

// runAfterMapLoad waits until the server answers again after a map change,
// sends the preset's post-load commands and verifies the mode actually
// applied (the server falls back when a map doesn't support the mode).
func (a *App) runAfterMapLoad(p *Preset) {
	time.Sleep(5 * time.Second)
	for i := 0; i < 12; i++ {
		if _, err := a.exec("echo strikeman-ready"); err == nil {
			for _, cmd := range p.PostCommands {
				a.exec(cmd)
				a.log("> %s", cmd)
			}
			if out, err := a.exec("game_mode"); err == nil {
				if m := reGameMode.FindStringSubmatch(out); m != nil && m[1] != strconv.Itoa(p.ExpectedMode) {
					a.warn("The map does not support %s — the server fell back to another mode (game_mode = %s).", p.Name, m[1])
					return
				}
			}
			a.log("Preset %s fully applied.", p.Name)
			return
		}
		time.Sleep(5 * time.Second)
	}
	a.log("Preset %s: server did not come back in time, post commands skipped.", p.Name)
}

// ---- Match control ----

func (a *App) StartWarmup(seconds int) error {
	return a.execAll(fmt.Sprintf("mp_warmuptime %d", seconds), "mp_warmup_start")
}

func (a *App) EndWarmup() error    { return a.execAll("mp_warmup_end") }
func (a *App) Pause() error        { return a.execAll("mp_pause_match") }
func (a *App) Unpause() error      { return a.execAll("mp_unpause_match") }
func (a *App) RestartRound() error { return a.execAll("mp_restartgame 1") }

// ---- Teams & players ----

func (a *App) SwapTeams() error     { return a.execAll("mp_swapteams") }
func (a *App) ScrambleTeams() error { return a.execAll("mp_scrambleteams") }

func (a *App) SetTeamNames(ct, t string) error {
	return a.execAll("mp_teamname_1 "+ct, "mp_teamname_2 "+t)
}

func (a *App) KickPlayer(userID string) error { return a.execAll("kickid " + userID) }

func (a *App) AddBot(team string) error { // team: "ct" or "t"
	// In the gamemode cfgs' quota modes ("competitive"/"fill") the engine
	// manages the bot count itself and kicks manually added bots again.
	// "normal" hands control back to us.
	return a.execAll("bot_quota_mode normal", "bot_join_after_player 0", "bot_add_"+team)
}

func (a *App) KickBots() error { return a.execAll("bot_kick") }

// ---- Toggles, announcements ----

// SetToggle flips one of the convars backing the UI switches.
func (a *App) SetToggle(cvar string, on bool) error {
	if !contains(toggleCvars, cvar) {
		return fmt.Errorf("%q is not a toggle", cvar)
	}
	value := "0"
	if on {
		value = "1"
	}
	return a.execAll(cvar + " " + value)
}

// Announce prints a message in every player's chat.
func (a *App) Announce(msg string) error {
	msg = strings.NewReplacer(`"`, "", ";", "", "\n", " ").Replace(strings.TrimSpace(msg))
	if msg == "" {
		return fmt.Errorf("nothing to announce")
	}
	return a.execAll(`say ` + msg)
}

// ---- Server info ----

type ServerInfo struct {
	Version     string `json:"version"`
	Build       int    `json:"build"`
	UpToDate    bool   `json:"upToDate"`
	Latest      int    `json:"latest"`
	UpdateNote  string `json:"updateNote"`
	CheckFailed bool   `json:"checkFailed"`
	UptimeSecs  int    `json:"uptimeSecs"`
	Addon       string `json:"addon"`
}

// GetServerInfo pairs the server's own version with Steam's "is this build
// current" endpoint, so an outdated server is visible before match night.
func (a *App) GetServerInfo() ServerInfo {
	info := ServerInfo{Build: a.build}
	st := a.GetStatus()
	info.Version = st.Version
	info.Build = a.build

	if out, err := a.exec("status_json"); err == nil {
		var sj struct {
			Uptime int `json:"process_uptime"`
			Server struct {
				Addon string `json:"addon"`
			} `json:"server"`
		}
		if start := strings.Index(out, "{"); start >= 0 {
			if end := strings.LastIndex(out, "}"); end > start {
				json.Unmarshal([]byte(out[start:end+1]), &sj)
			}
		}
		info.UptimeSecs = sj.Uptime
		info.Addon = sj.Server.Addon
	}

	if info.Build > 0 {
		upToDate, latest, note, err := CheckServerUpToDate(info.Build)
		if err != nil {
			info.CheckFailed = true
			a.log("Update check failed: %v", err)
		} else {
			info.UpToDate, info.Latest, info.UpdateNote = upToDate, latest, note
		}
	}
	return info
}

func (a *App) execAll(cmds ...string) error {
	for _, cmd := range cmds {
		a.log("> %s", cmd)
		if _, err := a.exec(cmd); err != nil {
			return err
		}
	}
	return nil
}
