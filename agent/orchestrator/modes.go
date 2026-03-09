package orchestrator

import "github.com/vigo999/ms-cli/agent/planner"

// RunMode determines execution strategy.
type RunMode int

const (
	ModeStandard RunMode = iota
	ModePlan
)

// String returns the mode name.
func (m RunMode) String() string {
	switch m {
	case ModePlan:
		return "plan"
	default:
		return "standard"
	}
}

// ParseRunMode parses a mode string.
func ParseRunMode(s string) RunMode {
	switch s {
	case "plan", "planning":
		return ModePlan
	default:
		return ModeStandard
	}
}

// PlanCallback receives plan lifecycle events.
// Implemented by the UI or app layer.
type PlanCallback interface {
	// OnPlanCreated is called when a plan is generated.
	// Return error to abort.
	OnPlanCreated(steps []planner.Step) error

	// OnPlanApproved is called when execution begins.
	OnPlanApproved(steps []planner.Step) error

	// OnStepStarted is called before each step executes.
	OnStepStarted(step planner.Step, index int) error

	// OnStepCompleted is called after each step finishes.
	OnStepCompleted(step planner.Step, index int, result string) error
}

// NoOpCallback is a default callback that does nothing.
type NoOpCallback struct{}

func (NoOpCallback) OnPlanCreated([]planner.Step) error                      { return nil }
func (NoOpCallback) OnPlanApproved([]planner.Step) error                     { return nil }
func (NoOpCallback) OnStepStarted(_ planner.Step, _ int) error               { return nil }
func (NoOpCallback) OnStepCompleted(_ planner.Step, _ int, _ string) error    { return nil }
