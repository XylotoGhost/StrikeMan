package app

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestExportNeverContainsThePassword(t *testing.T) {
	yes := true
	data, err := encodeServers([]PortableServer{{
		Name:          "Mylo Gaming",
		Host:          "192.168.178.66",
		Port:          27016,
		CollectionID:  "3070284539",
		StickyAdmin:   &yes,
		Sticky:        map[string]bool{"autokick": false},
		WarmupSeconds: 90,
	}})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if strings.Contains(strings.ToLower(string(data)), "password") {
		t.Fatalf("export mentions a password:\n%s", data)
	}

	// Round trip: everything else must survive.
	servers, err := decodeServers(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("got %d servers, want 1", len(servers))
	}
	s := servers[0]
	if s.Name != "Mylo Gaming" || s.Host != "192.168.178.66" || s.Port != 27016 {
		t.Errorf("identity lost: %+v", s)
	}
	if s.CollectionID != "3070284539" || s.WarmupSeconds != 90 {
		t.Errorf("settings lost: %+v", s)
	}
	if s.StickyAdmin == nil || !*s.StickyAdmin || s.Sticky["autokick"] {
		t.Errorf("sticky toggles lost: %+v", s)
	}
	if s.Password != "" {
		t.Errorf("decoded a password: %q", s.Password)
	}
}

// StrikeMan's own config.json uses the same "servers" key, so it imports as
// well — and a password left in it (the keyring-less fallback) is dropped.
func TestImportOwnConfigDropsPasswords(t *testing.T) {
	raw := `{
	  "servers": [
	    {"name": "Home", "host": "10.0.0.5", "port": 27015, "password": "hunter2"}
	  ],
	  "default": "Home"
	}`
	servers, err := decodeServers([]byte(raw))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(servers) != 1 || servers[0].Name != "Home" {
		t.Fatalf("unexpected servers: %+v", servers)
	}
	if servers[0].Password != "" {
		t.Errorf("password survived the import: %q", servers[0].Password)
	}
}

func TestDecodeServersRejectsJunk(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{"not json", "hello"},
		{"no servers", `{"format":1,"servers":[]}`},
		{"no host", `{"servers":[{"name":"x","port":27015}]}`},
		{"port out of range", `{"servers":[{"name":"x","host":"h","port":70000}]}`},
		{"newer format", `{"format":99,"servers":[{"name":"x","host":"h","port":27015}]}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := decodeServers([]byte(tc.raw)); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

// A file with a BOM (what a Windows editor writes) must still import, and an
// entry without a name falls back to its host rather than being dropped.
func TestDecodeServersToleratesBOMAndMissingName(t *testing.T) {
	raw := append([]byte{0xEF, 0xBB, 0xBF},
		[]byte(`{"servers":[{"host":"10.0.0.7","port":27015}]}`)...)
	servers, err := decodeServers(raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if servers[0].Name != "10.0.0.7" {
		t.Errorf("name = %q, want the host", servers[0].Name)
	}
}

func TestDecodeServersCaps(t *testing.T) {
	var many []PortableServer
	for i := 0; i < maxImportServers+10; i++ {
		many = append(many, PortableServer{Name: "s", Host: "h", Port: 27015})
	}
	data, _ := json.Marshal(serverFile{Format: 1, Servers: many})
	servers, err := decodeServers(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(servers) != maxImportServers {
		t.Errorf("imported %d servers, want the cap of %d", len(servers), maxImportServers)
	}
}

func TestExportFilename(t *testing.T) {
	tests := []struct {
		servers []PortableServer
		want    string
	}{
		{[]PortableServer{{Name: "Mylo Gaming!"}}, "strikeman-mylo-gaming.json"},
		{[]PortableServer{{Name: "  "}}, "strikeman-server.json"},
		{[]PortableServer{{Name: "a"}, {Name: "b"}}, "strikeman-servers.json"},
	}
	for _, tc := range tests {
		if got := exportFilename(tc.servers); got != tc.want {
			t.Errorf("exportFilename(%v) = %q, want %q", tc.servers, got, tc.want)
		}
	}
}

func TestTestServerValidatesBeforeDialling(t *testing.T) {
	a := New("dev")
	if _, err := a.TestServer("  ", 27015, ""); err == nil {
		t.Error("expected an error for an empty host")
	}
	if _, err := a.TestServer("10.0.0.1", 0, ""); err == nil {
		t.Error("expected an error for port 0")
	}
}
