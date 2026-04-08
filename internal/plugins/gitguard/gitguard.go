package gitguard

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/vitalvas/claudecode-filter/internal/hook"
	"github.com/vitalvas/claudecode-filter/internal/marker"
)

const markerName = "gitguard-allow"

// New creates the gitguard middleware.
func New() hook.Middleware {
	return func(next hook.Handler) hook.Handler {
		return func(input hook.Input) *hook.Result {
			switch input.HookEventName {
			case hook.EventPreToolUse:
				if result := handlePreToolUse(input); result != nil {
					return result
				}
			case hook.EventUserPromptSubmit:
				handleUserPromptSubmit(input)
			case hook.EventSessionEnd:
				handleSessionEnd(input)
			}

			return next(input)
		}
	}
}

func handlePreToolUse(input hook.Input) *hook.Result {
	if input.ToolName != "Bash" {
		return nil
	}

	var bashInput hook.BashToolInput
	if err := json.Unmarshal(input.ToolInput, &bashInput); err != nil {
		return nil
	}

	ops := detectBlockedOps(bashInput.Command)
	if len(ops) == 0 {
		return nil
	}

	if header, ok := containsBlockedCommitHeader(bashInput.Command); ok {
		return denyPreToolUse(fmt.Sprintf("commit messages must not contain '%s' headers", header))
	}

	if _, ok := marker.Consume(input.CWD, markerName); ok {
		return nil
	}

	return denyPreToolUse(fmt.Sprintf(
		"git %s requires explicit user approval. Ask the user to say \"ok git %s\" first.",
		strings.Join(ops, ", "),
		ops[0],
	))
}

var okGitRe = regexp.MustCompile(`(?i)\bok\s+git\s+[\w-]+`)

func handleUserPromptSubmit(input hook.Input) {
	if !okGitRe.MatchString(input.Prompt) {
		return
	}

	marker.Create(input.CWD, markerName, "1")
}

func handleSessionEnd(input hook.Input) {
	marker.Cleanup(input.CWD)
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
