package writeguard

import (
	"encoding/json"
	"fmt"
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
	homeDir := os.Getenv("HOME")

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

	for _, tc := range []struct {
		toolName string
		pathKey  string
		dir      string
	}{
		{toolName: "Write", pathKey: "file_path", dir: "plans"},
		{toolName: "Edit", pathKey: "file_path", dir: "plans"},
		{toolName: "NotebookEdit", pathKey: "notebook_path", dir: "plans"},
		{toolName: "Write", pathKey: "file_path", dir: "projects"},
		{toolName: "Edit", pathKey: "file_path", dir: "projects"},
		{toolName: "NotebookEdit", pathKey: "notebook_path", dir: "projects"},
	} {
		t.Run(fmt.Sprintf("blocks %s under $HOME/.claude/%s", tc.toolName, tc.dir), func(t *testing.T) {
			toolInput, _ := json.Marshal(map[string]string{
				tc.pathKey: filepath.Join(homeDir, ".claude", tc.dir, "work.md"),
			})
			result := h(hook.Input{
				HookEventName: hook.EventPreToolUse,
				ToolName:      tc.toolName,
				ToolInput:     toolInput,
			})

			require.NotNil(t, result)

			var output hook.PreToolUseOutputWrapper
			require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
			assert.Equal(t, hook.PermissionDeny, output.HookSpecificOutput.PermissionDecision)
			assert.Contains(t, output.HookSpecificOutput.PermissionDecisionReason, filepath.Join(homeDir, ".claude", tc.dir))
		})
	}

	t.Run("appends AGENTS.md contents to deny reason when it exists in cwd", func(t *testing.T) {
		gitRoot := setupGitRepo(t)
		require.NoError(t, os.WriteFile(filepath.Join(gitRoot, "AGENTS.md"), []byte("project rules here"), 0o644))

		toolInput, _ := json.Marshal(map[string]string{
			"file_path": filepath.Join(homeDir, ".claude", "projects", "work.md"),
		})
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
		assert.Contains(t, output.HookSpecificOutput.PermissionDecisionReason, "project rules here")
	})

	t.Run("omits AGENTS.md hint when file absent in cwd", func(t *testing.T) {
		gitRoot := setupGitRepo(t)

		toolInput, _ := json.Marshal(map[string]string{
			"file_path": filepath.Join(homeDir, ".claude", "projects", "work.md"),
		})
		result := h(hook.Input{
			HookEventName: hook.EventPreToolUse,
			CWD:           gitRoot,
			ToolName:      "Write",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PreToolUseOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.NotContains(t, output.HookSpecificOutput.PermissionDecisionReason, "AGENTS.md")
	})

	t.Run("blocks Write to $HOME/.claude/plans directory", func(t *testing.T) {
		toolInput, _ := json.Marshal(map[string]string{
			"file_path": filepath.Join(homeDir, ".claude", "plans"),
		})
		result := h(hook.Input{
			HookEventName: hook.EventPreToolUse,
			ToolName:      "Write",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)
	})

	t.Run("allows Write to sibling of $HOME/.claude/plans", func(t *testing.T) {
		toolInput, _ := json.Marshal(map[string]string{
			"file_path": filepath.Join(homeDir, ".claude", "plans-backup", "work.md"),
		})
		result := h(hook.Input{
			HookEventName: hook.EventPreToolUse,
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

	t.Run("blocks Bash read with marker", func(t *testing.T) {
		gitRoot := setupGitRepo(t)
		require.NoError(t, marker.Create(gitRoot, markerName, "1"))

		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "ls -la"})
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

	t.Run("blocks Write to marker file without readonly marker", func(t *testing.T) {
		gitRoot := setupGitRepo(t)

		toolInput, _ := json.Marshal(map[string]string{"file_path": filepath.Join(gitRoot, ".tmp", "claudecode-filter-readonly")})
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
		assert.Contains(t, output.HookSpecificOutput.PermissionDecisionReason, "marker")
	})

	t.Run("blocks Edit to marker file", func(t *testing.T) {
		gitRoot := setupGitRepo(t)

		toolInput, _ := json.Marshal(map[string]string{"file_path": filepath.Join(gitRoot, ".tmp", "claudecode-filter-gitguard-allow")})
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

	t.Run("blocks Bash rm of marker file", func(t *testing.T) {
		gitRoot := setupGitRepo(t)

		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "rm .tmp/claudecode-filter-readonly"})
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
		assert.Contains(t, output.HookSpecificOutput.PermissionDecisionReason, "marker")
	})

	t.Run("allows Bash command without marker reference", func(t *testing.T) {
		gitRoot := setupGitRepo(t)

		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "ls -la"})
		result := h(hook.Input{
			HookEventName: hook.EventPreToolUse,
			CWD:           gitRoot,
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		assert.Nil(t, result)
	})

	t.Run("allows Write to non-marker file", func(t *testing.T) {
		gitRoot := setupGitRepo(t)

		toolInput, _ := json.Marshal(map[string]string{"file_path": filepath.Join(gitRoot, "src", "main.go"), "content": "test"})
		result := h(hook.Input{
			HookEventName: hook.EventPreToolUse,
			CWD:           gitRoot,
			ToolName:      "Write",
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
