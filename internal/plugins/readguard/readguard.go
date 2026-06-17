package readguard

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/vitalvas/claudecode-filter/internal/hook"
	"github.com/vitalvas/claudecode-filter/internal/marker"
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
	{pattern: fmt.Sprintf("%s*", marker.Prefix), reason: "reading marker files is not allowed"},
	{pattern: "id_rsa*", reason: "reading private key files is not allowed"},
	{pattern: "id_ecdsa*", reason: "reading private key files is not allowed"},
	{pattern: "id_ed25519*", reason: "reading private key files is not allowed"},
}

var allowedPatterns = []string{
	"*.pub",
}

type blockedDir struct {
	path         string
	allowProject bool
}

func allowedReadDirs() []string {
	var dirs []string

	home := os.Getenv("HOME")
	if home != "" {
		dirs = append(dirs,
			filepath.Join(home, ".cargo", "registry", "src"),
			filepath.Join(home, ".claude"),
			filepath.Join(home, ".rustup"),
		)
	}

	out, err := exec.Command("go", "env", "GOMODCACHE").Output()
	if err == nil {
		if dir := strings.TrimSpace(string(out)); dir != "" {
			dirs = append(dirs, dir)
		}
	}

	uid := fmt.Sprintf("%d", os.Getuid())
	dirs = append(dirs, filepath.Join("/private/tmp", fmt.Sprintf("claude-%s", uid)))

	return dirs
}

// New creates the readguard middleware.
func New() hook.Middleware {
	blockedDirs := blockedDirectories()
	allowedDirs := allowedReadDirs()

	return func(next hook.Handler) hook.Handler {
		return func(input hook.Input) *hook.Result {
			if input.HookEventName == hook.EventPreToolUse || input.HookEventName == hook.EventPermissionRequest {
				if input.ToolName == "Read" {
					if result := handleRead(input, blockedDirs, allowedDirs); result != nil {
						return result
					}
				}

				if input.ToolName == "Bash" {
					if result := handleBashCd(input); result != nil {
						return result
					}
				}
			}

			return next(input)
		}
	}
}

func blockedDirectories() []blockedDir {
	home := os.Getenv("HOME")

	dirs := []blockedDir{
		{path: filepath.Join(home, ".ssh")},
	}

	defaultGoPath := filepath.Join(home, "go")
	if goPath := os.Getenv("GOPATH"); goPath != "" && goPath != defaultGoPath {
		dirs = append(dirs, blockedDir{path: defaultGoPath})
	}

	if goPath := os.Getenv("GOPATH"); goPath != "" {
		dirs = append(dirs, blockedDir{
			path:         filepath.Join(goPath, "src"),
			allowProject: true,
		})
	}

	return dirs
}

func handleRead(input hook.Input, blockedDirs []blockedDir, allowedDirs []string) *hook.Result {
	var readInput hook.ReadToolInput
	if err := json.Unmarshal(input.ToolInput, &readInput); err != nil {
		return nil
	}

	filePath := expandHome(readInput.FilePath)
	base := filepath.Base(filePath)

	if isAllowed(base) {
		return nil
	}

	for _, dir := range blockedDirs {
		if !isUnderDir(filePath, dir.path) {
			continue
		}

		if dir.allowProject && input.CWD != "" && isUnderDir(filePath, input.CWD) {
			continue
		}

		if dir.allowProject {
			if input.CWD != "" && marker.Exists(input.CWD, "allow-extread") {
				return askResult(input.HookEventName, fmt.Sprintf("reading files under %s outside project requires approval", dir.path))
			}

			return denyResult(input.HookEventName, fmt.Sprintf("reading files under %s outside project is not allowed", dir.path))
		}

		return denyResult(input.HookEventName, fmt.Sprintf("reading files under %s is not allowed", dir.path))
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
			return denyResult(input.HookEventName, r.reason)
		}
	}

	for _, dir := range allowedDirs {
		if isUnderDir(filePath, dir) || filePath == dir {
			return nil
		}
	}

	if input.CWD != "" && !isUnderDir(filePath, input.CWD) {
		if marker.Exists(input.CWD, "allow-extread") {
			return askResult(input.HookEventName, "reading files outside the project directory requires approval")
		}

		return denyResult(input.HookEventName, "reading files outside the project directory is not allowed")
	}

	return nil
}

// handleBashCd gates Bash commands that cd into a directory outside the
// current project. Such a command would otherwise execute against another
// repository (e.g. an agent worktree) without going through the read guard.
func handleBashCd(input hook.Input) *hook.Result {
	if input.CWD == "" {
		return nil
	}

	var bashInput hook.BashToolInput
	if err := json.Unmarshal(input.ToolInput, &bashInput); err != nil {
		return nil
	}

	for _, target := range cdTargets(bashInput.Command) {
		if !filepath.IsAbs(target) {
			continue
		}

		if isUnderDir(target, input.CWD) || target == input.CWD {
			continue
		}

		if marker.Exists(input.CWD, "allow-extread") {
			return askResult(input.HookEventName, fmt.Sprintf("changing directory to %s outside project requires approval", target))
		}

		return denyResult(input.HookEventName, fmt.Sprintf("changing directory to %s outside project is not allowed", target))
	}

	return nil
}

// cdTargets returns the directory arguments of every `cd` invocation in a
// command, splitting on the common shell separators that chain commands.
func cdTargets(command string) []string {
	var targets []string

	segments := strings.FieldsFunc(command, func(r rune) bool {
		return r == ';' || r == '&' || r == '|' || r == '\n'
	})

	for _, segment := range segments {
		fields := strings.Fields(segment)
		if len(fields) >= 2 && fields[0] == "cd" {
			targets = append(targets, expandHome(strings.Trim(fields[1], `"'`)))
		}
	}

	return targets
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

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home := os.Getenv("HOME")
		if home != "" {
			return filepath.Join(home, path[2:])
		}
	}

	return path
}

func isUnderDir(filePath, dir string) bool {
	rel, err := filepath.Rel(dir, filePath)
	if err != nil {
		return false
	}

	return len(rel) > 0 && !strings.HasPrefix(rel, "..")
}

func askResult(event, reason string) *hook.Result {
	if event == hook.EventPermissionRequest {
		return permissionRequestResult(hook.PermissionAsk, reason)
	}

	return preToolUseResult(hook.PermissionAsk, reason)
}

func denyResult(event, reason string) *hook.Result {
	if event == hook.EventPermissionRequest {
		return permissionRequestResult(hook.PermissionDeny, reason)
	}

	return preToolUseResult(hook.PermissionDeny, reason)
}

func preToolUseResult(decision, reason string) *hook.Result {
	output := hook.PreToolUseOutputWrapper{
		HookSpecificOutput: hook.PreToolUseOutput{
			HookEventName:            hook.EventPreToolUse,
			PermissionDecision:       decision,
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

func permissionRequestResult(decision, reason string) *hook.Result {
	output := hook.PermissionRequestOutputWrapper{
		HookSpecificOutput: hook.PermissionRequestOutput{
			HookEventName: hook.EventPermissionRequest,
			Decision: hook.PermissionDecision{
				Behavior: decision,
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
