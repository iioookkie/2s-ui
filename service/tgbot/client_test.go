package tgbot

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/shenaba/2s-ui/database"
	"github.com/shenaba/2s-ui/database/model"
	"github.com/shenaba/2s-ui/logger"

	"github.com/op/go-logging"
)

// One Telegram id has to resolve to exactly one client, because roleOf takes
// the first matching row: a second binding does not give that person a second
// account, it makes one of the two invisible to them.
func TestClientBoundTo(t *testing.T) {
	logger.InitLogger(logging.CRITICAL)

	dir := t.TempDir()
	if err := database.InitDB(filepath.Join(dir, "bind.db")); err != nil {
		t.Fatalf("init db: %v", err)
	}
	t.Cleanup(func() {
		if err := database.CloseDBForTest(); err != nil {
			t.Errorf("close db: %v", err)
		}
	})

	seed := func(name string, tgID int64, enable bool) {
		t.Helper()
		c := model.Client{
			Name: name, Enable: enable, TgId: tgID,
			Config: json.RawMessage(`{}`), Inbounds: json.RawMessage(`[]`), Links: json.RawMessage(`[]`),
		}
		if err := database.GetDB().Create(&c).Error; err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
	}
	seed("alice", 900, true)
	seed("bob", 0, true)
	// Disabled still counts: re-enabling it would bring the collision back
	// rather than resolve it, and roleOf resolves disabled clients anyway.
	seed("carol", 901, false)
	// A duplicate this check would refuse today, but an install upgrading from
	// before it can already carry one. Both holders have to be named, or the
	// operator unbinds the first and is told about the second on the retry.
	// dave is disabled so the ordering is also pinned: roleOf prefers the
	// enabled row, and this answer has to agree with it.
	seed("dave", 900, false)

	cases := []struct {
		name    string
		tgID    int64
		exclude string
		want    string
	}{
		{"a free id", 555, "bob", ""},
		// Two legacy holders, enabled one first -- the order roleOf resolves in.
		{"an id other clients hold", 900, "bob", "alice, dave"},
		// Re-confirming the id a client already has is not a conflict with
		// itself, or nobody could ever re-run their own binding -- but the
		// other holder still is one.
		{"the client's own id", 900, "alice", "dave"},
		{"a disabled client still holds its id", 901, "bob", "carol"},
		// 0 is the column default that every never-bound client carries, so a
		// lookup for it would match all of them. Unbinding is the one call that
		// passes 0 and it can never collide, so the helper has to answer "free"
		// rather than name the first unbound row it happens to find.
		{"zero is never a conflict", 0, "alice", ""},
	}

	for _, c := range cases {
		got, err := clientBoundTo(c.tgID, c.exclude)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s: clientBoundTo(%d, %q) = %q, want %q", c.name, c.tgID, c.exclude, got, c.want)
		}
	}
}

// A picker button carries the description (#261) but only a bounded part of
// it, and never at the cost of the name it is matched by.
func TestPickLabel(t *testing.T) {
	cases := []struct{ name, desc, want string }{
		{"gN3qa8ScNY5LqTz", "", "gN3qa8ScNY5LqTz"},
		{"gN3qa8ScNY5LqTz", "小龙", "gN3qa8ScNY5LqTz (小龙)"},
		{"gN3qa8ScNY5LqTz", "小龙的手机和平板电脑还有电视", "gN3qa8ScNY5LqTz (小龙的手机和平板电脑还…)"},
		{"a-very-long-client-name-indeed", "x", "a-very-long-client-name-indeed (x)"},
	}
	for _, c := range cases {
		if got := pickLabel(model.Client{Name: c.name, Desc: c.desc}); got != c.want {
			t.Errorf("pickLabel(%q, %q) = %q, want %q", c.name, c.desc, got, c.want)
		}
	}
}

// The bot finds a client by the description that names it to a person (#261),
// as the panel's own search does, alongside name, remark and group.
func TestClientSearch(t *testing.T) {
	logger.InitLogger(logging.CRITICAL)

	dir := t.TempDir()
	if err := database.InitDB(filepath.Join(dir, "search.db")); err != nil {
		t.Fatalf("init db: %v", err)
	}
	t.Cleanup(func() {
		if err := database.CloseDBForTest(); err != nil {
			t.Errorf("close db: %v", err)
		}
	})
	db := database.GetDB()
	for _, c := range []model.Client{
		{Name: "gN3qa8", Desc: "小龙"},
		{Name: "nprGX5", Remark: "十七"},
		{Name: "96dA7Q", Group: "family"},
	} {
		c.Inbounds = json.RawMessage(`[]`)
		c.Links = json.RawMessage(`[]`)
		c.Config = json.RawMessage(`{}`)
		if err := db.Create(&c).Error; err != nil {
			t.Fatalf("seed %s: %v", c.Name, err)
		}
	}

	for query, want := range map[string]string{
		"小龙": "gN3qa8", "qa8": "gN3qa8", "十七": "nprGX5", "fam": "96dA7Q",
	} {
		var names []string
		if err := clientSearch(query).Pluck("name", &names).Error; err != nil {
			t.Fatalf("search %q: %v", query, err)
		}
		if len(names) != 1 || names[0] != want {
			t.Errorf("search %q found %v, want [%s]", query, names, want)
		}
	}
}
