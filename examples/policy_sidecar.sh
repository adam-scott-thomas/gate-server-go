#!/bin/bash
# Example: gate-policy as sidecar consuming gate-server-go
#
# Architecture:
#   Agent -> gate-server-go:8090 -> filter tools by mode
#                |
#                v
#         gate-policy:8091 -> evaluate policy rules against visible tools
#
# This script demonstrates how gate-policy (Layer 2) would call
# gate-server-go (Layer 1) to get the mode-filtered tool list,
# then apply policy rules on top.
#
# In production, gate-policy's PolicyGate would:
# 1. Call POST /v1/filter to get mode-suppressed results
# 2. Run its own rule engine over the visible tools
# 3. Return a combined result: mode_suppressed + policy_denied + visible

BASE=http://localhost:8090

echo "=== Step 1: gate-policy registers tools via gate-server-go ==="
curl -s -X POST $BASE/v1/tools \
  -H 'Content-Type: application/json' \
  -d '{
    "tools": [
      {"name": "read_file",    "execution_class": "read_only"},
      {"name": "deploy",       "execution_class": "high_impact"},
      {"name": "send_slack",   "execution_class": "external_action"},
      {"name": "delete_table", "execution_class": "high_impact"}
    ]
  }'

echo -e "\n\n=== Step 2: gate-policy asks for filtered tools at current mode ==="
RESULT=$(curl -s -X POST $BASE/v1/filter \
  -H 'Content-Type: application/json' \
  -d '{"mode": 0.3}')
echo "$RESULT" | python -m json.tool

echo -e "\n=== Step 3: gate-policy applies its own rules ==="
echo "  (policy engine would further deny 'delete_table' based on"
echo "   role=developer + no human_approved, even though mode allows it)"

echo -e "\n=== Step 4: Before executing, validate via ingress ==="
curl -s -X POST $BASE/v1/validate \
  -H 'Content-Type: application/json' \
  -d '{"tool_name": "deploy", "mode": 0.3}' | python -m json.tool

echo -e "\n=== Step 5: Get envelope for approved tool ==="
curl -s -X POST $BASE/v1/envelope \
  -H 'Content-Type: application/json' \
  -d '{"tool_name": "read_file", "context_id": "policy_session_42", "mode": 0.3}' \
  | python -m json.tool
