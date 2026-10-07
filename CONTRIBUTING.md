# Contributing

## Development setup

Install the Go version declared in `go.mod` (currently Go 1.27.1), then run the service from the repository root:

```sh
go mod download
go run ./cmd
```

The exporter connects to `wss://demo.evcc.io/ws` by default and serves Prometheus metrics at `http://localhost:9070/metrics`. You can override the connection and listen address with environment variables:

```sh
EVCC_WS_URL=wss://your-evcc-host/ws PROMETHEUS_ADDR=:9070 go run ./cmd
```

Other supported settings are `LOG_LEVEL` (`debug`, `info`, `warn`, or `error`) and `GO_METRICS_ENABLED` (`true` or `false`).

## Before submitting

All code must be formatted and all tests must pass before a change is submitted. From the repository root, run:

```sh
gofmt -w ./cmd ./internal
go test ./...
```

New features must include tests for their behavior. Bug fixes should include a regression test where practical. Keep changes focused, update documentation when behavior or configuration changes, and make sure the README and tests agree with the implementation. Do not include unrelated formatting or generated files in a change.

Use a clear, descriptive pull request summary and mention any relevant behavior changes or limitations. If a check cannot be run, state that in the pull request.
