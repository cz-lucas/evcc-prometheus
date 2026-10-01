package evccprometheus

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// MessageToState reads websocket messages and merges valid updates into EVCC state.
func MessageToState(logger *slog.Logger, websocketMessages <-chan string, evccData *EVCCData) {
	for message := range websocketMessages {
		logger.Debug("Received message from EVCC", "payload", message)
		if err := mergeMessage(logger, message, evccData); err != nil {
			logger.Warn("Failed to parse message from EVCC", "error", err)
			continue
		}
	}
}

// mergeMessage decodes one JSON update and merges recognized fields into EVCC state.
func mergeMessage(logger *slog.Logger, message string, evccData *EVCCData) error {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal([]byte(message), &payload); err != nil {
		return fmt.Errorf("decode websocket message: %w", err)
	}

	for key, rawValue := range payload {
		logger.Debug("Applying EVCC field update", "field", key)
		if strings.Contains(key, ".") {
			// EVCC sends forecasts and loadpoints as flat dotted keys.
			if err := mergeDottedField(evccData, key, rawValue); err != nil {
				return fmt.Errorf("field %q: %w", key, err)
			}
			continue
		}

		target, ok := jsonFieldTarget(evccData, key)
		if !ok {
			logger.Debug("Ignoring unknown EVCC field", "field", key)
			continue
		}
		if err := json.Unmarshal(rawValue, target); err != nil {
			return fmt.Errorf("field %q: %w", key, err)
		}
	}

	return nil
}

// mergeDottedField routes a dotted EVCC key to its forecast or loadpoint handler.
func mergeDottedField(evccData *EVCCData, key string, rawValue json.RawMessage) error {
	parts := strings.Split(key, ".")
	if len(parts) < 2 {
		return errors.New("invalid dotted key")
	}

	switch parts[0] {
	case "forecast":
		if len(parts) != 2 {
			return nil
		}
		return mergeForecastField(evccData, parts[1:], rawValue)
	case "loadpoints":
		return mergeLoadpointField(evccData, parts[1:], rawValue)
	default:
		return nil
	}
}

// mergeForecastField initializes forecast state and updates the named forecast series.
func mergeForecastField(evccData *EVCCData, parts []string, rawValue json.RawMessage) error {
	if len(parts) == 0 {
		return errors.New("missing forecast field")
	}

	field := parts[0]
	if field == "solar" {
		if evccData.Forecast == nil {
			evccData.Forecast = &Forecast{}
		}
		if evccData.Forecast.Solar == nil {
			evccData.Forecast.Solar = &ForecastSolar{}
		}
		return decodeForecastSolar(rawValue, evccData.Forecast.Solar)
	}

	switch field {
	case "co2", "feedin", "grid", "planner", "temperature":
	default:
		return nil
	}
	if evccData.Forecast == nil {
		evccData.Forecast = &Forecast{}
	}
	var points *[]ForecastPoint
	switch field {
	case "co2":
		points = &evccData.Forecast.CO2
	case "feedin":
		points = &evccData.Forecast.FeedIn
	case "grid":
		points = &evccData.Forecast.Grid
	case "planner":
		points = &evccData.Forecast.Planner
	case "temperature":
		points = &evccData.Forecast.Temperature
	}
	return decodeForecastPoints(rawValue, points)
}

// mergeLoadpointField creates the indexed loadpoint if needed and applies one field update.
func mergeLoadpointField(evccData *EVCCData, parts []string, rawValue json.RawMessage) error {
	if len(parts) != 2 {
		return errors.New("invalid loadpoint key: expected loadpoints.<index>.<field>")
	}

	index, err := strconv.Atoi(parts[0])
	if err != nil {
		return fmt.Errorf("invalid loadpoint index %q: %w", parts[0], err)
	}

	if evccData.Loadpoints == nil {
		evccData.Loadpoints = make(map[int]*Loadpoint)
	}

	loadpoint := evccData.Loadpoints[index]
	if loadpoint == nil {
		loadpoint = &Loadpoint{}
		evccData.Loadpoints[index] = loadpoint
	}

	return unmarshalLoadpointField(loadpoint, parts[1], rawValue)
}

func decodeForecastSolar(rawValue json.RawMessage, target *ForecastSolar) error {
	var solarFields map[string]json.RawMessage
	if err := json.Unmarshal(rawValue, &solarFields); err != nil {
		return err
	}

	timeSeries, hasTimeSeries := solarFields["timeseries"]
	delete(solarFields, "timeseries")
	solarWithoutTimeSeries, err := json.Marshal(solarFields)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(solarWithoutTimeSeries, target); err != nil {
		return err
	}
	if !hasTimeSeries {
		return nil
	}
	if string(timeSeries) == "null" {
		target.TimeSeries = nil
		return nil
	}
	return decodeForecastPoints(timeSeries, &target.TimeSeries)
}

func decodeForecastPoints(rawValue json.RawMessage, target *[]ForecastPoint) error {
	var rawPoints [][]json.RawMessage
	if err := json.Unmarshal(rawValue, &rawPoints); err != nil {
		return err
	}

	points := make([]ForecastPoint, 0, len(rawPoints))

	for i, rawPoint := range rawPoints {

		// Throw error if the forecast point does not have 2 or 3 values
		if len(rawPoint) != 2 && len(rawPoint) != 3 {
			return fmt.Errorf("forecast point %d: expected 2 or 3 values, got %d", i, len(rawPoint))
		}

		// Decode the start time from the first element of the forecast point array.
		var startUnix int64
		if err := json.Unmarshal(rawPoint[0], &startUnix); err != nil {
			return fmt.Errorf("forecast point %d start: %w", i, err)
		}

		// Decode the end time from the second element of the forecast point array if it exists.
		var value float64
		if err := json.Unmarshal(rawPoint[len(rawPoint)-1], &value); err != nil {
			return fmt.Errorf("forecast point %d value: %w", i, err)
		}

		point := ForecastPoint{
			Start: time.Unix(startUnix, 0).UTC(),
			Value: value,
		}

		// If the forecast point has 3 values, decode the end time from the second element of the array.
		if len(rawPoint) == 3 {
			var endUnix int64
			if err := json.Unmarshal(rawPoint[1], &endUnix); err != nil {
				return fmt.Errorf("forecast point %d end: %w", i, err)
			}

			point.End = time.Unix(endUnix, 0).UTC()
		}

		points = append(points, point)
	}

	*target = points
	return nil
}

// decodeLoadpointPlan parses plan entries and their RFC3339 start and end times.
func decodeLoadpointPlan(rawValue json.RawMessage, target *[]LoadpointPlanEntry) error {
	var rawEntries []struct {
		Start string  `json:"start"`
		End   string  `json:"end"`
		Value float64 `json:"value"`
	}
	if err := json.Unmarshal(rawValue, &rawEntries); err != nil {
		return err
	}

	entries := make([]LoadpointPlanEntry, 0, len(rawEntries))
	for entryIndex, entry := range rawEntries {
		startTime, err := time.Parse(time.RFC3339, entry.Start)
		if err != nil {
			return fmt.Errorf("plan entry %d start: %w", entryIndex, err)
		}
		endTime, err := time.Parse(time.RFC3339, entry.End)
		if err != nil {
			return fmt.Errorf("plan entry %d end: %w", entryIndex, err)
		}
		entries = append(entries, LoadpointPlanEntry{Start: startTime, End: endTime, Value: entry.Value})
	}

	*target = entries
	return nil
}

// unmarshalLoadpointField decodes a recognized scalar field into its loadpoint target.
func unmarshalLoadpointField(loadpoint *Loadpoint, field string, rawValue json.RawMessage) error {
	if field == "plan" {
		return decodeLoadpointPlan(rawValue, &loadpoint.Plan)
	}

	value, ok := jsonFieldTarget(loadpoint, field)
	if !ok {
		return nil
	}
	if err := json.Unmarshal(rawValue, value); err == nil {
		return nil
	} else if boolTarget, isBool := value.(*bool); isBool {
		var stringValue string
		if stringErr := json.Unmarshal(rawValue, &stringValue); stringErr == nil {
			parsedValue, parseErr := strconv.ParseBool(stringValue)
			if parseErr != nil {
				return fmt.Errorf("invalid boolean string %q: %w", stringValue, parseErr)
			}
			*boolTarget = parsedValue
			return nil
		}
		return err
	} else {
		return err
	}
}

func jsonFieldTarget(target any, key string) (any, bool) {
	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return nil, false
	}

	value = value.Elem()
	if value.Kind() != reflect.Struct {
		return nil, false
	}

	for fieldIndex := 0; fieldIndex < value.NumField(); fieldIndex++ {
		field := value.Type().Field(fieldIndex)
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name == key && name != "-" && value.Field(fieldIndex).CanAddr() {
			return value.Field(fieldIndex).Addr().Interface(), true
		}
	}

	return nil, false
}
