package autoallow

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitalvas/claudecode-filter/internal/hook"
)

func TestHandleWebFetch(t *testing.T) {
	h := hook.BuildChain(New())

	t.Run("allows github.com", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.WebFetchToolInput{URL: "https://github.com/owner/repo"})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			ToolName:      "WebFetch",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PermissionRequestOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionAllow, output.HookSpecificOutput.Decision.Behavior)
	})

	t.Run("allows raw.githubusercontent.com", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.WebFetchToolInput{URL: "https://raw.githubusercontent.com/owner/repo/main/README.md"})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			ToolName:      "WebFetch",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PermissionRequestOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionAllow, output.HookSpecificOutput.Decision.Behavior)
	})

	t.Run("does not allow unknown domain", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.WebFetchToolInput{URL: "https://evil.com/something"})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			ToolName:      "WebFetch",
			ToolInput:     toolInput,
		})

		assert.Nil(t, result)
	})

	t.Run("does not allow on PreToolUse event", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.WebFetchToolInput{URL: "https://github.com/owner/repo"})
		result := h(hook.Input{
			HookEventName: hook.EventPreToolUse,
			ToolName:      "WebFetch",
			ToolInput:     toolInput,
		})

		assert.Nil(t, result)
	})
}
