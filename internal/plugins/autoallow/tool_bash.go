package autoallow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vitalvas/claudecode-filter/internal/hook"
)

var deniedBashPrefixes = []string{
	"ssh",
	"telnet",
}

var projectScopedPrefixes = []string{
	"mkdir",
}

var allowedBashPrefixes = []string{
	"cargo",
	"cat",
	"container",
	"curl",
	"date",
	"df",
	"diff",
	"dig",
	"docker",
	"du",
	"env",
	"file",
	"find",
	"fzf",
	"gh api",
	"gh issue list",
	"gh issue view",
	"gh label",
	"gh pr diff",
	"gh pr view",
	"gh repo",
	"gh run view",
	"git add",
	"git branch",
	"git checkout",
	"git cherry-pick",
	"git clean",
	"git commit",
	"git diff",
	"git fetch",
	"git log",
	"git merge",
	"git mv",
	"git pull",
	"git push",
	"git rebase",
	"git reset",
	"git restore",
	"git revert",
	"git rm",
	"git show",
	"git stash",
	"git status",
	"git switch",
	"git tag",
	"go build",
	"go clean",
	"go doc",
	"go env",
	"go fmt",
	"go get",
	"go list",
	"go mod",
	"go run",
	"go test",
	"go tool",
	"go vet",
	"gofmt",
	"gofumpt",
	"goimports",
	"golangci-lint run",
	"goreleaser",
	"grep",
	"head",
	"hostname",
	"hugo",
	"id",
	"jq",
	"ls",
	"lsof",
	"markdownlint",
	"rustc",
	"rustup",
	"nslookup",
	"ping",
	"ps",
	"pwd",
	"rg",
	"scp",
	"sed",
	"sort",
	"stat",
	"tail",
	"task",
	"tree",
	"uname",
	"uniq",
	"wc",
	"which",
	"whoami",
	"yake",
}

var envVarPrefix = regexp.MustCompile(`^(?:[A-Za-z_][A-Za-z0-9_]*=\S+\s+)+`)

func stripEnvVars(command string) string {
	return envVarPrefix.ReplaceAllString(command, "")
}

func matchesPrefix(command string, prefixes []string) string {
	cmd := stripEnvVars(command)

	for _, prefix := range prefixes {
		if cmd == prefix || strings.HasPrefix(cmd, fmt.Sprintf("%s ", prefix)) {
			return prefix
		}
	}

	return ""
}

func handleBashDeny(input hook.Input) *hook.Result {
	var bashInput hook.BashToolInput
	if err := json.Unmarshal(input.ToolInput, &bashInput); err != nil {
		return nil
	}

	if prefix := matchesPrefix(bashInput.Command, deniedBashPrefixes); prefix != "" {
		return denyPreToolUse(fmt.Sprintf("command '%s' is not allowed", prefix))
	}

	return nil
}

func handleBash(input hook.Input) *hook.Result {
	var bashInput hook.BashToolInput
	if err := json.Unmarshal(input.ToolInput, &bashInput); err != nil {
		return nil
	}

	if matchesPrefix(bashInput.Command, allowedBashPrefixes) != "" {
		if input.CWD != "" && hasAbsPathOutsideProject(stripEnvVars(bashInput.Command), input.CWD) {
			return denyPermissionRequest("command references paths outside the project directory")
		}

		return allowPermissionRequest()
	}

	if input.CWD != "" && isProjectScopedCommand(stripEnvVars(bashInput.Command), input.CWD) {
		return allowPermissionRequest()
	}

	return nil
}

func isProjectScopedCommand(command, cwd string) bool {
	if matchesPrefix(command, projectScopedPrefixes) != "" {
		return allPathArgsInProject(command, cwd)
	}

	return false
}

var quotedStrings = regexp.MustCompile(`"[^"]*"|'[^']*'`)

func hasAbsPathOutsideProject(command, cwd string) bool {
	cmd := quotedStrings.ReplaceAllString(command, "")
	args := strings.Fields(cmd)

	for _, arg := range args[1:] {
		if strings.HasPrefix(arg, "-") {
			continue
		}

		expanded := arg
		if strings.HasPrefix(expanded, "~/") {
			if home := os.Getenv("HOME"); home != "" {
				expanded = filepath.Join(home, expanded[2:])
			}
		}

		if !filepath.IsAbs(expanded) {
			continue
		}

		rel, err := filepath.Rel(cwd, filepath.Clean(expanded))
		if err != nil || strings.HasPrefix(rel, "..") {
			return true
		}
	}

	return false
}

func allPathArgsInProject(command, cwd string) bool {
	args := strings.Fields(command)

	for _, arg := range args[1:] {
		if strings.HasPrefix(arg, "-") {
			continue
		}

		path := arg
		if !filepath.IsAbs(path) {
			path = filepath.Join(cwd, path)
		}

		path = filepath.Clean(path)

		rel, err := filepath.Rel(cwd, path)
		if err != nil || strings.HasPrefix(rel, "..") {
			return false
		}
	}

	return true
}
