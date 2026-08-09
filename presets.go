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
		Description:  "Standard competitive rules, MR12 with halftime.",
		ExpectedMode: 1,
		Commands:     []string{"game_type 0", "game_mode 1"},
	},
	{
		ID:           "wingman",
		Name:         "Wingman 2v2",
		Description:  "Standard wingman rules, MR16, best on wingman maps.",
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
		Description:  "Wingman rules and maps, but with 3 players per team.",
		Wingman:      true,
		ExpectedMode: 2,
		Commands:     []string{"game_type 0", "game_mode 2"},
		PostCommands: []string{
			"mp_limitteams 0",
			"mp_autoteambalance 0",
		},
	},
}

// Official maps with wingman spawn support; used to filter the map dropdown
// while a wingman mode is active.
var WingmanMaps = []string{
	"de_brewery",
	"de_dogtown",
	"de_inferno",
	"de_nuke",
	"de_overpass",
	"de_vertigo",
}

func presetByID(id string) *Preset {
	for i := range Presets {
		if Presets[i].ID == id {
			return &Presets[i]
		}
	}
	return nil
}
