package evccprometheus

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestMergeMessageDecodesSolarTimeseriesTuples(t *testing.T) {
	var state EVCCData
	message := `{"forecast.solar":{"scale":0.75,"timeseries":[[1700000000,1700003600,0.42]]}}`

	if err := mergeMessage(slog.Default(), message, &state); err != nil {
		t.Fatalf("mergeMessage() error = %v", err)
	}
	if state.Forecast == nil || state.Forecast.Solar == nil {
		t.Fatal("solar forecast was not initialized")
	}
	if state.Forecast.Solar.Scale != 0.75 {
		t.Errorf("solar scale = %v, want 0.75", state.Forecast.Solar.Scale)
	}
	if len(state.Forecast.Solar.TimeSeries) != 1 {
		t.Fatalf("solar timeseries length = %d, want 1", len(state.Forecast.Solar.TimeSeries))
	}

	point := state.Forecast.Solar.TimeSeries[0]
	if want := time.Unix(1700000000, 0).UTC(); !point.Start.Equal(want) {
		t.Errorf("point start = %v, want %v", point.Start, want)
	}
	if want := time.Unix(1700003600, 0).UTC(); !point.End.Equal(want) {
		t.Errorf("point end = %v, want %v", point.End, want)
	}
	if point.Value != 0.42 {
		t.Errorf("point value = %v, want 0.42", point.Value)
	}
}

func TestMergeMessageIgnoresUnknownDottedFields(t *testing.T) {
	var state EVCCData
	message := `{"future.value":12,"forecast.future":[[1,2,3]]}`

	if err := mergeMessage(slog.Default(), message, &state); err != nil {
		t.Fatalf("mergeMessage() error = %v, want nil", err)
	}
	if state.Forecast != nil {
		t.Errorf("forecast = %#v, want nil for an unknown forecast field", state.Forecast)
	}
}

func TestMergeMessageRejectsExtraLoadpointPathSegments(t *testing.T) {
	var state EVCCData
	err := mergeMessage(slog.Default(), `{"loadpoints.0.plan.foo":[]}`, &state)
	if err == nil {
		t.Fatal("mergeMessage() error = nil, want invalid loadpoint path error")
	}
	if !strings.Contains(err.Error(), `field "loadpoints.0.plan.foo"`) {
		t.Errorf("mergeMessage() error = %v, want field context", err)
	}
}

func TestMergeMessageDecodesStringLoadpointBooleans(t *testing.T) {
	var state EVCCData
	err := mergeMessage(slog.Default(), `{"loadpoints.2.alwaysCharge":"true"}`, &state)
	if err != nil {
		t.Fatalf("mergeMessage() error = %v, want nil", err)
	}
	if !state.Loadpoints[2].AlwaysCharge {
		t.Error("loadpoints[2].AlwaysCharge = false, want true")
	}
}

func TestMergeMessageRejectsInvalidStringLoadpointBoolean(t *testing.T) {
	var state EVCCData
	err := mergeMessage(slog.Default(), `{"loadpoints.2.alwaysCharge":"sometimes"}`, &state)
	if err == nil {
		t.Fatal("mergeMessage() error = nil, want invalid boolean string error")
	}
	if !strings.Contains(err.Error(), `field "loadpoints.2.alwaysCharge": invalid boolean string "sometimes"`) {
		t.Errorf("mergeMessage() error = %v, want field and value context", err)
	}
}

func TestMergeMessageRejectsMalformedForecastTuple(t *testing.T) {
	var state EVCCData
	err := mergeMessage(slog.Default(), `{"forecast.co2":[[1700000000]]}`, &state)
	if err == nil {
		t.Fatal("mergeMessage() error = nil, want malformed tuple error")
	}
	if !strings.Contains(err.Error(), "forecast point 0: expected 2 or 3 values, got 1") {
		t.Errorf("mergeMessage() error = %v, want tuple context", err)
	}
}

func TestMergeMessageErrorPaths(t *testing.T) {
	tests := []struct {
		name    string
		message string
		wantErr string
	}{
		{name: "invalid json", message: `{"apiReady":`, wantErr: "decode websocket message"},
		{name: "invalid root field value", message: `{"apiReady":"bad"}`, wantErr: `field "apiReady"`},
		{name: "unknown root field", message: `{"future":1}`},
		{name: "unknown dotted namespace", message: `{"future.value":1}`},
		{name: "forecast extra path", message: `{"forecast.solar.extra":1}`},
		{name: "loadpoint missing field", message: `{"loadpoints.0":1}`, wantErr: "invalid loadpoint key"},
		{name: "invalid loadpoint index", message: `{"loadpoints.nope.mode":"off"}`, wantErr: "invalid loadpoint index"},
		{name: "unknown loadpoint field", message: `{"loadpoints.0.future":1}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := mergeMessage(slog.Default(), test.message, &EVCCData{})
			if test.wantErr == "" && err != nil {
				t.Fatalf("mergeMessage() error = %v, want nil", err)
			}
			if test.wantErr != "" && (err == nil || !strings.Contains(err.Error(), test.wantErr)) {
				t.Fatalf("mergeMessage() error = %v, want containing %q", err, test.wantErr)
			}
		})
	}
}

func TestDecodeForecastSolarCases(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "invalid solar object", value: `[`, wantErr: true},
		{name: "without timeseries", value: `{"scale":0.5}`},
		{name: "null timeseries", value: `{"timeseries":null}`},
		{name: "invalid timeseries", value: `{"timeseries":[[1]]}`, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var solar ForecastSolar
			err := decodeForecastSolar(json.RawMessage(test.value), &solar)
			if (err != nil) != test.wantErr {
				t.Fatalf("decodeForecastSolar() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestMergeForecastFields(t *testing.T) {
	for _, field := range []string{"co2", "feedin", "grid", "planner", "temperature"} {
		t.Run(field, func(t *testing.T) {
			var state EVCCData
			message := `{"forecast.` + field + `":[[1700000000,0.5]]}`
			if err := mergeMessage(slog.Default(), message, &state); err != nil {
				t.Fatalf("mergeMessage() error = %v", err)
			}
			if state.Forecast == nil {
				t.Fatal("forecast was not initialized")
			}
		})
	}
	if err := mergeForecastField(&EVCCData{}, nil, nil); err == nil {
		t.Fatal("mergeForecastField() error = nil, want missing field error")
	}
}

func TestDecodeForecastPointsInvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "invalid outer value", value: `{"not":"an array"}`},
		{name: "invalid start", value: `[["bad",0.5]]`},
		{name: "invalid value", value: `[[1,"bad"]]`},
		{name: "invalid end", value: `[[1,"bad",0.5]]`},
		{name: "empty tuple", value: `[[]]`},
		{name: "too many tuple values", value: `[[1,2,3,4]]`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var points []ForecastPoint
			if err := decodeForecastPoints(json.RawMessage(test.value), &points); err == nil {
				t.Fatal("decodeForecastPoints() error = nil, want error")
			}
		})
	}
}

func TestDecodeLoadpointPlanCases(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "invalid json", value: `[`},
		{name: "invalid start", value: `[{"start":"bad","end":"2024-01-01T01:00:00Z"}]`},
		{name: "invalid end", value: `[{"start":"2024-01-01T00:00:00Z","end":"bad"}]`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var plan []LoadpointPlanEntry
			if err := decodeLoadpointPlan(json.RawMessage(test.value), &plan); err == nil {
				t.Fatal("decodeLoadpointPlan() error = nil, want error")
			}
		})
	}
	var plan []LoadpointPlanEntry
	if err := decodeLoadpointPlan(json.RawMessage(`[{"start":"2024-01-01T00:00:00Z","end":"2024-01-01T01:00:00Z","value":2}]`), &plan); err != nil {
		t.Fatalf("decodeLoadpointPlan() error = %v", err)
	}
}

func TestLoadpointFieldBooleanFallbackAndPlanError(t *testing.T) {
	for _, value := range []string{`1`, `[]`} {
		if err := unmarshalLoadpointField(&Loadpoint{}, "alwaysCharge", json.RawMessage(value)); err == nil {
			t.Errorf("unmarshalLoadpointField(%s) error = nil, want type error", value)
		}
	}
	if err := unmarshalLoadpointField(&Loadpoint{}, "plan", json.RawMessage(`[{}]`)); err == nil {
		t.Fatal("unmarshalLoadpointField(plan) error = nil, want time parse error")
	}
}

func TestJSONFieldTargetKindsAndMisses(t *testing.T) {
	var nilData *EVCCData
	var scalar int
	var nilStruct *struct {
		Value int `json:"value"`
	}
	var ignored struct {
		Value int `json:"-"`
	}
	for name, target := range map[string]any{
		"nil pointer":        nilData,
		"non-pointer":        1,
		"pointer to scalar":  &scalar,
		"nil struct pointer": nilStruct,
	} {
		if _, ok := jsonFieldTarget(target, "value"); ok {
			t.Errorf("jsonFieldTarget(%s) unexpectedly found a field", name)
		}
	}
	if _, ok := jsonFieldTarget(&ignored, "value"); ok {
		t.Error("jsonFieldTarget() found a field tagged with '-' ")
	}
	if _, ok := jsonFieldTarget(&EVCCData{}, "unknown"); ok {
		t.Error("jsonFieldTarget() found an unknown field")
	}
}

func TestMessageToStateLogsParseFailuresWithContext(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug}))
	messages := make(chan string, 1)
	messages <- `{"apiReady":"not-a-bool"}`
	close(messages)

	MessageToState(logger, messages, &EVCCData{})

	logOutput := output.String()
	if !strings.Contains(logOutput, "level=WARN") {
		t.Errorf("log output does not contain a warning: %s", logOutput)
	}
	if strings.Contains(logOutput, "level=ERROR") {
		t.Errorf("log output contains an error instead of a warning: %s", logOutput)
	}
	if !strings.Contains(logOutput, `field \"apiReady\"`) {
		t.Errorf("log output does not identify the source field: %s", logOutput)
	}
	if !strings.Contains(logOutput, "level=DEBUG") || !strings.Contains(logOutput, "Applying EVCC field update") {
		t.Errorf("log output does not contain the field debug trace: %s", logOutput)
	}
}
