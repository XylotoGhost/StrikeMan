package app

import (
	"slices"
	"strings"
	"testing"

	"strikeman/internal/steam"
)

func TestOfficialMapsFiltersNoise(t *testing.T) {
	// Trimmed from a real `maps *` listing.
	out := `	ar_baggage
	cs_office
	cs_office_vanity
	de_dust2
	de_dust2_vanity
	de_inferno
	error
	lobby_mapveto
	prefabs/de_nuke/de_nuke_skybox02
	ui/buy_menu
	workshop_preview_dust2`

	got := officialMaps(out)
	want := []string{"ar_baggage", "cs_office", "de_dust2", "de_inferno"}
	if !slices.Equal(got, want) {
		t.Errorf("officialMaps() = %v, want %v", got, want)
	}
}

func TestWingmanMapClassification(t *testing.T) {
	// de_brewery loads only the wingman intro prefabs; de_inferno loads both
	// team and wingman intros; de_dust2 has no wingman spawns at all.
	if !slices.Contains(WingmanMaps, "de_brewery") {
		t.Error("de_brewery should support wingman")
	}
	if !slices.Contains(WingmanOnlyMaps, "de_brewery") {
		t.Error("de_brewery should be wingman-only")
	}
	if !slices.Contains(WingmanMaps, "de_inferno") {
		t.Error("de_inferno should support wingman")
	}
	if slices.Contains(WingmanOnlyMaps, "de_inferno") {
		t.Error("de_inferno also supports competitive, so it is not wingman-only")
	}
	if slices.Contains(WingmanMaps, "de_dust2") {
		t.Error("de_dust2 has no wingman support")
	}
}

func TestWingmanSupportFromWorkshopTags(t *testing.T) {
	maps := []steam.WorkshopMap{
		{ID: "1", Title: "Cache Minecraft (port)", Tags: []string{"Classic", "Map"}},
		{ID: "2", Title: "Some Wingman Map", Tags: []string{"Wingman", "Map"}},
		{ID: "3", Title: "Untagged Map"},
	}

	if title, ok := wingmanSupport(maps, "1"); ok {
		t.Errorf("map tagged Classic only should be rejected (title %q)", title)
	}
	if _, ok := wingmanSupport(maps, "2"); !ok {
		t.Error("map tagged Wingman should be accepted")
	}
	// Without tags we cannot tell, so it must pass and rely on the check
	// performed once the map is loaded.
	if _, ok := wingmanSupport(maps, "3"); !ok {
		t.Error("untagged map should be allowed through")
	}
	if _, ok := wingmanSupport(maps, "unknown"); !ok {
		t.Error("unknown map should be allowed through")
	}
}

// Each preset must state its own match rules rather than relying on whatever
// the previous mode left behind — that was the "wingman inherits overtime"
// class of bug.
func TestPresetsSetTheirOwnRules(t *testing.T) {
	for _, p := range Presets {
		post := strings.Join(p.PostCommands, " ")
		if !strings.Contains(post, "mp_friendlyfire") {
			t.Errorf("preset %q does not set friendly fire", p.ID)
		}
		if !strings.Contains(post, "mp_overtime_enable") {
			t.Errorf("preset %q does not set overtime", p.ID)
		}
		if p.NameKey == "" || p.DescKey == "" {
			t.Errorf("preset %q is missing translation keys", p.ID)
		}
	}
}

func TestWingmanPresetsHaveNoOvertime(t *testing.T) {
	for _, id := range []string{"wingman", "wingman3v3"} {
		p := presetByID(id)
		if p == nil {
			t.Fatalf("preset %q not found", id)
		}
		if !slices.Contains(p.PostCommands, "mp_overtime_enable 0") {
			t.Errorf("%q should turn overtime off: 8:8 is a draw in wingman", id)
		}
		if !p.Wingman {
			t.Errorf("%q should be marked as needing a wingman map", id)
		}
	}
}

func TestPremierDiffersFromCompetitiveOnlyByOvertime(t *testing.T) {
	comp, premier := presetByID("competitive"), presetByID("premier")
	if comp == nil || premier == nil {
		t.Fatal("competitive and premier presets must exist")
	}
	if !slices.Equal(comp.Commands, premier.Commands) {
		t.Errorf("both run game_mode 1: %v vs %v", comp.Commands, premier.Commands)
	}
	if !slices.Contains(comp.PostCommands, "mp_overtime_enable 0") {
		t.Error("competitive should end 12:12 as a draw")
	}
	if !slices.Contains(premier.PostCommands, "mp_overtime_enable 1") {
		t.Error("premier should play out a 12:12")
	}
}

func TestTogglesAreWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, tg := range Toggles {
		if seen[tg.ID] {
			t.Errorf("duplicate toggle id %q", tg.ID)
		}
		seen[tg.ID] = true
		if tg.LabelKey == "" {
			t.Errorf("toggle %q is missing a translation key", tg.ID)
		}
		if tg.Cvar == "" || len(tg.On) == 0 || len(tg.Off) == 0 {
			t.Errorf("toggle %q needs a convar and both command sets", tg.ID)
		}
		if toggleByID(tg.ID) == nil {
			t.Errorf("toggleByID cannot find %q", tg.ID)
		}
	}
}
