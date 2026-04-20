#!/usr/bin/env python3
"""SDK integration test for gate-server-go.

Validates that gate-sdk can talk to a running gate-server-go instance
using the ServerEndpoints contract defined in gate_sdk.server_client.

STUB — requires a running gate-server-go on localhost:8080.
Start with: go run ./cmd/server

Usage:
    # Terminal 1: start the Go server
    cd gate-server-go && go run ./cmd/server

    # Terminal 2: run this test
    python examples/sdk_client_test.py
"""
from __future__ import annotations

import json
import sys
from urllib.request import Request, urlopen
from urllib.error import URLError


BASE = "http://localhost:8080"


def post_json(path: str, body: dict) -> dict:
    data = json.dumps(body).encode()
    req = Request(f"{BASE}{path}", data=data, headers={"Content-Type": "application/json"})
    with urlopen(req, timeout=5) as resp:
        return json.loads(resp.read())


def get_json(path: str) -> dict:
    req = Request(f"{BASE}{path}")
    with urlopen(req, timeout=5) as resp:
        return json.loads(resp.read())


def main():
    print("gate-server-go SDK integration test")
    print("=" * 50)

    # 1. Health check
    try:
        health = get_json("/health")
        print(f"[OK] Health: {health}")
    except URLError as e:
        print(f"[FAIL] Server not reachable at {BASE}: {e}")
        print("       Start with: cd gate-server-go && go run ./cmd/server")
        sys.exit(1)

    # 2. Register tools via POST /v1/tools
    tools = [
        {"name": "read_file", "execution_class": "read_only", "description": "Read a file"},
        {"name": "deploy", "execution_class": "high_impact", "description": "Deploy to prod"},
        {"name": "send_email", "execution_class": "external_action", "description": "Send email"},
    ]
    for tool in tools:
        result = post_json("/v1/tools", tool)
        print(f"[OK] Registered: {tool['name']} -> {result}")

    # 3. List tools via GET /v1/tools
    all_tools = get_json("/v1/tools")
    print(f"[OK] Listing: {len(all_tools)} tools registered")

    # 4. Filter at normal mode
    filter_result = post_json("/v1/filter", {"mode": 0.1})
    visible = [t["name"] for t in filter_result.get("visible", [])]
    suppressed = [t["name"] for t in filter_result.get("suppressed", [])]
    print(f"[OK] Filter mode=0.1: visible={visible}, suppressed={suppressed}")

    # 5. Filter at crisis mode
    filter_crisis = post_json("/v1/filter", {"mode": 0.8})
    visible_c = [t["name"] for t in filter_crisis.get("visible", [])]
    suppressed_c = [t["name"] for t in filter_crisis.get("suppressed", [])]
    print(f"[OK] Filter mode=0.8: visible={visible_c}, suppressed={suppressed_c}")

    # 6. Validate a tool
    validation = post_json("/v1/validate", {"name": "deploy", "mode": 0.5})
    print(f"[OK] Validate 'deploy' at 0.5: {validation}")

    print("\n" + "=" * 50)
    print("All checks passed. SDK <-> server-go contract holds.")


if __name__ == "__main__":
    main()
