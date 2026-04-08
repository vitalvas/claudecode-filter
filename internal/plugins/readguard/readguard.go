package readguard

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vitalvas/claudecode-filter/internal/hook"
	"github.com/vitalvas/gokit/xstrings"
)

type rule struct {
	pattern   string
	matchFull bool
	reason    string
}

var blockedPatterns = []rule{
	{pattern: "*.key", reason: "reading *.key files is not allowed"},
	{pattern: ".env", reason: "reading .env files is not allowed"},
	{pattern: ".env.*", reason: "reading .env files is not allowed"},
	{pattern: "id_rsa*", reason: "reading private key files is not allowed"},
	{pattern: "id_ecdsa*", reason: "reading private key files is not allowed"},
	{pattern: "id_ed25519*", reason: "reading private key files is not allowed"},
}

var allowedPatterns = []string{
	"*.pub",
}

// New creates the readguard middleware.
func New() hook.Middleware {
	sshDir := filepath.Join(os.Getenv("HOME"), ".ssh")

	return func(next hook.Handler) hook.Handler {
		return func(input hook.Input) *hook.Result {
			if input.ToolName == "Read" {
				if input.HookEventName == hook.EventPreToolUse || input.HookEventName == hook.EventPermissionRequest {
					if result := handleRead(input, sshDir); result != nil {
						return result
					}
				}
			}

			return next(input)
		}
	}
}

func handleRead(input hook.Input, sshDir string) *hook.Result {
	var readInput hook.ReadToolInput
	if err := json.Unmarshal(input.ToolInput, &readInput); err != nil {
		return nil
	}

	filePath := readInput.FilePath
	base := filepath.Base(filePath)

	if isAllowed(base) {
		return nil
	}

	if sshDir != "" && isUnderDir(filePath, sshDir) {
		return denyPreToolUse(fmt.Sprintf("reading files under %s is not allowed", sshDir))
	}

	for _, r := range blockedPatterns {
		target := base
		if r.matchFull {
			target = filePath
		}

		matched, err := xstrings.GlobMatch(r.pattern, target)
		if err != nil {
			continue
		}

		if matched {
			return denyPreToolUse(r.reason)
		}
	}

	return nil
}

func isAllowed(base string) bool {
	for _, pattern := range allowedPatterns {
		matched, err := xstrings.GlobMatch(pattern, base)
		if err != nil {
			continue
		}

		if matched {
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

	return len(rel) > 0 && rel[0] != '.'
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
