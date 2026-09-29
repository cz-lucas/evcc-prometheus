package evccprometheus

import (
	"time"
)

type EVCCData struct {
	// General / system
	APIReady                   bool    `json:"apiReady"`
	Database                   string  `json:"database"`
	Config                     string  `json:"config"`
	Currency                   string  `json:"currency"`
	DemoMode                   bool    `json:"demoMode"`
	Experimental               bool    `json:"experimental"`
	BatteryMode                string  `json:"batteryMode"`
	BufferSoc                  float64 `json:"bufferSoc"`
	BufferStartSoc             float64 `json:"bufferStartSoc"`
	BatteryDischargeControl    bool    `json:"batteryDischargeControl"`
	BatteryGridChargeActive    bool    `json:"batteryGridChargeActive"`
	BatteryGridChargeLimit     float64 `json:"batteryGridChargeLimit"`
	BatteryGridDischarge       bool    `json:"batteryGridDischarge"`
	BatteryGridDischargeActive bool    `json:"batteryGridDischargeActive"`

	// Energy
	Battery   *Battery   `json:"battery"`
	PV        []PV       `json:"pv"`
	PVPower   float64    `json:"pvPower"`
	PVEnergy  float64    `json:"pvEnergy"`
	HomePower float64    `json:"homePower"`
	Consumers []Consumer `json:"consumers"`
	Grid      *Grid      `json:"grid"`

	// Energy shares
	GreenShareHome       float64 `json:"greenShareHome"`
	GreenShareLoadpoints float64 `json:"greenShareLoadpoints"`

	// Tariffs
	TariffGrid            float64 `json:"tariffGrid"`
	TariffCo2             float64 `json:"tariffCo2"`
	TariffSolar           float64 `json:"tariffSolar"`
	TariffPriceHome       float64 `json:"tariffPriceHome"`
	TariffCo2Home         float64 `json:"tariffCo2Home"`
	TariffPriceLoadpoints float64 `json:"tariffPriceLoadpoints"`
	TariffCo2Loadpoints   float64 `json:"tariffCo2Loadpoints"`

	// Forecast
	Forecast *Forecast `json:"forecast"`

	// Loadpoints
	Loadpoints map[int]*Loadpoint `json:"loadpoints"`

	// Remote
	Remote *Remote `json:"remote"`

	// Logging
	Log *LogMessage `json:"log"`

	// History
	HistoryUpdated *time.Time `json:"historyUpdated"`

	// EEBUS
	EEBus *EEBus `json:"eebus"`

	// Generic arrays from EVCC
	Aux          []any `json:"aux"`
	DeviceColors []any `json:"deviceColors"`
	Ext          []any `json:"ext"`
}

type Forecast struct {
	CO2         []ForecastPoint `json:"co2"`
	FeedIn      []ForecastPoint `json:"feedin"`
	Grid        []ForecastPoint `json:"grid"`
	Planner     []ForecastPoint `json:"planner"`
	Solar       *ForecastSolar  `json:"solar"`
	Temperature []ForecastPoint `json:"temperature"`
}

type ForecastPoint struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Value float64   `json:"value"`
}

type ForecastSolar struct {
	Scale            float64          `json:"scale"`
	Today            ForecastDayValue `json:"today"`
	Tomorrow         ForecastDayValue `json:"tomorrow"`
	DayAfterTomorrow ForecastDayValue `json:"dayAfterTomorrow"`
	TimeSeries       []ForecastPoint  `json:"timeseries"`
}

type ForecastDayValue struct {
	Energy   float64 `json:"energy"`
	Complete bool    `json:"complete"`
}

type Loadpoint struct {
	AlwaysCharge                 bool                   `json:"alwaysCharge"`
	BatteryBoost                 bool                   `json:"batteryBoost"`
	BatteryBoostLimit            float64                `json:"batteryBoostLimit"`
	ChargeDuration               int                    `json:"chargeDuration"`
	ChargePower                  float64                `json:"chargePower"`
	ChargeRemainingDuration      int                    `json:"chargeRemainingDuration"`
	ChargeRemainingEnergy        float64                `json:"chargeRemainingEnergy"`
	ChargedEnergy                float64                `json:"chargedEnergy"`
	Charging                     bool                   `json:"charging"`
	Connected                    bool                   `json:"connected"`
	ConnectedDuration            int                    `json:"connectedDuration"`
	EffectiveLimitSoc            float64                `json:"effectiveLimitSoc"`
	EffectiveMaxCurrent          float64                `json:"effectiveMaxCurrent"`
	EffectiveMinCurrent          float64                `json:"effectiveMinCurrent"`
	EffectiveMinSoc              float64                `json:"effectiveMinSoc"`
	EffectivePlanID              int                    `json:"effectivePlanId"`
	EffectivePlanSoc             float64                `json:"effectivePlanSoc"`
	EffectivePlanStrategy        *LoadpointPlanStrategy `json:"effectivePlanStrategy"`
	EffectivePlanTime            *time.Time             `json:"effectivePlanTime"`
	EffectivePriority            int                    `json:"effectivePriority"`
	Enabled                      bool                   `json:"enabled"`
	Last24HEnergy                float64                `json:"last24hEnergy"`
	Last7DEnergy                 float64                `json:"last7dEnergy"`
	LimitEnergy                  float64                `json:"limitEnergy"`
	LimitSoc                     float64                `json:"limitSoc"`
	MaxCurrent                   float64                `json:"maxCurrent"`
	MinCurrent                   float64                `json:"minCurrent"`
	MinSoc                       float64                `json:"minSoc"`
	MinSocNotReached             bool                   `json:"minSocNotReached"`
	Mode                         string                 `json:"mode"`
	Name                         string                 `json:"name"`
	OfferedCurrent               float64                `json:"offeredCurrent"`
	Plan                         []LoadpointPlanEntry   `json:"plan"`
	PlanOverrun                  float64                `json:"planOverrun"`
	PlanProjectedEnd             *time.Time             `json:"planProjectedEnd"`
	PlanProjectedStart           *time.Time             `json:"planProjectedStart"`
	Priority                     int                    `json:"priority"`
	SessionCo2PerKWh             float64                `json:"sessionCo2PerKWh"`
	SessionEnergy                float64                `json:"sessionEnergy"`
	SessionPrice                 float64                `json:"sessionPrice"`
	SessionPricePerKWh           float64                `json:"sessionPricePerKWh"`
	SessionSolarPercentage       float64                `json:"sessionSolarPercentage"`
	SmartCostActive              bool                   `json:"smartCostActive"`
	SmartCostLimit               float64                `json:"smartCostLimit"`
	SmartCostNextStart           *time.Time             `json:"smartCostNextStart"`
	SmartFeedInPriorityActive    bool                   `json:"smartFeedInPriorityActive"`
	SmartFeedInPriorityLimit     float64                `json:"smartFeedInPriorityLimit"`
	SmartFeedInPriorityNextStart *time.Time             `json:"smartFeedInPriorityNextStart"`
	SolarShare                   float64                `json:"solarShare"`
	Suggestion                   any                    `json:"suggestion"`
	TodayEnergy                  float64                `json:"todayEnergy"`
	VehicleClimaterActive        bool                   `json:"vehicleClimaterActive"`
	VehicleDetectionActive       bool                   `json:"vehicleDetectionActive"`
	VehicleLimitSoc              float64                `json:"vehicleLimitSoc"`
	VehicleName                  string                 `json:"vehicleName"`
	VehicleOdometer              float64                `json:"vehicleOdometer"`
	VehicleRange                 float64                `json:"vehicleRange"`
	VehicleSoc                   float64                `json:"vehicleSoc"`
	VehicleTitle                 string                 `json:"vehicleTitle"`
	VehicleWelcomeActive         bool                   `json:"vehicleWelcomeActive"`
}

type LoadpointPlanStrategy struct {
	Continuous   bool `json:"continuous"`
	Precondition int  `json:"precondition"`
}

type LoadpointPlanEntry struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Value float64   `json:"value"`
}

type Battery struct {
	Power    float64         `json:"power"`
	Capacity float64         `json:"capacity"`
	SOC      float64         `json:"soc"`
	Devices  []BatteryDevice `json:"devices"`
}

type BatteryDevice struct {
	Name         string  `json:"name"`
	Title        string  `json:"title"`
	Power        float64 `json:"power"`
	Capacity     float64 `json:"capacity"`
	SOC          float64 `json:"soc"`
	Controllable bool    `json:"controllable"`
}

type PV struct {
	Name   string  `json:"name"`
	Title  string  `json:"title"`
	Power  float64 `json:"power"`
	Energy float64 `json:"energy"`
}

type Consumer struct {
	Name   string  `json:"name"`
	Title  string  `json:"title"`
	Icon   string  `json:"icon"`
	Power  float64 `json:"power"`
	Energy float64 `json:"energy"`
}

type Grid struct {
	Name     string    `json:"name"`
	Power    float64   `json:"power"`
	Energy   float64   `json:"energy"`
	Currents []float64 `json:"currents"`
}

type Remote struct {
	Config RemoteConfig `json:"config"`
	Status RemoteStatus `json:"status"`
}

type RemoteConfig struct {
	Enabled bool `json:"enabled"`
}

type RemoteStatus struct {
	Connected    bool `json:"connected"`
	LoginBlocked bool `json:"loginBlocked"`
}

type LogMessage struct {
	Message string `json:"message"`
	Level   string `json:"level"`
}

type EEBus struct {
	Config EEBusConfig `json:"config"`
	Status EEBusStatus `json:"status"`
}

type EEBusConfig struct {
	Port        int         `json:"port"`
	ShipID      string      `json:"shipid"`
	Certificate Certificate `json:"certificate"`
}

type Certificate struct {
	Public  string `json:"public"`
	Private string `json:"private"`
}

type EEBusStatus struct {
	SKI string `json:"ski"`
	QR  string `json:"qr"`
}
