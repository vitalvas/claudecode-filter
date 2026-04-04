package autoallow

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vitalvas/claudecode-filter/internal/hook"
)

var allowedBashPrefixes = []string{
	"gh api",
	"gh repo",
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
	"go vet",
	"golangci-lint run",
	"markdownlint",
	"yake code",
	"yake policy run",
	"yake run",
	"yake tests",
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

	return nil
}
