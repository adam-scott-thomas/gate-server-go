#!/usr/bin/env python3
"""gate-server-go test client — verifies the running server against the spec.

Usage:
    docker build -t gated .
    docker run -p 8090:8090 -e GATE_SIGNING_KEY=test-key gated
    python test_client.py
"""

import json
import sys
import urllib.request

BASE = "http://localhost:8090"


def req(method, path, body=None):
    data = json.dumps(body).encode() if body else None
    r = urllib.request.Request(f"{BASE}{path}", data=data, method=method)
    r.add_header("Content-Type", "application/json")
    try:
        resp = urllib.request.urlopen(r)
        return resp.status, json.loads(resp.read())
    except urllib.error.HTTPError as e:
        return e.code, json.loads(e.read())


def check(label, condition, detail=""):
    status = "PASS" if condition else "FAIL"
    print(f"  [{status}] {label}" + (f" — {detail}" if detail and not condition else ""))
    return condition


def main():
    passed = 0
    failed = 0

    print("=== gate-server-go test client ===\n")

    # Health
    print("[1] Health check")
    code, data = req("GET", "/health")
    if check("GET /health returns 200", code == 200):
        passed += 1
    else:
        failed += 1
    if check("status is ok", data.get("status") == "ok"):
        passed += 1
    else:
        failed += 1

    # Register tools
    print("\n[2] Register tools")
    code, data = req("POST", "/v1/tools", {"tools": [
        {"name": "read_file",  "execution_class": "read_only"},
        {"name": "analyze",    "execution_class": "advisory"},
        {"name": "send_email", "execution_class": "external_action"},
        {"name": "write_db",   "execution_class": "state_mutation"},
        {"name": "deploy",     "execution_class": "high_impact"},
    ]})
    if check("registers 5 tools", data.get("registered") == 5):
        passed += 1
    else:
        failed += 1

    # Filter: normal
    print("\n[3] Filter — normal mode (0.1)")
    code, data = req("POST", "/v1/filter", {"mode": 0.1})
    if check("5 visible", len(data.get("visible", [])) == 5):
        passed += 1
    else:
        failed += 1
    if check("zone is normal", data.get("mode_zone") == "normal"):
        passed += 1
    else:
        failed += 1

    # Filter: elevated
    print("\n[4] Filter — elevated mode (0.5)")
    code, data = req("POST", "/v1/filter", {"mode": 0.5})
    if check("4 visible", len(data.get("visible", [])) == 4):
        passed += 1
    else:
        failed += 1
    if check("1 suppressed (deploy)", len(data.get("suppressed", [])) == 1):
        passed += 1
    else:
        failed += 1

    # Filter: crisis
    print("\n[5] Filter — crisis mode (0.9)")
    code, data = req("POST", "/v1/filter", {"mode": 0.9})
    if check("2 visible (read_only + advisory)", len(data.get("visible", [])) == 2):
        passed += 1
    else:
        failed += 1
    if check("3 suppressed", len(data.get("suppressed", [])) == 3):
        passed += 1
    else:
        failed += 1

    # Validate
    print("\n[6] Ingress validation")
    code, _ = req("POST", "/v1/validate", {"tool_name": "deploy", "mode": 0.5})
    if check("deploy at 0.5 → 403", code == 403):
        passed += 1
    else:
        failed += 1
    code, _ = req("POST", "/v1/validate", {"tool_name": "read_file", "mode": 0.9})
    if check("read_file at 0.9 → 200", code == 200):
        passed += 1
    else:
        failed += 1
    code, data = req("POST", "/v1/validate", {"tool_name": "fake", "mode": 0.1})
    if check("nonexistent → 404", code == 404):
        passed += 1
    else:
        failed += 1

    # Envelope
    print("\n[7] Envelope build + verify")
    code, env = req("POST", "/v1/envelope", {
        "tool_name": "read_file", "context_id": "test_sess", "mode": 0.5,
    })
    if check("builds envelope", code == 200):
        passed += 1
    else:
        failed += 1
    if check("cautious at elevated", env.get("execution_mode") == "cautious"):
        passed += 1
    else:
        failed += 1

    code, vr = req("POST", "/v1/envelope/verify", {"envelope": env})
    if check("verifies valid", vr.get("valid") is True):
        passed += 1
    else:
        failed += 1

    env["max_tool_calls"] = 9999
    code, vr = req("POST", "/v1/envelope/verify", {"envelope": env})
    if check("tampered → invalid", vr.get("valid") is False):
        passed += 1
    else:
        failed += 1

    # Threshold override
    print("\n[8] Threshold override")
    code, _ = req("PUT", "/v1/thresholds", {"high_impact": 0.20})
    if check("updates thresholds", code == 200):
        passed += 1
    else:
        failed += 1

    code, data = req("POST", "/v1/filter", {"mode": 0.25})
    suppressed_names = [t["name"] for t in data.get("suppressed", [])]
    if check("deploy suppressed at 0.25 after override", "deploy" in suppressed_names):
        passed += 1
    else:
        failed += 1

    # Summary
    total = passed + failed
    print(f"\n=== Results: {passed}/{total} passed ===")
    return 0 if failed == 0 else 1


if __name__ == "__main__":
    sys.exit(main())
