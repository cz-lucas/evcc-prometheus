# EVCC Metrics Refactor Plan

## Goal

Decouple EVCC message decoding, domain state, and Prometheus exposition so each part can be reused and tested independently. Support multiple entities in each category: energy consumers, grid meters, PV systems, charge points, and vehicles.

## Current State

- `EVCCData` models many EVCC fields, while only `battery_soc` is currently registered as an application metric.
- `MessageToState` merges WebSocket messages into the broad model and publishes a battery-specific `StateUpdate`.
- `Metrics.Update` and `RunMetricsUpdater` consume that update, coupling parser output to the current metrics implementation.
- The parser includes forecast and loadpoint-plan decoding, reflective root-field lookup, and tests for those features even though they are not currently exported as metrics.
- EVCC messages can be partial, including dotted loadpoint updates. The refactor must preserve current values when a message omits a field and handle entity removal deliberately.

## Target Design

Use a one-way dependency flow:

```text
WebSocket client -> EVCC decoder -> state store -> Prometheus collector -> HTTP registry
```

- **Decoder:** knows the EVCC wire format and produces typed domain updates. It has no Prometheus dependency.
- **State store:** applies partial updates, tracks entities by stable identifiers, and provides consistent snapshots. It has no EVCC JSON or Prometheus dependency.
- **Prometheus collector:** reads a snapshot through a small interface and emits metric families on scrape. It does not receive parser channels or mutate gauges from the parser goroutine.
- **Application wiring:** constructs the client, decoder, store, collector, registry, and server. Components depend on interfaces at their boundaries where substitution is useful.

Prefer a pull-based custom `prometheus.Collector` over the current update channel and updater goroutine. This removes the battery-only `StateUpdate` transport and lets additional collectors or consumers reuse the same store without changing the parser.

## Implementation Steps

1. **Define the metric contract.** List the metrics to expose for each category, their units, Prometheus names/types, and labels. Candidate values already represented in the model include consumer power/energy, grid power/energy/currents, PV power/energy, charge-point charging/connection/power/energy, and vehicle SOC/range/odometer. Confirm EVCC units before naming metrics. Keep labels bounded and stable: source/site ID where needed, plus configured entity IDs; avoid titles, URLs, and other mutable display values.
2. **Resolve entity identity and lifecycle.** Specify stable IDs for grid meters, consumers, PV systems, charge points, and vehicles. Vehicle values currently live on a loadpoint, so define how vehicle identity and association behave when a vehicle is disconnected or moved. Define how complete-list updates, partial updates, nulls, and removed entities affect stored state and exported series.
3. **Create narrow domain types.** Replace the all-purpose metrics-facing `EVCCData` / battery-only `StateUpdate` with typed state for only the selected metric inputs. Keep transport DTOs separate if they need to reflect EVCC JSON shape. Model collections by entity ID and include a source ID if multiple EVCC sources are supported.
4. **Extract EVCC decoding.** Replace metric-facing state mutation in `MessageToState` with a decoder that accepts WebSocket payloads and emits typed updates for the selected entities. Preserve partial-update semantics and useful field-path errors. Remove forecast, tariff, remote, EEBUS, logging, and other decoding/model paths that are not part of the approved metric contract. Retain only protocol parsing needed to reach the selected metric values.
5. **Implement the state store.** Add `Apply(update)` and `Snapshot()` behavior behind a small interface. Protect internal maps with a mutex and return detached snapshots so scrapes cannot race with updates. Keep the store independent of Prometheus and JSON decoding.
6. **Implement the Prometheus collector.** Register one collector that reads a store snapshot and emits the approved metrics for every entity. Define descriptors once, preserve stable metric names/labels, and make empty or removed entities stop emitting samples. Avoid a separate mutable gauge vector per parser update unless scrape-time collection proves unsuitable.
7. **Wire components in `main`.** Construct the decoder, store, collector, registry, and HTTP server explicitly. Keep `/metrics` on a private mux. Remove the `StateUpdate` channel and metrics updater goroutine; use a channel only if it separates the WebSocket reader from the decoder/store for a measured reason.
8. **Prune and migrate tests.** Replace broad-model parser tests with focused decoder tests for each retained entity, partial updates, invalid values, unknown fields, and entity removal. Add state-store tests for concurrent snapshots and update semantics, and collector tests that gather metrics for multiple IDs and verify names, units, labels, and values. Remove forecast/plan tests together with their deleted code. Run `go test -race ./...`.
9. **Document the output contract.** Update the README with the supported metrics, units, labels, source configuration, and any metric-name migration from `battery_soc`.

## Decisions To Confirm Before Implementation

- Keep the previously requested battery SOC metric in the initial scope, or replace it with the newly listed entity categories?
- Which exact measurements should be exported for each category, and what units does this EVCC endpoint return?
- What stable IDs are available for each entity, especially vehicles and grid meters?
- Will the exporter monitor multiple EVCC sources in the same process? If so, define the configured source label and source lifecycle.
- Should existing metric names remain compatible, or is a breaking rename acceptable?

## Completion Criteria

- The decoder and domain store do not import Prometheus packages.
- No full EVCC model fields or parser branches remain unless they feed an approved metric or are required by the protocol.
- All supported entities produce distinct series with stable labels; partial updates do not erase unrelated fields.
- Removed entities stop being exported, invalid updates do not corrupt valid state, and race-enabled tests pass.
