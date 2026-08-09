package app

// Carrying server definitions between machines, and checking one before it is
// saved. The file format is a small JSON document holding everything about a
// server except its RCON password.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"strikeman/internal/config"
	"strikeman/internal/rcon"
)

const (
	serverFileFormat = 1
	// A server file is a few hundred bytes; anything larger is not one.
	maxServerFileSize = 1 << 20
	maxImportServers  = 50
)

// PortableServer is a server definition without its password. Export and
// import both go through this type rather than config.Server, so a password
// cannot reach a file even if config.Server grows fields later.
type PortableServer struct {
	Name          string          `json:"name"`
	Host          string          `json:"host"`
	Port          int             `json:"port"`
	CollectionID  string          `json:"collectionId,omitempty"`
	StickyAdmin   *bool           `json:"stickyAdmin,omitempty"`
	Sticky        map[string]bool `json:"sticky,omitempty"`
	WarmupSeconds int             `json:"warmupSeconds,omitempty"`
}

// serverFile is what lands on disk. Its "servers" key is deliberately the same
// as StrikeMan's own config.json, so that file imports too — minus the
// passwords, which PortableServer has nowhere to put.
type serverFile struct {
	Format   int              `json:"format"`
	App      string           `json:"app"`
	Exported string           `json:"exported,omitempty"`
	Servers  []PortableServer `json:"servers"`
}

var fileFilters = []runtime.FileFilter{
	{DisplayName: "StrikeMan server file (*.json)", Pattern: "*.json"},
}

func encodeServers(servers []PortableServer) ([]byte, error) {
	if len(servers) == 0 {
		return nil, tErr("error.exportEmpty")
	}
	return json.MarshalIndent(serverFile{
		Format:   serverFileFormat,
		App:      "StrikeMan",
		Exported: time.Now().UTC().Format(time.RFC3339),
		Servers:  servers,
	}, "", "  ")
}

// decodeServers validates a server file and drops entries that could not be
// connected to anyway. Passwords are always empty: the caller asks for them.
func decodeServers(data []byte) ([]config.Server, error) {
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	var file serverFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, tErr("error.importParse", err.Error())
	}
	if file.Format > serverFileFormat {
		return nil, tErr("error.importFormat", strconv.Itoa(file.Format))
	}

	var out []config.Server
	for _, s := range file.Servers {
		if len(out) >= maxImportServers {
			break
		}
		host := strings.TrimSpace(s.Host)
		if host == "" || s.Port < 1 || s.Port > 65535 {
			continue
		}
		name := strings.TrimSpace(s.Name)
		if name == "" {
			name = host
		}
		out = append(out, config.Server{
			Name:          name,
			Host:          host,
			Port:          s.Port,
			CollectionID:  strings.TrimSpace(s.CollectionID),
			StickyAdmin:   s.StickyAdmin,
			Sticky:        s.Sticky,
			WarmupSeconds: s.WarmupSeconds,
		})
	}
	if len(out) == 0 {
		return nil, tErr("error.importEmpty")
	}
	return out, nil
}

// exportFilename suggests a name the user will recognise in their downloads.
func exportFilename(servers []PortableServer) string {
	if len(servers) != 1 {
		return "strikeman-servers.json"
	}
	slug := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			return r
		case r >= 'A' && r <= 'Z':
			return r + 32
		}
		return '-'
	}, servers[0].Name)
	slug = strings.Trim(slug, "-")
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	if slug == "" {
		return "strikeman-server.json"
	}
	return "strikeman-" + slug + ".json"
}

// ExportServers writes the given servers to a file the user picks. title comes
// from the frontend because the translations live there. Returns the path, or
// "" when the dialog was cancelled.
func (a *App) ExportServers(servers []PortableServer, title string) (string, error) {
	data, err := encodeServers(servers)
	if err != nil {
		return "", err
	}
	if a.ctx == nil {
		return "", tErr("error.noWindow")
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:                title,
		DefaultFilename:      exportFilename(servers),
		CanCreateDirectories: true,
		Filters:              fileFilters,
	})
	if err != nil || path == "" {
		return "", err
	}
	if !strings.EqualFold(filepath.Ext(path), ".json") {
		path += ".json"
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", err
	}
	a.logKey("log.serversExported", strconv.Itoa(len(servers)), path)
	return path, nil
}

// ImportServers reads a file the user picks and returns the servers in it,
// without saving anything: the caller puts them in front of the user first so
// the passwords can be filled in. Returns nil when the dialog was cancelled.
func (a *App) ImportServers(title string) ([]config.Server, error) {
	if a.ctx == nil {
		return nil, tErr("error.noWindow")
	}
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   title,
		Filters: fileFilters,
	})
	if err != nil || path == "" {
		return nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.Size() > maxServerFileSize {
		return nil, tErr("error.importTooBig")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	servers, err := decodeServers(data)
	if err != nil {
		return nil, err
	}
	a.logKey("log.serversImported", strconv.Itoa(len(servers)), path)
	return servers, nil
}

// TestServer checks a host, port and password without touching the saved
// config or the active connection. It only runs `status`, which reads, so this
// is safe to press during a match. Returns the server's hostname.
func (a *App) TestServer(host string, port int, password string) (string, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return "", tErr("error.hostRequired")
	}
	if port < 1 || port > 65535 {
		return "", tErr("error.portInvalid")
	}
	r := rcon.New(host, port, password)
	if err := r.Connect(); err != nil {
		return "", err
	}
	defer r.Close()

	// Twice at most: a busy server sometimes splits its answer across a gap
	// long enough that the first read stops before the hostname line. The
	// connection is proven either way, this is only about naming it.
	for attempt := 0; attempt < 2; attempt++ {
		out, err := r.Exec("status")
		if err != nil {
			return "", err
		}
		if name := parseStatus(out).Hostname; name != "" {
			return name, nil
		}
	}
	return host, nil
}
