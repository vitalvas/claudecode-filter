package autoallow

import (
	"encoding/json"

	"github.com/vitalvas/claudecode-filter/internal/hook"
)

// Plugin implements the auto-allow filter.
type Plugin struct{}

// New creates a new auto-allow filter plugin.
func New() hook.Filter {
	return &Plugin{}
}

func (p *Plugin) OnPreToolUse(_ hook.Input) *hook.Result {
	return nil
}

func (p *Plugin) OnPermissionRequest(input hook.Input) *hook.Result {
	switch input.ToolName {
	case "Bash":
		return p.handleBash(input)
	case "Read":
		return p.handleRead(input)
	default:
		return nil
	}
}

func (p *Plugin) OnUserPromptSubmit(_ hook.Input) *hook.Result {
	return nil
}

func (p *Plugin) OnSessionEnd(_ hook.Input) {}

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
