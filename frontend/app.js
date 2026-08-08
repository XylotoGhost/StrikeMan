// StrikeMan frontend. Talks to the Go backend via the Wails bindings
// (window.go.main.App) and listens for backend log events.

const $ = (id) => document.getElementById(id);
let App; // set once the Wails runtime is ready

// ---------- helpers ----------

function toast(msg, isError = false) {
  const el = $("toast");
  el.textContent = msg;
  el.className = "show" + (isError ? " error" : "");
  clearTimeout(el._timer);
  el._timer = setTimeout(() => (el.className = ""), 3000);
}

function logLine(text) {
  const out = $("console-out");
  out.textContent += text.endsWith("\n") ? text : text + "\n";
  out.scrollTop = out.scrollHeight;
}

// Wrap a backend call: toast on error, optional success message.
async function call(fn, okMsg) {
  try {
    const result = await fn();
    if (okMsg) toast(okMsg);
    return result;
  } catch (err) {
    toast(String(err), true);
    return undefined;
  }
}

// ---------- status polling ----------

async function refreshStatus() {
  const st = await App.GetStatus().catch(() => null);
  const online = st && st.connected;
  $("status-dot").className = "dot " + (online ? "online" : "offline");
  $("server-name").textContent = online ? st.hostname : "not connected";
  $("server-map").textContent = online ? st.map : "";
  $("server-players").textContent = online ? st.humans : "";
  renderPlayers(online ? st.players : []);
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
    kick.onclick = () => call(() => App.KickPlayer(p.userId), `Kicked ${p.name}`);
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
  const sel = $("map-select");
  sel.innerHTML = '<option value="">— current map —</option>';
  const officials = document.createElement("optgroup");
  officials.label = "Official";
  for (const name of maps.official) {
    officials.append(new Option(name, name));
  }
  sel.append(officials);
  if (maps.workshop.length) {
    const ws = document.createElement("optgroup");
    ws.label = "Workshop";
    for (const m of maps.workshop) {
      ws.append(new Option(m.title, "ws:" + m.id));
    }
    sel.append(ws);
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
    btn.innerHTML = `<b>${p.name}</b><small>${p.description}</small>`;
    btn.onclick = () => {
      const m = selectedMap();
      call(() => App.ApplyPreset(p.id, m.ref, m.workshop), `${p.name} — reloading map…`);
    };
    row.append(btn);
  }
}

// ---------- settings ----------

async function openSettings() {
  const cfg = await App.GetConfig();
  $("cfg-host").value = cfg.host || "";
  $("cfg-port").value = cfg.port || 27015;
  $("cfg-password").value = cfg.password || "";
  $("cfg-collection").value = cfg.collectionId || "";
  $("settings-dialog").showModal();
}

async function saveSettings() {
  const cfg = {
    host: $("cfg-host").value.trim(),
    port: parseInt($("cfg-port").value, 10) || 27015,
    password: $("cfg-password").value,
    collectionId: $("cfg-collection").value.trim(),
  };
  await call(() => App.SaveConfig(cfg), "Settings saved");
  await refreshStatus();
  await refreshMaps();
}

// ---------- wiring ----------

function wireEvents() {
  $("btn-settings").onclick = openSettings;
  $("btn-save-settings").onclick = saveSettings;

  $("btn-changemap").onclick = () => {
    const m = selectedMap();
    if (!m.ref) return toast("Pick a map first", true);
    call(() => App.ChangeMap(m.ref, m.workshop), "Changing map…");
  };
  $("btn-refreshmaps").onclick = refreshMaps;

  $("btn-warmup-start").onclick = () =>
    call(() => App.StartWarmup(parseInt($("warmup-seconds").value, 10) || 120), "Warmup started");
  $("btn-warmup-end").onclick = () => call(() => App.EndWarmup(), "Warmup ended");
  $("btn-pause").onclick = () => call(() => App.Pause(), "Match pauses at end of round/freezetime");
  $("btn-unpause").onclick = () => call(() => App.Unpause(), "Match unpaused");
  $("btn-restart").onclick = () => call(() => App.RestartRound(), "Round restarted");

  $("btn-swap").onclick = () => call(() => App.SwapTeams(), "Teams swapped");
  $("btn-scramble").onclick = () => call(() => App.ScrambleTeams(), "Teams scrambled");
  $("btn-teamnames").onclick = () =>
    call(() => App.SetTeamNames($("teamname-ct").value, $("teamname-t").value), "Team names set");
  $("btn-bot-ct").onclick = () => call(() => App.AddBot("ct"));
  $("btn-bot-t").onclick = () => call(() => App.AddBot("t"));
  $("btn-bot-kick").onclick = () => call(() => App.KickBots(), "Bots kicked");

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
}

async function init() {
  // Wait until the Wails runtime has injected the bindings.
  while (!window.go || !window.runtime) {
    await new Promise((r) => setTimeout(r, 50));
  }
  App = window.go.main.App;
  wireEvents();
  await loadPresets();

  const cfg = await App.GetConfig();
  if (!cfg.password) {
    openSettings();
  } else {
    await refreshStatus();
    await refreshMaps();
  }
  setInterval(refreshStatus, 5000);
}

init();
