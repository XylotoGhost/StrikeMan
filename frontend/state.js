// Application state and everything that talks to the Go backend.
// Deliberately free of DOM code: views.js renders what lives here.

export const api = {
  /** Resolves once Wails has injected the bindings. */
  async ready() {
    while (!window.go || !window.runtime) {
      await new Promise((r) => setTimeout(r, 50));
    }
    this.app = window.go.main.App;
    this.runtime = window.runtime;
    return this.app;
  },
};

export const state = {
  presets: [],
  toggles: [],
  maps: { official: [], wingman: [], wingmanOnly: [], workshop: [] },
  status: null,
  server: null, // active server view: sticky flags, warmup length

  selectedMode: null, // the mode you intend to play, not what runs
  userPickedMode: false, // stop following the server once you choose
  userPickedMap: false,

  // CS2 reports neither pause nor warmup state over RCON, so both are
  // tracked here from what StrikeMan itself did.
  paused: false,
  warmupEndsAt: null,
};

/** Resets the per-server bits when switching servers or saving settings. */
export function resetForServer() {
  state.status = null;
  state.selectedMode = null;
  state.userPickedMode = false;
  state.userPickedMap = false;
  state.paused = false;
  state.warmupEndsAt = null;
}

// ---- Derived values ----

export function isWingmanIntent() {
  const preset = state.presets.find((p) => p.id === state.selectedMode);
  return !!(preset && preset.wingman);
}

/** Which preset describes what the server is running right now. */
export function activePresetId(status = state.status) {
  if (!status || !status.connected || status.gameType !== 0) return null;
  const toggles = status.toggles || {};
  if (status.gameMode === 1) {
    // Premier and Competitive both run game_mode 1 and differ only in
    // whether a 12:12 is played out.
    return toggles.overtime === 1 ? "premier" : "competitive";
  }
  if (status.gameMode === 2) {
    return status.limitTeams === 0 ? "wingman3v3" : "wingman";
  }
  return null;
}

export function presetById(id) {
  return state.presets.find((p) => p.id === id) || null;
}

export function humansOnline() {
  return !!(state.status && state.status.connected && state.status.humans > 0);
}

export function warmupRunning() {
  return state.warmupEndsAt !== null && state.warmupEndsAt > Date.now();
}

export function warmupSecondsLeft() {
  if (!warmupRunning()) return 0;
  // Floor, not round: the game counts down the whole second it is in, so
  // rounding up showed one second more than the in-game timer.
  return Math.floor((state.warmupEndsAt - Date.now()) / 1000);
}

/** Maps offered for the selected mode, already filtered. */
export function visibleMaps() {
  const wingman = isWingmanIntent();
  const official = state.maps.official.filter((name) =>
    wingman
      ? state.maps.wingman.includes(name)
      : !state.maps.wingmanOnly.includes(name)
  );
  const workshop = state.maps.workshop.filter(
    (m) =>
      !wingman ||
      !(m.tags || []).length ||
      m.tags.some((tag) => tag.toLowerCase() === "wingman")
  );
  return { official, workshop, wingman };
}

// ---- Backend refreshes ----

export async function refreshStatus() {
  const [server, status] = await Promise.all([
    api.app.GetActiveServerConfig().catch(() => null),
    api.app.GetStatus().catch(() => null),
  ]);
  state.server = server;

  const previous = state.status;
  const mapChanged = (previous && previous.map) !== (status && status.map);
  const first = !previous;
  state.status = status;

  // Follow the server's mode until the user picks one, so the selection is
  // still right if the first poll caught a map change in progress.
  let modeChanged = false;
  const active = activePresetId(status);
  if (!state.userPickedMode && active && active !== state.selectedMode) {
    state.selectedMode = active;
    modeChanged = true;
  }

  if (status && status.connected && (mapChanged || first)) {
    state.paused = false; // a map change clears any pause
    modeChanged = true;
    // A map change restarts warmup. When StrikeMan triggered it, the backend
    // reports the exact length (the "warmup" event); this covers a map change
    // made elsewhere, where CS2 may still shorten it.
    if (!first && status.warmupOnline === 1 && status.humans > 0 && status.warmupTime > 0) {
      state.warmupEndsAt = Date.now() + status.warmupTime * 1000;
    } else {
      state.warmupEndsAt = null;
    }
  }
  return { modeChanged };
}

export async function refreshMaps() {
  state.maps = await api.app.GetMaps();
  return state.maps;
}
