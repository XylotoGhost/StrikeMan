package main

// App holds the backend state. Every exported method on it is callable
// from the frontend as window.go.main.App.<Method>.

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx    context.Context
	config Config
	rcon   *Rcon
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.config = LoadConfig()
	a.rcon = NewRcon(a.config.Host, a.config.Port, a.config.Password)
}

// log sends a line to the frontend console card.
func (a *App) log(format string, args ...any) {
	runtime.EventsEmit(a.ctx, "log", fmt.Sprintf(format, args...))
}

// ---- Config ----

func (a *App) GetConfig() Config {
	return a.config
}

func (a *App) SaveConfig(cfg Config) error {
	a.config = cfg
	a.rcon.Close()
	a.rcon = NewRcon(cfg.Host, cfg.Port, cfg.Password)
	return cfg.Save()
}

// ---- Raw console ----

func (a *App) RunCommand(cmd string) (string, error) {
	return a.rcon.Exec(cmd)
}

// ---- Status ----

type Player struct {
	UserID string `json:"userId"`
	Name   string `json:"name"`
	Ping   string `json:"ping"`
	Bot    bool   `json:"bot"`
}

type Status struct {
	Connected bool     `json:"connected"`
	Error     string   `json:"error"`
	Hostname  string   `json:"hostname"`
	Map       string   `json:"map"`
	Humans    string   `json:"humans"`
	Players   []Player `json:"players"`
}

var (
	reHostname   = regexp.MustCompile(`(?m)^hostname\s*:\s*(.+)$`)
	reMap        = regexp.MustCompile(`loaded spawngroup\(\s*1\)\s*:\s*SV:\s*\[1:\s*([^|\]]+?)\s*\|`)
	rePlayerLine = regexp.MustCompile(`^\s*(\d+)\s+(\S+)\s+(\d+)\s+\d+\s+(\w+)\s+\d+\s*(\S*)\s+'(.+)'\s*$`)
	rePlayerCnt  = regexp.MustCompile(`(?m)^players\s*:\s*(.+)$`)
)

func (a *App) GetStatus() Status {
	out, err := a.rcon.Exec("status")
	if err != nil {
		return Status{Error: err.Error()}
	}
	st := Status{Connected: true, Players: []Player{}}
	if m := reHostname.FindStringSubmatch(out); m != nil {
		st.Hostname = strings.TrimSpace(m[1])
	}
	if m := reMap.FindStringSubmatch(out); m != nil {
		st.Map = strings.TrimSpace(m[1])
	}
	if m := rePlayerCnt.FindStringSubmatch(out); m != nil {
		st.Humans = strings.TrimSpace(m[1])
	}
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
	Official []string      `json:"official"`
	Workshop []WorkshopMap `json:"workshop"`
}

var reMapName = regexp.MustCompile(`^(ar|cs|de)_[a-z0-9_]+$`)

// GetMaps asks the server which official maps it has and pairs that with the
// configured workshop collection.
func (a *App) GetMaps() (MapList, error) {
	list := MapList{Official: []string{}, Workshop: []WorkshopMap{}}
	out, err := a.rcon.Exec("maps *")
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

	if ws, err := FetchWorkshopMaps(a.config.CollectionID); err != nil {
		a.log("Workshop collection: %v", err)
	} else {
		list.Workshop = ws
	}
	return list, nil
}

// ChangeMap loads an official map name or a workshop file ID.
func (a *App) ChangeMap(ref string, workshop bool) error {
	cmd := "changelevel " + ref
	if workshop {
		cmd = "host_workshop_map " + ref
	}
	a.log("> %s", cmd)
	_, err := a.rcon.Exec(cmd)
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
	a.log("Applying preset: %s on %s", p.Name, mapRef)
	for _, cmd := range p.Commands {
		if _, err := a.rcon.Exec(cmd); err != nil {
			return err
		}
	}
	if err := a.ChangeMap(mapRef, workshop); err != nil {
		return err
	}
	if len(p.PostCommands) > 0 {
		go a.runAfterMapLoad(p)
	}
	return nil
}

// runAfterMapLoad waits until the server answers again after a map change,
// then sends the preset's post-load commands.
func (a *App) runAfterMapLoad(p *Preset) {
	time.Sleep(5 * time.Second)
	for i := 0; i < 12; i++ {
		if _, err := a.rcon.Exec("echo strikeman-ready"); err == nil {
			for _, cmd := range p.PostCommands {
				a.rcon.Exec(cmd)
				a.log("> %s", cmd)
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

func (a *App) EndWarmup() error   { return a.execAll("mp_warmup_end") }
func (a *App) Pause() error       { return a.execAll("mp_pause_match") }
func (a *App) Unpause() error     { return a.execAll("mp_unpause_match") }
func (a *App) RestartRound() error { return a.execAll("mp_restartgame 1") }

// ---- Teams & players ----

func (a *App) SwapTeams() error     { return a.execAll("mp_swapteams") }
func (a *App) ScrambleTeams() error { return a.execAll("mp_scrambleteams") }

func (a *App) SetTeamNames(ct, t string) error {
	return a.execAll("mp_teamname_1 "+ct, "mp_teamname_2 "+t)
}

func (a *App) KickPlayer(userID string) error { return a.execAll("kickid " + userID) }

func (a *App) AddBot(team string) error { // team: "ct" or "t"
	return a.execAll("bot_add_" + team)
}

func (a *App) KickBots() error { return a.execAll("bot_kick") }

func (a *App) execAll(cmds ...string) error {
	for _, cmd := range cmds {
		a.log("> %s", cmd)
		if _, err := a.rcon.Exec(cmd); err != nil {
			return err
		}
	}
	return nil
}
