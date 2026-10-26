package core

import (
	"context"
	"errors"
	"net"
	"os"
	"strings"
	"testing"

	E "github.com/sagernet/sing/common/exceptions"
)

// A cleanup that only reports its component as already closed or cancelled is
// not a teardown failure -- adapter.Scope.Close has said so since 1.14.4 -- but
// a real failure reported alongside one still is. Judging the error after the
// entry's name was wrapped around it dropped both, since errors.Is looks inside.
func TestScopeCloseDropsOnlyClosedErrors(t *testing.T) {
	scopes, _ := startedScopes(t)
	real := errors.New("disk on fire")
	scopes.Add("closed", func() error { return os.ErrClosed })
	scopes.Add("cancelled", func() error { return context.Canceled })
	scopes.Add("mixed", func() error { return E.Errors(net.ErrClosed, real) })

	err := scopes.Close()
	if err == nil {
		t.Fatal("the real failure next to a closed error was dropped")
	}
	if !errors.Is(err, real) {
		t.Fatalf("want the real failure kept, got %v", err)
	}
	if E.IsClosed(err) || E.IsCanceled(err) || strings.Contains(err.Error(), "close closed") {
		t.Fatalf("a closed or cancelled cleanup was reported: %v", err)
	}
}

func TestScopeCloseWithOnlyClosedErrorsSucceeds(t *testing.T) {
	scopes, _ := startedScopes(t)
	scopes.Add("closed", func() error { return net.ErrClosed })
	if err := scopes.Close(); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
}
