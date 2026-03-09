package planner

import "fmt"

// ValidationError describes a single validation problem.
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidateSteps checks steps for common issues.
// allowedTools is optional — pass nil to skip tool checks.
func ValidateSteps(steps []Step, allowedTools []string) []ValidationError {
	var errs []ValidationError

	if len(steps) == 0 {
		errs = append(errs, ValidationError{
			Field:   "steps",
			Message: "plan has no steps",
		})
		return errs
	}

	allowed := make(map[string]struct{}, len(allowedTools))
	for _, t := range allowedTools {
		allowed[t] = struct{}{}
	}

	for i, s := range steps {
		if s.Description == "" {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("steps[%d].description", i),
				Message: "step description is empty",
			})
		}
		if s.Tool != "" && len(allowedTools) > 0 {
			if _, ok := allowed[s.Tool]; !ok {
				errs = append(errs, ValidationError{
					Field:   fmt.Sprintf("steps[%d].tool", i),
					Message: fmt.Sprintf("tool %q is not allowed", s.Tool),
				})
			}
		}
	}

	return errs
}
