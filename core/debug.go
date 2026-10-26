package core

import (
	"os"
	"runtime/debug"

	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
)

func checkDebugOptions(options option.DebugOptions) error {
	if options.OOMKiller != nil {
		return E.New("legacy oom_killer in debug options is removed, use oom-killer service instead")
	}
	return nil
}

// runtimeDefaults are the process-wide runtime settings as they stood before
// any core applied its own, so that a field which is unset -- or a debug block
// that was removed -- puts the default back.
//
// sing-box only ever sets these, which is right when the process is sing-box:
// a changed config means a new process. Here the core is restarted in-process,
// so a gc_percent or memory_limit that was saved once would otherwise stay in
// force until the panel itself restarted -- and it is the whole panel's heap
// they govern, not just the core's.
var runtimeDefaults = captureRuntimeDefaults()

type runtimeSettings struct {
	gcPercent   int
	maxStack    int
	maxThreads  int
	memoryLimit int64
}

// Each setter returns the previous value, which is the only way to read it; the
// probe is put straight back. SetMemoryLimit reads without changing anything
// when given a negative value.
func captureRuntimeDefaults() runtimeSettings {
	gcPercent := debug.SetGCPercent(100)
	debug.SetGCPercent(gcPercent)
	maxStack := debug.SetMaxStack(1 << 30)
	debug.SetMaxStack(maxStack)
	maxThreads := debug.SetMaxThreads(10000)
	debug.SetMaxThreads(maxThreads)
	return runtimeSettings{
		gcPercent:   gcPercent,
		maxStack:    maxStack,
		maxThreads:  maxThreads,
		memoryLimit: debug.SetMemoryLimit(-1),
	}
}

func applyDebugOptions(options option.DebugOptions) {
	gcPercent := runtimeDefaults.gcPercent
	if options.GCPercent != nil {
		gcPercent = *options.GCPercent
	}
	debug.SetGCPercent(gcPercent)

	maxStack := runtimeDefaults.maxStack
	if options.MaxStack != nil {
		maxStack = *options.MaxStack
	}
	debug.SetMaxStack(maxStack)

	maxThreads := runtimeDefaults.maxThreads
	if options.MaxThreads != nil {
		maxThreads = *options.MaxThreads
	}
	debug.SetMaxThreads(maxThreads)

	// Per goroutine, so there is nothing left behind to restore.
	if options.PanicOnFault != nil {
		debug.SetPanicOnFault(*options.PanicOnFault)
	}

	// There is no getter, but the default is whatever GOTRACEBACK says ("" is
	// "single"), and the runtime never goes below that level anyway.
	traceBack := os.Getenv("GOTRACEBACK")
	if options.TraceBack != "" {
		traceBack = options.TraceBack
	}
	debug.SetTraceback(traceBack)

	memoryLimit := runtimeDefaults.memoryLimit
	if options.MemoryLimit.Value() != 0 {
		memoryLimit = int64(float64(options.MemoryLimit.Value()) / 1.5)
	}
	debug.SetMemoryLimit(memoryLimit)
}
