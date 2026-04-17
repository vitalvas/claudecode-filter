package gitguard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

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

func TestHandlePreToolUse(t *testing.T) {
	h := hook.BuildChain(New())
	gitRoot := setupGitRepo(t)

	t.Run("blocks git commit", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "git commit -m 'test'"})
		result := h(hook.Input{
			HookEventName: hook.EventPreToolUse,
			CWD:           gitRoot,
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PreToolUseOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionDeny, output.HookSpecificOutput.PermissionDecision)
		assert.Contains(t, output.HookSpecificOutput.PermissionDecisionReason, "git commit")
	})

	t.Run("blocks git push", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "git push origin main"})
		result := h(hook.Input{
			HookEventName: hook.EventPreToolUse,
			CWD:           gitRoot,
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PreToolUseOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionDeny, output.HookSpecificOutput.PermissionDecision)
	})

	t.Run("allows git status", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "git status"})
		result := h(hook.Input{
			HookEventName: hook.EventPreToolUse,
			CWD:           gitRoot,
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		assert.Nil(t, result)
	})

	t.Run("allows non-bash tool", func(t *testing.T) {
		result := h(hook.Input{
			HookEventName: hook.EventPreToolUse,
			CWD:           gitRoot,
			ToolName:      "Read",
		})

		assert.Nil(t, result)
	})

	t.Run("allows after marker consumed", func(t *testing.T) {
		h(hook.Input{
			HookEventName: hook.EventUserPromptSubmit,
			CWD:           gitRoot,
			Prompt:        "ok git commit",
		})

		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "git commit -m 'test'"})
		input := hook.Input{
			HookEventName: hook.EventPreToolUse,
			CWD:           gitRoot,
			ToolName:      "Bash",
			ToolInput:     toolInput,
		}

		result := h(input)
		assert.Nil(t, result)

		// Second attempt blocked again (one-time use)
		result = h(input)
		require.NotNil(t, result)

		var output hook.PreToolUseOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionDeny, output.HookSpecificOutput.PermissionDecision)
	})

	t.Run("blocks commit with Co-Authored-By", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{
			Command: "git commit -m \"$(cat <<'EOF'\nfeat: add feature\n\nCo-Authored-By: user <user@example.com>\nEOF\n)\"",
		})
		result := h(hook.Input{
			HookEventName: hook.EventPreToolUse,
			CWD:           gitRoot,
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PreToolUseOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionDeny, output.HookSpecificOutput.PermissionDecision)
		assert.Contains(t, output.HookSpecificOutput.PermissionDecisionReason, "co-authored-by:")
	})

	t.Run("blocks commit with Co-Authored-By even with marker", func(t *testing.T) {
		h(hook.Input{
			HookEventName: hook.EventUserPromptSubmit,
			CWD:           gitRoot,
			Prompt:        "ok git commit",
		})

		toolInput, _ := json.Marshal(hook.BashToolInput{
			Command: "git commit -m 'feat: thing\n\nCo-Authored-By: user <user@example.com>'",
		})
		result := h(hook.Input{
			HookEventName: hook.EventPreToolUse,
			CWD:           gitRoot,
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PreToolUseOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionDeny, output.HookSpecificOutput.PermissionDecision)
		assert.Contains(t, output.HookSpecificOutput.PermissionDecisionReason, "co-authored-by:")
	})

	t.Run("blocks commit with AI-assistant", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{
			Command: "git commit -m 'feat: add feature\n\nAI-assistant: Claude'",
		})
		result := h(hook.Input{
			HookEventName: hook.EventPreToolUse,
			CWD:           gitRoot,
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PreToolUseOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionDeny, output.HookSpecificOutput.PermissionDecision)
		assert.Contains(t, output.HookSpecificOutput.PermissionDecisionReason, "ai-assistant:")
	})

	t.Run("blocks commit with AI-assistant even with marker", func(t *testing.T) {
		h(hook.Input{
			HookEventName: hook.EventUserPromptSubmit,
			CWD:           gitRoot,
			Prompt:        "ok git commit",
		})

		toolInput, _ := json.Marshal(hook.BashToolInput{
			Command: "git commit -m 'fix: bug\n\nAI-Assistant: copilot'",
		})
		result := h(hook.Input{
			HookEventName: hook.EventPreToolUse,
			CWD:           gitRoot,
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PreToolUseOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionDeny, output.HookSpecificOutput.PermissionDecision)
		assert.Contains(t, output.HookSpecificOutput.PermissionDecisionReason, "ai-assistant:")
	})

	t.Run("any ok git unlocks any operation", func(t *testing.T) {
		h(hook.Input{
			HookEventName: hook.EventUserPromptSubmit,
			CWD:           gitRoot,
			Prompt:        "ok git push",
		})

		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "git commit -m 'test'"})
		result := h(hook.Input{
			HookEventName: hook.EventPreToolUse,
			CWD:           gitRoot,
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		assert.Nil(t, result)
	})
}

func TestHandleUserPromptSubmit(t *testing.T) {
	h := hook.BuildChain(New())
	gitRoot := setupGitRepo(t)

	t.Run("creates one-shot marker", func(t *testing.T) {
		h(hook.Input{
			HookEventName: hook.EventUserPromptSubmit,
			CWD:           gitRoot,
			Prompt:        "ok git commit",
		})

		val, ok := marker.Read(gitRoot, markerName)
		assert.True(t, ok)
		assert.Equal(t, "1", val)

		marker.Remove(gitRoot, markerName)
	})

	t.Run("creates timed marker with hours", func(t *testing.T) {
		h(hook.Input{
			HookEventName: hook.EventUserPromptSubmit,
			CWD:           gitRoot,
			Prompt:        "ok git commit for 1h",
		})

		val, ok := marker.Read(gitRoot, markerName)
		assert.True(t, ok)
		assert.NotEqual(t, "1", val)

		marker.Remove(gitRoot, markerName)
	})

	t.Run("creates timed marker with minutes", func(t *testing.T) {
		h(hook.Input{
			HookEventName: hook.EventUserPromptSubmit,
			CWD:           gitRoot,
			Prompt:        "ok git commit for 30m",
		})

		val, ok := marker.Read(gitRoot, markerName)
		assert.True(t, ok)
		assert.NotEqual(t, "1", val)

		marker.Remove(gitRoot, markerName)
	})

	t.Run("case insensitive", func(t *testing.T) {
		h(hook.Input{
			HookEventName: hook.EventUserPromptSubmit,
			CWD:           gitRoot,
			Prompt:        "OK GIT MERGE",
		})

		_, ok := marker.Consume(gitRoot, markerName)
		assert.True(t, ok)
	})

	t.Run("no match does nothing", func(t *testing.T) {
		h(hook.Input{
			HookEventName: hook.EventUserPromptSubmit,
			CWD:           gitRoot,
			Prompt:        "please fix the bug",
		})

		_, ok := marker.Consume(gitRoot, markerName)
		assert.False(t, ok)
	})
}

func TestTimedAllow(t *testing.T) {
	h := hook.BuildChain(New())
	gitRoot := setupGitRepo(t)

	t.Run("allows multiple operations within time", func(t *testing.T) {
		h(hook.Input{
			HookEventName: hook.EventUserPromptSubmit,
			CWD:           gitRoot,
			Prompt:        "ok git commit for 1h",
		})

		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "git commit -m 'first'"})
		input := hook.Input{
			HookEventName: hook.EventPreToolUse,
			CWD:           gitRoot,
			ToolName:      "Bash",
			ToolInput:     toolInput,
		}

		assert.Nil(t, h(input))
		assert.Nil(t, h(input))
		assert.Nil(t, h(input))

		marker.Remove(gitRoot, markerName)
	})

	t.Run("blocks after expiry", func(t *testing.T) {
		marker.Create(gitRoot, markerName, "1000000000")

		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "git commit -m 'test'"})
		result := h(hook.Input{
			HookEventName: hook.EventPreToolUse,
			CWD:           gitRoot,
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PreToolUseOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionDeny, output.HookSpecificOutput.PermissionDecision)
	})
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  time.Duration
	}{
		{name: "1 hour", input: "1h", want: time.Hour},
		{name: "2 hours", input: "2h", want: 2 * time.Hour},
		{name: "30 minutes", input: "30m", want: 30 * time.Minute},
		{name: "invalid unit", input: "1x", want: 0},
		{name: "zero", input: "0h", want: 0},
		{name: "too short", input: "h", want: 0},
		{name: "empty", input: "", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parseDuration(tt.input))
		})
	}
}

func TestHandleSessionEnd(t *testing.T) {
	h := hook.BuildChain(New())
	gitRoot := setupGitRepo(t)

	t.Run("cleans up markers", func(t *testing.T) {
		h(hook.Input{
			HookEventName: hook.EventUserPromptSubmit,
			CWD:           gitRoot,
			Prompt:        "ok git commit",
		})

		h(hook.Input{
			HookEventName: hook.EventSessionEnd,
			CWD:           gitRoot,
		})

		_, ok := marker.Consume(gitRoot, markerName)
		assert.False(t, ok)
	})
}
