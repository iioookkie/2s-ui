package service

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/shenaba/2s-ui/database"
	"github.com/shenaba/2s-ui/database/model"
	"github.com/shenaba/2s-ui/logger"

	"github.com/op/go-logging"
)

func TestPresenceObserve(t *testing.T) {
	const grace = 5 * time.Minute
	t0 := time.Unix(1_800_000_000, 0)
	at := func(d time.Duration) time.Time { return t0.Add(d) }
	expect := func(t *testing.T, gotOn, gotOff, wantOn, wantOff []string) {
		t.Helper()
		if !reflect.DeepEqual(gotOn, wantOn) || !reflect.DeepEqual(gotOff, wantOff) {
			t.Errorf("online %v offline %v, want online %v offline %v", gotOn, gotOff, wantOn, wantOff)
		}
	}

	t.Run("start-up is silent for one grace period", func(t *testing.T) {
		var p presenceTracker
		on, off := p.observe(at(0), []string{"alice"}, grace)
		expect(t, on, off, nil, nil)
		// Still warming up: a client that reconnects late after a restart is
		// not news either.
		on, off = p.observe(at(time.Minute), []string{"alice", "bob"}, grace)
		expect(t, on, off, nil, nil)
		on, off = p.observe(at(grace), []string{"alice", "bob", "carol"}, grace)
		expect(t, on, off, []string{"carol"}, nil)
	})

	t.Run("an absence shorter than the grace says nothing", func(t *testing.T) {
		var p presenceTracker
		p.observe(at(0), nil, grace)
		on, _ := p.observe(at(grace), []string{"alice"}, grace)
		expect(t, on, nil, []string{"alice"}, nil)
		// Idle for a few flushes, then traffic again: the flap this exists
		// to absorb.
		on, off := p.observe(at(grace+time.Minute), nil, grace)
		expect(t, on, off, nil, nil)
		on, off = p.observe(at(grace+4*time.Minute), []string{"alice"}, grace)
		expect(t, on, off, nil, nil)
	})

	t.Run("offline after the grace, and the return is announced", func(t *testing.T) {
		var p presenceTracker
		p.observe(at(0), nil, grace)
		p.observe(at(grace), []string{"alice", "bob"}, grace)
		on, off := p.observe(at(2*grace), []string{"bob"}, grace)
		expect(t, on, off, nil, []string{"alice"})
		// Reported once, not on every flush that follows.
		on, off = p.observe(at(2*grace+10*time.Second), []string{"bob"}, grace)
		expect(t, on, off, nil, nil)
		on, off = p.observe(at(3*grace), []string{"alice", "bob"}, grace)
		expect(t, on, off, []string{"alice"}, nil)
	})

	// What ObservePresence does while the events are off: turning them on
	// again must not announce everyone already online.
	t.Run("after a reset the start-up grace applies again", func(t *testing.T) {
		var p presenceTracker
		p.observe(at(0), nil, grace)
		on, _ := p.observe(at(grace), []string{"alice"}, grace)
		expect(t, on, nil, []string{"alice"}, nil)

		p.reset()
		on, off := p.observe(at(2*grace), []string{"alice", "bob"}, grace)
		expect(t, on, off, nil, nil)
		on, off = p.observe(at(3*grace), []string{"alice", "bob", "carol"}, grace)
		expect(t, on, off, []string{"carol"}, nil)
	})

	t.Run("batches are sorted", func(t *testing.T) {
		var p presenceTracker
		p.observe(at(0), nil, grace)
		on, _ := p.observe(at(grace), []string{"zed", "amy", "kim"}, grace)
		expect(t, on, nil, []string{"amy", "kim", "zed"}, nil)
	})
}

// Presence messages name each client with its description (#261), and a client
// deleted while online still drops out of the offline batch rather than being
// announced as gone.
func TestPresenceLabels(t *testing.T) {
	logger.InitLogger(logging.CRITICAL)

	dir := t.TempDir()
	if err := database.InitDB(filepath.Join(dir, "presence.db")); err != nil {
		t.Fatalf("init db: %v", err)
	}
	t.Cleanup(func() {
		if err := database.CloseDBForTest(); err != nil {
			t.Errorf("close db: %v", err)
		}
	})
	db := database.GetDB()
	for _, c := range []model.Client{
		{Name: "gN3qa8", Desc: "小龙", Enable: true},
		{Name: "nprGX5", Desc: "十七", Enable: true},
		{Name: "plain", Enable: true},
	} {
		c.Inbounds = json.RawMessage(`[]`)
		c.Links = json.RawMessage(`[]`)
		c.Config = json.RawMessage(`{}`)
		if err := db.Create(&c).Error; err != nil {
			t.Fatalf("seed %s: %v", c.Name, err)
		}
	}

	on, off := presenceLabels([]string{"gN3qa8", "plain"}, []string{"deleted", "nprGX5"})
	if want := []string{"gN3qa8 (小龙)", "plain"}; !reflect.DeepEqual(on, want) {
		t.Errorf("online %v, want %v", on, want)
	}
	if want := []string{"nprGX5 (十七)"}; !reflect.DeepEqual(off, want) {
		t.Errorf("offline %v, want %v", off, want)
	}

	// The bot's /online keeps a name it has no row for, bare.
	got := ClientLabels([]string{"nprGX5", "on-a-node"})
	if want := []string{"nprGX5 (十七)", "on-a-node"}; !reflect.DeepEqual(got, want) {
		t.Errorf("ClientLabels %v, want %v", got, want)
	}
}
