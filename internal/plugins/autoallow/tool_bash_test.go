package autoallow

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitalvas/claudecode-filter/internal/hook"
)

func TestHandleBash(t *testing.T) {
	p := New()

	t.Run("allows go test", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "go test ./..."})
		result := p.OnPermissionRequest(hook.Input{
			ToolName:  "Bash",
			ToolInput: toolInput,
		})

		require.NotNil(t, result)

		var output hook.PermissionRequestOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionAllow, output.HookSpecificOutput.Decision.Behavior)
	})

	t.Run("allows go build", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "go build ./..."})
		result := p.OnPermissionRequest(hook.Input{
			ToolName:  "Bash",
			ToolInput: toolInput,
		})

		require.NotNil(t, result)

		var output hook.PermissionRequestOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionAllow, output.HookSpecificOutput.Decision.Behavior)
	})

	t.Run("allows yake tests", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "yake tests"})
		result := p.OnPermissionRequest(hook.Input{
			ToolName:  "Bash",
			ToolInput: toolInput,
		})

		require.NotNil(t, result)

		var output hook.PermissionRequestOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionAllow, output.HookSpecificOutput.Decision.Behavior)
	})

	t.Run("allows golangci-lint run", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "golangci-lint run ./..."})
		result := p.OnPermissionRequest(hook.Input{
			ToolName:  "Bash",
			ToolInput: toolInput,
		})

		require.NotNil(t, result)

		var output hook.PermissionRequestOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionAllow, output.HookSpecificOutput.Decision.Behavior)
	})

	t.Run("does not allow unknown command", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "rm -rf /"})
		result := p.OnPermissionRequest(hook.Input{
			ToolName:  "Bash",
			ToolInput: toolInput,
		})

		assert.Nil(t, result)
	})

	t.Run("does not allow partial prefix match", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "go testing"})
		result := p.OnPermissionRequest(hook.Input{
			ToolName:  "Bash",
			ToolInput: toolInput,
		})

		assert.Nil(t, result)
	})
}
