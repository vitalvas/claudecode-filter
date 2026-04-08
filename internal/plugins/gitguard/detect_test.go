package gitguard

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainsBlockedCommitHeader(t *testing.T) {
	tests := []struct {
		name       string
		command    string
		wantHeader string
		wantFound  bool
	}{
		{
			name:       "commit with Co-Authored-By",
			command:    "git commit -m \"$(cat <<'EOF'\nfeat: add feature\n\nCo-Authored-By: user <user@example.com>\nEOF\n)\"",
			wantHeader: "co-authored-by:",
			wantFound:  true,
		},
		{
			name:       "commit with lowercase co-authored-by",
			command:    "git commit -m 'fix: something\n\nco-authored-by: user <user@example.com>'",
			wantHeader: "co-authored-by:",
			wantFound:  true,
		},
		{
			name:       "commit with mixed case CO-AUTHORED-BY",
			command:    "git commit -m 'feat: thing\n\nCO-AUTHORED-BY: user <user@example.com>'",
			wantHeader: "co-authored-by:",
			wantFound:  true,
		},
		{
			name:       "commit with AI-assistant",
			command:    "git commit -m 'feat: add feature\n\nAI-assistant: Claude'",
			wantHeader: "ai-assistant:",
			wantFound:  true,
		},
		{
			name:       "commit with lowercase ai-assistant",
			command:    "git commit -m 'fix: something\n\nai-assistant: copilot'",
			wantHeader: "ai-assistant:",
			wantFound:  true,
		},
		{
			name:       "commit with mixed case AI-ASSISTANT",
			command:    "git commit -m 'feat: thing\n\nAI-ASSISTANT: Claude'",
			wantHeader: "ai-assistant:",
			wantFound:  true,
		},
		{
			name:      "commit without blocked headers",
			command:   "git commit -m 'feat: add new feature'",
			wantFound: false,
		},
		{
			name:      "empty command",
			command:   "",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header, found := containsBlockedCommitHeader(tt.command)
			assert.Equal(t, tt.wantFound, found)

			if tt.wantFound {
				assert.Equal(t, tt.wantHeader, header)
			}
		})
	}
}

func TestDetectBlockedOps(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    []string
	}{
		{
			name:    "simple commit",
			command: "git commit -m 'test'",
			want:    []string{"commit"},
		},
		{
			name:    "simple push",
			command: "git push origin main",
			want:    []string{"push"},
		},
		{
			name:    "force push",
			command: "git push --force origin main",
			want:    []string{"push"},
		},
		{
			name:    "merge",
			command: "git merge feature-branch",
			want:    []string{"merge"},
		},
		{
			name:    "rebase",
			command: "git rebase main",
			want:    []string{"rebase"},
		},
		{
			name:    "cherry-pick",
			command: "git cherry-pick abc123",
			want:    []string{"cherry-pick"},
		},
		{
			name:    "revert",
			command: "git revert HEAD",
			want:    []string{"revert"},
		},
		{
			name:    "tag",
			command: "git tag v1.0.0",
			want:    []string{"tag"},
		},
		{
			name:    "commit and push chained",
			command: "git commit -m 'test' && git push",
			want:    []string{"commit", "push"},
		},
		{
			name:    "allowed: status",
			command: "git status",
			want:    nil,
		},
		{
			name:    "allowed: diff",
			command: "git diff",
			want:    nil,
		},
		{
			name:    "allowed: log",
			command: "git log --oneline",
			want:    nil,
		},
		{
			name:    "allowed: add",
			command: "git add .",
			want:    nil,
		},
		{
			name:    "allowed: reset",
			command: "git reset HEAD~1",
			want:    nil,
		},
		{
			name:    "allowed: restore",
			command: "git restore file.go",
			want:    nil,
		},
		{
			name:    "allowed: checkout",
			command: "git checkout main",
			want:    nil,
		},
		{
			name:    "allowed: fetch",
			command: "git fetch origin",
			want:    nil,
		},
		{
			name:    "allowed: stash",
			command: "git stash pop",
			want:    nil,
		},
		{
			name:    "allowed: branch",
			command: "git branch -D feature",
			want:    nil,
		},
		{
			name:    "allowed: clean",
			command: "git clean -fd",
			want:    nil,
		},
		{
			name:    "not git command",
			command: "go test ./...",
			want:    nil,
		},
		{
			name:    "empty command",
			command: "",
			want:    nil,
		},
		{
			name:    "git with flags before subcommand",
			command: "git -C /some/path commit -m 'test'",
			want:    []string{"commit"},
		},
		{
			name:    "commit after other command",
			command: "cd /path && git commit -m 'test'",
			want:    []string{"commit"},
		},
		{
			name:    "commit with amend",
			command: "git commit --amend",
			want:    []string{"commit"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectBlockedOps(tt.command)
			assert.Equal(t, tt.want, got)
		})
	}
}
