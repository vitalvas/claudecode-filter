package autoallow

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitalvas/claudecode-filter/internal/hook"
)

func TestHandleRead(t *testing.T) {
	h := hook.BuildChain(New())

	modCache := getGoModCache()
	if modCache == "" {
		t.Skip("GOMODCACHE not available")
	}

	t.Run("allows read from GOMODCACHE", func(t *testing.T) {
		toolInput, _ := json.Marshal(readToolInput{
			FilePath: fmt.Sprintf("%s/github.com/stretchr/testify/assert/assertions.go", modCache),
		})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			ToolName:      "Read",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PermissionRequestOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionAllow, output.HookSpecificOutput.Decision.Behavior)
	})

	t.Run("does not allow read outside GOMODCACHE", func(t *testing.T) {
		toolInput, _ := json.Marshal(readToolInput{
			FilePath: "/etc/passwd",
		})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			ToolName:      "Read",
			ToolInput:     toolInput,
		})

		assert.Nil(t, result)
	})
}
