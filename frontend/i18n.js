// Translations. Keys are grouped by the card they appear in; the backend
// sends the same keys for presets, toggles and log messages so those are
// translated too.
//
// t("preset.wingman.desc") -> "MR8. An 8:8 ends in a draw."
// Arguments are positional: t("log.switchedServer", ["Mylo"]) fills {0}.

const strings = {
  en: {
    "app.notConnected": "not connected",
    "app.settings": "Settings",
    "app.switchServer": "Switch server",
    "app.currentMap": "Current map",
    "app.humans": "Human players",
    "app.bots": "Bots",
    "app.running": "Command running…",
    "app.version": "StrikeMan",
    "app.update": "App update",
    "app.updateAvailable": "{0} available",
    "app.upToDate": "up to date",
    "app.updateCheckFailed": "check failed",
    "app.devBuild": "local build",
    "app.installUpdate": "Update",
    "app.serverLabel": "Server",
    "app.addServer": "Add a server",

    "game.title": "Game",
    "game.now": "Now running: {0} on {1}",
    "game.nowRounds": "Now running: {0} on {1} · max {2} rounds",
    "game.notConnected": "Not connected.",
    "game.running": "running",
    "game.apply": "▶ Apply mode & load map",
    "game.changeMapOnly": "Change map only",
    "game.changeMapOnlyHint": "Load the map without changing the mode",
    "game.refreshMaps": "Reload map list",
    "game.filterWingman":
      "Maps the server has run in Wingman come first. Others are offered untried — only maps it has refused are hidden.",
    "game.filterCompetitive":
      "Maps the server has run in Competitive come first. Others are offered untried — only maps it has refused are hidden.",
    "game.mapCurrent": "current",
    "game.groupConfirmed": "Plays this mode",
    "game.groupUntested": "Not tried yet",
    "game.groupWorkshop": "Workshop",

    "mode.competitive": "Competitive",
    "mode.wingman": "Wingman",

    "match.title": "Match",
    "match.warmup": "Warmup",
    "match.warmupStart": "▶ Start warmup",
    "match.warmupEnd": "⏹ End warmup · go live ({0})",
    "match.pause": "⏸ Pause match",
    "match.resume": "▶ Resume match",
    "match.restart": "↻ Restart match",
    "match.announcePlaceholder": "Announce to all players…",
    "match.say": "Say",
    "match.sticky": "Kept across presets and map changes",

    "teams.title": "Teams",
    "teams.swap": "⇄ Swap sides",
    "teams.scramble": "🎲 Scramble",
    "teams.ctName": "CT team name",
    "teams.tName": "T team name",
    "teams.setNames": "Set names",
    "teams.addBot": "+ Bot",
    "teams.removeBot": "− Bot",
    "teams.kickBots": "Kick bots",

    "players.title": "Players",
    "players.empty": "nobody on the server",
    "players.kick": "kick",
    "players.bot": "BOT",

    "server.title": "Server",
    "server.check": "Check",
    "server.version": "Version",
    "server.versionValue": "{0} (build {1})",
    "server.updates": "Updates",
    "server.upToDate": "up to date",
    "server.updateAvailable": "update available (build {0})",
    "server.checkFailed": "check failed",
    "server.notChecked": "not checked",
    "server.uptime": "Uptime",

    "console.title": "Console",
    "console.clear": "Clear",
    "console.placeholder": "rcon command… (Enter to send)",
    "console.noOutput": "(no output)",

    "settings.title": "Settings",
    "settings.add": "+ Add",
    "settings.remove": "Remove",
    "settings.name": "Name",
    "settings.namePlaceholder": "My server",
    "settings.host": "Server host",
    "settings.port": "RCON port",
    "settings.password": "RCON password",
    "settings.collection": "Workshop collection ID",
    "settings.collectionPlaceholder": "e.g. 3070284539",
    "settings.default": "Use as default server",
    "settings.sticky": "Keep admin toggles across presets and map changes",
    "settings.stickyHint":
      "Loading a map re-runs the game mode config on the server, which turns auto-kick back on. With this enabled StrikeMan re-applies your choice afterwards.",
    "settings.passwordHint":
      "Passwords are stored in your OS credential store, not in the config file.",
    "settings.language": "Language",
    "settings.languageAuto": "System language",
    "settings.cancel": "Cancel",
    "settings.save": "Save & connect",
    "settings.nameRequired": "Every server needs a name",
    "settings.nameUnique": "Server names must be unique",
    "settings.saved": "Settings saved",
    "settings.newServer": "New server",
    "settings.servers": "Servers",
    "settings.noServers": "No servers yet",
    "settings.import": "Import…",
    "settings.importHint":
      "Read a server file — everything except the password, which you fill in here.",
    "settings.imported": "{0} server(s) imported — enter the RCON password",
    "settings.export": "Export…",
    "settings.exportHint": "Write this server to a file. The password is never included.",
    "settings.exported": "Saved to {0}",
    "settings.test": "Test connection",
    "settings.testing": "Connecting…",
    "settings.testOk": "Connected to {0}",

    "setup.welcome": "Welcome to StrikeMan",
    "setup.lead":
      "Manage your Counter-Strike 2 server without the console. Point StrikeMan at it once and it remembers.",
    "setup.addTitle": "Connect your server",
    "setup.import": "Import from a file…",
    "setup.imported": "{0} server(s) read from the file — enter the RCON password",

    "dialog.importTitle": "Open a StrikeMan server file",
    "dialog.exportTitle": "Save the server file",

    "confirm.yes": "Do it",
    "confirm.cancel": "Cancel",
    "confirm.kickTitle": "Kick player?",
    "confirm.kickText": "{0} will be disconnected from the server.",
    "confirm.applyTitle": "Apply mode and load map?",
    "confirm.applyText":
      "{0} player(s) are on the server. Applying {1} reloads the map and interrupts the current match.",
    "confirm.mapTitle": "Change map?",
    "confirm.mapText":
      "{0} player(s) are on the server and will be moved to the new map.",
    "confirm.warmupTitle": "Start warmup?",
    "confirm.warmupText":
      "{0} player(s) are on the server. Starting warmup interrupts the running match.",
    "confirm.restartTitle": "Restart the match?",
    "confirm.restartText":
      "This resets the score to 0:0 and starts the match from round 1.",
    "confirm.swapTitle": "Swap sides?",
    "confirm.swapText": "CT and T swap immediately, mid-match.",
    "confirm.scrambleTitle": "Scramble teams?",
    "confirm.scrambleText":
      "Players are randomly redistributed across both teams.",
    "confirm.updateTitle": "Install the update?",
    "confirm.updateText":
      "StrikeMan downloads the new version, replaces itself and restarts. The server is not affected.",

    "toast.pickMode": "Pick a mode first",
    "toast.pickMap": "Pick a map first",
    "toast.applying": "{0} — reloading map…",
    "toast.changingMap": "Changing map…",
    "toast.warmupStarted": "Warmup started",
    "toast.warmupEnded": "Warmup ended — match is live",
    "toast.paused": "Match pauses at the end of this round",
    "toast.resumed": "Match resumed",
    "toast.restarted": "Match restarted",
    "toast.swapped": "Teams swapped",
    "toast.scrambled": "Teams scrambled",
    "toast.namesSet": "Team names set",
    "toast.botsKicked": "Bots kicked",
    "toast.kicked": "Kicked {0}",
    "toast.announced": "Announced",
    "toast.updating": "Downloading the update…",

    "preset.competitive.name": "Competitive 5v5",
    "preset.competitive.desc": "MR12. A 12:12 ends in a draw.",
    "preset.premier.name": "Premier 5v5",
    "preset.premier.desc": "MR12, but 12:12 goes to overtime.",
    "preset.wingman.name": "Wingman 2v2",
    "preset.wingman.desc": "MR8. An 8:8 ends in a draw.",
    "preset.wingman3v3.name": "Wingman 3v3",
    "preset.wingman3v3.desc": "Wingman maps, 3 players per team.",

    "toggle.friendlyfire": "Friendly fire",
    "toggle.overtime": "Overtime",
    "toggle.autokick": "Auto-kick (idle / team damage)",
    "toggle.autokick.hint":
      "Off means nobody is kicked for team damage or being idle.",
    "toggle.kickban": "Kicked players banned 15 min",
    "toggle.kickban.hint": "Off lets a kicked player rejoin immediately.",

    "state.on": "on",
    "state.off": "off",

    "log.switchedServer": "Switched to server: {0}",
    "log.applyingPreset": "Applying preset: {0} on {1}",
    "log.presetApplied": "Preset {0} fully applied.",
    "log.presetTimedOut":
      "Preset {0}: server did not come back in time, post commands skipped.",
    "log.modeFellBack":
      "The map does not support {0} — the server fell back to another mode (game_mode = {1}).",
    "log.keptToggle": "Kept {0} {1} (server had reset it).",
    "log.commandFailed": "Command {0} failed: {1}",
    "log.workshopFailed": "Workshop collection: {0}",
    "log.updateCheckFailed": "Update check failed: {0}",
    "log.configUnreadable": "{0}",
    "log.updateDownloading": "Downloading StrikeMan {0}…",
    "log.updateRestarting": "Update to {0} installed — restarting.",
    "log.updateNoChecksum": "No checksum published for this release; skipping verification.",
    "log.serversExported": "Exported {0} server(s) to {1}",
    "log.serversImported": "Imported {0} server(s) from {1}",
    "log.mapSupports": "Noted: {0} plays {1}.",
    "log.mapDoesNotSupport": "Noted: {0} does not play {1}.",
    "log.mapModeSaveFailed": "Could not remember what the map supports: {0}",
    "log.noMapsParsed":
      "The server answered `maps *` with {0} bytes but no map names — the map list is empty.",

    "error.noServer": "No server configured",
    "error.unknownServer": "Unknown server {0}",
    "error.unknownPreset": "Unknown preset {0}",
    "error.unknownToggle": "{0} is not a toggle",
    "error.noMapSelected": "No map selected and the current map is unknown",
    "error.emptyAnnounce": "Nothing to announce",
    "error.mapNotWingman":
      "{0} does not support Wingman — pick one of the wingman maps",
    "error.workshopNotWingman":
      "{0} is not tagged as a Wingman map on the workshop — the server would fall back to Competitive",
    "error.updateFailed": "Update failed: {0}",
    "error.noUpdate": "Already on the newest version",
    "error.updateNoAsset": "The release has no download named {0}",
    "error.updateChecksum": "The download did not match its checksum: {0}",
    "error.hostRequired": "Enter the server host",
    "error.portInvalid": "Enter a port between 1 and 65535",
    "error.exportEmpty": "Nothing to export",
    "error.importParse": "That is not a StrikeMan server file: {0}",
    "error.importEmpty": "No usable server in that file",
    "error.importFormat": "That file was written by a newer StrikeMan (format {0})",
    "error.importTooBig": "That file is too large to be a server file",
    "error.noWindow": "No window available for the file dialog",
  },

  de: {
    "app.notConnected": "nicht verbunden",
    "app.settings": "Einstellungen",
    "app.switchServer": "Server wechseln",
    "app.currentMap": "Aktuelle Map",
    "app.humans": "Menschliche Spieler",
    "app.bots": "Bots",
    "app.running": "Befehl läuft…",
    "app.version": "StrikeMan",
    "app.update": "App-Update",
    "app.updateAvailable": "{0} verfügbar",
    "app.upToDate": "aktuell",
    "app.updateCheckFailed": "Prüfung fehlgeschlagen",
    "app.devBuild": "lokaler Build",
    "app.installUpdate": "Aktualisieren",
    "app.serverLabel": "Server",
    "app.addServer": "Server hinzufügen",

    "game.title": "Spiel",
    "game.now": "Läuft gerade: {0} auf {1}",
    "game.nowRounds": "Läuft gerade: {0} auf {1} · max. {2} Runden",
    "game.notConnected": "Nicht verbunden.",
    "game.running": "aktiv",
    "game.apply": "▶ Modus anwenden & Map laden",
    "game.changeMapOnly": "Nur Map wechseln",
    "game.changeMapOnlyHint": "Map laden, ohne den Modus zu ändern",
    "game.refreshMaps": "Map-Liste neu laden",
    "game.filterWingman":
      "Maps, die der Server schon in Wingman gefahren hat, stehen oben. Der Rest wird ungetestet angeboten — ausgeblendet wird nur, was der Server abgelehnt hat.",
    "game.filterCompetitive":
      "Maps, die der Server schon in Competitive gefahren hat, stehen oben. Der Rest wird ungetestet angeboten — ausgeblendet wird nur, was der Server abgelehnt hat.",
    "game.mapCurrent": "aktuell",
    "game.groupConfirmed": "Spielt diesen Modus",
    "game.groupUntested": "Noch nicht getestet",
    "game.groupWorkshop": "Workshop",

    "mode.competitive": "Competitive",
    "mode.wingman": "Wingman",

    "match.title": "Match",
    "match.warmup": "Aufwärmen",
    "match.warmupStart": "▶ Aufwärmen starten",
    "match.warmupEnd": "⏹ Aufwärmen beenden · live ({0})",
    "match.pause": "⏸ Match pausieren",
    "match.resume": "▶ Match fortsetzen",
    "match.restart": "↻ Match neu starten",
    "match.announcePlaceholder": "Nachricht an alle Spieler…",
    "match.say": "Senden",
    "match.sticky": "Bleibt über Presets und Map-Wechsel hinweg erhalten",

    "teams.title": "Teams",
    "teams.swap": "⇄ Seiten tauschen",
    "teams.scramble": "🎲 Mischen",
    "teams.ctName": "CT-Teamname",
    "teams.tName": "T-Teamname",
    "teams.setNames": "Namen setzen",
    "teams.addBot": "+ Bot",
    "teams.removeBot": "− Bot",
    "teams.kickBots": "Bots entfernen",

    "players.title": "Spieler",
    "players.empty": "niemand auf dem Server",
    "players.kick": "kicken",
    "players.bot": "BOT",

    "server.title": "Server",
    "server.check": "Prüfen",
    "server.version": "Version",
    "server.versionValue": "{0} (Build {1})",
    "server.updates": "Updates",
    "server.upToDate": "aktuell",
    "server.updateAvailable": "Update verfügbar (Build {0})",
    "server.checkFailed": "Prüfung fehlgeschlagen",
    "server.notChecked": "nicht geprüft",
    "server.uptime": "Laufzeit",

    "console.title": "Konsole",
    "console.clear": "Leeren",
    "console.placeholder": "RCON-Befehl… (Enter zum Senden)",
    "console.noOutput": "(keine Ausgabe)",

    "settings.title": "Einstellungen",
    "settings.add": "+ Hinzufügen",
    "settings.remove": "Entfernen",
    "settings.name": "Name",
    "settings.namePlaceholder": "Mein Server",
    "settings.host": "Server-Host",
    "settings.port": "RCON-Port",
    "settings.password": "RCON-Passwort",
    "settings.collection": "Workshop-Sammlungs-ID",
    "settings.collectionPlaceholder": "z. B. 3070284539",
    "settings.default": "Als Standardserver verwenden",
    "settings.sticky": "Admin-Schalter über Presets und Map-Wechsel beibehalten",
    "settings.stickyHint":
      "Beim Laden einer Map führt der Server die Spielmodus-Konfiguration erneut aus und schaltet Auto-Kick wieder ein. Mit dieser Option setzt StrikeMan deine Wahl danach erneut.",
    "settings.passwordHint":
      "Passwörter liegen im Anmeldeinformationsspeicher deines Systems, nicht in der Konfigurationsdatei.",
    "settings.language": "Sprache",
    "settings.languageAuto": "Systemsprache",
    "settings.cancel": "Abbrechen",
    "settings.save": "Speichern & verbinden",
    "settings.nameRequired": "Jeder Server braucht einen Namen",
    "settings.nameUnique": "Servernamen müssen eindeutig sein",
    "settings.saved": "Einstellungen gespeichert",
    "settings.newServer": "Neuer Server",
    "settings.servers": "Server",
    "settings.noServers": "Noch keine Server",
    "settings.import": "Importieren…",
    "settings.importHint":
      "Liest eine Serverdatei — alles außer dem Passwort, das du hier einträgst.",
    "settings.imported": "{0} Server importiert — bitte das RCON-Passwort eintragen",
    "settings.export": "Exportieren…",
    "settings.exportHint":
      "Schreibt diesen Server in eine Datei. Das Passwort ist nie enthalten.",
    "settings.exported": "Gespeichert unter {0}",
    "settings.test": "Verbindung testen",
    "settings.testing": "Verbinde…",
    "settings.testOk": "Verbunden mit {0}",

    "setup.welcome": "Willkommen bei StrikeMan",
    "setup.lead":
      "Verwalte deinen Counter-Strike-2-Server ohne Konsole. Einmal eintragen, StrikeMan merkt es sich.",
    "setup.addTitle": "Server verbinden",
    "setup.import": "Aus Datei importieren…",
    "setup.imported":
      "{0} Server aus der Datei gelesen — bitte das RCON-Passwort eintragen",

    "dialog.importTitle": "StrikeMan-Serverdatei öffnen",
    "dialog.exportTitle": "Serverdatei speichern",

    "confirm.yes": "Ausführen",
    "confirm.cancel": "Abbrechen",
    "confirm.kickTitle": "Spieler kicken?",
    "confirm.kickText": "{0} wird vom Server getrennt.",
    "confirm.applyTitle": "Modus anwenden und Map laden?",
    "confirm.applyText":
      "{0} Spieler sind auf dem Server. {1} anzuwenden lädt die Map neu und unterbricht das laufende Match.",
    "confirm.mapTitle": "Map wechseln?",
    "confirm.mapText":
      "{0} Spieler sind auf dem Server und werden auf die neue Map verschoben.",
    "confirm.warmupTitle": "Aufwärmen starten?",
    "confirm.warmupText":
      "{0} Spieler sind auf dem Server. Das Aufwärmen unterbricht das laufende Match.",
    "confirm.restartTitle": "Match neu starten?",
    "confirm.restartText":
      "Das setzt den Punktestand auf 0:0 zurück und startet ab Runde 1.",
    "confirm.swapTitle": "Seiten tauschen?",
    "confirm.swapText": "CT und T tauschen sofort, mitten im Match.",
    "confirm.scrambleTitle": "Teams mischen?",
    "confirm.scrambleText":
      "Die Spieler werden zufällig auf beide Teams verteilt.",
    "confirm.updateTitle": "Update installieren?",
    "confirm.updateText":
      "StrikeMan lädt die neue Version, ersetzt sich selbst und startet neu. Der Server ist davon nicht betroffen.",

    "toast.pickMode": "Wähle zuerst einen Modus",
    "toast.pickMap": "Wähle zuerst eine Map",
    "toast.applying": "{0} — Map wird neu geladen…",
    "toast.changingMap": "Map wird gewechselt…",
    "toast.warmupStarted": "Aufwärmen gestartet",
    "toast.warmupEnded": "Aufwärmen beendet — das Match läuft",
    "toast.paused": "Das Match pausiert am Ende dieser Runde",
    "toast.resumed": "Match fortgesetzt",
    "toast.restarted": "Match neu gestartet",
    "toast.swapped": "Seiten getauscht",
    "toast.scrambled": "Teams gemischt",
    "toast.namesSet": "Teamnamen gesetzt",
    "toast.botsKicked": "Bots entfernt",
    "toast.kicked": "{0} gekickt",
    "toast.announced": "Gesendet",
    "toast.updating": "Update wird geladen…",

    "preset.competitive.name": "Competitive 5v5",
    "preset.competitive.desc": "MR12. 12:12 endet unentschieden.",
    "preset.premier.name": "Premier 5v5",
    "preset.premier.desc": "MR12, aber 12:12 geht in die Verlängerung.",
    "preset.wingman.name": "Wingman 2v2",
    "preset.wingman.desc": "MR8. 8:8 endet unentschieden.",
    "preset.wingman3v3.name": "Wingman 3v3",
    "preset.wingman3v3.desc": "Wingman-Maps, 3 Spieler pro Team.",

    "toggle.friendlyfire": "Eigenbeschuss",
    "toggle.overtime": "Verlängerung",
    "toggle.autokick": "Auto-Kick (inaktiv / Teamschaden)",
    "toggle.autokick.hint":
      "Aus bedeutet: niemand wird wegen Teamschaden oder Inaktivität gekickt.",
    "toggle.kickban": "Gekickte Spieler 15 Min. gesperrt",
    "toggle.kickban.hint":
      "Aus lässt einen gekickten Spieler sofort wieder beitreten.",

    "state.on": "ein",
    "state.off": "aus",

    "log.switchedServer": "Zu Server gewechselt: {0}",
    "log.applyingPreset": "Preset wird angewendet: {0} auf {1}",
    "log.presetApplied": "Preset {0} vollständig angewendet.",
    "log.presetTimedOut":
      "Preset {0}: Server war nicht rechtzeitig zurück, Folgebefehle übersprungen.",
    "log.modeFellBack":
      "Die Map unterstützt {0} nicht — der Server ist auf einen anderen Modus zurückgefallen (game_mode = {1}).",
    "log.keptToggle": "{0} auf {1} gehalten (der Server hatte es zurückgesetzt).",
    "log.commandFailed": "Befehl {0} fehlgeschlagen: {1}",
    "log.workshopFailed": "Workshop-Sammlung: {0}",
    "log.updateCheckFailed": "Update-Prüfung fehlgeschlagen: {0}",
    "log.configUnreadable": "{0}",
    "log.updateDownloading": "StrikeMan {0} wird geladen…",
    "log.updateRestarting": "Update auf {0} installiert — Neustart.",
    "log.updateNoChecksum":
      "Für dieses Release ist keine Prüfsumme veröffentlicht; Prüfung übersprungen.",
    "log.serversExported": "{0} Server nach {1} exportiert",
    "log.serversImported": "{0} Server aus {1} importiert",
    "log.mapSupports": "Notiert: {0} spielt {1}.",
    "log.mapDoesNotSupport": "Notiert: {0} spielt {1} nicht.",
    "log.mapModeSaveFailed":
      "Konnte nicht speichern, welche Modi die Map unterstützt: {0}",
    "log.noMapsParsed":
      "Der Server hat auf `maps *` {0} Bytes geantwortet, aber keine Map-Namen — die Map-Liste ist leer.",

    "error.noServer": "Kein Server konfiguriert",
    "error.unknownServer": "Unbekannter Server {0}",
    "error.unknownPreset": "Unbekanntes Preset {0}",
    "error.unknownToggle": "{0} ist kein Schalter",
    "error.noMapSelected": "Keine Map gewählt und die aktuelle Map ist unbekannt",
    "error.emptyAnnounce": "Nichts zu senden",
    "error.mapNotWingman":
      "{0} unterstützt Wingman nicht — wähle eine der Wingman-Maps",
    "error.workshopNotWingman":
      "{0} ist im Workshop nicht als Wingman-Map markiert — der Server würde auf Competitive zurückfallen",
    "error.updateFailed": "Update fehlgeschlagen: {0}",
    "error.noUpdate": "Bereits auf der neuesten Version",
    "error.updateNoAsset": "Das Release enthält keinen Download namens {0}",
    "error.updateChecksum": "Der Download stimmt nicht mit seiner Prüfsumme überein: {0}",
    "error.hostRequired": "Bitte den Server-Host eintragen",
    "error.portInvalid": "Bitte einen Port zwischen 1 und 65535 eintragen",
    "error.exportEmpty": "Nichts zu exportieren",
    "error.importParse": "Das ist keine StrikeMan-Serverdatei: {0}",
    "error.importEmpty": "Kein brauchbarer Server in dieser Datei",
    "error.importFormat":
      "Diese Datei stammt aus einem neueren StrikeMan (Format {0})",
    "error.importTooBig": "Diese Datei ist zu groß für eine Serverdatei",
    "error.noWindow": "Kein Fenster für den Dateidialog verfügbar",
  },
};

export const languages = [
  { code: "", labelKey: "settings.languageAuto" },
  { code: "en", label: "English" },
  { code: "de", label: "Deutsch" },
];

let current = "en";

/** Picks the language to use: an explicit choice, else the OS/browser one. */
export function resolveLanguage(preferred) {
  if (strings[preferred]) return preferred;
  const system = (navigator.language || "en").slice(0, 2).toLowerCase();
  return strings[system] ? system : "en";
}

export function setLanguage(lang) {
  current = resolveLanguage(lang);
  document.documentElement.lang = current;
  return current;
}

export function getLanguage() {
  return current;
}

/**
 * An argument can itself be a translation key: preset and toggle names travel
 * from the backend as keys so they stay localised, tagged the same way its
 * errors are. Without this they printed raw, as in "Preset
 * preset.competitive.name fully applied".
 */
function resolveArg(arg) {
  const value = String(arg);
  return value.startsWith("i18n:") ? t(value.slice(5)) : value;
}

/** Translate a key, filling {0}, {1}, … from args. Falls back to English. */
export function t(key, args = []) {
  const table = strings[current] || strings.en;
  let text = table[key];
  if (text === undefined) text = strings.en[key];
  if (text === undefined) return key; // visible on purpose: a missing key
  return text.replace(/\{(\d+)\}/g, (match, i) =>
    args[i] === undefined ? match : resolveArg(args[i])
  );
}

/**
 * Backend errors arrive as plain strings. Ours are tagged "i18n:key" with
 * unit-separator delimited arguments (see tErr in app.go); anything else is
 * a raw server or network message and is shown as-is.
 */
export function translateError(err) {
  const text = String(err && err.message ? err.message : err);
  if (!text.startsWith("i18n:")) return text;
  const [key, ...args] = text.slice(5).split("\x1f");
  return t(key, args);
}

/** Fills every element tagged with data-i18n / -title / -placeholder. */
export function applyTranslations(root = document) {
  root.querySelectorAll("[data-i18n]").forEach((el) => {
    el.textContent = t(el.dataset.i18n);
  });
  root.querySelectorAll("[data-i18n-title]").forEach((el) => {
    el.title = t(el.dataset.i18nTitle);
  });
  root.querySelectorAll("[data-i18n-placeholder]").forEach((el) => {
    el.placeholder = t(el.dataset.i18nPlaceholder);
  });
  root.querySelectorAll("[data-i18n-aria]").forEach((el) => {
    el.setAttribute("aria-label", t(el.dataset.i18nAria));
  });
}

/** Development aid: keys present in English but missing from another table. */
export function missingKeys(lang) {
  const table = strings[lang] || {};
  return Object.keys(strings.en).filter((key) => table[key] === undefined);
}
