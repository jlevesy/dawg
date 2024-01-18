package testutil

import (
	"testing"
	"time"
)

func WaitForSignal(t *testing.T, maxDuration time.Duration, signal <-chan struct{}) {
	t.Helper()

	select {
	case <-time.After(maxDuration):
		t.Fatal("wait exceeded maximum duration, failing the test")
		return
	case <-signal:
		return
	}
}

func Retry(t *testing.T, maxAttempt int, delay time.Duration, do func() bool) {
	t.Helper()

	for i := 0; i < maxAttempt; i++ {
		if do() {
			return
		}

		time.Sleep(delay)
	}

	t.Fatal("maximum attempt exhausted for a retry")
}
