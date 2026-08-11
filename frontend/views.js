// Rendering and small UI primitives. Reads from state.js, never calls the
// backend itself: handlers are passed in from app.js.

import { t, translateError, languages } from "./i18n.js";
import {
  state,
  activePresetId,
  isWingmanIntent,
  presetById,
  visibleMaps,
  warmupRunning,
  warmupSecondsLeft,
} from "./state.js";

export const $ = (id) => document.getElementById(id);

// ---- Primitives ----

export function toast(message, isError = false) {
  const el = $("toast");
  el.textContent = message;
  el.className = "show" + (isError ? " error" : "");
  clearTimeout(el._timer);
  el._timer = setTimeout(() => (el.className = ""), 3500);
}

export function toastError(err) {
  toast(translateError(err), true);
}

export function logLine(text) {
  const out = $("console-out");
  out.textContent += text.endsWith("\n") ? text : text + "\n";
  out.scrollTop = out.scrollHeight;
}

let busyCount = 0;
export function setBusy(busy) {
  busyCount += busy ? 1 : -1;
  $("spinner").classList.toggle("hidden", busyCount <= 0);
}

/** Modal yes/no. Resolves true when confirmed. */
export function confirmAction(titleKey, textKey, args = []) {
  return new Promise((resolve) => {
    const dialog = $("confirm-dialog");
    $("confirm-title").textContent = t(titleKey);
    $("confirm-text").textContent = t(textKey, args);
    const finish = (ok) => {
      dialog.close();
      $("confirm-yes").onclick = null;
      $("confirm-no").onclick = null;
      resolve(ok);
    };
    $("confirm-yes").onclick = () => finish(true);
    $("confirm-no").onclick = () => finish(false);
    dialog.showModal();
  });
}

// ---- Header ----

export function renderHeader() {
  const st = state.status;
  const online = !!(st && st.connected);
  $("status-dot").className = "dot " + (online ? "online" : "offline");
  $("server-name").textContent = online
    ? st.hostname
    : st && st.error
      ? translateError(st.error)
      : t("app.notConnected");
  $("server-map").textContent = online && st.map ? st.map : "–";
  $("count-humans").textContent = online ? st.humans : 0;
  $("count-bots").textContent = online ? st.bots : 0;
}

// ---- Game card ----

export function renderModeButtons(onPick) {
  const row = $("mode-buttons");
  if (!row.childElementCount) {
    for (const preset of state.presets) {
      const button = document.createElement("button");
      button.className = "preset";
      button.dataset.id = preset.id;
      button.innerHTML = `<b></b><small></small>`;
      button.onclick = () => onPick(preset.id);
      row.append(button);
    }
  }
  const active = activePresetId();
  for (const button of row.children) {
    const preset = presetById(button.dataset.id);
    const name = button.querySelector("b");
    name.textContent = t(preset.nameKey);
    // The "● running" badge is drawn by b::after, and attr() only reads the
    // attributes of the element the pseudo-element belongs to.
    name.dataset.runningLabel = t("game.running");
    button.querySelector("small").textContent = t(preset.descKey);
    button.classList.toggle("selected", button.dataset.id === state.selectedMode);
    button.classList.toggle("active", button.dataset.id === active);
  }
}

export function renderNowLine() {
  const st = state.status;
  if (!st || !st.connected) {
    $("now-line").textContent = t("game.notConnected");
    return;
  }
  const preset = presetById(activePresetId());
  const modeName = preset ? t(preset.nameKey) : `game_mode ${st.gameMode}`;
  const map = st.map || "?";
  $("now-line").textContent = st.maxRounds
    ? t("game.nowRounds", [modeName, map, st.maxRounds])
    : t("game.now", [modeName, map]);
}

export function renderMapSelect() {
  const select = $("map-select");
  const previous = select.value;
  const { confirmed, untested, workshop, wingman } = visibleMaps();
  const current = state.status ? state.status.map : "";

  select.innerHTML = "";
  $("map-filter-hint").textContent = t(
    wingman ? "game.filterWingman" : "game.filterCompetitive"
  );

  const addGroup = (label, options) => {
    if (!options.length) return;
    const group = document.createElement("optgroup");
    group.label = label;
    for (const [text, value] of options) group.append(new Option(text, value));
    select.append(group);
  };
  const officialOption = (name) => [
    name === current ? `${name}  ● ${t("game.mapCurrent")}` : name,
    name,
  ];

  addGroup(t("game.groupConfirmed"), confirmed.map(officialOption));
  addGroup(t("game.groupUntested"), untested.map(officialOption));
  addGroup(
    t("game.groupWorkshop"),
    workshop.map((map) => [map.title, "ws:" + map.id])
  );

  // Keep the current map visible even when the filter would hide it.
  if (current && !confirmed.includes(current) && !untested.includes(current)) {
    select.append(new Option(`${current}  ● ${t("game.mapCurrent")}`, current));
  }

  const stillThere = [...select.options].some((o) => o.value === previous);
  if (state.userPickedMap && stillThere) {
    select.value = previous;
  } else {
    select.value = current || (select.options[0] ? select.options[0].value : "");
    state.userPickedMap = false;
  }
}

export function selectedMap() {
  const value = $("map-select").value;
  if (value.startsWith("ws:")) return { ref: value.slice(3), workshop: true };
  return { ref: value, workshop: false };
}

// ---- Match card ----

export function renderWarmupButton() {
  const button = $("btn-warmup");
  if (!warmupRunning()) {
    state.warmupEndsAt = null;
    button.textContent = t("match.warmupStart");
    return;
  }
  const left = warmupSecondsLeft();
  const clock = `${Math.floor(left / 60)}:${String(left % 60).padStart(2, "0")}`;
  button.textContent = t("match.warmupEnd", [clock]);
}

export function renderPauseButton() {
  $("btn-pause").textContent = state.paused ? t("match.resume") : t("match.pause");
}

export function renderToggles(onChange) {
  const row = $("switches");
  if (!row.childElementCount) {
    for (const toggle of state.toggles) {
      const label = document.createElement("label");
      label.className = "switch";
      const input = document.createElement("input");
      input.type = "checkbox";
      input.id = "tg-" + toggle.id;
      input.onchange = () => onChange(toggle.id, input.checked);
      const knob = document.createElement("span");
      const text = document.createElement("span");
      text.className = "switch-label";
      label.append(input, knob, text);
      if (toggle.admin) {
        const pin = document.createElement("span");
        pin.className = "pin hidden";
        pin.id = "pin-" + toggle.id;
        pin.textContent = "📌";
        label.append(pin);
      }
      row.append(label);
    }
  }

  const sticky = (state.server && state.server.sticky) || {};
  const stickyOn = !state.server || state.server.stickyAdmin !== false;
  for (const toggle of state.toggles) {
    const input = $("tg-" + toggle.id);
    const label = input.parentElement;
    label.querySelector(".switch-label").textContent = t(toggle.labelKey);
    label.title = toggle.hintKey ? t(toggle.hintKey) : "";
    const value = state.status && state.status.toggles ? state.status.toggles[toggle.id] : -1;
    input.disabled = !state.status || !state.status.connected || value === undefined || value < 0;
    if (value >= 0) input.checked = value === 1;
    const pin = $("pin-" + toggle.id);
    if (pin) {
      pin.classList.toggle("hidden", !(stickyOn && toggle.id in sticky));
      pin.title = t("match.sticky");
    }
  }
}

// ---- Players ----

export function renderPlayers(onKick) {
  const players = state.status && state.status.connected ? state.status.players : [];
  const list = $("player-list");
  $("player-count").textContent = players.length ? `(${players.length})` : "";
  if (!players.length) {
    list.innerHTML = "";
    const empty = document.createElement("li");
    empty.className = "empty";
    empty.textContent = t("players.empty");
    list.append(empty);
    return;
  }
  list.innerHTML = "";
  for (const player of players) {
    const row = document.createElement("li");
    const name = document.createElement("span");
    name.className = "name";
    name.textContent = player.name;
    row.append(name);
    if (player.bot) {
      const tag = document.createElement("span");
      tag.className = "bot-tag";
      tag.textContent = t("players.bot");
      row.append(tag);
    }
    const ping = document.createElement("span");
    ping.className = "ping";
    ping.textContent = `${player.ping} ms`;
    const kick = document.createElement("button");
    kick.className = "kick";
    kick.textContent = t("players.kick");
    kick.onclick = () => onKick(player);
    row.append(ping, kick);
    list.append(row);
  }
}

// ---- Server card ----

function formatUptime(seconds) {
  if (!seconds) return "–";
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (days) return `${days}d ${hours}h`;
  if (hours) return `${hours}h ${minutes}m`;
  return `${minutes}m`;
}

export function renderServerInfo(info) {
  $("info-version").textContent = info.version
    ? t("server.versionValue", [info.version, info.build])
    : "–";
  $("info-uptime").textContent = formatUptime(info.uptimeSecs);
  const updates = $("info-update");
  if (info.checkFailed) {
    updates.textContent = t("server.checkFailed");
    updates.className = "";
  } else if (info.upToDate) {
    updates.textContent = t("server.upToDate");
    updates.className = "ok";
  } else {
    updates.textContent = t("server.updateAvailable", [info.latest]);
    updates.className = "warn";
  }
}

/** StrikeMan's own version and whether a newer release is waiting. */
export function renderAppUpdate(version, update) {
  $("info-appversion").textContent = version || "–";
  const status = $("info-appupdate");
  const button = $("btn-update");

  button.classList.add("hidden");
  if (!update) {
    status.textContent = "";
    status.className = "";
    return;
  }
  if (update.error) {
    status.textContent = "· " + t("app.updateCheckFailed");
    status.className = "note";
    return;
  }
  if (update.available) {
    status.textContent = "· " + t("app.updateAvailable", [update.latest]);
    status.className = "note warn";
    button.classList.remove("hidden");
    return;
  }
  // A local build has no release version to compare against.
  status.textContent = "· " + (version === "dev" ? t("app.devBuild") : t("app.upToDate"));
  status.className = version === "dev" ? "note" : "note ok";
}

// ---- Settings dialog ----

export function renderServerSelect(config, activeName) {
  const select = $("server-select");
  const servers = config.servers || [];
  select.innerHTML = "";
  for (const server of servers) {
    select.append(new Option(server.name, server.name));
  }
  select.value = activeName;
}

export function renderLanguageSelect(id, currentValue) {
  const select = $(id);
  select.innerHTML = "";
  for (const language of languages) {
    select.append(new Option(language.label || t(language.labelKey), language.code));
  }
  select.value = currentValue || "";
}

/** A one-line outcome under a form: a test result, an import summary. */
export function setResult(id, message, kind = "") {
  const el = $(id);
  el.textContent = message;
  el.className = "result" + (kind ? " " + kind : "");
}

export function renderSettingsList(draft, index, onPick) {
  const list = $("srv-list");
  list.innerHTML = "";
  if (!draft.servers.length) {
    const empty = document.createElement("li");
    empty.className = "empty";
    empty.textContent = t("settings.noServers");
    list.append(empty);
    return;
  }
  draft.servers.forEach((server, i) => {
    const item = document.createElement("li");
    const button = document.createElement("button");
    button.type = "button";
    button.className = i === index ? "selected" : "";
    const name = document.createElement("span");
    name.className = "srv-name";
    name.textContent = server.name || t("settings.newServer");
    if (server.name && server.name === draft.default) {
      const star = document.createElement("span");
      star.className = "srv-star";
      star.textContent = " ★";
      star.title = t("settings.default");
      name.append(star);
    }
    const addr = document.createElement("span");
    addr.className = "srv-addr";
    addr.textContent = server.host ? `${server.host}:${server.port || 27015}` : "–";
    button.append(name, addr);
    button.onclick = () => onPick(i);
    item.append(button);
    list.append(item);
  });
}

const serverFieldIds = [
  "cfg-name",
  "cfg-host",
  "cfg-port",
  "cfg-password",
  "cfg-collection",
  "cfg-default",
  "cfg-sticky",
];

export function renderSettingsFields(draft, index) {
  const server = draft.servers[index];
  for (const id of serverFieldIds) $(id).disabled = !server;
  for (const id of ["btn-srv-test", "btn-srv-export", "btn-srv-remove"]) {
    $(id).disabled = !server;
  }
  setResult("srv-result", "");
  if (!server) {
    for (const id of serverFieldIds) {
      const field = $(id);
      if (field.type === "checkbox") field.checked = false;
      else field.value = "";
    }
    return;
  }
  $("cfg-name").value = server.name || "";
  $("cfg-host").value = server.host || "";
  $("cfg-port").value = server.port || 27015;
  $("cfg-password").value = server.password || "";
  $("cfg-collection").value = server.collectionId || "";
  $("cfg-default").checked = server.name === draft.default;
  $("cfg-sticky").checked = server.stickyAdmin !== false;
}

// ---- First-launch setup ----

export function showSetup(visible) {
  $("setup").classList.toggle("hidden", !visible);
}

export function setupServer() {
  return {
    name: $("setup-name").value.trim(),
    host: $("setup-host").value.trim(),
    port: parseInt($("setup-port").value, 10) || 27015,
    password: $("setup-password").value,
    collectionId: "",
  };
}

export function fillSetup(server) {
  $("setup-name").value = server.name || "";
  $("setup-host").value = server.host || "";
  $("setup-port").value = server.port || 27015;
  $("setup-password").value = server.password || "";
}
