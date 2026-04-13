package autoallow

import (
	"encoding/json"

	"github.com/vitalvas/claudecode-filter/internal/hook"
)

// New creates the autoallow middleware.
func New() hook.Middleware {
	return func(next hook.Handler) hook.Handler {
		return func(input hook.Input) *hook.Result {
			if input.HookEventName == hook.EventPreToolUse {
				if input.ToolName == "Bash" {
					if result := handleBashDeny(input); result != nil {
						return result
					}
				}

				return next(input)
			}

			if input.HookEventName != hook.EventPermissionRequest {
				return next(input)
			}

			switch input.ToolName {
			case "Bash":
				if result := handleBashDeny(input); result != nil {
					return result
				}

				if result := handleBash(input); result != nil {
					return result
				}
			case "Read":
				if result := handleRead(input); result != nil {
					return result
				}
			case "WebFetch":
				if result := handleWebFetch(input); result != nil {
					return result
				}
			case "WebSearch":
				return allowPermissionRequest()
			}

			return next(input)
		}
	}
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

func allowPermissionRequest() *hook.Result {
	output := hook.PermissionRequestOutputWrapper{
		HookSpecificOutput: hook.PermissionRequestOutput{
			HookEventName: hook.EventPermissionRequest,
			Decision: hook.PermissionDecision{
				Behavior: hook.PermissionAllow,
			},
		},
	}

	data, err := json.Marshal(output)
	if err != nil {
		return nil
	}

	return &hook.Result{
		Stdout: string(data),
	}
}
