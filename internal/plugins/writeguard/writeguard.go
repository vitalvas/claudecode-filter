package writeguard

import (
	"encoding/json"
	"fmt"
	"os"
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

	if result := blockProtectedDirWrite(input); result != nil {
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
		return denyPreToolUse("readonly sandbox mode")
	}

	if input.ToolName == "Bash" {
		return denyPreToolUse("readonly sandbox mode")
	}

	return nil
}

func blockProtectedDirWrite(input hook.Input) *hook.Result {
	if !writeTools[input.ToolName] {
		return nil
	}

	var ti struct {
		FilePath     string `json:"file_path"`
		NotebookPath string `json:"notebook_path"`
	}

	if err := json.Unmarshal(input.ToolInput, &ti); err != nil {
		return nil
	}

	filePath := ti.FilePath
	if filePath == "" {
		filePath = ti.NotebookPath
	}
	filePath = resolvePath(filePath, input.CWD)

	home := os.Getenv("HOME")
	if home == "" {
		return nil
	}

	protectedDirs := []string{
		filepath.Join(home, ".claude", "plans"),
		filepath.Join(home, ".claude", "projects"),
	}

	for _, protectedDir := range protectedDirs {
		if isUnderDir(filePath, protectedDir) {
			reason := fmt.Sprintf("modifying files under %s is not allowed", protectedDir)
			return denyPreToolUse(withAgentsHint(reason, input.CWD))
		}
	}

	return nil
}

// withAgentsHint appends the contents of the project's AGENTS.md to the deny
// reason, but only when that file exists in the project root (cwd). It is used
// to steer a trapped agent back to the project's own guidance.
func withAgentsHint(reason, cwd string) string {
	if cwd == "" {
		return reason
	}

	data, err := os.ReadFile(filepath.Join(cwd, "AGENTS.md"))
	if err != nil {
		return reason
	}

	return fmt.Sprintf("%s\n\n%s", reason, data)
}

func resolvePath(path, cwd string) string {
	if strings.HasPrefix(path, "~/") {
		if home := os.Getenv("HOME"); home != "" {
			path = filepath.Join(home, path[2:])
		}
	}

	if !filepath.IsAbs(path) && cwd != "" {
		path = filepath.Join(cwd, path)
	}

	return filepath.Clean(path)
}

func isUnderDir(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}

	parentPrefix := fmt.Sprintf("..%c", filepath.Separator)

	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, parentPrefix))
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
