package autoallow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vitalvas/claudecode-filter/internal/hook"
)

func TestOnPermissionRequestOtherTools(t *testing.T) {
	p := New()

	t.Run("ignores non-bash non-read tool", func(t *testing.T) {
		result := p.OnPermissionRequest(hook.Input{
			ToolName: "Write",
		})

		assert.Nil(t, result)
	})
}

func TestOnPreToolUse(t *testing.T) {
	p := New()

	t.Run("returns nil", func(t *testing.T) {
		result := p.OnPreToolUse(hook.Input{})
		assert.Nil(t, result)
	})
}

func TestOnUserPromptSubmit(t *testing.T) {
	p := New()

	t.Run("returns nil", func(t *testing.T) {
		result := p.OnUserPromptSubmit(hook.Input{Prompt: "test"})
		assert.Nil(t, result)
	})
}

func TestOnSessionEnd(t *testing.T) {
	p := New()

	t.Run("does nothing", func(_ *testing.T) {
		p.OnSessionEnd(hook.Input{})
	})
}
