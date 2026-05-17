package writeguard

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vitalvas/claudecode-filter/internal/hook"
	"github.com/vitalvas/claudecode-filter/internal/marker"
)

const markerName = "readonly"

var protectedFiles = []string{
	".yake.yaml",
}

var writeTools = map[string]bool{
	"Write":        true,
	"Edit":         true,
	"NotebookEdit": true,
}

// New creates the writeguard middleware.
func New() hook.Middleware {
	return func(next hook.Handler) hook.Handler {
		return func(input hook.Input) *hook.Result {
			if input.HookEventName == hook.EventPreToolUse || input.HookEventName == hook.EventPermissionRequest {
				if result := handleWriteGuard(input); result != nil {
					return result
				}
			}

			return next(input)
		}
	}
}

func handleWriteGuard(input hook.Input) *hook.Result {
	if result := blockProtectedFileWrite(input); result != nil {
		return result
	}

	if result := blockMarkerFileWrite(input); result != nil {
		return result
	}

	if input.CWD == "" {
		return nil
	}

	if !marker.Exists(input.CWD, markerName) {
		return nil
	}

	if writeTools[input.ToolName] {
		return denyPreToolUse(fmt.Sprintf("writes are blocked: readonly marker is active (remove %s to unblock)", markerPath()))
	}

	if input.ToolName == "Bash" {
		return denyPreToolUse(fmt.Sprintf("bash is blocked: readonly marker is active (remove %s to unblock)", markerPath()))
	}

	return nil
}

func blockProtectedFileWrite(input hook.Input) *hook.Result {
	switch input.ToolName {
	case "Write", "Edit":
		var ti struct {
			FilePath string `json:"file_path"`
		}

		if err := json.Unmarshal(input.ToolInput, &ti); err != nil {
			return nil
		}

		base := filepath.Base(ti.FilePath)
		for _, protected := range protectedFiles {
			if base == protected {
				return denyPreToolUse(fmt.Sprintf("modifying %s is not allowed", protected))
			}
		}
	}

	return nil
}

func blockMarkerFileWrite(input hook.Input) *hook.Result {
	switch input.ToolName {
	case "Write", "Edit":
		var ti struct {
			FilePath string `json:"file_path"`
		}

		if err := json.Unmarshal(input.ToolInput, &ti); err != nil {
			return nil
		}

		if strings.HasPrefix(filepath.Base(ti.FilePath), marker.Prefix) {
			return denyPreToolUse("modifying marker files is not allowed")
		}
	case "Bash":
		var bashInput hook.BashToolInput
		if err := json.Unmarshal(input.ToolInput, &bashInput); err != nil {
			return nil
		}

		if bashTargetsMarkerFile(bashInput.Command) {
			return denyPreToolUse("modifying marker files is not allowed")
		}
	}

	return nil
}

func bashTargetsMarkerFile(command string) bool {
	args := strings.Fields(command)

	for _, arg := range args {
		if strings.Contains(arg, marker.Prefix) {
			return true
		}
	}

	return false
}

func markerPath() string {
	return fmt.Sprintf(".tmp/claudecode-filter-%s", markerName)
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
