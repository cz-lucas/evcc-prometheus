package evccprometheus

import (
	"strings"
	"testing"
)

// TestDecoderPreservesPartialFieldsAndReplacesEntityLists checks patch retention and complete-list removal.
func TestDecoderPreservesPartialFieldsAndReplacesEntityLists(t *testing.T) {
	decoder, store := NewDecoder(), NewStateStore()
	for _, message := range []string{
		`{"consumers":[{"name":"water","power":400,"energy":2.5},{"name":"dryer","power":900}]}`,
		`{"consumers":[{"name":"water","power":425}]}`,
	} {
		update, err := decoder.Decode(message)
		if err != nil {
			t.Fatalf("Decode(): %v", err)
		}
		store.Apply(update)
	}
	snapshot := store.Snapshot()
	if got := snapshot.Consumers["water"].Power.Value; got != 425 {
		t.Errorf("power = %v, want 425", got)
	}
	if got := snapshot.Consumers["water"].Energy.Value; got != 2.5 {
		t.Errorf("energy = %v, want retained 2.5", got)
	}
	if _, ok := snapshot.Consumers["dryer"]; ok {
		t.Error("omitted consumer was not removed")
	}
}

// TestDecoderAppliesLoadpointPatchesAndNullClearsVehicle checks partial updates and vehicle clearing.
func TestDecoderAppliesLoadpointPatchesAndNullClearsVehicle(t *testing.T) {
	decoder, store := NewDecoder(), NewStateStore()
	for _, message := range []string{`{"loadpoints.3.name":"garage","loadpoints.3.connected":"true","loadpoints.3.vehicleName":"Car","loadpoints.3.vehicleSoc":65,"loadpoints.3.chargePower":7000}`} {
		update, err := decoder.Decode(message)
		if err != nil {
			t.Fatalf("Decode(): %v", err)
		}
		store.Apply(update)
	}
	update, err := decoder.Decode(`{"loadpoints.3.connected":false}`)
	if err != nil {
		t.Fatalf("Decode() disconnect: %v", err)
	}
	store.Apply(update)
	point := store.Snapshot().ChargePoints["3"]
	if !point.Connected.Set || point.Connected.Value {
		t.Errorf("connected = %#v, want false", point.Connected)
	}
	if !point.Power.Set || point.Power.Value != 7000 {
		t.Errorf("power = %#v, want retained 7000", point.Power)
	}
	if !point.VehicleSOC.Set || !point.VehicleSOC.Null {
		t.Errorf("vehicle SOC = %#v, want cleared", point.VehicleSOC)
	}
	if !point.Name.Set || point.Name.Value != "garage" {
		t.Errorf("loadpoint name = %#v, want garage", point.Name)
	}
	if !point.VehicleName.Set || !point.VehicleName.Null {
		t.Errorf("vehicle name = %#v, want cleared on disconnect", point.VehicleName)
	}
	if update, err = decoder.Decode(`{"loadpoints.3.vehicleName":null}`); err != nil {
		t.Fatalf("Decode() null vehicle name: %v", err)
	} else {
		store.Apply(update)
	}
	if got := store.Snapshot().ChargePoints["3"].VehicleName; !got.Set || !got.Null {
		t.Errorf("vehicle name after explicit null = %#v, want cleared", got)
	}
}

// TestDecoderVehicleReassignmentClearsPreviousVehicleValues prevents stale telemetry crossing vehicle IDs.
func TestDecoderVehicleReassignmentClearsPreviousVehicleValues(t *testing.T) {
	decoder, store := NewDecoder(), NewStateStore()
	for _, message := range []string{
		`{"loadpoints.1.vehicleName":"car-a","loadpoints.1.vehicleSoc":80,"loadpoints.1.vehicleRange":250}`,
		`{"loadpoints.1.vehicleName":"car-b"}`,
	} {
		update, err := decoder.Decode(message)
		if err != nil {
			t.Fatalf("Decode(): %v", err)
		}
		store.Apply(update)
	}
	point := store.Snapshot().ChargePoints["1"]
	if !point.VehicleSOC.Set || !point.VehicleSOC.Null {
		t.Errorf("vehicle SOC = %#v, want cleared on reassignment", point.VehicleSOC)
	}
	if !point.VehicleRange.Set || !point.VehicleRange.Null {
		t.Errorf("vehicle range = %#v, want cleared on reassignment", point.VehicleRange)
	}
}

// TestDecoderInvalidUpdateIsNotApplied verifies malformed values do not alter valid stored state.
func TestDecoderInvalidUpdateIsNotApplied(t *testing.T) {
	decoder, store := NewDecoder(), NewStateStore()
	valid, err := decoder.Decode(`{"pv":[{"name":"roof","power":100}]}`)
	if err != nil {
		t.Fatal(err)
	}
	store.Apply(valid)
	invalid, err := decoder.Decode(`{"pv":[{"name":"roof","power":"bad"}]}`)
	if err == nil || !strings.Contains(err.Error(), `field "pv"`) {
		t.Fatalf("Decode() error = %v, want field context", err)
	}
	store.Apply(invalid)
	if got := store.Snapshot().PVSystems["roof"].Power.Value; got != 100 {
		t.Errorf("power after invalid update = %v, want 100", got)
	}
}

// TestDecoderUnknownFieldsNullListsAndInvalidIDs checks ignored extensions and lifecycle edge cases.
func TestDecoderUnknownFieldsNullListsAndInvalidIDs(t *testing.T) {
	decoder := NewDecoder()
	update, err := decoder.Decode(`{"future":1,"loadpoints.0.future":2,"pv":null}`)
	if err != nil {
		t.Fatalf("Decode(): %v", err)
	}
	if !update.ReplacePV || len(update.PVSystems) != 0 {
		t.Errorf("null PV update = %#v", update)
	}
	if _, err := decoder.Decode(`{"loadpoints.bad.connected":true}`); err == nil {
		t.Fatal("accepted invalid loadpoint ID")
	}
	remove, err := decoder.Decode(`{"loadpoints.0":null}`)
	if err != nil || len(remove.RemoveChargePoints) != 1 {
		t.Fatalf("loadpoint removal = %#v, %v", remove, err)
	}
	store := NewStateStore()
	store.Apply(StateUpdate{ChargePoints: map[string]ChargePointState{"0": {Connected: Field[bool]{Set: true, Value: true}}}})
	store.Apply(remove)
	if _, ok := store.Snapshot().ChargePoints["0"]; ok {
		t.Error("explicitly removed loadpoint remains in state")
	}
}
