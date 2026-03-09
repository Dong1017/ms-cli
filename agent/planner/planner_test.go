package planner

import (
	"context"
	"testing"

	"github.com/vigo999/ms-cli/integrations/llm"
)

// mockProvider returns a fixed response.
type mockProvider struct {
	content string
	err     error
}

func (m *mockProvider) Name() string { return "mock" }
func (m *mockProvider) Complete(_ context.Context, _ *llm.CompletionRequest) (*llm.CompletionResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &llm.CompletionResponse{Content: m.content}, nil
}
func (m *mockProvider) CompleteStream(_ context.Context, _ *llm.CompletionRequest) (llm.StreamIterator, error) {
	return nil, nil
}
func (m *mockProvider) SupportsTools() bool        { return false }
func (m *mockProvider) AvailableModels() []llm.ModelInfo { return nil }

func TestPlan_JSON(t *testing.T) {
	provider := &mockProvider{
		content: `[{"description":"Read the file","tool":"read"},{"description":"Edit the code","tool":"edit"}]`,
	}
	p := New(provider, DefaultConfig())
	steps, err := p.Plan(context.Background(), "fix the bug", []string{"read", "edit", "shell"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}
	if steps[0].Tool != "read" {
		t.Errorf("expected tool 'read', got %q", steps[0].Tool)
	}
	if steps[1].Description != "Edit the code" {
		t.Errorf("expected 'Edit the code', got %q", steps[1].Description)
	}
}

func TestPlan_LinesFallback(t *testing.T) {
	provider := &mockProvider{
		content: "Here is the plan:\n1. Read the file\n2. Fix the bug\n3. Run tests\n",
	}
	p := New(provider, DefaultConfig())
	steps, err := p.Plan(context.Background(), "fix the bug", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(steps))
	}
}

func TestPlan_EmptyGoal(t *testing.T) {
	p := New(&mockProvider{}, DefaultConfig())
	_, err := p.Plan(context.Background(), "", nil)
	if err == nil {
		t.Fatal("expected error for empty goal")
	}
}

func TestPlan_MaxSteps(t *testing.T) {
	provider := &mockProvider{
		content: `[{"description":"s1"},{"description":"s2"},{"description":"s3"},{"description":"s4"}]`,
	}
	cfg := DefaultConfig()
	cfg.MaxSteps = 2
	p := New(provider, cfg)
	steps, err := p.Plan(context.Background(), "do stuff", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps (capped), got %d", len(steps))
	}
}

func TestValidateSteps_Empty(t *testing.T) {
	errs := ValidateSteps(nil, nil)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
}

func TestValidateSteps_UnknownTool(t *testing.T) {
	steps := []Step{{Description: "do it", Tool: "nope"}}
	errs := ValidateSteps(steps, []string{"read", "edit"})
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
}

func TestParseSteps_JSON(t *testing.T) {
	input := `Some preamble\n[{"description":"step one","tool":"read"}]\nMore text`
	steps, err := parseSteps(input, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
}

func TestParseSteps_Lines(t *testing.T) {
	input := "1. First step\n2. Second step\n- Third step\n"
	steps, err := parseSteps(input, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(steps))
	}
}

func TestRefine(t *testing.T) {
	provider := &mockProvider{
		content: `[{"description":"Improved step","tool":"shell"}]`,
	}
	p := New(provider, DefaultConfig())
	steps, err := p.Refine(context.Background(), "goal", []Step{{Description: "old step"}}, "make it better")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if steps[0].Description != "Improved step" {
		t.Errorf("expected 'Improved step', got %q", steps[0].Description)
	}
}
