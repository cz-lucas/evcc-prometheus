package evccprometheus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type Decoder struct{}

// NewDecoder creates a decoder for EVCC fields used by the exported metrics.
func NewDecoder() *Decoder { return &Decoder{} }

// Decode returns no update if any recognized field is invalid, so callers can apply messages atomically.
func (d *Decoder) Decode(message string) (StateUpdate, error) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal([]byte(message), &payload); err != nil {
		return StateUpdate{}, fmt.Errorf("decode websocket message: %w", err)
	}
	if payload == nil {
		return StateUpdate{}, fmt.Errorf("decode websocket message: expected a JSON object")
	}
	update := StateUpdate{}
	for key, raw := range payload {
		var err error
		switch key {
		case "battery":
			err = decodeBatteries(raw, &update)
		case "pv":
			err = decodePVSystems(raw, &update)
		case "consumers":
			err = decodeConsumers(raw, &update)
		case "grid":
			err = decodeGrid(raw, &update)
		default:
			if strings.HasPrefix(key, "loadpoints.") {
				err = decodeLoadpoint(key, raw, &update)
			}
		}
		if err != nil {
			return StateUpdate{}, fmt.Errorf("field %q: %w", key, err)
		}
	}
	return update, nil
}

// decodeBatteries extracts named battery-device SoC values from an EVCC battery object.
func decodeBatteries(raw json.RawMessage, update *StateUpdate) error {
	if isNull(raw) {
		update.ReplaceBattery = true
		update.Batteries = map[string]BatteryState{}
		return nil
	}
	object, err := decodeObject(raw)
	if err != nil {
		return err
	}
	devices, ok := object["devices"]
	if !ok {
		return nil
	}
	update.ReplaceBattery = true
	update.Batteries = make(map[string]BatteryState)
	return decodeEntityList(devices, func(fields map[string]json.RawMessage, id string) (string, BatteryState, error) {
		soc, err := decodeNumber(fields, "soc")
		return id, BatteryState{SOC: soc}, err
	}, &update.Batteries)
}

// decodePVSystems converts the complete PV meter list into typed entity updates.
func decodePVSystems(raw json.RawMessage, update *StateUpdate) error {
	update.ReplacePV = true
	update.PVSystems = make(map[string]PVState)
	return decodeEntityList(raw, func(fields map[string]json.RawMessage, id string) (string, PVState, error) {
		power, err := decodeNumber(fields, "power")
		if err != nil {
			return "", PVState{}, err
		}
		energy, err := decodeNumber(fields, "energy")
		return id, PVState{Power: power, Energy: energy}, err
	}, &update.PVSystems)
}

// decodeConsumers converts the complete consumer meter list into typed entity updates.
func decodeConsumers(raw json.RawMessage, update *StateUpdate) error {
	update.ReplaceConsumer = true
	update.Consumers = make(map[string]ConsumerState)
	return decodeEntityList(raw, func(fields map[string]json.RawMessage, id string) (string, ConsumerState, error) {
		power, err := decodeNumber(fields, "power")
		if err != nil {
			return "", ConsumerState{}, err
		}
		energy, err := decodeNumber(fields, "energy")
		return id, ConsumerState{Power: power, Energy: energy}, err
	}, &update.Consumers)
}

// decodeGrid extracts the grid meter's power, energy, and phase currents.
func decodeGrid(raw json.RawMessage, update *StateUpdate) error {
	update.ReplaceGrid = true
	update.GridMeters = make(map[string]GridState)
	if isNull(raw) {
		return nil
	}
	fields, err := decodeObject(raw)
	if err != nil {
		return err
	}
	id := "grid"
	if name, ok := fields["name"]; ok && !isNull(name) {
		if err := json.Unmarshal(name, &id); err != nil {
			return fmt.Errorf("name: %w", err)
		}
		if id == "" {
			id = "grid"
		}
	}
	power, err := decodeNumber(fields, "power")
	if err != nil {
		return err
	}
	energy, err := decodeNumber(fields, "energy")
	if err != nil {
		return err
	}
	currents, err := decodeNumberSlice(fields, "currents")
	if err != nil {
		return err
	}
	update.GridMeters[id] = GridState{Power: power, Energy: energy, Currents: currents}
	return nil
}

// A recognized list is a complete category snapshot; its names become stable state keys.
func decodeEntityList[T any](raw json.RawMessage, decode func(map[string]json.RawMessage, string) (string, T, error), target *map[string]T) error {
	if isNull(raw) {
		return nil
	}
	var entities []json.RawMessage
	if err := json.Unmarshal(raw, &entities); err != nil {
		return err
	}
	for index, entity := range entities {
		fields, err := decodeObject(entity)
		if err != nil {
			return fmt.Errorf("entity %d: %w", index, err)
		}
		nameRaw, ok := fields["name"]
		if !ok || isNull(nameRaw) {
			return fmt.Errorf("entity %d: missing name", index)
		}
		var id string
		if err := json.Unmarshal(nameRaw, &id); err != nil {
			return fmt.Errorf("entity %d name: %w", index, err)
		}
		if id == "" {
			return fmt.Errorf("entity %d: empty name", index)
		}
		id, state, err := decode(fields, id)
		if err != nil {
			return fmt.Errorf("entity %q: %w", id, err)
		}
		(*target)[id] = state
	}
	return nil
}

// decodeLoadpoint decodes one dotted loadpoint field or an explicit entity removal.
func decodeLoadpoint(key string, raw json.RawMessage, update *StateUpdate) error {
	parts := strings.Split(key, ".")
	if len(parts) != 2 && len(parts) != 3 {
		return fmt.Errorf("invalid loadpoint key: expected loadpoints.<index>[.<field>]")
	}
	index, err := strconv.Atoi(parts[1])
	if err != nil || index < 0 {
		return fmt.Errorf("invalid loadpoint index %q", parts[1])
	}
	id := strconv.Itoa(index)
	if len(parts) == 2 {
		// EVCC uses a null loadpoint entry to signal entity removal.
		if !isNull(raw) {
			return fmt.Errorf("loadpoint removal must be null")
		}
		update.RemoveChargePoints = append(update.RemoveChargePoints, id)
		return nil
	}
	if update.ChargePoints == nil {
		update.ChargePoints = make(map[string]ChargePointState)
	}
	point := update.ChargePoints[id]
	switch parts[2] {
	case "name":
		point.Name, err = decodeValue[string](raw)
	case "charging":
		point.Charging, err = decodeBoolean(raw)
	case "connected":
		point.Connected, err = decodeBoolean(raw)
	case "chargePower":
		point.Power, err = decodeValue[float64](raw)
	case "chargedEnergy":
		point.ChargedEnergy, err = decodeValue[float64](raw)
	case "vehicleName":
		point.VehicleName, err = decodeValue[string](raw)
	case "vehicleSoc":
		point.VehicleSOC, err = decodeValue[float64](raw)
	case "vehicleRange":
		point.VehicleRange, err = decodeValue[float64](raw)
	case "vehicleOdometer":
		point.VehicleOdometer, err = decodeValue[float64](raw)
	default:
		return nil
	}
	if err != nil {
		return fmt.Errorf("%s: %w", parts[2], err)
	}
	update.ChargePoints[id] = point
	return nil
}

// decodeNumber decodes an optional numeric field while preserving null and omission.
func decodeNumber(fields map[string]json.RawMessage, key string) (Field[float64], error) {
	raw, ok := fields[key]
	if !ok {
		return Field[float64]{}, nil
	}
	return decodeValue[float64](raw)
}

// decodeNumberSlice decodes an optional numeric slice while preserving null and omission.
func decodeNumberSlice(fields map[string]json.RawMessage, key string) (Field[[]float64], error) {
	raw, ok := fields[key]
	if !ok {
		return Field[[]float64]{}, nil
	}
	return decodeValue[[]float64](raw)
}

// decodeBoolean accepts JSON booleans and EVCC's string-encoded booleans.
func decodeBoolean(raw json.RawMessage) (Field[bool], error) {
	if isNull(raw) {
		return Field[bool]{Set: true, Null: true}, nil
	}
	var value bool
	if err := json.Unmarshal(raw, &value); err == nil {
		return Field[bool]{Value: value, Set: true}, nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return Field[bool]{}, err
	}
	value, err := strconv.ParseBool(text)
	if err != nil {
		return Field[bool]{}, fmt.Errorf("invalid boolean string %q: %w", text, err)
	}
	return Field[bool]{Value: value, Set: true}, nil
}

// decodeValue converts a JSON value while distinguishing null from an omitted field.
func decodeValue[T any](raw json.RawMessage) (Field[T], error) {
	if isNull(raw) {
		return Field[T]{Set: true, Null: true}, nil
	}
	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		return Field[T]{}, err
	}
	return Field[T]{Value: value, Set: true}, nil
}

// decodeObject requires an object and returns its fields for metric-specific decoding.
func decodeObject(raw json.RawMessage) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	if fields == nil {
		return nil, fmt.Errorf("expected a JSON object")
	}
	return fields, nil
}

// isNull reports whether a raw JSON value is the null literal.
func isNull(raw json.RawMessage) bool { return bytes.Equal(bytes.TrimSpace(raw), []byte("null")) }
