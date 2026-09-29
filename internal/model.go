package evccprometheus

import (
	"time"
)

type EVCCMessage struct {
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
	//	Forecast *Forecast `json:"forecast"`

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
