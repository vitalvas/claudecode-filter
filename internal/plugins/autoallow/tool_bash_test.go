package autoallow

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitalvas/claudecode-filter/internal/hook"
)

func TestHandleBash(t *testing.T) {
	h := hook.BuildChain(New())

	t.Run("allows go test", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "go test ./..."})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PermissionRequestOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionAllow, output.HookSpecificOutput.Decision.Behavior)
	})

	t.Run("allows go build", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "go build ./..."})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PermissionRequestOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionAllow, output.HookSpecificOutput.Decision.Behavior)
	})

	t.Run("allows yake tests", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "yake tests"})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PermissionRequestOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionAllow, output.HookSpecificOutput.Decision.Behavior)
	})

	t.Run("allows golangci-lint run", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "golangci-lint run ./..."})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PermissionRequestOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionAllow, output.HookSpecificOutput.Decision.Behavior)
	})

	t.Run("allows gh api", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "gh api repos/owner/repo/pulls"})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PermissionRequestOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionAllow, output.HookSpecificOutput.Decision.Behavior)
	})

	t.Run("allows gh repo view", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "gh repo view owner/repo"})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PermissionRequestOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionAllow, output.HookSpecificOutput.Decision.Behavior)
	})

	t.Run("does not allow unknown command", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "rm -rf /"})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		assert.Nil(t, result)
	})

	t.Run("does not allow partial prefix match", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "go testing"})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		assert.Nil(t, result)
	})
}
