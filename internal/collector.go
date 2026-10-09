package evccprometheus

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

type MetricsCollector struct {
	state       StateReader
	descriptors metricDescriptors
}
type metricDescriptors struct {
	batterySOC, pvPower, pvEnergy, consumerPower, consumerEnergy                   *prometheus.Desc
	gridPower, gridEnergy, gridCurrent                                             *prometheus.Desc
	chargePointCharging, chargePointConnected, chargePointPower, chargePointEnergy *prometheus.Desc
	vehicleSOC, vehicleRange, vehicleOdometer                                      *prometheus.Desc
}

// NewMetricsCollector creates descriptors and binds collection to a state reader and source ID.
func NewMetricsCollector(state StateReader) *MetricsCollector {
	entity := func(name, help, label1, label2 string, labels ...string) *prometheus.Desc {
		allLabels := []string{label1, label2}
		allLabels = append(allLabels, labels...)
		return prometheus.NewDesc(name, help, allLabels, nil)
	}
	return &MetricsCollector{state: state, descriptors: metricDescriptors{
		batterySOC:           entity("evcc_battery_soc", "Battery state of charge in percent.", "battery_id", "name"),
		pvPower:              entity("evcc_pv_power_watts", "PV system power in watts.", "pv_id", "name"),
		pvEnergy:             entity("evcc_pv_energy_kwh", "PV system energy in kilowatt-hours.", "pv_id", "name"),
		consumerPower:        entity("evcc_consumer_power_watts", "Consumer power in watts.", "consumer_id", "name"),
		consumerEnergy:       entity("evcc_consumer_energy_kwh", "Consumer energy in kilowatt-hours.", "consumer_id", "name"),
		gridPower:            entity("evcc_grid_power_watts", "Grid meter power in watts.", "grid_id", "name"),
		gridEnergy:           entity("evcc_grid_energy_kwh", "Grid meter energy in kilowatt-hours.", "grid_id", "name"),
		gridCurrent:          entity("evcc_grid_current_amperes", "Grid current in amperes.", "grid_id", "phase", "name"),
		chargePointCharging:  entity("evcc_chargepoint_charging", "Whether a charge point is charging, as 0 or 1.", "chargepoint_id", "name"),
		chargePointConnected: entity("evcc_chargepoint_connected", "Whether a vehicle is connected to a charge point, as 0 or 1.", "chargepoint_id", "name"),
		chargePointPower:     entity("evcc_chargepoint_power_watts", "Charge point power in watts.", "chargepoint_id", "name"),
		chargePointEnergy:    entity("evcc_chargepoint_charged_energy_kwh", "Energy charged by a charge point in kilowatt-hours.", "chargepoint_id", "name"),
		vehicleSOC:           entity("evcc_vehicle_soc_percent", "Vehicle state of charge in percent, associated with a loadpoint.", "chargepoint_id", "name"),
		vehicleRange:         entity("evcc_vehicle_range_kilometers", "Vehicle range in kilometers, associated with a loadpoint.", "chargepoint_id", "name"),
		vehicleOdometer:      entity("evcc_vehicle_odometer_kilometers", "Vehicle odometer in kilometers, associated with a loadpoint.", "chargepoint_id", "name"),
	}}
}

// Describe sends every metric descriptor so Prometheus can validate collector registration.
func (c *MetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, desc := range c.descriptorList() {
		ch <- desc
	}
}

// Collect emits samples for fields present in one detached state snapshot.
func (c *MetricsCollector) Collect(ch chan<- prometheus.Metric) {
	// Each scrape uses one consistent snapshot; absent and removed values emit no series.
	snapshot := c.state.Snapshot()
	for id, value := range snapshot.Batteries {
		collectFloat(ch, c.descriptors.batterySOC, value.SOC, 1, id, fieldString(value.Name))
	}
	for id, value := range snapshot.PVSystems {
		collectFloat(ch, c.descriptors.pvPower, value.Power, 0, id, fieldString(value.Name))
		collectFloat(ch, c.descriptors.pvEnergy, value.Energy, 2, id, fieldString(value.Name))
	}
	for id, value := range snapshot.Consumers {
		collectFloat(ch, c.descriptors.consumerPower, value.Power, 0, id, fieldString(value.Name))
		collectFloat(ch, c.descriptors.consumerEnergy, value.Energy, 2, id, fieldString(value.Name))
	}
	for id, value := range snapshot.GridMeters {
		collectFloat(ch, c.descriptors.gridPower, value.Power, 0, id, fieldString(value.Name))
		collectFloat(ch, c.descriptors.gridEnergy, value.Energy, 2, id, fieldString(value.Name))
		if value.Currents.Set && !value.Currents.Null {
			for phase, current := range value.Currents.Value {
				ch <- prometheus.MustNewConstMetric(c.descriptors.gridCurrent, prometheus.GaugeValue, roundFloat(current, 3), id, strconv.Itoa(phase+1), fieldString(value.Name))
			}
		}
	}
	for id, point := range snapshot.ChargePoints {
		entityID := id
		if point.Name.Set && !point.Name.Null && point.Name.Value != "" {
			entityID = point.Name.Value
		}
		name := fieldString(point.Name)
		collectBool(ch, c.descriptors.chargePointCharging, point.Charging, entityID, name)
		collectBool(ch, c.descriptors.chargePointConnected, point.Connected, entityID, name)
		collectFloat(ch, c.descriptors.chargePointPower, point.Power, 0, entityID, name)
		collectFloat(ch, c.descriptors.chargePointEnergy, point.ChargedEnergy, 2, entityID, name)
		collectFloat(ch, c.descriptors.vehicleSOC, point.VehicleSOC, 1, entityID, name)
		collectFloat(ch, c.descriptors.vehicleRange, point.VehicleRange, 0, entityID, name)
		collectFloat(ch, c.descriptors.vehicleOdometer, point.VehicleOdometer, 0, entityID, name)
	}
}

// descriptorList returns all descriptors owned by this collector.
func (c *MetricsCollector) descriptorList() []*prometheus.Desc {
	d := c.descriptors
	return []*prometheus.Desc{d.batterySOC, d.pvPower, d.pvEnergy, d.consumerPower, d.consumerEnergy, d.gridPower, d.gridEnergy, d.gridCurrent, d.chargePointCharging, d.chargePointConnected, d.chargePointPower, d.chargePointEnergy, d.vehicleSOC, d.vehicleRange, d.vehicleOdometer}
}

// collectFloat emits a gauge when its field is present and non-null.
func collectFloat(ch chan<- prometheus.Metric, desc *prometheus.Desc, value Field[float64], precision uint, labels ...string) {
	if value.Set && !value.Null {
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, roundFloat(value.Value, precision), labels...)
	}
}

// collectBool converts a present boolean field to a 0-or-1 gauge.
func collectBool(ch chan<- prometheus.Metric, desc *prometheus.Desc, value Field[bool], labels ...string) {
	if !value.Set || value.Null {
		return
	}
	number := 0.0
	if value.Value {
		number = 1
	}
	ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, number, labels...)
}

func fieldString(value Field[string]) string {
	if value.Set && !value.Null {
		return value.Value
	}
	return ""
}
