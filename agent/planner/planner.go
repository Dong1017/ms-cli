// Package planner owns task decomposition and planning strategy selection.
package planner

import (
	"context"
	"fmt"

	"github.com/vigo999/ms-cli/integrations/llm"
)

// Config controls planner behavior.
type Config struct {
	MaxSteps    int
	Temperature float32
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		MaxSteps:    10,
		Temperature: 0.3,
	}
}

// Planner decomposes a goal into executable steps using an LLM.
type Planner struct {
	provider llm.Provider
	config   Config
}

// New creates a Planner.
func New(provider llm.Provider, cfg Config) *Planner {
	if cfg.MaxSteps <= 0 {
		cfg.MaxSteps = 10
	}
	return &Planner{
		provider: provider,
		config:   cfg,
	}
}

// Plan generates steps for the given goal.
func (p *Planner) Plan(ctx context.Context, goal string, tools []string) ([]Step, error) {
	if goal == "" {
		return nil, fmt.Errorf("goal cannot be empty")
	}

	prompt := buildPlanPrompt(goal, tools)

	resp, err := p.provider.Complete(ctx, &llm.CompletionRequest{
		Messages:    []llm.Message{llm.NewUserMessage(prompt)},
		Temperature: p.config.Temperature,
		MaxTokens:   2000,
	})
	if err != nil {
		return nil, fmt.Errorf("llm completion: %w", err)
	}

	steps, err := parseSteps(resp.Content, p.config.MaxSteps)
	if err != nil {
		return nil, fmt.Errorf("parse plan: %w", err)
	}

	if errs := ValidateSteps(steps, tools); len(errs) > 0 {
		return nil, fmt.Errorf("validate plan: %s", errs[0].Error())
	}

	return steps, nil
}

// Refine takes existing steps and feedback, returns improved steps.
func (p *Planner) Refine(ctx context.Context, goal string, steps []Step, feedback string) ([]Step, error) {
	prompt := buildRefinePrompt(goal, steps, feedback)

	resp, err := p.provider.Complete(ctx, &llm.CompletionRequest{
		Messages:    []llm.Message{llm.NewUserMessage(prompt)},
		Temperature: p.config.Temperature,
		MaxTokens:   2000,
	})
	if err != nil {
		return nil, fmt.Errorf("llm completion: %w", err)
	}

	refined, err := parseSteps(resp.Content, p.config.MaxSteps)
	if err != nil {
		return nil, fmt.Errorf("parse refined plan: %w", err)
	}

	return refined, nil
}
