package evccprometheus

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

// TestMetricsCollectorExportsEntitiesAndRemovesReplacedEntities checks samples, labels, and list reconciliation.
func TestMetricsCollectorExportsEntitiesAndRemovesReplacedEntities(t *testing.T) {
	store := NewStateStore()
	store.Apply(StateUpdate{
		Batteries: map[string]BatteryState{"home": {SOC: Field[float64]{Value: 72.5, Set: true}}}, ReplaceBattery: true,
		PVSystems: map[string]PVState{"roof": {Power: Field[float64]{Value: 1250, Set: true}}, "shed": {Power: Field[float64]{Value: 350, Set: true}}}, ReplacePV: true,
		GridMeters: map[string]GridState{"main": {Currents: Field[[]float64]{Value: []float64{4, 5, 6}, Set: true}}}, ReplaceGrid: true,
		ChargePoints: map[string]ChargePointState{"0": {Name: Field[string]{Value: "garage-wallbox", Set: true}, Charging: Field[bool]{Value: true, Set: true}, VehicleSOC: Field[float64]{Value: 61, Set: true}}},
	})
	registry := prometheus.NewRegistry()
	registry.MustRegister(NewMetricsCollector(store, "garage"))
	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("Gather(): %v", err)
	}
	values := map[string]map[string]float64{}
	for _, family := range families {
		values[family.GetName()] = map[string]float64{}
		for _, metric := range family.GetMetric() {
			labels := map[string]string{}
			for _, label := range metric.GetLabel() {
				labels[label.GetName()] = label.GetValue()
			}
			key := labels["source_id"] + "/" + labels["pv_id"] + labels["battery_id"] + labels["grid_id"] + labels["chargepoint_id"] + labels["phase"]
			values[family.GetName()][key] = metric.GetGauge().GetValue()
		}
	}
	for metric, test := range map[string]struct {
		key  string
		want float64
	}{
		"battery_soc": {"garage/home", 72.5}, "evcc_pv_power_watts": {"garage/roof", 1250},
		"evcc_pv_power_watts_shed": {"garage/shed", 350}, "evcc_grid_current_amperes": {"garage/main2", 5},
		"evcc_chargepoint_charging": {"garage/garage-wallbox", 1}, "evcc_vehicle_soc_percent": {"garage/garage-wallbox", 61},
	} {
		name := metric
		if name == "evcc_pv_power_watts_shed" {
			name = "evcc_pv_power_watts"
		}
		if got := values[name][test.key]; got != test.want {
			t.Errorf("%s[%s] = %v, want %v", name, test.key, got, test.want)
		}
	}
	store.Apply(StateUpdate{PVSystems: map[string]PVState{}, ReplacePV: true})
	families, err = registry.Gather()
	if err != nil {
		t.Fatalf("Gather() after removal: %v", err)
	}
	for _, family := range families {
		if family.GetName() == "evcc_pv_power_watts" {
			t.Fatalf("removed PV still exported: %v", family)
		}
	}
}
