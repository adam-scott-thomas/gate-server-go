package gate_test

import (
	"testing"

	"github.com/adam-scott-thomas/gate-server-go/internal/gate"
)

func tools() []gate.Tool {
	return []gate.Tool{
		{Name: "read_file", ExecutionClass: gate.ReadOnly, Description: "Read a file"},
		{Name: "analyze", ExecutionClass: gate.Advisory, Description: "Analyze data"},
		{Name: "send_email", ExecutionClass: gate.ExternalAction, Description: "Send email"},
		{Name: "write_db", ExecutionClass: gate.StateMutation, Description: "Write to DB"},
		{Name: "deploy", ExecutionClass: gate.HighImpact, Description: "Deploy to production"},
	}
}

func TestFilterNormalMode(t *testing.T) {
	g := gate.New(gate.DefaultThresholds())
	g.AddTools(tools())
	result := g.Filter(0.1)

	if len(result.Visible) != 5 {
		t.Errorf("normal mode: want 5 visible, got %d", len(result.Visible))
	}
	if len(result.Suppressed) != 0 {
		t.Errorf("normal mode: want 0 suppressed, got %d", len(result.Suppressed))
	}
	if result.ModeZone != "normal" {
		t.Errorf("want zone normal, got %s", result.ModeZone)
	}
}

func TestFilterElevatedMode(t *testing.T) {
	g := gate.New(gate.DefaultThresholds())
	g.AddTools(tools())
	result := g.Filter(0.5)

	visibleNames := map[string]bool{}
	for _, t := range result.Visible {
		visibleNames[t.Name] = true
	}
	suppressedNames := map[string]bool{}
	for _, t := range result.Suppressed {
		suppressedNames[t.Name] = true
	}

	// high_impact suppressed at >0.35
	if !suppressedNames["deploy"] {
		t.Error("deploy should be suppressed at 0.5")
	}
	// read_only and advisory never suppressed
	if !visibleNames["read_file"] {
		t.Error("read_file should be visible at 0.5")
	}
	if !visibleNames["analyze"] {
		t.Error("analyze should be visible at 0.5")
	}
	// external_action and state_mutation suppressed at >0.65, so visible at 0.5
	if !visibleNames["send_email"] {
		t.Error("send_email should be visible at 0.5")
	}
	if !visibleNames["write_db"] {
		t.Error("write_db should be visible at 0.5")
	}
	if result.ModeZone != "elevated" {
		t.Errorf("want zone elevated, got %s", result.ModeZone)
	}
}

func TestFilterCrisisMode(t *testing.T) {
	g := gate.New(gate.DefaultThresholds())
	g.AddTools(tools())
	result := g.Filter(0.9)

	if len(result.Visible) != 2 {
		t.Errorf("crisis mode: want 2 visible (read_only + advisory), got %d", len(result.Visible))
	}
	if len(result.Suppressed) != 3 {
		t.Errorf("crisis mode: want 3 suppressed, got %d", len(result.Suppressed))
	}
	if result.ModeZone != "crisis" {
		t.Errorf("want zone crisis, got %s", result.ModeZone)
	}
}

func TestModeClamp(t *testing.T) {
	g := gate.New(gate.DefaultThresholds())
	g.AddTools(tools())

	// Negative clamped to 0
	r1 := g.Filter(-5.0)
	if r1.Mode != 0.0 {
		t.Errorf("negative mode should clamp to 0, got %f", r1.Mode)
	}

	// Over 1 clamped to 1
	r2 := g.Filter(99.0)
	if r2.Mode != 1.0 {
		t.Errorf("mode >1 should clamp to 1, got %f", r2.Mode)
	}
}

func TestUnrecognizedClassTreatedAsHighImpact(t *testing.T) {
	g := gate.New(gate.DefaultThresholds())
	g.AddTools([]gate.Tool{
		{Name: "mystery", ExecutionClass: "made_up_class"},
	})

	// At elevated mode, high_impact (threshold 0.35) should be suppressed
	result := g.Filter(0.5)
	if len(result.Suppressed) != 1 || result.Suppressed[0].Name != "mystery" {
		t.Error("unrecognized execution class should be treated as high_impact and suppressed at 0.5")
	}
}

func TestValidateProposal(t *testing.T) {
	g := gate.New(gate.DefaultThresholds())
	g.AddTools(tools())

	tests := []struct {
		name    string
		tool    string
		mode    float64
		wantErr error
	}{
		{"accepted in normal", "deploy", 0.1, nil},
		{"suppressed in elevated", "deploy", 0.5, gate.ErrSuppressed},
		{"not found", "nonexistent", 0.1, gate.ErrToolNotFound},
		{"read_only always ok", "read_file", 0.99, nil},
		{"external_action in crisis", "send_email", 0.8, gate.ErrSuppressed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := g.ValidateProposal(tt.tool, tt.mode)
			if err != tt.wantErr {
				t.Errorf("ValidateProposal(%q, %f) = %v, want %v", tt.tool, tt.mode, err, tt.wantErr)
			}
		})
	}
}

func TestAddToolsDedup(t *testing.T) {
	g := gate.New(gate.DefaultThresholds())
	g.AddTools([]gate.Tool{
		{Name: "x", ExecutionClass: gate.ReadOnly},
	})
	g.AddTools([]gate.Tool{
		{Name: "x", ExecutionClass: gate.HighImpact},
	})

	all := g.Tools()
	if len(all) != 1 {
		t.Fatalf("expected 1 tool after dedup, got %d", len(all))
	}
	if all[0].ExecutionClass != gate.HighImpact {
		t.Errorf("expected overwritten class high_impact, got %s", all[0].ExecutionClass)
	}
}

func TestCustomThresholds(t *testing.T) {
	hi := 0.20
	g := gate.New(gate.Thresholds{
		gate.ReadOnly:       nil,
		gate.Advisory:       nil,
		gate.ExternalAction: nil,
		gate.StateMutation:  nil,
		gate.HighImpact:     &hi,
	})
	g.AddTools(tools())

	result := g.Filter(0.25)
	// Only high_impact suppressed, everything else visible (thresholds are nil)
	if len(result.Suppressed) != 1 {
		t.Errorf("custom threshold: want 1 suppressed, got %d", len(result.Suppressed))
	}
	if len(result.Visible) != 4 {
		t.Errorf("custom threshold: want 4 visible, got %d", len(result.Visible))
	}
}

func TestBoundaryThresholds(t *testing.T) {
	g := gate.New(gate.DefaultThresholds())
	g.AddTools([]gate.Tool{
		{Name: "deploy", ExecutionClass: gate.HighImpact},
	})

	// Exactly at threshold (0.35) — should NOT be suppressed (rule is mode > threshold)
	result := g.Filter(0.35)
	if len(result.Suppressed) != 0 {
		t.Error("mode exactly at threshold should not suppress (> not >=)")
	}

	// Just above threshold
	result2 := g.Filter(0.351)
	if len(result2.Suppressed) != 1 {
		t.Error("mode above threshold should suppress")
	}
}
