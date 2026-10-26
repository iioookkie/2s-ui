package service

import (
	"encoding/json"
	"testing"

	"github.com/shenaba/2s-ui/database/model"
)

// A TLS config is stored with its SNI trimmed and without a blank one, whoever
// wrote it: the SPA does the same, but apiv2 posts whatever it was given.
func TestTlsSaveNormalizesServerName(t *testing.T) {
	serverName := func(server string) (any, bool) {
		row := model.Tls{Server: json.RawMessage(server)}
		normalizeServerName(&row)
		var m map[string]interface{}
		if err := json.Unmarshal(row.Server, &m); err != nil {
			t.Fatalf("%s: %v", server, err)
		}
		v, ok := m["server_name"]
		return v, ok
	}
	if v, ok := serverName(`{"enabled":true,"server_name":" example.com "}`); !ok || v != "example.com" {
		t.Errorf("padded: got %#v (present %v), want example.com", v, ok)
	}
	if v, ok := serverName(`{"enabled":true,"server_name":"  "}`); ok {
		t.Errorf("blank: want the key dropped, got %#v", v)
	}

	// Untouched when there is nothing to fix, down to the bytes: a rewrite
	// would reorder the keys of every TLS config on every save.
	const clean = `{"server_name":"example.com","enabled":true}`
	row := model.Tls{Server: json.RawMessage(clean)}
	normalizeServerName(&row)
	if string(row.Server) != clean {
		t.Errorf("a clean config was rewritten: %s", row.Server)
	}
}
