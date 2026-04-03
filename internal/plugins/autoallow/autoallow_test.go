package autoallow

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

	t.Run("passes unknown tool in PermissionRequest", func(t *testing.T) {
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			ToolName:      "Write",
		})

		assert.Nil(t, result)
	})
}
