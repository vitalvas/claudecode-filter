package readguard

import (
	"encoding/json"
	"path/filepath"

	"github.com/vitalvas/claudecode-filter/internal/hook"
)

var blockedPatterns = []string{
	"*.key",
}

// New creates the readguard middleware.
func New() hook.Middleware {
	return func(next hook.Handler) hook.Handler {
		return func(input hook.Input) *hook.Result {
			if input.ToolName == "Read" {
				if input.HookEventName == hook.EventPreToolUse || input.HookEventName == hook.EventPermissionRequest {
					if result := handleRead(input); result != nil {
						return result
					}
				}
			}

			return next(input)
		}
	}
}

func handleRead(input hook.Input) *hook.Result {
	var readInput hook.ReadToolInput
	if err := json.Unmarshal(input.ToolInput, &readInput); err != nil {
		return nil
	}

	base := filepath.Base(readInput.FilePath)

	for _, pattern := range blockedPatterns {
		matched, err := filepath.Match(pattern, base)
		if err != nil {
			continue
		}

		if matched {
			return denyPreToolUse("reading *.key files is not allowed")
		}
	}

	return nil
}

func denyPreToolUse(reason string) *hook.Result {
	output := hook.PreToolUseOutputWrapper{
		HookSpecificOutput: hook.PreToolUseOutput{
			HookEventName:            hook.EventPreToolUse,
			PermissionDecision:       hook.PermissionDeny,
			PermissionDecisionReason: reason,
		},
	}

	data, err := json.Marshal(output)
	if err != nil {
		return &hook.Result{
			Stderr:   reason,
			ExitCode: 2,
		}
	}

	return &hook.Result{
		Stdout: string(data),
	}
}
