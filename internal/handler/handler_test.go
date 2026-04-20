package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/adam-scott-thomas/gate-server-go/internal/envelope"
	"github.com/adam-scott-thomas/gate-server-go/internal/gate"
	"github.com/adam-scott-thomas/gate-server-go/internal/handler"
)

const testKey = "test-signing-key"

func setup() (*handler.Handler, *http.ServeMux) {
	g := gate.New(gate.DefaultThresholds())
	h := handler.New(g, testKey)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/tools", h.RegisterTools)
	mux.HandleFunc("POST /v1/filter", h.Filter)
	mux.HandleFunc("POST /v1/validate", h.Validate)
	mux.HandleFunc("POST /v1/envelope", h.Envelope)
	mux.HandleFunc("POST /v1/envelope/verify", h.VerifyEnvelope)
	mux.HandleFunc("PUT /v1/thresholds", h.SetThresholds)
	mux.HandleFunc("GET /v1/tools", h.ListTools)
	mux.HandleFunc("GET /health", h.Health)
	return h, mux
}

func post(mux *http.ServeMux, path string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func put(mux *http.ServeMux, path string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("PUT", path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func get(mux *http.ServeMux, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", path, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func decode(w *httptest.ResponseRecorder) map[string]any {
	var out map[string]any
	json.NewDecoder(w.Body).Decode(&out)
	return out
}

// === Full Demo Flow ===
// This test walks through the entire gate-server-go lifecycle:
// register → list → filter at 3 modes → validate → envelope → verify → threshold override

func TestFullDemoFlow(t *testing.T) {
	_, mux := setup()

	// Step 1: Health check
	w := get(mux, "/health")
	if w.Code != 200 {
		t.Fatalf("health: want 200, got %d", w.Code)
	}
	health := decode(w)
	if health["status"] != "ok" {
		t.Errorf("health status: want ok, got %v", health["status"])
	}

	// Step 2: Register tools
	w = post(mux, "/v1/tools", map[string]any{
		"tools": []map[string]any{
			{"name": "read_file", "execution_class": "read_only", "description": "Read a file"},
			{"name": "analyze", "execution_class": "advisory", "description": "Analyze data"},
			{"name": "send_email", "execution_class": "external_action", "description": "Send email"},
			{"name": "write_db", "execution_class": "state_mutation", "description": "Write to DB"},
			{"name": "deploy", "execution_class": "high_impact", "description": "Deploy to prod"},
		},
	})
	if w.Code != 200 {
		t.Fatalf("register: want 200, got %d", w.Code)
	}
	reg := decode(w)
	if reg["registered"] != float64(5) {
		t.Errorf("registered: want 5, got %v", reg["registered"])
	}

	// Step 3: List tools
	w = get(mux, "/v1/tools")
	if w.Code != 200 {
		t.Fatalf("list: want 200, got %d", w.Code)
	}

	// Step 4: Filter — Normal mode (0.1)
	w = post(mux, "/v1/filter", map[string]any{"mode": 0.1})
	if w.Code != 200 {
		t.Fatalf("filter normal: want 200, got %d", w.Code)
	}
	fr := decode(w)
	visible := fr["visible"].([]any)
	if len(visible) != 5 {
		t.Errorf("normal: want 5 visible, got %d", len(visible))
	}
	if fr["mode_zone"] != "normal" {
		t.Errorf("normal: want zone normal, got %v", fr["mode_zone"])
	}

	// Step 5: Filter — Elevated mode (0.5)
	w = post(mux, "/v1/filter", map[string]any{"mode": 0.5})
	fr = decode(w)
	visible = fr["visible"].([]any)
	suppressed := fr["suppressed"].([]any)
	if len(visible) != 4 {
		t.Errorf("elevated: want 4 visible, got %d", len(visible))
	}
	if len(suppressed) != 1 {
		t.Errorf("elevated: want 1 suppressed (deploy), got %d", len(suppressed))
	}

	// Step 6: Filter — Crisis mode (0.9)
	w = post(mux, "/v1/filter", map[string]any{"mode": 0.9})
	fr = decode(w)
	visible = fr["visible"].([]any)
	suppressed = fr["suppressed"].([]any)
	if len(visible) != 2 {
		t.Errorf("crisis: want 2 visible, got %d", len(visible))
	}
	if len(suppressed) != 3 {
		t.Errorf("crisis: want 3 suppressed, got %d", len(suppressed))
	}

	// Step 7: Validate — deploy at elevated (should be denied)
	w = post(mux, "/v1/validate", map[string]any{"tool_name": "deploy", "mode": 0.5})
	if w.Code != 403 {
		t.Errorf("validate deploy at 0.5: want 403, got %d", w.Code)
	}
	v := decode(w)
	if v["reason"] != "execution_class_suppressed" {
		t.Errorf("validate reason: want execution_class_suppressed, got %v", v["reason"])
	}

	// Step 8: Validate — read_file at crisis (should be accepted)
	w = post(mux, "/v1/validate", map[string]any{"tool_name": "read_file", "mode": 0.9})
	if w.Code != 200 {
		t.Errorf("validate read_file at 0.9: want 200, got %d", w.Code)
	}

	// Step 9: Validate — nonexistent tool
	w = post(mux, "/v1/validate", map[string]any{"tool_name": "nope", "mode": 0.1})
	if w.Code != 404 {
		t.Errorf("validate nonexistent: want 404, got %d", w.Code)
	}

	// Step 10: Build envelope
	w = post(mux, "/v1/envelope", map[string]any{
		"tool_name":  "read_file",
		"context_id": "demo_session",
		"mode":       0.5,
	})
	if w.Code != 200 {
		t.Fatalf("envelope: want 200, got %d", w.Code)
	}
	envData := decode(w)
	if envData["execution_mode"] != "cautious" {
		t.Errorf("envelope at 0.5: want cautious, got %v", envData["execution_mode"])
	}
	if envData["max_tool_calls"] != float64(10) {
		t.Errorf("envelope at 0.5: want max_tool_calls 10, got %v", envData["max_tool_calls"])
	}

	// Step 11: Verify that envelope
	w = post(mux, "/v1/envelope/verify", map[string]any{"envelope": envData})
	if w.Code != 200 {
		t.Fatalf("verify: want 200, got %d", w.Code)
	}
	vr := decode(w)
	if vr["valid"] != true {
		t.Error("envelope should verify as valid")
	}

	// Step 12: Tamper and re-verify
	envData["max_tool_calls"] = float64(9999)
	w = post(mux, "/v1/envelope/verify", map[string]any{"envelope": envData})
	vr = decode(w)
	if vr["valid"] != false {
		t.Error("tampered envelope should fail verification")
	}

	// Step 13: Override thresholds — tighten high_impact to 0.20
	w = put(mux, "/v1/thresholds", map[string]*float64{
		"high_impact": ptr(0.20),
	})
	if w.Code != 200 {
		t.Fatalf("thresholds: want 200, got %d", w.Code)
	}

	// Step 14: Filter again — deploy now suppressed even at 0.25
	w = post(mux, "/v1/filter", map[string]any{"mode": 0.25})
	fr = decode(w)
	suppressed = fr["suppressed"].([]any)
	found := false
	for _, s := range suppressed {
		tool := s.(map[string]any)
		if tool["name"] == "deploy" {
			found = true
		}
	}
	if !found {
		t.Error("deploy should be suppressed at 0.25 after threshold override to 0.20")
	}
}

func TestRegisterEmptyTools(t *testing.T) {
	_, mux := setup()
	w := post(mux, "/v1/tools", map[string]any{"tools": []any{}})
	if w.Code != 400 {
		t.Errorf("empty tools: want 400, got %d", w.Code)
	}
}

func TestEnvelopeWithoutSigningKey(t *testing.T) {
	g := gate.New(gate.DefaultThresholds())
	h := handler.New(g, "") // no key
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/envelope", h.Envelope)

	w := post(mux, "/v1/envelope", map[string]any{
		"tool_name": "x", "context_id": "s", "mode": 0.1,
	})
	if w.Code != 503 {
		t.Errorf("no signing key: want 503, got %d", w.Code)
	}
}

func TestFilterInvalidJSON(t *testing.T) {
	_, mux := setup()
	req := httptest.NewRequest("POST", "/v1/filter", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Errorf("invalid json: want 400, got %d", w.Code)
	}
}

// Verify envelope round-trip: build via handler, decode, verify via handler
func TestEnvelopeRoundTrip(t *testing.T) {
	_, mux := setup()

	// Build
	w := post(mux, "/v1/envelope", map[string]any{
		"tool_name": "deploy", "context_id": "rt_test", "mode": 0.1,
	})
	var env envelope.Envelope
	json.NewDecoder(w.Body).Decode(&env)

	if env.ExecutionMode != "standard" {
		t.Errorf("normal mode envelope: want standard, got %s", env.ExecutionMode)
	}
	if env.Branching != "auto" {
		t.Errorf("normal mode branching: want auto, got %s", env.Branching)
	}

	// Verify via handler
	w = post(mux, "/v1/envelope/verify", map[string]any{"envelope": env})
	vr := decode(w)
	if vr["valid"] != true {
		t.Error("round-trip envelope should verify")
	}
}

func ptr(f float64) *float64 { return &f }
