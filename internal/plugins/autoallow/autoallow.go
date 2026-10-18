package autoallow

import (
	"encoding/json"
	"strings"

	"github.com/vitalvas/claudecode-filter/internal/hook"
)

const sharedMemoryToolPrefix = "mcp__shared-memory__"

// New creates the autoallow middleware.
func New() hook.Middleware {
	return func(next hook.Handler) hook.Handler {
		return func(input hook.Input) *hook.Result {
			// Pure-allow tools: Claude Code gates these on PreToolUse, so the
			// allow decision must be emitted there (and on PermissionRequest).
			if input.HookEventName == hook.EventPreToolUse || input.HookEventName == hook.EventPermissionRequest {
				if strings.HasPrefix(input.ToolName, sharedMemoryToolPrefix) {
					return allowResult(input.HookEventName)
				}

				switch input.ToolName {
				case "Read", "Grep", "Glob":
					if result := handleRead(input); result != nil {
						return result
					}
				case "WebSearch":
					return allowResult(input.HookEventName)
				}
			}

			// Bash deny runs on PreToolUse; Bash/WebFetch allow is
			// PermissionRequest-only (their deny paths are request-shaped).
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
			case "WebFetch":
				if result := handleWebFetch(input); result != nil {
					return result
				}
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

func denyPermissionRequest(reason string) *hook.Result {
	output := hook.PermissionRequestOutputWrapper{
		HookSpecificOutput: hook.PermissionRequestOutput{
			HookEventName: hook.EventPermissionRequest,
			Decision: hook.PermissionDecision{
				Behavior: hook.PermissionDeny,
			},
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

// allowResult emits an allow decision in the output shape for the given event.
// Claude Code gates Read/Bash/etc. on PreToolUse, so a PermissionRequest-shaped
// allow is ignored there; the shape must match the event.
func allowResult(event string) *hook.Result {
	if event == hook.EventPreToolUse {
		output := hook.PreToolUseOutputWrapper{
			HookSpecificOutput: hook.PreToolUseOutput{
				HookEventName:      hook.EventPreToolUse,
				PermissionDecision: hook.PermissionAllow,
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
