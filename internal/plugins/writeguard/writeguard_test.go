package writeguard

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

func TestWriteguard(t *testing.T) {
	h := hook.BuildChain(New())

	t.Run("allows write without marker", func(t *testing.T) {
		gitRoot := setupGitRepo(t)

		toolInput, _ := json.Marshal(map[string]string{"file_path": "/test/file.go", "content": "test"})
		result := h(hook.Input{
			HookEventName: hook.EventPreToolUse,
			CWD:           gitRoot,
			ToolName:      "Write",
			ToolInput:     toolInput,
		})

		assert.Nil(t, result)
	})

	t.Run("blocks Write with marker", func(t *testing.T) {
		gitRoot := setupGitRepo(t)
		require.NoError(t, marker.Create(gitRoot, markerName, "1"))

		toolInput, _ := json.Marshal(map[string]string{"file_path": "/test/file.go", "content": "test"})
		result := h(hook.Input{
			HookEventName: hook.EventPreToolUse,
			CWD:           gitRoot,
			ToolName:      "Write",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PreToolUseOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionDeny, output.HookSpecificOutput.PermissionDecision)
		assert.Contains(t, output.HookSpecificOutput.PermissionDecisionReason, "readonly")
	})

	t.Run("blocks Edit with marker", func(t *testing.T) {
		gitRoot := setupGitRepo(t)
		require.NoError(t, marker.Create(gitRoot, markerName, "1"))

		toolInput, _ := json.Marshal(map[string]string{"file_path": "/test/file.go"})
		result := h(hook.Input{
			HookEventName: hook.EventPreToolUse,
			CWD:           gitRoot,
			ToolName:      "Edit",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PreToolUseOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionDeny, output.HookSpecificOutput.PermissionDecision)
	})

	t.Run("blocks NotebookEdit with marker", func(t *testing.T) {
		gitRoot := setupGitRepo(t)
		require.NoError(t, marker.Create(gitRoot, markerName, "1"))

		toolInput, _ := json.Marshal(map[string]string{"file_path": "/test/notebook.ipynb"})
		result := h(hook.Input{
			HookEventName: hook.EventPreToolUse,
			CWD:           gitRoot,
			ToolName:      "NotebookEdit",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PreToolUseOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionDeny, output.HookSpecificOutput.PermissionDecision)
	})

	t.Run("blocks Bash write with marker", func(t *testing.T) {
		gitRoot := setupGitRepo(t)
		require.NoError(t, marker.Create(gitRoot, markerName, "1"))

		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "rm -rf dir"})
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

	t.Run("allows Bash read with marker", func(t *testing.T) {
		gitRoot := setupGitRepo(t)
		require.NoError(t, marker.Create(gitRoot, markerName, "1"))

		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "ls -la"})
		result := h(hook.Input{
			HookEventName: hook.EventPreToolUse,
			CWD:           gitRoot,
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		assert.Nil(t, result)
	})

	t.Run("allows Read with marker", func(t *testing.T) {
		gitRoot := setupGitRepo(t)
		require.NoError(t, marker.Create(gitRoot, markerName, "1"))

		toolInput, _ := json.Marshal(hook.ReadToolInput{FilePath: "/test/file.go"})
		result := h(hook.Input{
			HookEventName: hook.EventPreToolUse,
			CWD:           gitRoot,
			ToolName:      "Read",
			ToolInput:     toolInput,
		})

		assert.Nil(t, result)
	})

	t.Run("blocks on PermissionRequest too", func(t *testing.T) {
		gitRoot := setupGitRepo(t)
		require.NoError(t, marker.Create(gitRoot, markerName, "1"))

		toolInput, _ := json.Marshal(map[string]string{"file_path": "/test/file.go", "content": "test"})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			CWD:           gitRoot,
			ToolName:      "Write",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PreToolUseOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionDeny, output.HookSpecificOutput.PermissionDecision)
	})
}
