package autoallow

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vitalvas/claudecode-filter/internal/hook"
)

var projectScopedPrefixes = []string{
	"mkdir",
}

var allowedBashPrefixes = []string{
	"curl",
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
	"goimports",
	"golangci-lint run",
	"grep",
	"hugo",
	"ls",
	"lsof",
	"markdownlint",
	"rg",
	"tree",
	"yake",
}

func handleBash(input hook.Input) *hook.Result {
	var bashInput hook.BashToolInput
	if err := json.Unmarshal(input.ToolInput, &bashInput); err != nil {
		return nil
	}

	for _, prefix := range allowedBashPrefixes {
		if bashInput.Command == prefix || strings.HasPrefix(bashInput.Command, fmt.Sprintf("%s ", prefix)) {
			return allowPermissionRequest()
		}
	}

	if input.CWD != "" && isProjectScopedCommand(bashInput.Command, input.CWD) {
		return allowPermissionRequest()
	}

	return nil
}

func isProjectScopedCommand(command, cwd string) bool {
	for _, prefix := range projectScopedPrefixes {
		if command == prefix || strings.HasPrefix(command, fmt.Sprintf("%s ", prefix)) {
			return allPathArgsInProject(command, cwd)
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
