package autoallow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitalvas/claudecode-filter/internal/hook"
)

func TestHandleRead(t *testing.T) {
	h := hook.BuildChain(New())

	dirs := getAllowedReadDirs()
	if len(dirs) == 0 {
		t.Skip("no allowed read dirs available")
	}

	t.Run("allows read from allowed dir", func(t *testing.T) {
		toolInput, _ := json.Marshal(readToolInput{
			FilePath: fmt.Sprintf("%s/github.com/stretchr/testify/assert/assertions.go", dirs[0]),
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

	t.Run("allows read from cargo registry", func(t *testing.T) {
		home := os.Getenv("HOME")
		if home == "" {
			t.Skip("HOME not set")
		}

		toolInput, _ := json.Marshal(readToolInput{
			FilePath: filepath.Join(home, ".cargo", "registry", "src", "index/some-crate/src/lib.rs"),
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

	t.Run("does not allow read outside allowed dirs", func(t *testing.T) {
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
