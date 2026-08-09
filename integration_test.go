package main

// End-to-end test against a real CS2 server. It changes the map and match
// settings, so it only runs when pointed at a server on purpose:
//
//	STRIKEMAN_TEST_HOST=192.168.178.66 STRIKEMAN_TEST_PORT=27016 \
//	STRIKEMAN_TEST_PASSWORD=... go test -run TestLive -v
//
// Skipped everywhere else, including CI.

import (
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

func liveApp(t *testing.T) (*App, string) {
	t.Helper()
	host := os.Getenv("STRIKEMAN_TEST_HOST")
	password := os.Getenv("STRIKEMAN_TEST_PASSWORD")
	if host == "" || password == "" {
		t.Skip("set STRIKEMAN_TEST_HOST and STRIKEMAN_TEST_PASSWORD to run the live test")
	}
	port, _ := strconv.Atoi(os.Getenv("STRIKEMAN_TEST_PORT"))
	if port == 0 {
		port = 27015
	}

	server := Server{Name: "test", Host: host, Port: port, Password: password}
	app := NewApp()
	app.config = Config{Servers: []Server{server}, Default: server.Name}
	app.active = server.Name
	app.rcon = NewRcon(host, port, password)
	return app, server.Name
}

// waitFor polls until check passes or the deadline runs out.
func waitFor(t *testing.T, what string, timeout time.Duration, check func() (bool, string)) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var last string
	for time.Now().Before(deadline) {
		ok, detail := check()
		if ok {
			return
		}
		last = detail
		time.Sleep(3 * time.Second)
	}
	t.Fatalf("timed out waiting for %s (last state: %s)", what, last)
}

func TestLivePresetApply(t *testing.T) {
	app, serverName := liveApp(t)
	defer app.rcon.Close()

	// Where the server is now, so it can be put back.
	before := app.GetStatus()
	if before.Error != "" {
		t.Fatalf("cannot reach the server: %s", before.Error)
	}
	t.Logf("before: map=%s game_mode=%d humans=%d", before.Map, before.GameMode, before.Humans)
	if before.Humans == 0 {
		t.Log("nobody on the server: the post-load warmup is expected to be skipped")
	}

	// Keep auto-kick off across the preset, the way the UI does.
	app.config.Servers[0].Sticky = map[string]bool{"autokick": false}
	_ = serverName

	// Push the server away from the target state so the preset has something
	// to correct: overtime on (wingman should turn it off), auto-kick on
	// (sticky should turn it back off) and CS2's warmup shortcut back at its
	// default (the post-load warmup should clear it). Seeding all three keeps
	// the assertions from passing on state that was already correct.
	if err := app.execAll(
		"mp_overtime_enable 1",
		"mp_autokick 1",
		"mp_warmuptime_all_players_connected 15",
	); err != nil {
		t.Fatalf("seeding drift: %v", err)
	}

	targetMap := before.Map
	if !slices.Contains(WingmanMaps, targetMap) {
		targetMap = "de_nuke"
	}
	if err := app.ApplyPreset("wingman", targetMap, false); err != nil {
		t.Fatalf("ApplyPreset: %v", err)
	}

	// runAfterMapLoad waits for the map, applies rules, then warmup.
	waitFor(t, "the preset to land", 90*time.Second, func() (bool, string) {
		st := app.GetStatus()
		ok := st.Connected && st.GameMode == 2 && st.MaxRounds == 16 &&
			st.Toggles["overtime"] == 0 && st.Toggles["autokick"] == 0
		return ok, formatState(st)
	})

	st := app.GetStatus()
	if st.Map != targetMap {
		t.Errorf("map = %q, want %q", st.Map, targetMap)
	}
	if st.MaxRounds != 16 {
		t.Errorf("maxRounds = %d, want 16 (wingman is MR8)", st.MaxRounds)
	}
	if st.Toggles["overtime"] != 0 {
		t.Error("wingman must not inherit overtime: 8:8 is a draw")
	}
	if st.Toggles["autokick"] != 0 {
		t.Error("sticky auto-kick was not restored after the map load")
	}

	// The warmup StrikeMan starts after a map load only applies with players
	// present, and disables CS2's shortcut so the countdown cannot drift.
	if before.Humans > 0 {
		out, err := app.exec("mp_warmuptime_all_players_connected")
		if err != nil {
			t.Fatalf("reading warmup convar: %v", err)
		}
		if readToggle(out, "mp_warmuptime_all_players_connected") != 0 {
			t.Errorf("post-load warmup should disable the shortcut, got %q", strings.TrimSpace(out))
		}
	}

	t.Logf("after: %s", formatState(st))
}

func formatState(st Status) string {
	return strings.Join([]string{
		"map=" + st.Map,
		"game_mode=" + strconv.Itoa(st.GameMode),
		"maxrounds=" + strconv.Itoa(st.MaxRounds),
		"overtime=" + strconv.Itoa(st.Toggles["overtime"]),
		"autokick=" + strconv.Itoa(st.Toggles["autokick"]),
		"humans=" + strconv.Itoa(st.Humans),
	}, " ")
}
