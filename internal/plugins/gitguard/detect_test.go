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
		{
			name:    "stash push is not blocked",
			command: "git stash push -m 'wip'",
			want:    nil,
		},
		{
			name:    "stash pop is not blocked",
			command: "git stash pop",
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectBlockedOps(tt.command)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDetectDeniedCommitType(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		wantType  string
		wantFound bool
	}{
		{
			name:      "denied: style",
			command:   "git commit -m 'style: format code'",
			wantType:  "style",
			wantFound: true,
		},
		{
			name:      "denied: refactor",
			command:   "git commit -m 'refactor: extract method'",
			wantType:  "refactor",
			wantFound: true,
		},
		{
			name:      "denied: test",
			command:   "git commit -m 'test: add unit tests'",
			wantType:  "test",
			wantFound: true,
		},
		{
			name:      "denied: build",
			command:   "git commit -m 'build: update deps'",
			wantType:  "build",
			wantFound: true,
		},
		{
			name:      "denied: ci",
			command:   "git commit -m 'ci: fix pipeline'",
			wantType:  "ci",
			wantFound: true,
		},
		{
			name:      "denied: refactor with scope",
			command:   "git commit -m 'refactor(auth): simplify logic'",
			wantType:  "refactor",
			wantFound: true,
		},
		{
			name:      "denied: breaking change indicator",
			command:   "git commit -m 'feat!: rewrite API'",
			wantType:  "!",
			wantFound: true,
		},
		{
			name:      "denied: breaking change with scope",
			command:   "git commit -m 'fix(auth)!: change token format'",
			wantType:  "!",
			wantFound: true,
		},
		{
			name:      "allowed: feat",
			command:   "git commit -m 'feat: add feature'",
			wantFound: false,
		},
		{
			name:      "allowed: fix",
			command:   "git commit -m 'fix: resolve bug'",
			wantFound: false,
		},
		{
			name:      "allowed: perf",
			command:   "git commit -m 'perf: optimize query'",
			wantFound: false,
		},
		{
			name:      "allowed: deps",
			command:   "git commit -m 'deps: update modules'",
			wantFound: false,
		},
		{
			name:      "allowed: revert",
			command:   "git commit -m 'revert: undo change'",
			wantFound: false,
		},
		{
			name:      "allowed: docs",
			command:   "git commit -m 'docs: update readme'",
			wantFound: false,
		},
		{
			name:      "allowed: chore",
			command:   "git commit -m 'chore: cleanup'",
			wantFound: false,
		},
		{
			name:      "non-conventional commit",
			command:   "git commit -m 'random message'",
			wantFound: false,
		},
		{
			name:      "no commit message",
			command:   "git push origin main",
			wantFound: false,
		},
		{
			name:      "empty command",
			command:   "",
			wantFound: false,
		},
		{
			name:      "double quoted message",
			command:   `git commit -m "refactor: extract method"`,
			wantType:  "refactor",
			wantFound: true,
		},
		{
			name:      "heredoc commit message",
			command:   "git commit -m \"$(cat <<'EOF'\nstyle: format code\nEOF\n)\"",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commitType, found := detectDeniedCommitType(tt.command)
			assert.Equal(t, tt.wantFound, found)

			if tt.wantFound {
				assert.Equal(t, tt.wantType, commitType)
			}
		})
	}
}
