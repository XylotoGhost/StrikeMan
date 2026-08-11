package app

// End-to-end test against a real CS2 server. It changes the map and match
// settings, so it only runs when pointed at a server on purpose:
//
//	STRIKEMAN_TEST_HOST=192.168.178.66 STRIKEMAN_TEST_PORT=27016 \
//	STRIKEMAN_TEST_PASSWORD=... go test -run TestLive -v
//
// Skipped everywhere else, including CI.

import (
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"strikeman/internal/config"
	"strikeman/internal/rcon"
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

	server := config.Server{Name: "test", Host: host, Port: port, Password: password}
	app := New("test")
	app.config = config.Config{Servers: []config.Server{server}, Default: server.Name}
	app.active = server.Name
	app.rcon = rcon.New(host, port, password)
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

	// Stay on the current map when it is known to play wingman, otherwise
	// fall back to one that has been seeded as such.
	targetMap := before.Map
	if known, supported := supportsMode(app.mapModes(), targetMap, modeWingman); !known || !supported {
		targetMap = "de_brewery"
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

// Adding one bot must add exactly one bot, and kicking them must make them
// stay gone. bot_quota is both a cap and an auto-fill target, so getting this
// wrong lets a single click fill the server.
func TestLiveBotsDoNotAutoFill(t *testing.T) {
	app, _ := liveApp(t)
	defer app.rcon.Close()

	if st := app.GetStatus(); st.Error != "" {
		t.Fatalf("cannot reach the server: %s", st.Error)
	}

	if err := app.KickBots(); err != nil {
		t.Fatalf("KickBots: %v", err)
	}
	waitFor(t, "the server to be free of bots", 30*time.Second, func() (bool, string) {
		st := app.GetStatus()
		return st.Bots == 0, formatState(st)
	})

	// Bots must not creep back while the quota says zero.
	time.Sleep(8 * time.Second)
	if st := app.GetStatus(); st.Bots != 0 {
		t.Fatalf("bots came back after being kicked: %d", st.Bots)
	}

	for expected := 1; expected <= 3; expected++ {
		if err := app.AddBot(); err != nil {
			t.Fatalf("AddBot: %v", err)
		}
		waitFor(t, "exactly one more bot", 20*time.Second, func() (bool, string) {
			st := app.GetStatus()
			return st.Bots == expected, formatState(st)
		})
		// Give the server a moment to top up, which is the bug being guarded
		// against: the count must hold.
		time.Sleep(6 * time.Second)
		st := app.GetStatus()
		if st.Bots != expected {
			t.Fatalf("after adding bot %d the server ended up with %d bots", expected, st.Bots)
		}
		t.Logf("added bot %d: bots=%d", expected, st.Bots)
	}

	// And removing takes exactly one away again.
	if err := app.RemoveBot(); err != nil {
		t.Fatalf("RemoveBot: %v", err)
	}
	waitFor(t, "one bot fewer", 20*time.Second, func() (bool, string) {
		st := app.GetStatus()
		return st.Bots == 2, formatState(st)
	})

	if err := app.KickBots(); err != nil {
		t.Fatalf("final KickBots: %v", err)
	}
	waitFor(t, "bots to be cleared again", 30*time.Second, func() (bool, string) {
		st := app.GetStatus()
		return st.Bots == 0, formatState(st)
	})
}

func formatState(st Status) string {
	return strings.Join([]string{
		"map=" + st.Map,
		"game_mode=" + strconv.Itoa(st.GameMode),
		"maxrounds=" + strconv.Itoa(st.MaxRounds),
		"overtime=" + strconv.Itoa(st.Toggles["overtime"]),
		"autokick=" + strconv.Itoa(st.Toggles["autokick"]),
		"humans=" + strconv.Itoa(st.Humans),
		"bots=" + strconv.Itoa(st.Bots),
	}, " ")
}
