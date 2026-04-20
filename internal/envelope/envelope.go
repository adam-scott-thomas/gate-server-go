package envelope

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/adam-scott-thomas/gate-server-go/internal/gate"
)

// Envelope is a signed, frozen permission set for a tool invocation.
type Envelope struct {
	EnvelopeID    string   `json:"envelope_id"`
	ContextID     string   `json:"context_id"`
	ToolName      string   `json:"tool_name"`
	AllowedTools  []string `json:"allowed_tools"`
	MaxToolCalls  int      `json:"max_tool_calls"`
	MaxRetries    int      `json:"max_retries"`
	BudgetSeconds int      `json:"budget_seconds"`
	ExecutionMode string   `json:"execution_mode"`
	DryRun        bool     `json:"dry_run"`
	Branching     string   `json:"branching"`
	HumanApproved bool     `json:"human_approved"`
	CreatedAt     float64  `json:"created_at"` // Unix timestamp, signed to prevent replay
	Signature     string   `json:"signature"`
}

// Build creates a signed envelope adjusted for the current mode.
func Build(toolName string, contextID string, mode float64, signingKey string) Envelope {
	zone := gate.ModeZone(mode)

	maxCalls := 20
	budget := 30
	execMode := "standard"
	branching := "auto"

	switch zone {
	case "elevated":
		maxCalls = 10
		budget = 15
		execMode = "cautious"
		branching = "deny"
	case "crisis":
		maxCalls = 5
		budget = 7
		execMode = "minimal"
		branching = "deny"
	}

	env := Envelope{
		EnvelopeID:    fmt.Sprintf("env_%s_%s", contextID, toolName),
		ContextID:     contextID,
		ToolName:      toolName,
		AllowedTools:  []string{toolName},
		MaxToolCalls:  maxCalls,
		MaxRetries:    1,
		BudgetSeconds: budget,
		ExecutionMode: execMode,
		DryRun:        false,
		Branching:     branching,
		HumanApproved: false,
		CreatedAt:     float64(time.Now().UnixNano()) / 1e9,
	}

	env.Signature = sign(env, signingKey)
	return env
}

// Verify checks the envelope signature against the given key.
func Verify(env Envelope, signingKey string) bool {
	expected := sign(env, signingKey)
	return hmac.Equal([]byte(expected), []byte(env.Signature))
}

func sign(env Envelope, key string) string {
	canonical := map[string]any{
		"envelope_id":    env.EnvelopeID,
		"context_id":     env.ContextID,
		"tool_name":      env.ToolName,
		"allowed_tools":  env.AllowedTools,
		"max_tool_calls": env.MaxToolCalls,
		"budget_seconds": env.BudgetSeconds,
		"execution_mode": env.ExecutionMode,
		"branching":      env.Branching,
		"human_approved": env.HumanApproved,
		"created_at":     env.CreatedAt,
	}
	data, _ := json.Marshal(canonical) // Go marshals map[string]any with keys sorted alphabetically
	hash := sha256.Sum256(data)
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(hash[:])
	return hex.EncodeToString(mac.Sum(nil))
}
