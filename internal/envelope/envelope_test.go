package envelope_test

import (
	"testing"

	"github.com/adam-scott-thomas/gate-server-go/internal/envelope"
)

func TestBuildAndVerify(t *testing.T) {
	env := envelope.Build("read_file", "session_1", 0.1, "secret-key")

	if env.ToolName != "read_file" {
		t.Errorf("want tool_name read_file, got %s", env.ToolName)
	}
	if env.ExecutionMode != "standard" {
		t.Errorf("normal mode should be standard, got %s", env.ExecutionMode)
	}
	if env.MaxToolCalls != 20 {
		t.Errorf("normal max_tool_calls should be 20, got %d", env.MaxToolCalls)
	}
	if env.BudgetSeconds != 30 {
		t.Errorf("normal budget should be 30, got %d", env.BudgetSeconds)
	}
	if env.Branching != "auto" {
		t.Errorf("normal branching should be auto, got %s", env.Branching)
	}
	if !envelope.Verify(env, "secret-key") {
		t.Error("envelope should verify with correct key")
	}
	if envelope.Verify(env, "wrong-key") {
		t.Error("envelope should not verify with wrong key")
	}
}

func TestBuildElevated(t *testing.T) {
	env := envelope.Build("send_email", "s2", 0.5, "key")

	if env.ExecutionMode != "cautious" {
		t.Errorf("elevated mode should be cautious, got %s", env.ExecutionMode)
	}
	if env.MaxToolCalls != 10 {
		t.Errorf("elevated max_tool_calls should be 10, got %d", env.MaxToolCalls)
	}
	if env.BudgetSeconds != 15 {
		t.Errorf("elevated budget should be 15, got %d", env.BudgetSeconds)
	}
	if env.Branching != "deny" {
		t.Errorf("elevated branching should be deny, got %s", env.Branching)
	}
}

func TestBuildCrisis(t *testing.T) {
	env := envelope.Build("read_file", "s3", 0.9, "key")

	if env.ExecutionMode != "minimal" {
		t.Errorf("crisis mode should be minimal, got %s", env.ExecutionMode)
	}
	if env.MaxToolCalls != 5 {
		t.Errorf("crisis max_tool_calls should be 5, got %d", env.MaxToolCalls)
	}
	if env.BudgetSeconds != 7 {
		t.Errorf("crisis budget should be 7, got %d", env.BudgetSeconds)
	}
}

func TestTamperedEnvelopeFails(t *testing.T) {
	env := envelope.Build("deploy", "s1", 0.1, "key")
	env.MaxToolCalls = 9999 // tamper
	if envelope.Verify(env, "key") {
		t.Error("tampered envelope should fail verification")
	}
}
