// StrikeMan frontend. Talks to the Go backend via the Wails bindings
// (window.go.main.App) and listens for backend log events.

const $ = (id) => document.getElementById(id);
let App; // set once the Wails runtime is ready

// UI state
let mapsData = { official: [], wingman: [], workshop: [] };
let lastStatus = null;
let userPickedMap = false; // user chose a map; stop following the current one
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

function isWingman(st) {
  return st && st.gameType === 0 && st.gameMode === 2;
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
  highlightPreset(online ? st : null);

  const modeChanged = !lastStatus || !st || isWingman(lastStatus) !== isWingman(st);
  const mapChanged = (lastStatus && lastStatus.map) !== (st && st.map);
  lastStatus = st;
  if (online && (modeChanged || mapChanged)) rebuildMapSelect();
}

function highlightPreset(st) {
  document.querySelectorAll(".preset").forEach((btn) => {
    let active = false;
    if (st && st.gameType === 0) {
      if (btn.dataset.id === "competitive") active = st.gameMode === 1;
      if (btn.dataset.id === "wingman") active = st.gameMode === 2 && st.limitTeams !== 0;
      if (btn.dataset.id === "wingman3v3") active = st.gameMode === 2 && st.limitTeams === 0;
    }
    btn.classList.toggle("active", active);
  });
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
    kick.onclick = () => action(() => App.KickPlayer(p.userId), `Kicked ${p.name}`);
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

// ---------- maps & presets ----------

async function refreshMaps() {
  const maps = await call(() => App.GetMaps());
  if (!maps) return;
  mapsData = maps;
  rebuildMapSelect();
}

// Rebuilds the dropdown: official maps (filtered to wingman maps while a
// wingman mode is active) + workshop maps. Unless the user picked a map
// themselves, the selection follows the server's current map.
function rebuildMapSelect() {
  const sel = $("map-select");
  const previous = sel.value;
  const wingmanActive = isWingman(lastStatus);
  const current = lastStatus ? lastStatus.map : "";
  sel.innerHTML = "";

  let official = mapsData.official;
  if (wingmanActive) {
    official = official.filter((m) => mapsData.wingman.includes(m));
  }
  $("map-filter-hint").textContent = wingmanActive
    ? "Wingman is active — showing wingman maps only."
    : "";

  const officials = document.createElement("optgroup");
  officials.label = "Official";
  for (const name of official) {
    officials.append(new Option(name === current ? name + "  ● current" : name, name));
  }
  sel.append(officials);

  if (mapsData.workshop.length) {
    const ws = document.createElement("optgroup");
    ws.label = "Workshop";
    for (const m of mapsData.workshop) {
      const opt = new Option(m.title, "ws:" + m.id);
      const noWingman = (m.tags || []).length && !m.tags.some((t) => t.toLowerCase() === "wingman");
      if (wingmanActive && noWingman) {
        opt.disabled = true;
        opt.text += "  (no wingman)";
      }
      ws.append(opt);
    }
    sel.append(ws);
  }

  // Current map not in the list (e.g. a workshop map or filtered out).
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

async function loadPresets() {
  const presets = await App.GetPresets();
  const row = $("preset-buttons");
  row.innerHTML = "";
  for (const p of presets) {
    const btn = document.createElement("button");
    btn.className = "preset";
    btn.dataset.id = p.id;
    btn.innerHTML = `<b>${p.name}</b><small>${p.description}</small>`;
    btn.onclick = async () => {
      const m = selectedMap();
      const ok = await call(() => App.ApplyPreset(p.id, m.ref, m.workshop), `${p.name} — reloading map…`);
      if (ok !== undefined) userPickedMap = false;
    };
    row.append(btn);
  }
}

// ---------- settings ----------

let editCfg = { servers: [], default: "" };
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
  editCfg = { servers: (cfg.servers || []).map((s) => ({ ...s })), default: cfg.default || "" };
  if (!editCfg.servers.length) addServer();
  editIdx = 0;
  renderServerList();
  showServerFields();
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
  await call(() => App.SaveConfig(editCfg), "Settings saved");
  await refreshServerSelect();
  userPickedMap = false;
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
    lastStatus = null;
    await refreshStatus();
    await refreshMaps();
  };

  $("map-select").onchange = () => (userPickedMap = true);
  $("btn-changemap").onclick = async () => {
    const m = selectedMap();
    if (!m.ref) return toast("Pick a map first", true);
    const ok = await call(() => App.ChangeMap(m.ref, m.workshop), "Changing map…");
    if (ok !== undefined) userPickedMap = false;
  };
  $("btn-refreshmaps").onclick = refreshMaps;

  $("btn-warmup-start").onclick = () =>
    action(() => App.StartWarmup(parseInt($("warmup-seconds").value, 10) || 120), "Warmup started");
  $("btn-warmup-end").onclick = () => action(() => App.EndWarmup(), "Warmup ended");
  $("btn-pause").onclick = () => action(() => App.Pause(), "Match pauses at end of round/freezetime");
  $("btn-unpause").onclick = () => action(() => App.Unpause(), "Match unpaused");
  $("btn-restart").onclick = () => action(() => App.RestartRound(), "Round restarted");

  $("btn-swap").onclick = () => action(() => App.SwapTeams(), "Teams swapped");
  $("btn-scramble").onclick = () => action(() => App.ScrambleTeams(), "Teams scrambled");
  $("btn-teamnames").onclick = () =>
    call(() => App.SetTeamNames($("teamname-ct").value, $("teamname-t").value), "Team names set");
  $("btn-bot-ct").onclick = () => action(() => App.AddBot("ct"));
  $("btn-bot-t").onclick = () => action(() => App.AddBot("t"));
  $("btn-bot-kick").onclick = () => action(() => App.KickBots(), "Bots kicked");

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
  wireEvents();
  await loadPresets();
  await refreshServerSelect();

  const cfg = await App.GetConfig();
  if (!(cfg.servers || []).length) {
    openSettings();
  } else {
    await refreshStatus();
    await refreshMaps();
  }
  setInterval(refreshStatus, 5000);
}

init();
