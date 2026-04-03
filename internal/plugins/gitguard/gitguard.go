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

// Plugin implements the git guard filter.
type Plugin struct{}

// New creates a new git guard filter plugin.
func New() hook.Filter {
	return &Plugin{}
}

func (p *Plugin) OnPreToolUse(input hook.Input) *hook.Result {
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

func (p *Plugin) OnUserPromptSubmit(input hook.Input) *hook.Result {
	if !okGitRe.MatchString(input.Prompt) {
		return nil
	}

	marker.Create(input.CWD, markerName, "1")

	return nil
}

func (p *Plugin) OnSessionEnd(input hook.Input) {
	marker.Cleanup(input.CWD)
}

func denyPreToolUse(reason string) *hook.Result {
	output := hook.Output{
		HookSpecificOutput: hook.PreToolUseOutput{
			HookEventName:            "PreToolUse",
			PermissionDecision:       "deny",
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
