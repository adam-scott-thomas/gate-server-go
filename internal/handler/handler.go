package handler

import (
	"encoding/json"
	"net/http"

	"github.com/adam-scott-thomas/gate-server-go/internal/envelope"
	"github.com/adam-scott-thomas/gate-server-go/internal/gate"
)

// Handler serves the Gate HTTP API.
type Handler struct {
	gate       *gate.Gate
	signingKey string
}

// New creates a Handler wrapping a Gate instance.
func New(g *gate.Gate, signingKey string) *Handler {
	return &Handler{gate: g, signingKey: signingKey}
}

// RegisterTools adds tools to the gate.
// POST /v1/tools  body: {"tools": [...]}
func (h *Handler) RegisterTools(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Tools []gate.Tool `json:"tools"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if len(req.Tools) == 0 {
		writeErr(w, http.StatusBadRequest, "empty_tools", "at least one tool required")
		return
	}
	h.gate.AddTools(req.Tools)
	writeJSON(w, http.StatusOK, map[string]any{"registered": len(req.Tools)})
}

// ListTools returns all registered tools.
// GET /v1/tools
func (h *Handler) ListTools(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"tools": h.gate.Tools()})
}

// Filter applies mode-based suppression.
// POST /v1/filter  body: {"mode": 0.5}
func (h *Handler) Filter(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Mode float64 `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	result := h.gate.Filter(req.Mode)
	writeJSON(w, http.StatusOK, result)
}

// Validate checks a tool proposal against the gate.
// POST /v1/validate  body: {"tool_name": "deploy", "mode": 0.5}
func (h *Handler) Validate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ToolName string  `json:"tool_name"`
		Mode     float64 `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	err := h.gate.ValidateProposal(req.ToolName, req.Mode)
	if err == nil {
		writeJSON(w, http.StatusOK, map[string]any{"accepted": true})
		return
	}
	status := http.StatusForbidden
	if err == gate.ErrToolNotFound {
		status = http.StatusNotFound
	}
	writeJSON(w, status, map[string]any{"accepted": false, "reason": err.Error()})
}

// Envelope builds a signed authorization envelope.
// POST /v1/envelope  body: {"tool_name": "read_file", "context_id": "sess1", "mode": 0.5}
func (h *Handler) Envelope(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ToolName  string  `json:"tool_name"`
		ContextID string  `json:"context_id"`
		Mode      float64 `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if h.signingKey == "" {
		writeErr(w, http.StatusServiceUnavailable, "no_signing_key", "server has no signing key configured")
		return
	}
	env := envelope.Build(req.ToolName, req.ContextID, req.Mode, h.signingKey)
	writeJSON(w, http.StatusOK, env)
}

// VerifyEnvelope checks an envelope signature.
// POST /v1/envelope/verify  body: {"envelope": {...}, "signing_key": "..."}
func (h *Handler) VerifyEnvelope(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Envelope envelope.Envelope `json:"envelope"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if h.signingKey == "" {
		writeErr(w, http.StatusServiceUnavailable, "no_signing_key", "server has no signing key configured")
		return
	}
	valid := envelope.Verify(req.Envelope, h.signingKey)
	writeJSON(w, http.StatusOK, map[string]any{"valid": valid})
}

// SetThresholds overrides suppression thresholds.
// PUT /v1/thresholds  body: {"high_impact": 0.20, "external_action": 0.50}
func (h *Handler) SetThresholds(w http.ResponseWriter, r *http.Request) {
	var raw map[string]*float64
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	t := gate.Thresholds{}
	for k, v := range raw {
		t[gate.ExecutionClass(k)] = v
	}
	h.gate.SetThresholds(t)
	writeJSON(w, http.StatusOK, map[string]any{"updated": true, "thresholds": h.gate.GetThresholds()})
}

// Health returns server status.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "tools": len(h.gate.Tools())})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]string{"error": code, "message": msg})
}
