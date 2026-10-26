package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/shenaba/2s-ui/database"
	"github.com/shenaba/2s-ui/database/model"
)

// A base config sing-box cannot parse is refused before it is written. The save
// restarts the core after the commit, and one it could not parse left the core
// down -- retried every few seconds by the watchdog -- with the bad value
// already stored.
func TestABaseConfigTheCoreCannotParseIsRefused(t *testing.T) {
	s := maintenanceEnv(t)
	const stored = `{"log":{"disabled":true}}`
	if err := s.SettingService.SetConfig(stored); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	for name, config := range map[string]string{
		// GetConfig reads an empty string as "the stored config", which used to
		// validate the old one and then store this as null.
		"empty":            ``,
		"null":             `null`,
		"array":            `[]`,
		"memory limit":     `{"log":{"disabled":true},"experimental":{"debug":{"memory_limit":"512mb512mb"}}}`,
		"fractional gc":    `{"log":{"disabled":true},"experimental":{"debug":{"gc_percent":80.5}}}`,
		"legacy oom":       `{"log":{"disabled":true},"experimental":{"debug":{"oom_killer":true}}}`,
		"unknown log knob": `{"log":{"disabled":true,"levle":"info"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			err := save(t, s, "config", "set", config)
			if err == nil || !strings.Contains(err.Error(), "invalid config") {
				t.Fatalf("want the save refused as an invalid config, got %v", err)
			}
			got, err := s.SettingService.GetConfig()
			if err != nil {
				t.Fatal(err)
			}
			if strings.Join(strings.Fields(got), "") != stored {
				t.Fatalf("the refused config was written: %s", got)
			}
		})
	}
}

// A stored row the core cannot read keeps it down, but must not also refuse
// every base save: raising the log level is how an operator finds that row.
// Saves made while the core is down are never parsed, so such a row can exist.
func TestABadRowDoesNotBlockTheBaseConfig(t *testing.T) {
	s := maintenanceEnv(t)
	bad := model.Outbound{Type: "direct", Tag: "broken", Options: json.RawMessage(`{"bogus":true}`)}
	if err := database.GetDB().Create(&bad).Error; err != nil {
		t.Fatalf("seed row: %v", err)
	}
	whole, err := s.GetConfig(`{"log":{"level":"debug"}}`)
	if err != nil {
		t.Fatal(err)
	}
	if corePtr.ValidateConfig(*whole) == nil {
		t.Fatal("the seeded row was meant to be unreadable, so the test proves nothing")
	}
	if err := s.validateBaseConfig([]byte(`{"log":{"level":"debug"}}`)); err != nil {
		t.Fatalf("a base config was refused for a row it does not contain: %v", err)
	}
}

func TestABaseConfigTheCoreCanParsePassesValidation(t *testing.T) {
	s := maintenanceEnv(t)
	config := `{"log":{"level":"info"},"experimental":{"debug":{"listen":"127.0.0.1:6060","memory_limit":"512mb","gc_percent":80}}}`
	if err := s.validateBaseConfig([]byte(config)); err != nil {
		t.Fatalf("a valid config was refused: %v", err)
	}
}
