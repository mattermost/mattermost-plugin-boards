// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package notifysubscriptions

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

func newTestNotifier(t *testing.T) *notifier {
	t.Helper()
	return &notifier{logger: mlog.CreateConsoleTestLogger(t)}
}

func TestRunLoopOnce_NormalReturn(t *testing.T) {
	n := newTestNotifier(t)
	called := false
	finished := n.runLoopOnce(func() { called = true })
	require.True(t, called, "fn should have been invoked")
	require.True(t, finished, "runLoopOnce should report finished=true on normal return")
}

func TestRunLoopOnce_RecoversFromPanic(t *testing.T) {
	n := newTestNotifier(t)
	require.NotPanics(t, func() {
		finished := n.runLoopOnce(func() { panic("boom") })
		require.False(t, finished, "runLoopOnce should report finished=false on panic")
	})
}

func TestSafeLoop_RestartsAfterPanic(t *testing.T) {
	n := newTestNotifier(t)
	calls := 0
	fn := func() {
		calls++
		if calls < 3 {
			panic("transient")
		}
	}

	deadline := time.After(2 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatalf("safeLoop simulation did not converge; calls=%d", calls)
		default:
		}
		if n.runLoopOnce(fn) {
			break
		}
	}
	require.Equal(t, 3, calls, "loop should restart after each panic until normal return")
}
