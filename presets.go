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

var Presets = []Preset{
	{
		ID:           "competitive",
		Name:         "Competitive 5v5",
		Description:  "MR12. A 12:12 ends in a draw.",
		ExpectedMode: 1,
		Commands:     []string{"game_type 0", "game_mode 1"},
		PostCommands: []string{
			"mp_overtime_enable 0", // undo a previous Premier round
		},
	},
	{
		ID:           "premier",
		Name:         "Premier 5v5",
		Description:  "MR12, but 12:12 goes to overtime.",
		ExpectedMode: 1,
		Commands:     []string{"game_type 0", "game_mode 1"},
		PostCommands: []string{
			"mp_overtime_enable 1",
			"mp_overtime_maxrounds 6", // MR3 per overtime
			"mp_overtime_limit 0",     // keep playing overtimes until decided
		},
	},
	{
		ID:           "wingman",
		Name:         "Wingman 2v2",
		Description:  "MR16 on wingman maps.",
		Wingman:      true,
		ExpectedMode: 2,
		Commands:     []string{"game_type 0", "game_mode 2"},
		PostCommands: []string{
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
			"mp_limitteams 0",
			"mp_autoteambalance 0",
		},
	},
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
