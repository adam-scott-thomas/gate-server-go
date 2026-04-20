#!/bin/bash
# gate-server-go demo — register tools, filter, validate, envelope
# Requires: gate-server-go running on localhost:8090
#   docker build -t gated . && docker run -p 8090:8090 -e GATE_SIGNING_KEY=demo-key gated

BASE=http://localhost:8090

echo "=== Health ==="
curl -s $BASE/health | python -m json.tool

echo -e "\n=== Register Tools ==="
curl -s -X POST $BASE/v1/tools \
  -H 'Content-Type: application/json' \
  -d '{
    "tools": [
      {"name": "read_file",  "execution_class": "read_only",       "description": "Read a file"},
      {"name": "analyze",    "execution_class": "advisory",         "description": "Analyze data"},
      {"name": "send_email", "execution_class": "external_action",  "description": "Send email"},
      {"name": "write_db",   "execution_class": "state_mutation",   "description": "Write to DB"},
      {"name": "deploy",     "execution_class": "high_impact",      "description": "Deploy to prod"}
    ]
  }' | python -m json.tool

echo -e "\n=== Filter: Normal (mode=0.1) ==="
curl -s -X POST $BASE/v1/filter \
  -H 'Content-Type: application/json' \
  -d '{"mode": 0.1}' | python -m json.tool

echo -e "\n=== Filter: Crisis (mode=0.9) ==="
curl -s -X POST $BASE/v1/filter \
  -H 'Content-Type: application/json' \
  -d '{"mode": 0.9}' | python -m json.tool

echo -e "\n=== Validate: deploy at mode=0.5 (should be denied) ==="
curl -s -X POST $BASE/v1/validate \
  -H 'Content-Type: application/json' \
  -d '{"tool_name": "deploy", "mode": 0.5}' | python -m json.tool

echo -e "\n=== Validate: read_file at mode=0.9 (should be accepted) ==="
curl -s -X POST $BASE/v1/validate \
  -H 'Content-Type: application/json' \
  -d '{"tool_name": "read_file", "mode": 0.9}' | python -m json.tool

echo -e "\n=== Build Envelope: read_file at mode=0.5 ==="
ENVELOPE=$(curl -s -X POST $BASE/v1/envelope \
  -H 'Content-Type: application/json' \
  -d '{"tool_name": "read_file", "context_id": "session_1", "mode": 0.5}')
echo "$ENVELOPE" | python -m json.tool

echo -e "\n=== Verify Envelope ==="
curl -s -X POST $BASE/v1/envelope/verify \
  -H 'Content-Type: application/json' \
  -d "{\"envelope\": $ENVELOPE}" | python -m json.tool

echo -e "\n=== Override Thresholds: tighten high_impact to 0.20 ==="
curl -s -X PUT $BASE/v1/thresholds \
  -H 'Content-Type: application/json' \
  -d '{"high_impact": 0.20}' | python -m json.tool

echo -e "\n=== Filter after threshold change (mode=0.25, deploy now suppressed) ==="
curl -s -X POST $BASE/v1/filter \
  -H 'Content-Type: application/json' \
  -d '{"mode": 0.25}' | python -m json.tool
