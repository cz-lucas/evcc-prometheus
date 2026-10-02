package evccprometheus

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

// TestMetricsCollectorExportsEntitiesAndRemovesReplacedEntities checks samples, labels, and list reconciliation.
func TestMetricsCollectorExportsRoundedMetricsAndRemovesReplacedEntities(t *testing.T) {
	store := NewStateStore()
	store.Apply(StateUpdate{
		Batteries: map[string]BatteryState{"home": {SOC: Field[float64]{Value: 72.56, Set: true}}}, ReplaceBattery: true,
		PVSystems: map[string]PVState{
			"roof": {Power: Field[float64]{Value: 1250.6, Set: true}, Energy: Field[float64]{Value: 2.345, Set: true}},
			"shed": {Power: Field[float64]{Value: 350, Set: true}},
		}, ReplacePV: true,
		Consumers:  map[string]ConsumerState{"water": {Power: Field[float64]{Value: 400.6, Set: true}, Energy: Field[float64]{Value: 1.236, Set: true}}},
		GridMeters: map[string]GridState{"main": {Power: Field[float64]{Value: -120.6, Set: true}, Energy: Field[float64]{Value: 4.567, Set: true}, Currents: Field[[]float64]{Value: []float64{4.1236, 5.1236, 6.1236}, Set: true}}}, ReplaceGrid: true,
		ChargePoints: map[string]ChargePointState{"0": {
			Name: Field[string]{Value: "garage-wallbox", Set: true}, Charging: Field[bool]{Value: true, Set: true},
			Power: Field[float64]{Value: 7000.6, Set: true}, ChargedEnergy: Field[float64]{Value: 3.456, Set: true},
			VehicleSOC: Field[float64]{Value: 61.26, Set: true}, VehicleRange: Field[float64]{Value: 212.7, Set: true},
			VehicleOdometer: Field[float64]{Value: 12345.7, Set: true},
		}},
	})
	registry := prometheus.NewRegistry()
	registry.MustRegister(NewMetricsCollector(store))
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
			key := labels["pv_id"] + labels["consumer_id"] + labels["battery_id"] + labels["grid_id"] + labels["chargepoint_id"] + labels["phase"]
			values[family.GetName()][key] = metric.GetGauge().GetValue()
		}
	}
	for metric, test := range map[string]struct {
		key  string
		want float64
	}{
		"battery_soc": {"home", 72.6}, "evcc_pv_power_watts": {"roof", 1251},
		"evcc_pv_power_watts_shed": {"shed", 350}, "evcc_pv_energy_kwh": {"roof", 2.35},
		"evcc_consumer_power_watts": {"water", 401}, "evcc_consumer_energy_kwh": {"water", 1.24},
		"evcc_grid_power_watts": {"main", -121}, "evcc_grid_energy_kwh": {"main", 4.57},
		"evcc_grid_current_amperes": {"main2", 5.124},
		"evcc_chargepoint_charging": {"garage-wallbox", 1}, "evcc_chargepoint_power_watts": {"garage-wallbox", 7001},
		"evcc_chargepoint_charged_energy_kwh": {"garage-wallbox", 3.46}, "evcc_vehicle_soc_percent": {"garage-wallbox", 61.3},
		"evcc_vehicle_range_kilometers": {"garage-wallbox", 213}, "evcc_vehicle_odometer_kilometers": {"garage-wallbox", 12346},
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
