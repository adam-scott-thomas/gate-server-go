package gate

import "errors"

var (
	ErrToolNotFound = errors.New("tool_not_found")
	ErrSuppressed   = errors.New("execution_class_suppressed")
)

// ValidateProposal checks if a tool request should be accepted.
// Returns nil if accepted, ErrToolNotFound or ErrSuppressed otherwise.
func (g *Gate) ValidateProposal(name string, mode float64) error {
	mode = clamp(mode)
	g.mu.RLock()
	defer g.mu.RUnlock()
	for _, t := range g.tools {
		if t.Name == name {
			if g.isSuppressed(t.ExecutionClass, mode) {
				return ErrSuppressed
			}
			return nil
		}
	}
	return ErrToolNotFound
}
