package gitguard

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

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

	if detectNoVerify(bashInput.Command) {
		return denyPreToolUse("git --no-verify is blocked because git hooks must run")
	}

	ops := detectBlockedOps(bashInput.Command)
	if len(ops) == 0 {
		return nil
	}

	if header, ok := containsBlockedCommitHeader(bashInput.Command); ok {
		return denyPreToolUse(fmt.Sprintf("commit messages must not contain '%s' headers", header))
	}

	if commitType, ok := detectDeniedCommitType(bashInput.Command); ok {
		if commitType == "!" {
			return denyPreToolUse("breaking change indicator '!' is not allowed in commit messages")
		}

		return denyPreToolUse(fmt.Sprintf(
			"commit type '%s' is not allowed. Use one of: feat, fix, perf, deps, revert, docs, chore",
			commitType,
		))
	}

	if isAllowed(input.CWD) {
		return nil
	}

	return denyPreToolUse(fmt.Sprintf(
		"git %s requires explicit user approval. Ask the user to say \"ok git %s\" (one-shot) or \"ok git %s for 1h\" (timed).",
		strings.Join(ops, ", "),
		ops[0],
		ops[0],
	))
}

var okGitRe = regexp.MustCompile(`(?i)\bok\s+git\s+[\w-]+(?:\s+for\s+(\d+[hm]))?`)

func handleUserPromptSubmit(input hook.Input) {
	matches := okGitRe.FindStringSubmatch(input.Prompt)
	if matches == nil {
		return
	}

	value := "1"

	if duration := matches[1]; duration != "" {
		d := parseDuration(duration)
		if d > 0 {
			value = strconv.FormatInt(time.Now().Add(d).Unix(), 10)
		}
	}

	marker.Create(input.CWD, markerName, value)
}

func parseDuration(s string) time.Duration {
	if len(s) < 2 {
		return 0
	}

	val, err := strconv.Atoi(s[:len(s)-1])
	if err != nil || val <= 0 {
		return 0
	}

	switch s[len(s)-1] {
	case 'h':
		return time.Duration(val) * time.Hour
	case 'm':
		return time.Duration(val) * time.Minute
	}

	return 0
}

func isAllowed(cwd string) bool {
	val, ok := marker.Read(cwd, markerName)
	if !ok {
		return false
	}

	// One-shot: value is "1", consume it
	if val == "1" {
		marker.Remove(cwd, markerName)

		return true
	}

	// Timed: value is expiry timestamp
	expiry, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		marker.Remove(cwd, markerName)

		return false
	}

	if time.Now().Unix() > expiry {
		marker.Remove(cwd, markerName)

		return false
	}

	return true
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
