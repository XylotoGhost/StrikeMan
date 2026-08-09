package main

// Everything that changes what is being played: the map list, applying a
// preset, warmup, and the match and team commands.

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ---- Maps ----

type MapList struct {
	Official    []string      `json:"official"`
	Wingman     []string      `json:"wingman"`
	WingmanOnly []string      `json:"wingmanOnly"`
	Workshop    []WorkshopMap `json:"workshop"`
}

var reMapName = regexp.MustCompile(`^(ar|cs|de)_[a-z0-9_]+$`)

// officialMaps picks the playable maps out of a `maps *` listing, dropping
// the prefabs, UI scenes and _vanity variants the server also reports.
func officialMaps(out string) []string {
	maps := []string{}
	for _, line := range strings.Split(out, "\n") {
		name := strings.TrimSpace(line)
		if reMapName.MatchString(name) && !strings.HasSuffix(name, "_vanity") {
			maps = append(maps, name)
		}
	}
	sort.Strings(maps)
	return maps
}

// GetMaps asks the server which official maps it has and pairs that with the
// configured workshop collection.
func (a *App) GetMaps() (MapList, error) {
	list := MapList{
		Official:    []string{},
		Wingman:     WingmanMaps,
		WingmanOnly: WingmanOnlyMaps,
		Workshop:    []WorkshopMap{},
	}
	out, err := a.exec("maps *")
	if err != nil {
		return list, err
	}
	list.Official = officialMaps(out)

	collectionID := ""
	if s, ok := a.server(); ok {
		collectionID = s.CollectionID
	}
	ws, err := FetchWorkshopMaps(collectionID)
	if err != nil {
		a.logKey("log.workshopFailed", err.Error())
		return list, nil
	}
	sort.Slice(ws, func(i, j int) bool {
		return strings.ToLower(ws[i].Title) < strings.ToLower(ws[j].Title)
	})
	list.Workshop = ws

	a.mu.Lock()
	a.workshop = ws
	a.mu.Unlock()
	return list, nil
}

// wingmanSupport reports whether a workshop map can be ruled out for wingman
// from its Steam tags. Untagged maps pass: the post-load check covers them.
func wingmanSupport(maps []WorkshopMap, id string) (title string, supported bool) {
	for _, m := range maps {
		if m.ID != id {
			continue
		}
		if len(m.Tags) > 0 && !m.HasTag("wingman") {
			return m.Title, false
		}
		return m.Title, true
	}
	return "", true
}

// checkWingmanMap refuses maps that are known not to support wingman.
func (a *App) checkWingmanMap(mapRef string, workshop bool) error {
	if workshop {
		a.mu.Lock()
		known := a.workshop
		a.mu.Unlock()
		if title, ok := wingmanSupport(known, mapRef); !ok {
			return tErr("error.workshopNotWingman", title)
		}
		return nil
	}
	if slices.Contains(WingmanMaps, mapRef) {
		return nil
	}
	return tErr("error.mapNotWingman", mapRef)
}

// ChangeMap loads an official map name or a workshop file ID.
func (a *App) ChangeMap(ref string, workshop bool) error {
	if err := a.changeMap(ref, workshop); err != nil {
		return err
	}
	go a.runAfterMapLoad(nil) // the new map resets sticky toggles and warmup
	return nil
}

func (a *App) changeMap(ref string, workshop bool) error {
	cmd := "changelevel " + ref
	if workshop {
		cmd = "host_workshop_map " + ref
	}
	a.log("> %s", cmd)
	_, err := a.exec(cmd)
	return err
}

// ---- Presets ----

func (a *App) GetPresets() []Preset { return Presets }

// ApplyPreset sets the mode convars, reloads the map (mode changes only apply
// on map load) and, once the server is back, runs the preset's post commands.
func (a *App) ApplyPreset(id, mapRef string, workshop bool) error {
	p := presetByID(id)
	if p == nil {
		return tErr("error.unknownPreset", id)
	}
	if mapRef == "" {
		st := a.GetStatus()
		if st.Map == "" {
			return tErr("error.noMapSelected")
		}
		mapRef = st.Map
	}
	if p.Wingman {
		if err := a.checkWingmanMap(mapRef, workshop); err != nil {
			return err
		}
	}
	a.logKey("log.applyingPreset", p.NameKey, mapRef)
	for _, cmd := range p.Commands {
		if _, err := a.exec(cmd); err != nil {
			return err
		}
	}
	if err := a.changeMap(mapRef, workshop); err != nil {
		return err
	}
	go a.runAfterMapLoad(p)
	return nil
}

// runAfterMapLoad waits until the server answers again after a map change,
// then applies the preset's match rules and restarts warmup on our terms,
// because loading a map re-runs the gamemode config and resets both. With a
// nil preset it is a plain map change and only the warmup part applies.
func (a *App) runAfterMapLoad(p *Preset) {
	time.Sleep(mapLoadFirstWait)
	for i := 0; i < mapLoadMaxAttempts; i++ {
		if _, err := a.exec("echo strikeman-ready"); err != nil {
			time.Sleep(mapLoadPollWait)
			continue
		}
		if p != nil {
			for _, cmd := range p.PostCommands {
				a.log("> %s", cmd)
				if _, err := a.exec(cmd); err != nil {
					a.logKey("log.commandFailed", cmd, err.Error())
				}
			}
		}
		// Sticky admin toggles are not re-sent here: the GetStatus below (and
		// every status poll) restores whichever ones the map load reset, and
		// only those that actually drifted.
		a.restartWarmupAfterLoad()

		if p == nil {
			return
		}
		a.verifyMode(p)
		return
	}
	if p != nil {
		a.logKey("log.presetTimedOut", p.NameKey)
	}
}

// restartWarmupAfterLoad gives the new map the warmup StrikeMan promises:
// exactly the configured length, with CS2's "everyone connected" shortcut
// disabled so the countdown in the UI stays true. Skipped on an empty server,
// where CS2 does not warm up either. Players take a moment to come back after
// a changelevel, so allow for that before deciding nobody is here.
func (a *App) restartWarmupAfterLoad() {
	for attempt := 0; attempt < playerReturnTries; attempt++ {
		if a.GetStatus().Humans > 0 {
			if s, ok := a.server(); ok {
				a.startWarmup(s.Warmup())
			}
			return
		}
		time.Sleep(playerReturnWait)
	}
}

// verifyMode warns when the server silently fell back to another mode, which
// happens on maps that do not support the one the preset asked for.
func (a *App) verifyMode(p *Preset) {
	out, err := a.exec("game_mode")
	if err != nil {
		return
	}
	if m := reGameMode.FindStringSubmatch(out); m != nil && m[1] != strconv.Itoa(p.ExpectedMode) {
		a.warnKey("log.modeFellBack", p.NameKey, m[1])
		return
	}
	a.logKey("log.presetApplied", p.NameKey)
}

// ---- Warmup ----

// StartWarmup runs a warmup and remembers the length for the next map load.
func (a *App) StartWarmup(seconds int) error {
	if err := a.SetWarmupSeconds(seconds); err != nil {
		return err
	}
	return a.startWarmup(seconds)
}

// SetWarmupSeconds stores the warmup length so it survives a restart and is
// available when a map load starts warmup on its own.
func (a *App) SetWarmupSeconds(seconds int) error {
	if seconds <= 0 {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	s := a.config.serverByName(a.active)
	if s == nil || s.WarmupSeconds == seconds {
		return nil
	}
	s.WarmupSeconds = seconds
	return a.config.Save()
}

// startWarmup runs a warmup of exactly the requested length. CS2 would
// otherwise cut it short once everyone has connected
// (mp_warmuptime_all_players_connected), which would make StrikeMan's
// countdown lie, so that shortcut is disabled first. The frontend is told how
// long the warmup is rather than having to guess.
func (a *App) startWarmup(seconds int) error {
	err := a.execAll(
		"mp_warmuptime_all_players_connected 0",
		"mp_warmup_pausetimer 0",
		fmt.Sprintf("mp_warmuptime %d", seconds),
		"mp_warmup_start",
	)
	if err == nil {
		a.emit("warmup", seconds)
	}
	return err
}

// ---- Match control ----

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

// Announce prints a message in every player's chat.
func (a *App) Announce(msg string) error {
	msg = strings.NewReplacer(`"`, "", ";", "", "\n", " ").Replace(strings.TrimSpace(msg))
	if msg == "" {
		return tErr("error.emptyAnnounce")
	}
	return a.execAll("say " + msg)
}
