package evccprometheus

import (
	"sync"
	"testing"
)

// TestStateStoreSnapshotDetachmentAndConcurrentAccess checks copy isolation and synchronized access.
func TestStateStoreSnapshotDetachmentAndConcurrentAccess(t *testing.T) {
	store := NewStateStore()
	store.Apply(StateUpdate{GridMeters: map[string]GridState{"main": {Currents: Field[[]float64]{Value: []float64{1, 2, 3}, Set: true}}}})
	snapshot := store.Snapshot()
	currents := snapshot.GridMeters["main"].Currents.Value
	currents[0] = 99
	if got := store.Snapshot().GridMeters["main"].Currents.Value[0]; got != 1 {
		t.Fatalf("stored current changed: %v", got)
	}
	var wait sync.WaitGroup
	for worker := 0; worker < 4; worker++ {
		wait.Add(1)
		go func(worker int) {
			defer wait.Done()
			for iteration := 0; iteration < 100; iteration++ {
				store.Apply(StateUpdate{ChargePoints: map[string]ChargePointState{"0": {Power: Field[float64]{Value: float64(worker*100 + iteration), Set: true}}}})
				_ = store.Snapshot()
			}
		}(worker)
	}
	wait.Wait()
}
