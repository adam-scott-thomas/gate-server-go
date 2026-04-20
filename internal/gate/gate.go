package gate

import "sync"

// ExecutionClass defines tool side-effect severity.
type ExecutionClass string

const (
	ReadOnly       ExecutionClass = "read_only"
	Advisory       ExecutionClass = "advisory"
	ExternalAction ExecutionClass = "external_action"
	StateMutation  ExecutionClass = "state_mutation"
	HighImpact     ExecutionClass = "high_impact"
)

// Tool represents a registered tool in the gate.
type Tool struct {
	Name           string            `json:"name"`
	ExecutionClass ExecutionClass    `json:"execution_class"`
	Description    string            `json:"description,omitempty"`
	Inputs         map[string]string `json:"inputs,omitempty"`
}

// Thresholds maps execution classes to suppression thresholds.
// A nil value means the tool is never suppressed.
type Thresholds map[ExecutionClass]*float64

// DefaultThresholds returns the spec-defined defaults.
func DefaultThresholds() Thresholds {
	ea := 0.65
	sm := 0.65
	hi := 0.35
	return Thresholds{
		ReadOnly:       nil,
		Advisory:       nil,
		ExternalAction: &ea,
		StateMutation:  &sm,
		HighImpact:     &hi,
	}
}

// ModeZone returns the zone name for a mode value.
func ModeZone(mode float64) string {
	switch {
	case mode <= 0.35:
		return "normal"
	case mode <= 0.65:
		return "elevated"
	default:
		return "crisis"
	}
}

// FilterResult is the output of a filter operation.
type FilterResult struct {
	Visible    []Tool     `json:"visible"`
	Suppressed []Tool     `json:"suppressed"`
	Mode       float64    `json:"mode"`
	ModeZone   string     `json:"mode_zone"`
	Thresholds Thresholds `json:"thresholds"`
}

// Gate holds the tool registry and applies suppression logic.
type Gate struct {
	mu         sync.RWMutex
	tools      []Tool
	thresholds Thresholds
}

// New creates a Gate with the given thresholds.
func New(t Thresholds) *Gate {
	return &Gate{thresholds: t}
}

// AddTools registers tools. Duplicate names are overwritten.
func (g *Gate) AddTools(tools []Tool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, t := range tools {
		if !validClass(t.ExecutionClass) {
			t.ExecutionClass = HighImpact
		}
		replaced := false
		for i, existing := range g.tools {
			if existing.Name == t.Name {
				g.tools[i] = t
				replaced = true
				break
			}
		}
		if !replaced {
			g.tools = append(g.tools, t)
		}
	}
}

// Tools returns a copy of all registered tools.
func (g *Gate) Tools() []Tool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make([]Tool, len(g.tools))
	copy(out, g.tools)
	return out
}

// Filter applies mode-based suppression and returns the result.
func (g *Gate) Filter(mode float64) FilterResult {
	mode = clamp(mode)
	g.mu.RLock()
	defer g.mu.RUnlock()

	var visible, suppressed []Tool
	for _, t := range g.tools {
		if g.isSuppressed(t.ExecutionClass, mode) {
			suppressed = append(suppressed, t)
		} else {
			visible = append(visible, t)
		}
	}
	return FilterResult{
		Visible:    visible,
		Suppressed: suppressed,
		Mode:       mode,
		ModeZone:   ModeZone(mode),
		Thresholds: g.thresholds,
	}
}

// IsSuppressed checks if a specific tool name is suppressed at the given mode.
func (g *Gate) IsSuppressed(name string, mode float64) (bool, error) {
	mode = clamp(mode)
	g.mu.RLock()
	defer g.mu.RUnlock()
	for _, t := range g.tools {
		if t.Name == name {
			return g.isSuppressed(t.ExecutionClass, mode), nil
		}
	}
	return false, ErrToolNotFound
}

func (g *Gate) isSuppressed(ec ExecutionClass, mode float64) bool {
	thresh, ok := g.thresholds[ec]
	if !ok || thresh == nil {
		return false
	}
	return mode > *thresh
}

// SetThresholds merges new thresholds into the gate.
func (g *Gate) SetThresholds(t Thresholds) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for k, v := range t {
		g.thresholds[k] = v
	}
}

// GetThresholds returns a copy of the current thresholds.
func (g *Gate) GetThresholds() Thresholds {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := Thresholds{}
	for k, v := range g.thresholds {
		out[k] = v
	}
	return out
}

func clamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func validClass(ec ExecutionClass) bool {
	switch ec {
	case ReadOnly, Advisory, ExternalAction, StateMutation, HighImpact:
		return true
	}
	return false
}
