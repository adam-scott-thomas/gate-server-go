# gate-server-go

[![status](https://img.shields.io/badge/status-v0.1.0-blue)]()
[![tests](https://img.shields.io/badge/tests-19_passing-brightgreen)]()
[![license](https://img.shields.io/badge/license-Apache_2.0-green)]()

> Go reimplementation of gate-server. Same wire protocol, lower latency.

Drop-in replacement for the Python `gate-server` when you need single-digit-ms
filter responses or a static binary. Implements the same OpenAPI 3.1 contract,
so `gate-sdk`, `gatectl`, and the dashboards all talk to it unchanged.

## Build

```bash
go build -o gated ./cmd/gated
./gated -addr :8090 -key $GATE_SIGNING_KEY
```

Or via Docker:

```bash
docker build -t gate-server-go .
docker run -p 8090:8090 -e GATE_SIGNING_KEY=changeme gate-server-go
```

## Endpoints

```
POST /v1/tools             register tool manifest
POST /v1/filter            filter at current mode
POST /v1/validate          validate proposal against mode
POST /v1/envelope/build    issue HMAC-signed envelope
POST /v1/envelope/verify   verify envelope
GET  /v1/mode              current mode + zone
POST /v1/mode              set mode
GET  /health               liveness
```

Full schema: `openapi.yaml`.

## Quick example

```bash
curl -X POST localhost:8090/v1/tools \
  -H 'Content-Type: application/json' \
  -d '{"tools":[{"name":"deploy","execution_class":"high_impact"}]}'

curl -X POST localhost:8090/v1/filter \
  -H 'Content-Type: application/json' \
  -d '{"mode":0.85}'
# {"visible":[],"suppressed":[{"name":"deploy","execution_class":"high_impact"}],...}
```

## Layout

- `cmd/gated` — binary entry point
- `internal/gate` — zone/threshold/filter logic
- `internal/envelope` — HMAC envelope build + verify
- `internal/handler` — HTTP handlers

## Tests

```bash
go test ./...
```

19 tests across gate logic, envelope signing, and handler responses.

## Parity

- Threshold defaults match `maelstrom-gate` SPEC.md §5
- Envelope signatures verify cross-language (tested by `gate-test`'s
  cross-language conformance vectors)

## How it fits

Layer 1 (transport) in [Maelstrom Gate](https://github.com/adam-scott-thomas/maelstrom-gate).
Swap `gate-server` for `gate-server-go` behind the same URL and every client
keeps working.

## License

Apache-2.0.
