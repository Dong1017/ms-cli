package planner

import (
	"encoding/json"
	"regexp"
	"strings"
)

// parseSteps extracts Steps from LLM output.
// Tries JSON first, falls back to line-based parsing.
func parseSteps(content string, maxSteps int) ([]Step, error) {
	if steps, err := parseJSON(content, maxSteps); err == nil && len(steps) > 0 {
		return steps, nil
	}
	return parseLines(content, maxSteps), nil
}

func parseJSON(content string, maxSteps int) ([]Step, error) {
	raw := extractJSONArray(content)
	if raw == "" {
		return nil, errNoJSON
	}

	var parsed []Step
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, err
	}

	if len(parsed) > maxSteps {
		parsed = parsed[:maxSteps]
	}
	return parsed, nil
}

func parseLines(content string, maxSteps int) []Step {
	var steps []Step
	for _, line := range strings.Split(content, "\n") {
		if len(steps) >= maxSteps {
			break
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if desc := matchStepLine(line); desc != "" {
			steps = append(steps, Step{Description: desc})
		}
	}
	return steps
}

var stepPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^\d+[.)]\s*(.+)$`),
	regexp.MustCompile(`^-\s*(.+)$`),
	regexp.MustCompile(`^\*\s*(.+)$`),
	regexp.MustCompile(`^Step\s+\d+[:.]\s*(.+)$`),
}

func matchStepLine(line string) string {
	for _, re := range stepPatterns {
		if m := re.FindStringSubmatch(line); m != nil {
			return strings.TrimSpace(m[1])
		}
	}
	return ""
}

func extractJSONArray(text string) string {
	start := strings.Index(text, "[")
	if start == -1 {
		return ""
	}

	depth := 0
	for i := start; i < len(text); i++ {
		switch text[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return text[start : i+1]
			}
		}
	}
	return ""
}

type parseError string

func (e parseError) Error() string { return string(e) }

const errNoJSON = parseError("no JSON array found in content")
