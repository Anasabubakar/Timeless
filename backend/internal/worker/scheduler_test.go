package worker

import "testing"

func TestSchedulerTickIsFinerThanResyncInterval(t *testing.T) {
	// If the tick were >= the resync interval, an integration could sit
	// stale for up to a full extra tick before the scheduler notices it's
	// due — the tick must be strictly finer-grained than the interval it's
	// polling for.
	if schedulerTick >= resyncInterval {
		t.Errorf("schedulerTick (%v) must be smaller than resyncInterval (%v)", schedulerTick, resyncInterval)
	}
}
