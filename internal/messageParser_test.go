package evccprometheus

import (
	"bytes"
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
	err := mergeMessage(slog.Default(), `{"forecast.co2":[[1700000000,1700003600]]}`, &state)
	if err == nil {
		t.Fatal("mergeMessage() error = nil, want malformed tuple error")
	}
	if !strings.Contains(err.Error(), "forecast point 0: expected 3 values, got 2") {
		t.Errorf("mergeMessage() error = %v, want tuple context", err)
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
