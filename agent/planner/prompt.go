package planner

import (
	"fmt"
	"strings"
)

func buildPlanPrompt(goal string, tools []string) string {
	var toolDesc string
	if len(tools) > 0 {
		toolDesc = strings.Join(tools, ", ")
	} else {
		toolDesc = "read, write, edit, grep, glob, shell"
	}

	return fmt.Sprintf(`You are a planning assistant. Create a step-by-step plan for the following task.

Task: %s

Available tools: %s

Create a plan with the following format:
1. Each step should be a clear, actionable instruction
2. Specify which tool to use for each step (if applicable)
3. Steps should be in logical order
4. Keep the plan concise (3-10 steps)

Format your response as a JSON array of steps:
[
  {"description": "Step 1 description", "tool": "tool_name"},
  {"description": "Step 2 description", "tool": "tool_name"}
]`, goal, toolDesc)
}

func buildRefinePrompt(goal string, steps []Step, feedback string) string {
	var sb strings.Builder
	for i, s := range steps {
		fmt.Fprintf(&sb, "%d. %s", i+1, s.Description)
		if s.Tool != "" {
			fmt.Fprintf(&sb, " (tool: %s)", s.Tool)
		}
		sb.WriteByte('\n')
	}

	return fmt.Sprintf(`Given the following plan and feedback, refine the plan.

Original Goal: %s

Current Plan:
%s
Feedback: %s

Please provide an improved plan in the same JSON format.`, goal, sb.String(), feedback)
}
