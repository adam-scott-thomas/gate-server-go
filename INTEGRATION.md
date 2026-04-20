# gate-server-go — Integration Intent

How gate-server-go connects to the Maelstrom Gate ecosystem.

## Dependency Direction

```
Layer 0:  gate-core          gate-server-go implements this spec
Layer 1:  gate-server-go <-- THIS PROJECT
          gate-sdk           could wrap server for remote mode
Layer 2:  gate-policy        calls /v1/filter, /v1/validate, /v1/thresholds
          gate-compliance    would consume /v1/envelope/verify
Layer 3:  gate-ctf           could use server as challenge backend
          dashboard          would hit all endpoints for visualization
```

## gate-core (Layer 0) — Implemented

gate-server-go is a standalone Go implementation of the Gate spec.
It does NOT import gate-core Python — it reimplements the spec in Go.
This means:
- Same suppression rules, same execution classes, same thresholds
- Same envelope schema and HMAC-SHA256 signing
- Same ingress validation logic
- Cross-language verification: an envelope signed by gate-core (Python)
  can be verified by gate-server-go (Go), and vice versa

## gate-sdk (Layer 1) — Ready to Consume

gate-sdk currently wraps gate-core locally. A remote mode would:

```python
from gate_sdk import GateClient

# Local mode (current)
client = GateClient(mode=0.0)

# Remote mode (via gate-server-go)
client = GateClient(mode=0.0, remote="http://localhost:8090")
# client.filter() -> POST /v1/filter
# client.validate() -> POST /v1/validate
```

The OpenAPI spec (`openapi.yaml`) provides the contract for this.

## gate-policy (Layer 2) — Two Integration Paths

### Path A: Sidecar (recommended for dev)

gate-policy runs alongside gate-server-go. It calls:
1. `POST /v1/filter` to get mode-suppressed tools
2. Applies its own policy rules on the visible set
3. Returns combined result

See `examples/policy_sidecar.sh` for the full flow.

### Path B: Threshold manipulation (lightweight)

gate-policy translates policy rules into threshold overrides:
1. Policy says "deny high_impact for role=developer" -> `PUT /v1/thresholds {"high_impact": 0.0}`
2. Policy says "allow external_action for role=admin" -> `PUT /v1/thresholds {"external_action": null}`

This is lossy (policies are richer than thresholds) but requires zero
code changes in gate-server-go.

## gate-ctf (Layer 3) — Challenge Backend

gate-ctf could use gate-server-go as the live backend for challenges:
- Players register tools via `POST /v1/tools`
- Challenge sets a mode, players must figure out which tools survive
- Envelope signing challenges: build valid envelopes, detect tampered ones
- Threshold puzzles: find the right thresholds to allow specific tool combos

## API Contract

Full OpenAPI 3.1 spec at `openapi.yaml`. All 8 endpoints documented with
request/response schemas. Any client in any language can generate bindings.
