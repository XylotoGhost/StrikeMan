package main

// App holds the backend state. Every exported method on it is callable from
// the frontend as window.go.main.App.<Method>.
//
// Concurrency: Wails calls the bound methods from its own goroutines and
// runAfterMapLoad runs in one of ours, so every field below is guarded by mu.
// The lock is never held across an RCON round trip or an HTTP request — those
// can take seconds — so the pattern is: read state under the lock, do the I/O,
// then write results back under the lock.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context

	mu          sync.Mutex
	config      Config
	active      string // name of the currently selected server
	rcon        *Rcon
	canBatch    *bool         // whether the server accepts semicolon-batched commands
	build       int           // server build number from `status`, for the update check
	workshop    []WorkshopMap // last fetched workshop list, for tag lookups
	lastEnforce map[string]time.Time
}

func NewApp() *App {
	return &App{lastEnforce: map[string]time.Time{}}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	cleanupOldExecutable() // left behind by a previous self-update
	cfg, err := LoadConfig()
	if err != nil {
		// Never silently fall back to "no servers configured": that looks
		// like the settings vanished.
		a.warnKey("log.configUnreadable", err.Error())
	}

	a.mu.Lock()
	a.config = cfg
	a.active = cfg.Default
	if cfg.serverByName(a.active) == nil && len(cfg.Servers) > 0 {
		a.active = cfg.Servers[0].Name
	}
	a.mu.Unlock()

	a.connectActive()
	fitWindowToScreen(ctx)
}

func (a *App) shutdown(context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.rcon != nil {
		a.rcon.Close()
		a.rcon = nil
	}
}

// connectActive points the RCON client at the selected server.
func (a *App) connectActive() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.rcon != nil {
		a.rcon.Close()
		a.rcon = nil
	}
	a.canBatch = nil
	if s := a.config.serverByName(a.active); s != nil {
		a.rcon = NewRcon(s.Host, s.Port, s.Password)
	}
}

// server returns a copy of the active server's settings.
func (a *App) server() (Server, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if s := a.config.serverByName(a.active); s != nil {
		return *s, true
	}
	return Server{}, false
}

// ---- Logging ----

// emit publishes an event to the frontend. Without a UI context — before
// start-up, or when driven from a test — there is nowhere to send it.
func (a *App) emit(name string, data any) {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, name, data)
}

// log sends a raw line to the console card. Used for RCON echoes, which are
// server output and stay untranslated.
func (a *App) log(format string, args ...any) {
	a.emit("log", fmt.Sprintf(format, args...))
}

// logKey sends a translatable message: the frontend formats key with args.
func (a *App) logKey(key string, args ...string) {
	a.emit("logkey", localized{Key: key, Args: args})
}

// warnKey is logKey plus an error toast.
func (a *App) warnKey(key string, args ...string) {
	a.emit("warnkey", localized{Key: key, Args: args})
}

type localized struct {
	Key  string   `json:"key"`
	Args []string `json:"args"`
}

// tErr builds an error the frontend translates. Errors cross the Wails
// boundary as plain strings, so the key and its arguments are encoded into
// one; see translateError() in the frontend.
func tErr(key string, args ...string) error {
	if len(args) == 0 {
		return errors.New("i18n:" + key)
	}
	return errors.New("i18n:" + key + "\x1f" + strings.Join(args, "\x1f"))
}

// ---- RCON plumbing ----

func (a *App) exec(cmd string) (string, error) {
	a.mu.Lock()
	r := a.rcon
	a.mu.Unlock()
	if r == nil {
		return "", tErr("error.noServer")
	}
	return r.Exec(cmd)
}

// execAll runs commands in order, echoing each to the console card.
func (a *App) execAll(cmds ...string) error {
	for _, cmd := range cmds {
		a.log("> %s", cmd)
		if _, err := a.exec(cmd); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) RunCommand(cmd string) (string, error) {
	return a.exec(cmd)
}

// ---- Config & server selection ----

func (a *App) GetConfig() Config {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.config
}

func (a *App) SaveConfig(cfg Config) error {
	a.mu.Lock()
	a.config = cfg
	if cfg.serverByName(a.active) == nil {
		a.active = cfg.Default
		if cfg.serverByName(a.active) == nil && len(cfg.Servers) > 0 {
			a.active = cfg.Servers[0].Name
		}
	}
	err := a.config.Save()
	a.mu.Unlock()
	if err != nil {
		return err
	}
	a.connectActive()
	return nil
}

func (a *App) GetActiveServer() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.active
}

// ServerView is what the UI polls every few seconds. It deliberately leaves
// out the RCON password: only the settings dialog needs that, via GetConfig.
type ServerView struct {
	Name          string          `json:"name"`
	StickyAdmin   bool            `json:"stickyAdmin"`
	Sticky        map[string]bool `json:"sticky"`
	WarmupSeconds int             `json:"warmupSeconds"`
}

func (a *App) GetActiveServerConfig() ServerView {
	s, ok := a.server()
	if !ok {
		return ServerView{Sticky: map[string]bool{}}
	}
	sticky := map[string]bool{}
	for k, v := range s.Sticky {
		sticky[k] = v
	}
	return ServerView{
		Name:          s.Name,
		StickyAdmin:   s.StickyEnabled(),
		Sticky:        sticky,
		WarmupSeconds: s.Warmup(),
	}
}

func (a *App) SelectServer(name string) error {
	a.mu.Lock()
	if a.config.serverByName(name) == nil {
		a.mu.Unlock()
		return tErr("error.unknownServer", name)
	}
	a.active = name
	a.mu.Unlock()

	a.connectActive()
	a.logKey("log.switchedServer", name)
	return nil
}

// ---- Language ----

// GetLanguage returns the stored UI language, or "" to follow the OS.
func (a *App) GetLanguage() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.config.Language
}

func (a *App) SetLanguage(lang string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.config.Language = lang
	return a.config.Save()
}

// ---- Server info ----

type ServerInfo struct {
	Version     string `json:"version"`
	Build       int    `json:"build"`
	UpToDate    bool   `json:"upToDate"`
	Latest      int    `json:"latest"`
	UpdateNote  string `json:"updateNote"`
	CheckFailed bool   `json:"checkFailed"`
	UptimeSecs  int    `json:"uptimeSecs"`
	Addon       string `json:"addon"`
}

// GetServerInfo pairs the server's own version with Steam's "is this build
// current" endpoint, so an outdated server is visible before match night.
func (a *App) GetServerInfo() ServerInfo {
	st := a.GetStatus()

	a.mu.Lock()
	build := a.build
	a.mu.Unlock()

	info := ServerInfo{Version: st.Version, Build: build}

	if out, err := a.exec("status_json"); err == nil {
		var sj struct {
			Uptime int `json:"process_uptime"`
			Server struct {
				Addon string `json:"addon"`
			} `json:"server"`
		}
		if start := strings.Index(out, "{"); start >= 0 {
			if end := strings.LastIndex(out, "}"); end > start {
				json.Unmarshal([]byte(out[start:end+1]), &sj)
			}
		}
		info.UptimeSecs = sj.Uptime
		info.Addon = sj.Server.Addon
	}

	if info.Build > 0 {
		upToDate, latest, note, err := CheckServerUpToDate(info.Build)
		if err != nil {
			info.CheckFailed = true
			a.logKey("log.updateCheckFailed", err.Error())
		} else {
			info.UpToDate, info.Latest, info.UpdateNote = upToDate, latest, note
		}
	}
	return info
}
