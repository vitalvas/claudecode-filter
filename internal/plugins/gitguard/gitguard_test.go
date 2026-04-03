package gitguard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitalvas/claudecode-filter/internal/hook"
	"github.com/vitalvas/claudecode-filter/internal/marker"
)

func setupGitRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0o755))

	return dir
}

func TestOnPreToolUse(t *testing.T) {
	p := New()
	gitRoot := setupGitRepo(t)

	t.Run("blocks git commit", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "git commit -m 'test'"})
		result := p.OnPreToolUse(hook.Input{
			CWD:       gitRoot,
			ToolName:  "Bash",
			ToolInput: toolInput,
		})

		require.NotNil(t, result)

		var output hook.Output
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, "deny", output.HookSpecificOutput.PermissionDecision)
		assert.Contains(t, output.HookSpecificOutput.PermissionDecisionReason, "git commit")
	})

	t.Run("blocks git push", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "git push origin main"})
		result := p.OnPreToolUse(hook.Input{
			CWD:       gitRoot,
			ToolName:  "Bash",
			ToolInput: toolInput,
		})

		require.NotNil(t, result)

		var output hook.Output
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, "deny", output.HookSpecificOutput.PermissionDecision)
	})

	t.Run("allows git status", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "git status"})
		result := p.OnPreToolUse(hook.Input{
			CWD:       gitRoot,
			ToolName:  "Bash",
			ToolInput: toolInput,
		})

		assert.Nil(t, result)
	})

	t.Run("allows non-bash tool", func(t *testing.T) {
		result := p.OnPreToolUse(hook.Input{
			CWD:      gitRoot,
			ToolName: "Read",
		})

		assert.Nil(t, result)
	})

	t.Run("allows after marker consumed", func(t *testing.T) {
		p.OnUserPromptSubmit(hook.Input{
			CWD:    gitRoot,
			Prompt: "ok git commit",
		})

		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "git commit -m 'test'"})
		input := hook.Input{
			CWD:       gitRoot,
			ToolName:  "Bash",
			ToolInput: toolInput,
		}

		result := p.OnPreToolUse(input)
		assert.Nil(t, result)

		// Second attempt blocked again (one-time use)
		result = p.OnPreToolUse(input)
		require.NotNil(t, result)

		var output hook.Output
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, "deny", output.HookSpecificOutput.PermissionDecision)
	})

	t.Run("any ok git unlocks any operation", func(t *testing.T) {
		p.OnUserPromptSubmit(hook.Input{
			CWD:    gitRoot,
			Prompt: "ok git push",
		})

		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "git commit -m 'test'"})
		result := p.OnPreToolUse(hook.Input{
			CWD:       gitRoot,
			ToolName:  "Bash",
			ToolInput: toolInput,
		})

		assert.Nil(t, result)
	})
}

func TestOnUserPromptSubmit(t *testing.T) {
	p := New()
	gitRoot := setupGitRepo(t)

	t.Run("creates marker", func(t *testing.T) {
		p.OnUserPromptSubmit(hook.Input{
			CWD:    gitRoot,
			Prompt: "ok git commit",
		})

		_, ok := marker.Consume(gitRoot, markerName)
		assert.True(t, ok)
	})

	t.Run("case insensitive", func(t *testing.T) {
		p.OnUserPromptSubmit(hook.Input{
			CWD:    gitRoot,
			Prompt: "OK GIT MERGE",
		})

		_, ok := marker.Consume(gitRoot, markerName)
		assert.True(t, ok)
	})

	t.Run("no match does nothing", func(t *testing.T) {
		result := p.OnUserPromptSubmit(hook.Input{
			CWD:    gitRoot,
			Prompt: "please fix the bug",
		})

		assert.Nil(t, result)

		_, ok := marker.Consume(gitRoot, markerName)
		assert.False(t, ok)
	})
}

func TestOnSessionEnd(t *testing.T) {
	p := New()
	gitRoot := setupGitRepo(t)

	t.Run("cleans up markers", func(t *testing.T) {
		p.OnUserPromptSubmit(hook.Input{
			CWD:    gitRoot,
			Prompt: "ok git commit",
		})

		p.OnSessionEnd(hook.Input{CWD: gitRoot})

		_, ok := marker.Consume(gitRoot, markerName)
		assert.False(t, ok)
	})
}
