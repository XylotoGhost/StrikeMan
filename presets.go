package main

// Game mode presets. Commands run before the map reload (game_type/game_mode
// only apply on map load), PostCommands run once the new map is up.

type Preset struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Wingman      bool     `json:"wingman"` // needs a wingman-capable map
	ExpectedMode int      `json:"-"`       // game_mode value once the map is up
	Commands     []string `json:"-"`
	PostCommands []string `json:"-"`
}

// PostCommands spell out each preset's match rules instead of trusting the
// gamemode config, so applying a preset always lands on a known state. The
// values match what the server's own configs set (verified over RCON):
// competitive is MR12 without overtime, wingman is MR8 with overtime.
var Presets = []Preset{
	{
		ID:           "competitive",
		Name:         "Competitive 5v5",
		Description:  "MR12. A 12:12 ends in a draw.",
		ExpectedMode: 1,
		Commands:     []string{"game_type 0", "game_mode 1"},
		PostCommands: []string{
			"mp_friendlyfire 1",
			"mp_overtime_enable 0",
		},
	},
	{
		ID:           "premier",
		Name:         "Premier 5v5",
		Description:  "MR12, but 12:12 goes to overtime.",
		ExpectedMode: 1,
		Commands:     []string{"game_type 0", "game_mode 1"},
		PostCommands: []string{
			"mp_friendlyfire 1",
			"mp_overtime_enable 1",
			"mp_overtime_maxrounds 6", // MR3 per overtime
			"mp_overtime_limit 0",     // keep playing overtimes until decided
		},
	},
	{
		ID:           "wingman",
		Name:         "Wingman 2v2",
		Description:  "MR8. An 8:8 ends in a draw.",
		Wingman:      true,
		ExpectedMode: 2,
		Commands:     []string{"game_type 0", "game_mode 2"},
		PostCommands: []string{
			"mp_friendlyfire 1",
			// The server's wingman config turns overtime on; standard
			// wingman is a draw at 8:8, so turn it back off.
			"mp_overtime_enable 0",
			"mp_limitteams 1", // re-lock team size after a 3v3 session
		},
	},
	{
		ID:           "wingman3v3",
		Name:         "Wingman 3v3",
		Description:  "Wingman maps, 3 players per team.",
		Wingman:      true,
		ExpectedMode: 2,
		Commands:     []string{"game_type 0", "game_mode 2"},
		PostCommands: []string{
			"mp_friendlyfire 1",
			"mp_overtime_enable 0", // as in wingman 2v2: 8:8 is a draw
			"mp_limitteams 0",
			"mp_autoteambalance 0",
		},
	},
}

// Toggle is a switch in the Match card. Match rules follow whichever preset
// is applied; admin toggles can be kept sticky per server, because a map
// load re-runs the gamemode config and would otherwise undo them.
type Toggle struct {
	ID    string   `json:"id"`
	Label string   `json:"label"`
	Hint  string   `json:"hint"`
	Admin bool     `json:"admin"`
	Cvar  string   `json:"cvar"` // convar read back to show the state
	On    []string `json:"-"`
	Off   []string `json:"-"`
}

var Toggles = []Toggle{
	{
		ID: "friendlyfire", Label: "Friendly fire", Cvar: "mp_friendlyfire",
		On:  []string{"mp_friendlyfire 1"},
		Off: []string{"mp_friendlyfire 0"},
	},
	{
		ID: "overtime", Label: "Overtime", Cvar: "mp_overtime_enable",
		On:  []string{"mp_overtime_enable 1"},
		Off: []string{"mp_overtime_enable 0"},
	},
	{
		ID: "autokick", Label: "Auto-kick (idle / team damage)", Admin: true,
		Hint: "Off means nobody is kicked for team damage or being idle.",
		Cvar: "mp_autokick",
		On:   []string{"mp_autokick 1", "mp_spawnprotectiontime 5"},
		Off:  []string{"mp_autokick 0", "mp_spawnprotectiontime 0"},
	},
	{
		ID: "kickban", Label: "Kicked players banned 15 min", Admin: true,
		Hint: "Off lets a kicked player rejoin immediately.",
		Cvar: "sv_kick_ban_duration",
		On:   []string{"sv_kick_ban_duration 15", "sv_vote_kick_ban_duration 15"},
		Off:  []string{"sv_kick_ban_duration 0", "sv_vote_kick_ban_duration 0"},
	},
}

func toggleByID(id string) *Toggle {
	for i := range Toggles {
		if Toggles[i].ID == id {
			return &Toggles[i]
		}
	}
	return nil
}

// Official maps with wingman spawn support, and the subset that has *only*
// wingman spawns. Verified by loading a map and looking at which intro
// prefabs the server reports in `status`: de_inferno loads both team and
// wingman intros, de_brewery only the wingman ones.
var WingmanMaps = []string{
	"de_brewery",
	"de_dogtown",
	"de_inferno",
	"de_nuke",
	"de_overpass",
	"de_vertigo",
}

var WingmanOnlyMaps = []string{
	"de_brewery",
	"de_dogtown",
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func presetByID(id string) *Preset {
	for i := range Presets {
		if Presets[i].ID == id {
			return &Presets[i]
		}
	}
	return nil
}
