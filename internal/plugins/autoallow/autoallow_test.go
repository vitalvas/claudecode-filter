package autoallow

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitalvas/claudecode-filter/internal/hook"
)

func TestPassthrough(t *testing.T) {
	h := hook.BuildChain(New())

	t.Run("passes non-PermissionRequest events", func(t *testing.T) {
		result := h(hook.Input{
			HookEventName: hook.EventPreToolUse,
			ToolName:      "Bash",
		})

		assert.Nil(t, result)
	})

	t.Run("allows WebSearch in PermissionRequest", func(t *testing.T) {
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			ToolName:      "WebSearch",
		})

		require.NotNil(t, result)

		var output hook.PermissionRequestOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionAllow, output.HookSpecificOutput.Decision.Behavior)
	})

	t.Run("passes unknown tool in PermissionRequest", func(t *testing.T) {
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			ToolName:      "Write",
		})

		assert.Nil(t, result)
	})
}
