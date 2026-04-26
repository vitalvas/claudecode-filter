package sandbox

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vitalvas/claudecode-filter/internal/hook"
	"github.com/vitalvas/claudecode-filter/internal/marker"
)

const markerName = "allow-rootread"

var allowedRoots []string

var allowedExceptions = []string{}

func init() {
	home := os.Getenv("HOME")
	if home != "" {
		allowedRoots = append(allowedRoots, filepath.Join(home, ".claude"))
		allowedRoots = append(allowedRoots, filepath.Join(home, "workspace"))
	}

	uid := fmt.Sprintf("%d", os.Getuid())
	allowedRoots = append(allowedRoots, filepath.Join("/private/tmp", fmt.Sprintf("claude-%s", uid)))
}

// New creates the sandbox middleware.
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
			}

			return next(input)
		}
	}
}

func handlePreToolUse(input hook.Input) *hook.Result {
	path := extractPath(input)
	if path == "" {
		return nil
	}

	if isTmpPath(path) {
		return denyPreToolUse("The /tmp dir is prohibited. Use ${PROJECTROOT}/.tmp/ for it.")
	}

	if isAllowedPath(path) {
		return nil
	}

	if input.CWD != "" && isUnderDir(path, input.CWD) {
		return nil
	}

	if input.CWD != "" && marker.Exists(input.CWD, markerName) {
		return askPreToolUse(fmt.Sprintf("accessing %s outside sandbox requires approval", path))
	}

	return denyPreToolUse(
		"accessing files outside sandbox is blocked. Say \"ok allow rootread\" to enable with approval.",
	)
}

func extractPath(input hook.Input) string {
	switch input.ToolName {
	case "Read", "Write", "Edit":
		var ti struct {
			FilePath string `json:"file_path"`
		}

		if err := json.Unmarshal(input.ToolInput, &ti); err != nil {
			return ""
		}

		return ti.FilePath
	case "Bash":
		var ti hook.BashToolInput
		if err := json.Unmarshal(input.ToolInput, &ti); err != nil {
			return ""
		}

		return extractBashPath(ti.Command)
	}

	return ""
}

var heredocPattern = regexp.MustCompile(`<<-?\s*'?(\w+)'?[\s\S]*`)

func extractBashPath(command string) string {
	cmd := heredocPattern.ReplaceAllString(command, "")
	args := strings.Fields(cmd)

	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}

		if filepath.IsAbs(arg) {
			return arg
		}
	}

	return ""
}

func isTmpPath(path string) bool {
	return strings.HasPrefix(path, "/tmp/") || path == "/tmp" ||
		strings.HasPrefix(path, "/private/tmp/") || path == "/private/tmp"
}

func isAllowedPath(path string) bool {
	for _, root := range allowedRoots {
		if isUnderDir(path, root) || path == root {
			return true
		}
	}

	for _, exception := range allowedExceptions {
		if isUnderDir(path, exception) || path == exception {
			return true
		}
	}

	return false
}

func isUnderDir(filePath, dir string) bool {
	rel, err := filepath.Rel(dir, filePath)
	if err != nil {
		return false
	}

	return len(rel) > 0 && !strings.HasPrefix(rel, "..")
}

var okAllowRootreadRe = regexp.MustCompile(`(?i)\bok\s+allow\s+rootread\b`)

func handleUserPromptSubmit(input hook.Input) {
	if !okAllowRootreadRe.MatchString(input.Prompt) {
		return
	}

	marker.Create(input.CWD, markerName, "1")
}

func askPreToolUse(reason string) *hook.Result {
	output := hook.PreToolUseOutputWrapper{
		HookSpecificOutput: hook.PreToolUseOutput{
			HookEventName:            hook.EventPreToolUse,
			PermissionDecision:       hook.PermissionAsk,
			PermissionDecisionReason: reason,
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
