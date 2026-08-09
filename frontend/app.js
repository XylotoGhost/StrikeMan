// StrikeMan frontend. Talks to the Go backend via the Wails bindings
// (window.go.main.App) and listens for backend log events.

const $ = (id) => document.getElementById(id);
let App; // set once the Wails runtime is ready

// UI state
let mapsData = { official: [], wingman: [], wingmanOnly: [], workshop: [] };
let presets = [];
let lastStatus = null;
let selectedMode = null; // preset id the user intends to play, not what runs
let userPickedMode = false; // user chose a mode; stop following the server's
let userPickedMap = false; // user chose a map; stop following the current one
let paused = false; // no readable pause state in CS2, so track it here
let busyCount = 0;

// ---------- helpers ----------

function toast(msg, isError = false) {
  const el = $("toast");
  el.textContent = msg;
  el.className = "show" + (isError ? " error" : "");
  clearTimeout(el._timer);
  el._timer = setTimeout(() => (el.className = ""), 3500);
}

function logLine(text) {
  const out = $("console-out");
  out.textContent += text.endsWith("\n") ? text : text + "\n";
  out.scrollTop = out.scrollHeight;
}

// Modal yes/no. Returns a promise resolving to true when confirmed.
function confirmAction(title, text) {
  return new Promise((resolve) => {
    const dlg = $("confirm-dialog");
    $("confirm-title").textContent = title;
    $("confirm-text").textContent = text;
    const done = (ok) => {
      dlg.close();
      $("confirm-yes").onclick = null;
      $("confirm-no").onclick = null;
      resolve(ok);
    };
    $("confirm-yes").onclick = () => done(true);
    $("confirm-no").onclick = () => done(false);
    dlg.showModal();
  });
}

// True when people are actually playing — used to ask before disruptive acts.
function humansOnline() {
  return lastStatus && lastStatus.connected && lastStatus.humans > 0;
}

// Confirm only while humans are on the server.
async function confirmIfBusy(title, text) {
  if (!humansOnline()) return true;
  return confirmAction(title, text);
}

// Wrap a backend call: spinner while running, toast on error.
async function call(fn, okMsg) {
  busyCount++;
  $("spinner").classList.remove("hidden");
  try {
    const result = await fn();
    if (okMsg) toast(okMsg);
    return result;
  } catch (err) {
    toast(String(err), true);
    return undefined;
  } finally {
    busyCount--;
    if (busyCount === 0) $("spinner").classList.add("hidden");
  }
}

// A player/match action: run it, then refresh the status right away so the
// effect is visible without waiting for the next poll.
async function action(fn, okMsg) {
  const result = await call(fn, okMsg);
  await refreshStatus();
  return result;
}

// ---------- status polling ----------

function isWingmanMode(st) {
  return st && st.gameType === 0 && st.gameMode === 2;
}

// Which preset best describes what the server is running right now.
function activePresetId(st) {
  if (!st || !st.connected || st.gameType !== 0) return null;
  if (st.gameMode === 1) return "competitive";
  if (st.gameMode === 2) return st.limitTeams === 0 ? "wingman3v3" : "wingman";
  return null;
}

async function refreshStatus() {
  const st = await App.GetStatus().catch(() => null);
  const online = st && st.connected;
  $("status-dot").className = "dot " + (online ? "online" : "offline");
  $("server-name").textContent = online ? st.hostname : (st && st.error ? st.error : "not connected");
  $("server-map").textContent = online && st.map ? st.map : "–";
  $("count-humans").textContent = online ? st.humans : 0;
  $("count-bots").textContent = online ? st.bots : 0;
  renderPlayers(online ? st.players : []);

  const prevMap = lastStatus && lastStatus.map;
  const firstStatus = !lastStatus;
  lastStatus = st;

  // The mode selection follows the server until the user picks one, so it
  // stays right even if the first poll caught the server mid map change.
  const active = activePresetId(st);
  let modeChanged = false;
  if (!userPickedMode && active && active !== selectedMode) {
    selectedMode = active;
    modeChanged = true;
  }
  renderModeButtons();
  renderNowLine();
  syncToggles(st);
  if (online && (st.map !== prevMap || firstStatus)) {
    paused = false; // a map change clears any pause
    updatePauseButton();
    modeChanged = true;
  }
  if (modeChanged) rebuildMapSelect();
  refreshGameState();
}

function renderNowLine() {
  const st = lastStatus;
  if (!st || !st.connected) {
    $("now-line").textContent = "Not connected.";
    return;
  }
  const active = activePresetId(st);
  const p = presets.find((x) => x.id === active);
  const modeName = p ? p.name : `game_mode ${st.gameMode}`;
  const rounds = st.maxRounds ? ` · max ${st.maxRounds} rounds` : "";
  $("now-line").textContent = `Now running: ${modeName} on ${st.map || "?"}${rounds}`;
}

function syncToggles(st) {
  for (const cvar of ["mp_friendlyfire", "mp_overtime_enable", "mp_autokick"]) {
    const el = $("tg-" + cvar);
    const value = st ? st[{ mp_friendlyfire: "friendlyFire", mp_overtime_enable: "overtime", mp_autokick: "autokick" }[cvar]] : -1;
    el.disabled = !st || !st.connected || value < 0;
    if (value >= 0) el.checked = value === 1;
  }
}

function renderPlayers(players) {
  const list = $("player-list");
  $("player-count").textContent = players.length ? `(${players.length})` : "";
  if (!players.length) {
    list.innerHTML = '<li class="empty">nobody on the server</li>';
    return;
  }
  list.innerHTML = "";
  for (const p of players) {
    const li = document.createElement("li");
    const name = document.createElement("span");
    name.className = "name";
    name.textContent = p.name;
    const ping = document.createElement("span");
    ping.className = "ping";
    ping.textContent = p.ping + " ms";
    const kick = document.createElement("button");
    kick.className = "kick";
    kick.textContent = "kick";
    kick.onclick = async () => {
      if (!(await confirmAction("Kick player?", `${p.name} will be disconnected from the server.`))) return;
      action(() => App.KickPlayer(p.userId), `Kicked ${p.name}`);
    };
    li.append(name);
    if (p.bot) {
      const tag = document.createElement("span");
      tag.className = "bot-tag";
      tag.textContent = "BOT";
      li.append(tag);
    }
    li.append(ping, kick);
    list.append(li);
  }
}

// ---------- live score (GSI) ----------

async function refreshGameState() {
  const gs = await App.GetGameState().catch(() => null);
  const live = gs && gs.live;
  $("sb-live").classList.toggle("hidden", !live);
  $("sb-hint").classList.toggle("hidden", !!live);
  $("info-gsi").textContent = live ? "connected" : "waiting for server";
  if (!live) return;
  $("sb-ct").textContent = gs.scoreCt;
  $("sb-t").textContent = gs.scoreT;
  const bits = [];
  if (gs.roundNum) bits.push(`Round ${gs.roundNum + 1}`);
  if (gs.phase) bits.push(gs.phase);
  if (gs.bomb) bits.push("bomb " + gs.bomb);
  $("sb-meta").textContent = bits.join(" · ");
}

// ---------- maps & modes ----------

async function refreshMaps() {
  const maps = await call(() => App.GetMaps());
  if (!maps) return;
  mapsData = maps;
  rebuildMapSelect();
}

function wantsWingman() {
  const p = presets.find((x) => x.id === selectedMode);
  return !!(p && p.wingman);
}

// Annotation next to an official map name — only says what the current
// filter does not already imply.
function mapNote(name, wingmanView) {
  if (wingmanView) {
    return mapsData.wingmanOnly.includes(name) ? " · wingman only" : "";
  }
  return mapsData.wingman.includes(name) ? " · also wingman" : "";
}

// The dropdown follows the *selected* mode, so wingman maps are visible
// while competitive is still running (and the other way round).
function rebuildMapSelect() {
  const sel = $("map-select");
  const previous = sel.value;
  const wingman = wantsWingman();
  const current = lastStatus ? lastStatus.map : "";
  sel.innerHTML = "";

  const official = mapsData.official.filter((m) =>
    wingman ? mapsData.wingman.includes(m) : !mapsData.wingmanOnly.includes(m)
  );
  $("map-filter-hint").textContent = wingman
    ? "Showing maps that support Wingman."
    : "Showing maps that support Competitive.";

  const officials = document.createElement("optgroup");
  officials.label = "Official";
  for (const name of official) {
    const label = name + mapNote(name, wingman) + (name === current ? "  ● current" : "");
    officials.append(new Option(label, name));
  }
  sel.append(officials);

  const workshop = mapsData.workshop.filter(
    (m) => !wingman || !(m.tags || []).length || m.tags.some((t) => t.toLowerCase() === "wingman")
  );
  if (workshop.length) {
    const ws = document.createElement("optgroup");
    ws.label = "Workshop";
    for (const m of workshop) {
      const untagged = wingman && !(m.tags || []).length;
      ws.append(new Option(m.title + (untagged ? " · wingman unknown" : ""), "ws:" + m.id));
    }
    sel.append(ws);
  }

  // Keep the current map visible even when the filter would hide it.
  if (current && !official.includes(current)) {
    sel.append(new Option(current + "  ● current", current));
  }

  if (userPickedMap && [...sel.options].some((o) => o.value === previous)) {
    sel.value = previous;
  } else {
    sel.value = current || (sel.options[0] ? sel.options[0].value : "");
    userPickedMap = false;
  }
}

function selectedMap() {
  const v = $("map-select").value;
  if (v.startsWith("ws:")) return { ref: v.slice(3), workshop: true };
  return { ref: v, workshop: false };
}

function renderModeButtons() {
  const row = $("mode-buttons");
  const active = activePresetId(lastStatus);
  if (!row.childElementCount) {
    for (const p of presets) {
      const btn = document.createElement("button");
      btn.className = "preset";
      btn.dataset.id = p.id;
      btn.innerHTML = `<b>${p.name}</b><small>${p.description}</small>`;
      btn.onclick = () => {
        selectedMode = p.id;
        userPickedMode = true;
        renderModeButtons();
        rebuildMapSelect();
      };
      row.append(btn);
    }
  }
  for (const btn of row.children) {
    btn.classList.toggle("selected", btn.dataset.id === selectedMode);
    btn.classList.toggle("active", btn.dataset.id === active);
  }
}

async function applyMode() {
  const m = selectedMap();
  if (!selectedMode) return toast("Pick a mode first", true);
  if (!m.ref) return toast("Pick a map first", true);
  const p = presets.find((x) => x.id === selectedMode);
  if (!(await confirmIfBusy("Apply mode and load map?", `${lastStatus.humans} player(s) are on the server. Applying ${p.name} reloads the map and interrupts the current match.`)))
    return;
  const ok = await call(() => App.ApplyPreset(selectedMode, m.ref, m.workshop), `${p.name} — reloading map…`);
  if (ok !== undefined) userPickedMap = false;
}

// ---------- match controls ----------

function updatePauseButton() {
  $("btn-pause").textContent = paused ? "▶ Resume match" : "⏸ Pause match";
}

async function togglePause() {
  if (paused) {
    const ok = await action(() => App.Unpause(), "Match resumed");
    if (ok !== undefined) paused = false;
  } else {
    const ok = await action(() => App.Pause(), "Match pauses at the end of this round");
    if (ok !== undefined) paused = true;
  }
  updatePauseButton();
}

// ---------- settings ----------

let editCfg = { servers: [], default: "", gsiPort: 3838 };
let editIdx = -1;

function renderServerList() {
  const list = $("srv-list");
  list.innerHTML = "";
  editCfg.servers.forEach((s, i) => {
    const label = s.name + (s.name === editCfg.default ? "  ★" : "");
    list.append(new Option(label, i));
  });
  list.value = editIdx;
}

function showServerFields() {
  const s = editCfg.servers[editIdx];
  const has = !!s;
  for (const id of ["cfg-name", "cfg-host", "cfg-port", "cfg-password", "cfg-collection", "cfg-default"])
    $(id).disabled = !has;
  if (!has) return;
  $("cfg-name").value = s.name || "";
  $("cfg-host").value = s.host || "";
  $("cfg-port").value = s.port || 27015;
  $("cfg-password").value = s.password || "";
  $("cfg-collection").value = s.collectionId || "";
  $("cfg-default").checked = s.name === editCfg.default;
}

async function openSettings() {
  const cfg = await App.GetConfig();
  editCfg = {
    servers: (cfg.servers || []).map((s) => ({ ...s })),
    default: cfg.default || "",
    gsiPort: cfg.gsiPort || 3838,
  };
  if (!editCfg.servers.length) addServer();
  editIdx = 0;
  renderServerList();
  showServerFields();
  $("cfg-gsiport").value = editCfg.gsiPort;
  const setup = await App.GetGSISetup().catch(() => null);
  if (setup) {
    $("gsi-path").textContent = setup.path + setup.fileName;
    $("gsi-content").textContent = setup.content;
  }
  $("settings-dialog").showModal();
}

function addServer() {
  editCfg.servers.push({ name: "New server", host: "", port: 27015, password: "", collectionId: "" });
  if (editCfg.servers.length === 1) editCfg.default = "New server";
  editIdx = editCfg.servers.length - 1;
  renderServerList();
  showServerFields();
}

function removeServer() {
  if (editIdx < 0) return;
  const removed = editCfg.servers.splice(editIdx, 1)[0];
  if (removed && removed.name === editCfg.default) {
    editCfg.default = editCfg.servers[0] ? editCfg.servers[0].name : "";
  }
  editIdx = Math.min(editIdx, editCfg.servers.length - 1);
  renderServerList();
  showServerFields();
}

async function saveSettings() {
  const names = editCfg.servers.map((s) => s.name.trim());
  if (names.some((n) => !n)) return toast("Every server needs a name", true);
  if (new Set(names).size !== names.length) return toast("Server names must be unique", true);
  if (!editCfg.default && names.length) editCfg.default = names[0];
  editCfg.gsiPort = parseInt($("cfg-gsiport").value, 10) || 0;
  await call(() => App.SaveConfig(editCfg), "Settings saved");
  await refreshServerSelect();
  userPickedMap = false;
  userPickedMode = false;
  selectedMode = null;
  await refreshStatus();
  await refreshMaps();
}

async function refreshServerSelect() {
  const cfg = await App.GetConfig();
  const active = await App.GetActiveServer();
  const sel = $("server-select");
  sel.innerHTML = "";
  for (const s of cfg.servers || []) sel.append(new Option(s.name, s.name));
  sel.value = active;
  sel.classList.toggle("hidden", (cfg.servers || []).length < 2);
}

// ---------- server info ----------

function formatUptime(secs) {
  if (!secs) return "–";
  const d = Math.floor(secs / 86400);
  const h = Math.floor((secs % 86400) / 3600);
  const m = Math.floor((secs % 3600) / 60);
  return d ? `${d}d ${h}h` : h ? `${h}h ${m}m` : `${m}m`;
}

async function refreshServerInfo() {
  const info = await call(() => App.GetServerInfo());
  if (!info) return;
  $("info-version").textContent = info.version ? `${info.version} (build ${info.build})` : "–";
  $("info-uptime").textContent = formatUptime(info.uptimeSecs);
  const el = $("info-update");
  if (info.checkFailed) {
    el.textContent = "check failed";
    el.className = "";
  } else if (info.upToDate) {
    el.textContent = "up to date";
    el.className = "ok";
  } else {
    el.textContent = `update available (build ${info.latest})`;
    el.className = "warn";
  }
}

// ---------- wiring ----------

function wireEvents() {
  $("btn-settings").onclick = openSettings;
  $("btn-save-settings").onclick = saveSettings;
  $("btn-srv-add").onclick = addServer;
  $("btn-srv-remove").onclick = removeServer;

  $("srv-list").onchange = () => {
    editIdx = parseInt($("srv-list").value, 10);
    showServerFields();
  };
  $("cfg-name").oninput = () => {
    const s = editCfg.servers[editIdx];
    if (!s) return;
    if (s.name === editCfg.default) editCfg.default = $("cfg-name").value;
    s.name = $("cfg-name").value;
    renderServerList();
  };
  $("cfg-host").oninput = () => (editCfg.servers[editIdx].host = $("cfg-host").value.trim());
  $("cfg-port").oninput = () => (editCfg.servers[editIdx].port = parseInt($("cfg-port").value, 10) || 27015);
  $("cfg-password").oninput = () => (editCfg.servers[editIdx].password = $("cfg-password").value);
  $("cfg-collection").oninput = () => (editCfg.servers[editIdx].collectionId = $("cfg-collection").value.trim());
  $("cfg-default").onchange = () => {
    if ($("cfg-default").checked) editCfg.default = editCfg.servers[editIdx].name;
    else if (editCfg.default === editCfg.servers[editIdx].name) editCfg.default = "";
    renderServerList();
  };

  $("server-select").onchange = async () => {
    await call(() => App.SelectServer($("server-select").value));
    userPickedMap = false;
    userPickedMode = false;
    selectedMode = null;
    lastStatus = null;
    paused = false;
    await refreshStatus();
    await refreshMaps();
  };

  $("map-select").onchange = () => (userPickedMap = true);
  $("btn-apply").onclick = applyMode;
  $("btn-changemap").onclick = async () => {
    const m = selectedMap();
    if (!m.ref) return toast("Pick a map first", true);
    if (!(await confirmIfBusy("Change map?", `${lastStatus.humans} player(s) are on the server and will be moved to the new map.`)))
      return;
    const ok = await call(() => App.ChangeMap(m.ref, m.workshop), "Changing map…");
    if (ok !== undefined) userPickedMap = false;
  };
  $("btn-refreshmaps").onclick = refreshMaps;

  $("btn-warmup-start").onclick = () =>
    action(() => App.StartWarmup(parseInt($("warmup-seconds").value, 10) || 120), "Warmup started");
  $("btn-warmup-end").onclick = async () => {
    if (!(await confirmIfBusy("End warmup and go live?", "The match starts immediately for everyone on the server."))) return;
    action(() => App.EndWarmup(), "Warmup ended — match is live");
  };
  $("btn-pause").onclick = togglePause;
  $("btn-restart").onclick = async () => {
    if (!(await confirmAction("Restart the match?", "This resets the score to 0:0 and starts the match from round 1."))) return;
    action(() => App.RestartRound(), "Match restarted");
  };

  for (const cvar of ["mp_friendlyfire", "mp_overtime_enable", "mp_autokick"]) {
    $("tg-" + cvar).onchange = (e) => action(() => App.SetToggle(cvar, e.target.checked));
  }

  const announce = () => {
    const msg = $("announce-in").value.trim();
    if (!msg) return;
    $("announce-in").value = "";
    call(() => App.Announce(msg), "Announced");
  };
  $("btn-announce").onclick = announce;
  $("announce-in").addEventListener("keydown", (e) => e.key === "Enter" && announce());

  $("btn-swap").onclick = async () => {
    if (!(await confirmIfBusy("Swap sides?", "CT and T swap immediately, mid-match."))) return;
    action(() => App.SwapTeams(), "Teams swapped");
  };
  $("btn-scramble").onclick = async () => {
    if (!(await confirmIfBusy("Scramble teams?", "Players are randomly redistributed across both teams."))) return;
    action(() => App.ScrambleTeams(), "Teams scrambled");
  };
  $("btn-teamnames").onclick = () =>
    call(() => App.SetTeamNames($("teamname-ct").value, $("teamname-t").value), "Team names set");
  $("btn-bot-ct").onclick = () => action(() => App.AddBot("ct"));
  $("btn-bot-t").onclick = () => action(() => App.AddBot("t"));
  $("btn-bot-kick").onclick = () => action(() => App.KickBots(), "Bots kicked");

  $("btn-refreshinfo").onclick = refreshServerInfo;
  $("btn-clearlog").onclick = () => ($("console-out").textContent = "");

  $("console-in").addEventListener("keydown", async (e) => {
    if (e.key !== "Enter") return;
    const cmd = e.target.value.trim();
    if (!cmd) return;
    e.target.value = "";
    logLine("> " + cmd);
    const out = await call(() => App.RunCommand(cmd));
    if (out !== undefined) logLine(out || "(no output)");
  });

  window.runtime.EventsOn("log", logLine);
  window.runtime.EventsOn("warn", (msg) => {
    toast(msg, true);
    logLine("⚠ " + msg);
  });
}

async function init() {
  // Wait until the Wails runtime has injected the bindings.
  while (!window.go || !window.runtime) {
    await new Promise((r) => setTimeout(r, 50));
  }
  App = window.go.main.App;
  presets = await App.GetPresets();
  wireEvents();
  renderModeButtons();
  updatePauseButton();
  await refreshServerSelect();

  const cfg = await App.GetConfig();
  if (!(cfg.servers || []).length) {
    openSettings();
  } else {
    await refreshStatus();
    await refreshMaps();
    refreshServerInfo();
  }
  setInterval(refreshStatus, 5000);
}

init();
