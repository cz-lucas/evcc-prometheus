package evccprometheus

import (
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MetricsCollector struct {
	state       StateReader
	sourceID    string
	descriptors metricDescriptors
}
type metricDescriptors struct {
	batterySOC, pvPower, pvEnergy, consumerPower, consumerEnergy                   *prometheus.Desc
	gridPower, gridEnergy, gridCurrent                                             *prometheus.Desc
	chargePointCharging, chargePointConnected, chargePointPower, chargePointEnergy *prometheus.Desc
	vehicleSOC, vehicleRange, vehicleOdometer                                      *prometheus.Desc
}

// NewMetricsCollector creates descriptors and binds collection to a state reader and source ID.
func NewMetricsCollector(state StateReader, sourceID string) *MetricsCollector {
	entity := func(name, help, label string) *prometheus.Desc {
		return prometheus.NewDesc("evcc_"+name, help, []string{"source_id", label}, nil)
	}
	return &MetricsCollector{state: state, sourceID: sourceID, descriptors: metricDescriptors{
		batterySOC:           prometheus.NewDesc("battery_soc", "Battery state of charge in percent.", []string{"source_id", "battery_id"}, nil),
		pvPower:              entity("pv_power_watts", "PV system power in watts.", "pv_id"),
		pvEnergy:             entity("pv_energy_kwh", "PV system energy in kilowatt-hours.", "pv_id"),
		consumerPower:        entity("consumer_power_watts", "Consumer power in watts.", "consumer_id"),
		consumerEnergy:       entity("consumer_energy_kwh", "Consumer energy in kilowatt-hours.", "consumer_id"),
		gridPower:            entity("grid_power_watts", "Grid meter power in watts.", "grid_id"),
		gridEnergy:           entity("grid_energy_kwh", "Grid meter energy in kilowatt-hours.", "grid_id"),
		gridCurrent:          prometheus.NewDesc("evcc_grid_current_amperes", "Grid current in amperes.", []string{"source_id", "grid_id", "phase"}, nil),
		chargePointCharging:  entity("chargepoint_charging", "Whether a charge point is charging, as 0 or 1.", "chargepoint_id"),
		chargePointConnected: entity("chargepoint_connected", "Whether a vehicle is connected to a charge point, as 0 or 1.", "chargepoint_id"),
		chargePointPower:     entity("chargepoint_power_watts", "Charge point power in watts.", "chargepoint_id"),
		chargePointEnergy:    entity("chargepoint_charged_energy_kwh", "Energy charged by a charge point in kilowatt-hours.", "chargepoint_id"),
		vehicleSOC:           entity("vehicle_soc_percent", "Vehicle state of charge in percent, associated with a loadpoint.", "chargepoint_id"),
		vehicleRange:         entity("vehicle_range_kilometers", "Vehicle range in kilometers, associated with a loadpoint.", "chargepoint_id"),
		vehicleOdometer:      entity("vehicle_odometer_kilometers", "Vehicle odometer in kilometers, associated with a loadpoint.", "chargepoint_id"),
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
		collectFloat(ch, c.descriptors.batterySOC, value.SOC, c.sourceID, id)
	}
	for id, value := range snapshot.PVSystems {
		collectFloat(ch, c.descriptors.pvPower, value.Power, c.sourceID, id)
		collectFloat(ch, c.descriptors.pvEnergy, value.Energy, c.sourceID, id)
	}
	for id, value := range snapshot.Consumers {
		collectFloat(ch, c.descriptors.consumerPower, value.Power, c.sourceID, id)
		collectFloat(ch, c.descriptors.consumerEnergy, value.Energy, c.sourceID, id)
	}
	for id, value := range snapshot.GridMeters {
		collectFloat(ch, c.descriptors.gridPower, value.Power, c.sourceID, id)
		collectFloat(ch, c.descriptors.gridEnergy, value.Energy, c.sourceID, id)
		if value.Currents.Set && !value.Currents.Null {
			for phase, current := range value.Currents.Value {
				ch <- prometheus.MustNewConstMetric(c.descriptors.gridCurrent, prometheus.GaugeValue, current, c.sourceID, id, strconv.Itoa(phase+1))
			}
		}
	}
	for id, point := range snapshot.ChargePoints {
		entityID := id
		if point.Name.Set && !point.Name.Null && point.Name.Value != "" {
			entityID = point.Name.Value
		}
		collectBool(ch, c.descriptors.chargePointCharging, point.Charging, c.sourceID, entityID)
		collectBool(ch, c.descriptors.chargePointConnected, point.Connected, c.sourceID, entityID)
		collectFloat(ch, c.descriptors.chargePointPower, point.Power, c.sourceID, entityID)
		collectFloat(ch, c.descriptors.chargePointEnergy, point.ChargedEnergy, c.sourceID, entityID)
		collectFloat(ch, c.descriptors.vehicleSOC, point.VehicleSOC, c.sourceID, entityID)
		collectFloat(ch, c.descriptors.vehicleRange, point.VehicleRange, c.sourceID, entityID)
		collectFloat(ch, c.descriptors.vehicleOdometer, point.VehicleOdometer, c.sourceID, entityID)
	}
}

// descriptorList returns all descriptors owned by this collector.
func (c *MetricsCollector) descriptorList() []*prometheus.Desc {
	d := c.descriptors
	return []*prometheus.Desc{d.batterySOC, d.pvPower, d.pvEnergy, d.consumerPower, d.consumerEnergy, d.gridPower, d.gridEnergy, d.gridCurrent, d.chargePointCharging, d.chargePointConnected, d.chargePointPower, d.chargePointEnergy, d.vehicleSOC, d.vehicleRange, d.vehicleOdometer}
}

// collectFloat emits a gauge when its field is present and non-null.
func collectFloat(ch chan<- prometheus.Metric, desc *prometheus.Desc, value Field[float64], labels ...string) {
	if value.Set && !value.Null {
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, value.Value, labels...)
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

// PrometheusHandler serves the supplied registry from a private /metrics-only mux.
func PrometheusHandler(registry *prometheus.Registry) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	return mux
}
