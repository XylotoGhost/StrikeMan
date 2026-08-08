package main

// Game mode presets. Commands run before the map reload (game_type/game_mode
// only apply on map load), PostCommands run once the new map is up.

type Preset struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Commands     []string `json:"-"`
	PostCommands []string `json:"-"`
}

var Presets = []Preset{
	{
		ID:          "competitive",
		Name:        "Competitive 5v5",
		Description: "Standard competitive rules, MR12 with halftime.",
		Commands:    []string{"game_type 0", "game_mode 1"},
	},
	{
		ID:          "wingman",
		Name:        "Wingman 2v2",
		Description: "Standard wingman rules, MR16, best on wingman maps.",
		Commands:    []string{"game_type 0", "game_mode 2"},
	},
	{
		ID:          "wingman3v3",
		Name:        "Wingman 3v3",
		Description: "Wingman rules and maps, but with 3 players per team.",
		Commands:    []string{"game_type 0", "game_mode 2"},
		PostCommands: []string{
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
