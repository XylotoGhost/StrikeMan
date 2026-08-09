package app

// Game modes and switches, as data. Every label is a translation key: the
// frontend renders the text, so presets and toggles are localised too.

import "time"

// Timings for the map-load follow-up and sticky enforcement.
const (
	mapLoadFirstWait   = 5 * time.Second  // let the server actually drop the map
	mapLoadPollWait    = 5 * time.Second  // between "are you back yet" probes
	mapLoadMaxAttempts = 12               // ~1 minute of waiting in total
	playerReturnWait   = 4 * time.Second  // players reconnecting after a changelevel
	playerReturnTries  = 3                //
	stickyThrottle     = 20 * time.Second // don't re-send a convar that won't stick
)

type Preset struct {
	ID           string   `json:"id"`
	NameKey      string   `json:"nameKey"`
	DescKey      string   `json:"descKey"`
	Wingman      bool     `json:"wingman"` // needs a wingman-capable map
	ExpectedMode int      `json:"-"`       // game_mode value once the map is up
	Commands     []string `json:"-"`
	PostCommands []string `json:"-"`
}

// PostCommands spell out each preset's match rules instead of trusting the
// gamemode config, so applying a preset always lands on a known state. The
// values match what the server's own configs set (verified over RCON):
// competitive is MR12 without overtime, wingman is MR8.
var Presets = []Preset{
	{
		ID:           "competitive",
		NameKey:      "preset.competitive.name",
		DescKey:      "preset.competitive.desc",
		ExpectedMode: 1,
		Commands:     []string{"game_type 0", "game_mode 1"},
		PostCommands: []string{
			"mp_friendlyfire 1",
			"mp_overtime_enable 0",
		},
	},
	{
		ID:           "premier",
		NameKey:      "preset.premier.name",
		DescKey:      "preset.premier.desc",
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
		NameKey:      "preset.wingman.name",
		DescKey:      "preset.wingman.desc",
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
		NameKey:      "preset.wingman3v3.name",
		DescKey:      "preset.wingman3v3.desc",
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

func presetByID(id string) *Preset {
	for i := range Presets {
		if Presets[i].ID == id {
			return &Presets[i]
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

// Toggle is a switch in the Match card. Match rules follow whichever preset
// is applied; admin toggles can be kept sticky per server, because a map
// load re-runs the gamemode config and would otherwise undo them.
type Toggle struct {
	ID       string   `json:"id"`
	LabelKey string   `json:"labelKey"`
	HintKey  string   `json:"hintKey"`
	Admin    bool     `json:"admin"`
	Cvar     string   `json:"cvar"` // convar read back to show the state
	On       []string `json:"-"`
	Off      []string `json:"-"`
}

var Toggles = []Toggle{
	{
		ID: "friendlyfire", LabelKey: "toggle.friendlyfire", Cvar: "mp_friendlyfire",
		On:  []string{"mp_friendlyfire 1"},
		Off: []string{"mp_friendlyfire 0"},
	},
	{
		ID: "overtime", LabelKey: "toggle.overtime", Cvar: "mp_overtime_enable",
		On:  []string{"mp_overtime_enable 1"},
		Off: []string{"mp_overtime_enable 0"},
	},
	{
		ID: "autokick", LabelKey: "toggle.autokick", HintKey: "toggle.autokick.hint",
		Admin: true, Cvar: "mp_autokick",
		On:  []string{"mp_autokick 1", "mp_spawnprotectiontime 5"},
		Off: []string{"mp_autokick 0", "mp_spawnprotectiontime 0"},
	},
	{
		ID: "kickban", LabelKey: "toggle.kickban", HintKey: "toggle.kickban.hint",
		Admin: true, Cvar: "sv_kick_ban_duration",
		On:  []string{"sv_kick_ban_duration 15", "sv_vote_kick_ban_duration 15"},
		Off: []string{"sv_kick_ban_duration 0", "sv_vote_kick_ban_duration 0"},
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
