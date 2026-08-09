// Wiring: connects the UI controls to the backend and keeps the views in
// sync with the polled state.

import {
  t,
  setLanguage,
  applyTranslations,
  translateError,
  getLanguage,
} from "./i18n.js";
import {
  api,
  state,
  resetForServer,
  refreshStatus,
  refreshMaps,
  humansOnline,
  presetById,
  warmupRunning,
} from "./state.js";
import * as ui from "./views.js";

const POLL_INTERVAL = 5000;

// ---- Backend call helpers ----

/** Runs a backend call with the spinner on, turning failures into a toast. */
async function call(fn, okKey, okArgs = []) {
  ui.setBusy(true);
  try {
    const result = await fn();
    if (okKey) ui.toast(t(okKey, okArgs));
    return { ok: true, result };
  } catch (err) {
    ui.toastError(err);
    return { ok: false };
  } finally {
    ui.setBusy(false);
  }
}

/** A call whose effect should show up immediately, not at the next poll. */
async function action(fn, okKey, okArgs = []) {
  const outcome = await call(fn, okKey, okArgs);
  await poll();
  return outcome;
}

/** Only ask for confirmation while people are actually playing. */
async function confirmIfBusy(titleKey, textKey, extraArgs = []) {
  if (!humansOnline()) return true;
  return ui.confirmAction(titleKey, textKey, [state.status.humans, ...extraArgs]);
}

// ---- Rendering ----

function renderAll({ rebuildMaps = false } = {}) {
  ui.renderHeader();
  ui.renderNowLine();
  ui.renderModeButtons(onPickMode);
  ui.renderToggles(onToggle);
  ui.renderPlayers(onKickPlayer);
  ui.renderWarmupButton();
  ui.renderPauseButton();
  if (rebuildMaps) ui.renderMapSelect();
}

// ---- Polling ----

let pollTimer = null;

/** One status refresh. Self-scheduling so polls can never overlap. */
async function poll() {
  try {
    const { modeChanged } = await refreshStatus();
    renderAll({ rebuildMaps: modeChanged });
  } catch (err) {
    ui.toastError(err);
  }
}

function schedulePolling() {
  clearTimeout(pollTimer);
  const loop = async () => {
    await poll();
    pollTimer = setTimeout(loop, POLL_INTERVAL);
  };
  pollTimer = setTimeout(loop, POLL_INTERVAL);
}

// ---- Handlers ----

function onPickMode(id) {
  state.selectedMode = id;
  state.userPickedMode = true;
  ui.renderModeButtons(onPickMode);
  ui.renderMapSelect();
}

async function onToggle(id, on) {
  await action(() => api.app.SetToggle(id, on));
}

async function onKickPlayer(player) {
  const ok = await ui.confirmAction("confirm.kickTitle", "confirm.kickText", [player.name]);
  if (!ok) return;
  await action(() => api.app.KickPlayer(player.userId), "toast.kicked", [player.name]);
}

async function onApplyPreset() {
  const map = ui.selectedMap();
  if (!state.selectedMode) return ui.toast(t("toast.pickMode"), true);
  if (!map.ref) return ui.toast(t("toast.pickMap"), true);
  const preset = presetById(state.selectedMode);
  const name = t(preset.nameKey);
  if (!(await confirmIfBusy("confirm.applyTitle", "confirm.applyText", [name]))) return;
  const { ok } = await call(
    () => api.app.ApplyPreset(state.selectedMode, map.ref, map.workshop),
    "toast.applying",
    [name]
  );
  if (ok) state.userPickedMap = false;
}

async function onChangeMap() {
  const map = ui.selectedMap();
  if (!map.ref) return ui.toast(t("toast.pickMap"), true);
  if (!(await confirmIfBusy("confirm.mapTitle", "confirm.mapText"))) return;
  const { ok } = await call(
    () => api.app.ChangeMap(map.ref, map.workshop),
    "toast.changingMap"
  );
  if (ok) state.userPickedMap = false;
}

async function onToggleWarmup() {
  if (warmupRunning()) {
    const { ok } = await action(() => api.app.EndWarmup(), "toast.warmupEnded");
    if (ok) state.warmupEndsAt = null;
  } else {
    if (!(await confirmIfBusy("confirm.warmupTitle", "confirm.warmupText"))) return;
    const seconds = parseInt(ui.$("warmup-seconds").value, 10) || 120;
    await action(() => api.app.StartWarmup(seconds), "toast.warmupStarted");
    // warmupEndsAt comes from the backend's "warmup" event.
  }
  ui.renderWarmupButton();
}

async function onTogglePause() {
  if (state.paused) {
    const { ok } = await action(() => api.app.Unpause(), "toast.resumed");
    if (ok) state.paused = false;
  } else {
    const { ok } = await action(() => api.app.Pause(), "toast.paused");
    if (ok) state.paused = true;
  }
  ui.renderPauseButton();
}

async function onRefreshServerInfo() {
  const { ok, result } = await call(() => api.app.GetServerInfo());
  if (ok) ui.renderServerInfo(result);
}

async function onSelectServer(name) {
  await call(() => api.app.SelectServer(name));
  resetForServer();
  await poll();
  await loadMaps();
}

async function loadMaps() {
  const { ok } = await call(() => refreshMaps());
  if (ok) ui.renderMapSelect();
}

// ---- Settings dialog ----

let draft = { servers: [], default: "" };
let draftIndex = -1;

async function openSettings() {
  const config = await api.app.GetConfig();
  draft = {
    servers: (config.servers || []).map((s) => ({ ...s })),
    default: config.default || "",
  };
  if (!draft.servers.length) addServer();
  draftIndex = 0;
  ui.renderSettingsList(draft, draftIndex);
  ui.renderSettingsFields(draft, draftIndex);
  ui.renderLanguageSelect(await api.app.GetLanguage());
  ui.$("settings-dialog").showModal();
}

function addServer() {
  draft.servers.push({
    name: t("settings.newServer"),
    host: "",
    port: 27015,
    password: "",
    collectionId: "",
  });
  if (draft.servers.length === 1) draft.default = draft.servers[0].name;
  draftIndex = draft.servers.length - 1;
  ui.renderSettingsList(draft, draftIndex);
  ui.renderSettingsFields(draft, draftIndex);
}

function removeServer() {
  if (draftIndex < 0) return;
  const [removed] = draft.servers.splice(draftIndex, 1);
  if (removed && removed.name === draft.default) {
    draft.default = draft.servers[0] ? draft.servers[0].name : "";
  }
  draftIndex = Math.min(draftIndex, draft.servers.length - 1);
  ui.renderSettingsList(draft, draftIndex);
  ui.renderSettingsFields(draft, draftIndex);
}

async function saveSettings() {
  const names = draft.servers.map((s) => s.name.trim());
  if (names.some((n) => !n)) return ui.toast(t("settings.nameRequired"), true);
  if (new Set(names).size !== names.length) {
    return ui.toast(t("settings.nameUnique"), true);
  }
  if (!draft.default && names.length) draft.default = names[0];

  const language = ui.$("cfg-language").value;
  await call(() => api.app.SetLanguage(language));
  setLanguage(language);
  applyTranslations();

  await call(() => api.app.SaveConfig(draft), "settings.saved");
  resetForServer();
  await refreshServerSelect();
  await poll();
  await loadMaps();
  renderAll({ rebuildMaps: true });
}

async function refreshServerSelect() {
  const [config, active] = await Promise.all([
    api.app.GetConfig(),
    api.app.GetActiveServer(),
  ]);
  ui.renderServerSelect(config, active);
}

// ---- Wiring ----

function bindField(id, apply) {
  ui.$(id).oninput = () => {
    const server = draft.servers[draftIndex];
    if (server) apply(server, ui.$(id).value);
  };
}

function wireEvents() {
  ui.$("btn-settings").onclick = openSettings;
  ui.$("btn-save-settings").onclick = saveSettings;
  ui.$("btn-srv-add").onclick = addServer;
  ui.$("btn-srv-remove").onclick = removeServer;

  ui.$("srv-list").onchange = () => {
    draftIndex = parseInt(ui.$("srv-list").value, 10);
    ui.renderSettingsFields(draft, draftIndex);
  };
  ui.$("cfg-name").oninput = () => {
    const server = draft.servers[draftIndex];
    if (!server) return;
    if (server.name === draft.default) draft.default = ui.$("cfg-name").value;
    server.name = ui.$("cfg-name").value;
    ui.renderSettingsList(draft, draftIndex);
  };
  bindField("cfg-host", (s, v) => (s.host = v.trim()));
  bindField("cfg-port", (s, v) => (s.port = parseInt(v, 10) || 27015));
  bindField("cfg-password", (s, v) => (s.password = v));
  bindField("cfg-collection", (s, v) => (s.collectionId = v.trim()));
  ui.$("cfg-default").onchange = () => {
    const server = draft.servers[draftIndex];
    if (!server) return;
    if (ui.$("cfg-default").checked) draft.default = server.name;
    else if (draft.default === server.name) draft.default = "";
    ui.renderSettingsList(draft, draftIndex);
  };
  ui.$("cfg-sticky").onchange = () => {
    const server = draft.servers[draftIndex];
    if (server) server.stickyAdmin = ui.$("cfg-sticky").checked;
  };

  ui.$("server-select").onchange = (e) => onSelectServer(e.target.value);

  ui.$("map-select").onchange = () => (state.userPickedMap = true);
  ui.$("btn-apply").onclick = onApplyPreset;
  ui.$("btn-changemap").onclick = onChangeMap;
  ui.$("btn-refreshmaps").onclick = loadMaps;

  ui.$("btn-warmup").onclick = onToggleWarmup;
  ui.$("warmup-seconds").onchange = () =>
    call(() => api.app.SetWarmupSeconds(parseInt(ui.$("warmup-seconds").value, 10) || 120));
  ui.$("btn-pause").onclick = onTogglePause;
  ui.$("btn-restart").onclick = async () => {
    if (!(await ui.confirmAction("confirm.restartTitle", "confirm.restartText"))) return;
    action(() => api.app.RestartRound(), "toast.restarted");
  };

  const announce = () => {
    const message = ui.$("announce-in").value.trim();
    if (!message) return;
    ui.$("announce-in").value = "";
    call(() => api.app.Announce(message), "toast.announced");
  };
  ui.$("btn-announce").onclick = announce;
  ui.$("announce-in").addEventListener("keydown", (e) => e.key === "Enter" && announce());

  ui.$("btn-swap").onclick = async () => {
    if (!(await confirmIfBusy("confirm.swapTitle", "confirm.swapText"))) return;
    action(() => api.app.SwapTeams(), "toast.swapped");
  };
  ui.$("btn-scramble").onclick = async () => {
    if (!(await confirmIfBusy("confirm.scrambleTitle", "confirm.scrambleText"))) return;
    action(() => api.app.ScrambleTeams(), "toast.scrambled");
  };
  ui.$("btn-teamnames").onclick = () =>
    call(
      () => api.app.SetTeamNames(ui.$("teamname-ct").value, ui.$("teamname-t").value),
      "toast.namesSet"
    );
  ui.$("btn-bot-add").onclick = () => action(() => api.app.AddBot());
  ui.$("btn-bot-remove").onclick = () => action(() => api.app.RemoveBot());
  ui.$("btn-bot-kick").onclick = () => action(() => api.app.KickBots(), "toast.botsKicked");

  ui.$("btn-refreshinfo").onclick = onRefreshServerInfo;
  ui.$("btn-clearlog").onclick = () => (ui.$("console-out").textContent = "");

  ui.$("console-in").addEventListener("keydown", async (e) => {
    if (e.key !== "Enter") return;
    const command = e.target.value.trim();
    if (!command) return;
    e.target.value = "";
    ui.logLine("> " + command);
    const { ok, result } = await call(() => api.app.RunCommand(command));
    if (ok) ui.logLine(result || t("console.noOutput"));
  });

  api.runtime.EventsOn("log", ui.logLine);
  api.runtime.EventsOn("logkey", (msg) => ui.logLine(t(msg.key, msg.args)));
  api.runtime.EventsOn("warnkey", (msg) => {
    const text = t(msg.key, msg.args);
    ui.toast(text, true);
    ui.logLine("⚠ " + text);
  });
  // The backend also starts warmup after a map load and reports its length.
  api.runtime.EventsOn("warmup", (seconds) => {
    state.warmupEndsAt = Date.now() + seconds * 1000;
    ui.renderWarmupButton();
  });
}

// ---- Start-up ----

async function init() {
  await api.ready();

  setLanguage(await api.app.GetLanguage().catch(() => ""));
  applyTranslations();

  [state.presets, state.toggles] = await Promise.all([
    api.app.GetPresets(),
    api.app.GetToggles(),
  ]);

  wireEvents();
  renderAll();
  setInterval(ui.renderWarmupButton, 1000);
  await refreshServerSelect();

  const config = await api.app.GetConfig();
  if (!(config.servers || []).length) {
    openSettings();
    return;
  }
  await poll();
  await loadMaps();
  onRefreshServerInfo();
  schedulePolling();
}

init().catch((err) => {
  // Nothing is rendered yet at this point, so fall back to the raw message.
  document.body.textContent = translateError(err);
});
