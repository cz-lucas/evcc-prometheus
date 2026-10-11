# EVCC Prometheus exporter

Exports selected [EVCC](https://evcc.io/) WebSocket state as Prometheus gauges. EVCC's state schema documents power in W, energy in kWh, current in A, SoC in percent, and vehicle range and odometer in km. Values are exported without unit conversion.

## Installation

### Docker

1. Set `EVCC_WS_URL` in `docker-compose.yml` to your EVCC WebSocket URL, including `ws://` (or `wss://` for https) and the `/ws` path.
2. Start the exporter with `docker compose up -d`.
3. Scrape metrics at `http://<docker-host>:9070/metrics`.

### Home Assistant OS

1. In Home Assistant, open **Settings > Add-ons > Add-on Store**, select the menu, and choose **Repositories**.
2. Add `https://github.com/cz-lucas/homeassistant-addons` as a repository.
3. Find **EVCC Prometheus** in the add-on store and install it.
4. In the add-on configuration, check that the URL is set correctly. Adjust `log_level` and `go_metrics_enabled` if needed.

### Bare metal

1. Download the latest [release](https://github.com/cz-lucas/evcc-prometheus/releases)
2. Make it executable with `chmod +x ./exporter-amd64`
3. Configure it using the environtment variables

``` bash
export EVCC_WS_URL="ws://your.evcc.instance.internal:7070"
export PROMETHEUS_ADDR=9070
export LOG_LEVEL="info"
export GO_METRICS_ENABLED=true
```

4. Start it with `./exporter-amd64`

## Configuration

| Environment variable | Default | Description |
| -------------------- | ----------------------- | ----------------------------------------- |
| `EVCC_WS_URL` | `wss://demo.evcc.io/ws` | EVCC WebSocket URL |
| `PROMETHEUS_ADDR` | `:9070` | HTTP listen address; metrics are served at `/metrics` |
| `LOG_LEVEL` | `info` | Loglevel (Allowed values: `debug`, `info`, `warn`, `error`) |
| `GO_METRICS_ENABLED` | `true` | Show go system metrics on `/metrics` |

The `ws://` or `wss://` at the beginning and the `/ws` at the end of the EVCC-URL is required.

> NOTE: When you change the port using the env-variable you need to also update the healthchck for the docker-container.  
> It's recommended to do port mapping using the ports-attribute in docker.

## Metrics

All metrics are gauges. Every series includes `source_id` and its entity ID label. The legacy metric `battery_soc` keeps its name and value semantics; its label set now also includes `source_id`.

| Metric | Entity label | Meaning |
| --- | --- | --- |
| `evcc_home_power_watts` | None | House power consumption, W |
| `evcc_battery_soc` | `id` | Battery state of charge, percent |
| `evcc_battery_power_watts` | `id` | Battery power, W |
| `evcc_battery_energy_kwh` | `id` | Battery energy, kWh |
| `evcc_pv_power_watts` | `id` | PV meter power, W |
| `evcc_pv_energy_kwh` | `id` | PV meter energy, kWh |
| `evcc_consumer_power_watts` | `id` | Consumer meter power, W |
| `evcc_consumer_energy_kwh` | `id` | Consumer meter energy, kWh |
| `evcc_grid_power_watts` | `id` | Grid meter power, W |
| `evcc_grid_energy_kwh` | `id` | Grid meter energy, kWh |
| `evcc_grid_current_amperes` | `id`, `phase` | Grid current per phase, A; phase labels start at `1` |
| `evcc_chargepoint_charging` | `id` | Charging state, `0` or `1` |
| `evcc_chargepoint_connected` | `id` | Vehicle connection state, `0` or `1` |
| `evcc_chargepoint_power_watts` | `id` | Charge point power, W |
| `evcc_chargepoint_charged_energy_kwh` | `id` | Energy charged in the current session, kWh |
| `evcc_vehicle_soc_percent` | `id` | Vehicle SoC, percent |
| `evcc_vehicle_range_kilometers` | `id` | Vehicle range, km |
| `evcc_vehicle_odometer_kilometers` | `id` | Vehicle odometer, km |
| `evcc_connected` | None | State of the websocket connection to evcc |

Meter IDs use EVCC's stable device `name`; a missing grid meter name uses `grid`. Charge point IDs use the configured loadpoint `name`, falling back to its WebSocket index until the name arrives. Vehicle metrics use the associated charge point ID because EVCC reports vehicle values on loadpoints; the mutable vehicle name is not a label.
> When renaming a metric you need to restart the exporter.

EVCC list updates replace that entity category: entities omitted frrom the new list stop being exported, while omitted fields on retained IDs preserve their previous values. A `null` field clears only that value; a `null` entity list clears the category. Vehicle metrics clear when the loadpoint disconnects or its vehicle name is cleared. A `loadpoints.<index>: null` update removes that charge point.

## TODO / Not yet implemented

- Add metrics like CO2 and electricity price
- Add support for heatpumps
- Write E2E test with real EVCC instance in demo mode (maybe)
- Extend README (In progress)
