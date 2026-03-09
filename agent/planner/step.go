package planner

// Step is a single unit of work produced by the planner.
// It is consumed by workflow/engine for execution.
type Step struct {
	Description string         `json:"description"`
	Tool        string         `json:"tool,omitempty"`
	Params      map[string]any `json:"params,omitempty"`
	DependsOn   []string       `json:"depends_on,omitempty"`
}
