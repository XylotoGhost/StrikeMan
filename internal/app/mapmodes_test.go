package app

import (
	"testing"

	"strikeman/internal/config"
	"strikeman/internal/steam"
)

// isolateConfig points the config file at a temp directory so a test that
// saves cannot touch the real one.
func isolateConfig(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("APPDATA", dir)         // Windows
	t.Setenv("XDG_CONFIG_HOME", dir) // Linux
	t.Setenv("HOME", dir)            // macOS
	t.Setenv("XDG_RUNTIME_DIR", dir) // keeps stray helpers out of the real one
}

// The regression that started this: de_poseidon and de_eldorado support
// Wingman, were missing from the hand-written list, and so were both hidden
// from the Wingman map list and refused if picked anyway — while the server
// was running Wingman on one of them.
func TestUnknownMapsAreOfferedNotRefused(t *testing.T) {
	a := New("test")
	for _, name := range []string{"de_poseidon", "de_eldorado", "de_fachwerk"} {
		if err := a.checkWingmanMap(name, false); err != nil {
			t.Errorf("%s: unknown maps must be allowed through, got %v", name, err)
		}
	}
}

func TestLearnedSupportDecidesTheAnswer(t *testing.T) {
	isolateConfig(t)
	a := New("test")

	// The server ran wingman on it, so it plays wingman.
	a.learnMapMode("de_poseidon", modeWingman, true)
	if err := a.checkWingmanMap("de_poseidon", false); err != nil {
		t.Errorf("a map seen running wingman must be allowed: %v", err)
	}

	// The server fell back off wingman, so it does not.
	a.learnMapMode("de_dust2", modeWingman, false)
	if err := a.checkWingmanMap("de_dust2", false); err == nil {
		t.Error("a map that fell back off wingman must be refused")
	}
}

// A workshop map tagged Wingman by its uploader but which the server would
// not actually run in wingman: once observed, the tag stops mattering.
func TestObservedSupportOverridesWorkshopTags(t *testing.T) {
	isolateConfig(t)
	a := New("test")
	a.workshop = []steam.WorkshopMap{
		{ID: "999", Title: "Claims To Be Wingman", Tags: []string{"Wingman", "Map"}},
	}
	if err := a.checkWingmanMap("999", true); err != nil {
		t.Fatalf("before anything is known the tags decide: %v", err)
	}

	a.learnMapMode("999", modeWingman, false)
	if err := a.checkWingmanMap("999", true); err == nil {
		t.Error("a workshop map observed to fail wingman must be refused despite its tags")
	}
}

func TestSeedsCoverTheWingmanOnlyMaps(t *testing.T) {
	modes := mergedMapModes(nil)
	for _, name := range []string{"de_brewery", "de_dogtown"} {
		known, supported := supportsMode(modes, name, modeWingman)
		if !known || !supported {
			t.Errorf("%s should be seeded as playing wingman", name)
		}
		known, supported = supportsMode(modes, name, modeCompetitive)
		if !known || supported {
			t.Errorf("%s should be seeded as not playing competitive", name)
		}
	}
}

// A seed is only a starting point: if the server ever says otherwise, the
// server is right.
func TestLearnedValuesOverrideSeeds(t *testing.T) {
	modes := mergedMapModes(map[string]config.MapModes{
		"de_brewery": {Competitive: 1},
	})
	known, supported := supportsMode(modes, "de_brewery", modeCompetitive)
	if !known || !supported {
		t.Error("an observation must override the seeded value")
	}
	// The half of the seed that was not contradicted stays.
	if known, supported := supportsMode(modes, "de_brewery", modeWingman); !known || !supported {
		t.Error("the untouched half of the seed should survive")
	}
}

func TestSupportsModeReportsUnknown(t *testing.T) {
	modes := mergedMapModes(nil)
	if known, _ := supportsMode(modes, "de_neverheardofit", modeWingman); known {
		t.Error("a map nobody has played must read as unknown")
	}
	if known, _ := supportsMode(modes, "de_brewery", modeCompetitive); !known {
		t.Error("a seeded map must read as known")
	}
}

// Whatever the server is running right now is proof the loaded map supports
// it, however the map got there.
func TestObserveMapModeLearnsFromAPoll(t *testing.T) {
	isolateConfig(t)
	a := New("test")

	a.observeMapMode(Status{Connected: true, Map: "de_poseidon", GameType: 0, GameMode: modeWingman})
	if known, supported := supportsMode(a.mapModes(), "de_poseidon", modeWingman); !known || !supported {
		t.Error("a wingman match in progress should teach us the map plays wingman")
	}

	// Nothing conclusive in these, so nothing should be recorded.
	a.observeMapMode(Status{Connected: false, Map: "de_mirage", GameMode: modeWingman})
	a.observeMapMode(Status{Connected: true, Map: "", GameMode: modeWingman})
	a.observeMapMode(Status{Connected: true, Map: "de_mirage", GameType: 1, GameMode: modeWingman})
	if known, _ := supportsMode(a.mapModes(), "de_mirage", modeWingman); known {
		t.Error("nothing should be learned from a disconnected or non-standard status")
	}
}
