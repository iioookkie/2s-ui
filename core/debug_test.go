package core

import (
	"encoding/json"
	"runtime/debug"
	"testing"

	"github.com/sagernet/sing-box/option"
)

func debugOptionsOf(t *testing.T, raw string) option.DebugOptions {
	t.Helper()
	var options option.DebugOptions
	if err := json.Unmarshal([]byte(raw), &options); err != nil {
		t.Fatal(err)
	}
	return options
}

// The core restarts in-process, so a debug block that is removed -- or a field
// that is cleared -- has to put the runtime back the way it was. sing-box only
// ever sets these, which left a saved gc_percent or memory_limit governing the
// whole panel until the process restarted.
func TestApplyDebugOptionsRestoresDefaults(t *testing.T) {
	t.Cleanup(func() { applyDebugOptions(option.DebugOptions{}) })

	applyDebugOptions(debugOptionsOf(t, `{"gc_percent": 37, "memory_limit": "300mb", "max_threads": 5000}`))
	if got := debug.SetGCPercent(37); got != 37 {
		t.Fatalf("gc percent = %d, want 37", got)
	}
	if got := debug.SetMemoryLimit(-1); got != int64(float64(300*1024*1024)/1.5) {
		t.Fatalf("memory limit = %d, want 300mb/1.5", got)
	}

	applyDebugOptions(option.DebugOptions{})
	if got := debug.SetGCPercent(runtimeDefaults.gcPercent); got != runtimeDefaults.gcPercent {
		t.Errorf("gc percent after removal = %d, want the default %d", got, runtimeDefaults.gcPercent)
	}
	if got := debug.SetMemoryLimit(-1); got != runtimeDefaults.memoryLimit {
		t.Errorf("memory limit after removal = %d, want the default %d", got, runtimeDefaults.memoryLimit)
	}
	if got := debug.SetMaxThreads(runtimeDefaults.maxThreads); got != runtimeDefaults.maxThreads {
		t.Errorf("max threads after removal = %d, want the default %d", got, runtimeDefaults.maxThreads)
	}
}
