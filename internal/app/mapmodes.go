package app

// Which maps can be played in which mode.
//
// This used to be two hand-written lists, and they went stale the moment CS2
// shipped new maps: de_poseidon and de_eldorado support Wingman but were
// missing, so StrikeMan hid them from the Wingman list *and* refused to load
// them — while the server was happily running Wingman on one of them.
//
// So nothing is assumed. The server itself says what a map supports, in two
// ways, both conclusive:
//
//   - it is running game_mode M on map X and has not fallen back, therefore X
//     supports M;
//   - a preset asked for mode M and the server came back on another mode,
//     therefore X does not support M.
//
// Anything else stays unknown, and an unknown map is offered rather than
// hidden: hiding a map that plays fine is the worse of the two mistakes, and
// trying it is what turns it into knowledge.

import (
	"strikeman/internal/config"
)

// game_mode values, as reported by the server.
const (
	modeCompetitive = 1
	modeWingman     = 2
)

// seedMapModes is not a map pool — it is the two facts that cannot be learned
// by watching, because a map has to be loaded before it can teach us anything
// and these two would be offered for Competitive until someone tried. Both
// were verified by loading them and reading the intro prefabs the server
// reports in `status`: they carry wingman intros and no team select.
var seedMapModes = map[string]config.MapModes{
	"de_brewery": {Wingman: 1, Competitive: -1},
	"de_dogtown": {Wingman: 1, Competitive: -1},
}

// mapModes returns everything known about maps, seeds included.
func (a *App) mapModes() map[string]config.MapModes {
	a.mu.Lock()
	defer a.mu.Unlock()
	return mergedMapModes(a.config.MapModes)
}

func mergedMapModes(learned map[string]config.MapModes) map[string]config.MapModes {
	out := make(map[string]config.MapModes, len(seedMapModes)+len(learned))
	for name, modes := range seedMapModes {
		out[name] = modes
	}
	// Learned values win: if a seed is ever wrong, playing the map corrects it.
	for name, modes := range learned {
		seed := out[name]
		if modes.Competitive != 0 {
			seed.Competitive = modes.Competitive
		}
		if modes.Wingman != 0 {
			seed.Wingman = modes.Wingman
		}
		out[name] = seed
	}
	return out
}

// supportsMode reports what is known about one map and mode: yes, no, or
// unknown.
func supportsMode(modes map[string]config.MapModes, ref string, mode int) (known, supported bool) {
	m, ok := modes[ref]
	if !ok {
		return false, false
	}
	value := m.Competitive
	if mode == modeWingman {
		value = m.Wingman
	}
	switch value {
	case 1:
		return true, true
	case -1:
		return true, false
	}
	return false, false
}

// learnMapMode records what the server has just demonstrated. Writes only on
// a change, so the status poll does not rewrite the config every few seconds.
func (a *App) learnMapMode(ref string, mode int, supported bool) {
	if ref == "" || (mode != modeCompetitive && mode != modeWingman) {
		return
	}
	value := 1
	if !supported {
		value = -1
	}

	a.mu.Lock()
	if a.config.MapModes == nil {
		a.config.MapModes = map[string]config.MapModes{}
	}
	entry := a.config.MapModes[ref]
	current := &entry.Competitive
	if mode == modeWingman {
		current = &entry.Wingman
	}
	if *current == value {
		a.mu.Unlock()
		return
	}
	*current = value
	a.config.MapModes[ref] = entry
	err := a.config.Save()
	a.mu.Unlock()

	if err != nil {
		a.logKey("log.mapModeSaveFailed", err.Error())
		return
	}
	key := "log.mapSupports"
	if !supported {
		key = "log.mapDoesNotSupport"
	}
	a.logKey(key, ref, tkey(modeNameKey(mode)))
}

func modeNameKey(mode int) string {
	if mode == modeWingman {
		return "mode.wingman"
	}
	return "mode.competitive"
}

// observeMapMode is the passive half: whatever the server happens to be
// running right now is proof that the loaded map supports it, whether or not
// StrikeMan was the one that set it up.
func (a *App) observeMapMode(st Status) {
	if !st.Connected || st.Map == "" || st.GameType != 0 {
		return
	}
	if st.GameMode == modeCompetitive || st.GameMode == modeWingman {
		a.learnMapMode(st.Map, st.GameMode, true)
	}
}
