package sandbox

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

func TestSandbox(t *testing.T) {
	h := hook.BuildChain(New())
	home := os.Getenv("HOME")
	gitRoot := setupGitRepo(t)

	t.Run("allows read in workspace", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.ReadToolInput{
			FilePath: filepath.Join(home, "workspace", "project", "main.go"),
		})
		result := h(hook.Input{
			HookEventName: hook.EventPreToolUse,
			CWD:           gitRoot,
			ToolName:      "Read",
			ToolInput:     toolInput,
		})

		assert.Nil(t, result)
	})

	t.Run("denies read outside workspace", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.ReadToolInput{
			FilePath: "/etc/passwd",
		})
		result := h(hook.Input{
			HookEventName: hook.EventPreToolUse,
			CWD:           gitRoot,
			ToolName:      "Read",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PreToolUseOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionDeny, output.HookSpecificOutput.PermissionDecision)
		assert.Contains(t, output.HookSpecificOutput.PermissionDecisionReason, "sandbox")
	})

	t.Run("denies /tmp with specific message", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.ReadToolInput{
			FilePath: "/tmp/somefile",
		})
		result := h(hook.Input{
			HookEventName: hook.EventPreToolUse,
			CWD:           gitRoot,
			ToolName:      "Read",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PreToolUseOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionDeny, output.HookSpecificOutput.PermissionDecision)
		assert.Contains(t, output.HookSpecificOutput.PermissionDecisionReason, "/tmp")
		assert.Contains(t, output.HookSpecificOutput.PermissionDecisionReason, ".tmp/")
	})

	t.Run("denies Write outside workspace", func(t *testing.T) {
		toolInput, _ := json.Marshal(map[string]string{"file_path": "/etc/hosts", "content": "test"})
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
	})

	t.Run("denies Bash with absolute path outside workspace", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "cat /etc/passwd"})
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

	t.Run("allows Bash without absolute path", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "go test ./..."})
		result := h(hook.Input{
			HookEventName: hook.EventPreToolUse,
			CWD:           gitRoot,
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		assert.Nil(t, result)
	})

	t.Run("allows after marker with ask", func(t *testing.T) {
		markerRoot := setupGitRepo(t)

		h(hook.Input{
			HookEventName: hook.EventUserPromptSubmit,
			CWD:           markerRoot,
			Prompt:        "ok allow rootread",
		})

		toolInput, _ := json.Marshal(hook.ReadToolInput{FilePath: "/etc/hosts"})
		result := h(hook.Input{
			HookEventName: hook.EventPreToolUse,
			CWD:           markerRoot,
			ToolName:      "Read",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PreToolUseOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionAsk, output.HookSpecificOutput.PermissionDecision)
	})

	t.Run("marker is not consumed", func(t *testing.T) {
		markerRoot := setupGitRepo(t)
		require.NoError(t, marker.Create(markerRoot, markerName, "1"))

		toolInput, _ := json.Marshal(hook.ReadToolInput{FilePath: "/etc/hosts"})

		result := h(hook.Input{
			HookEventName: hook.EventPreToolUse,
			CWD:           markerRoot,
			ToolName:      "Read",
			ToolInput:     toolInput,
		})
		require.NotNil(t, result)

		// Second call should also ask, not deny
		result = h(hook.Input{
			HookEventName: hook.EventPreToolUse,
			CWD:           markerRoot,
			ToolName:      "Read",
			ToolInput:     toolInput,
		})
		require.NotNil(t, result)

		var output hook.PreToolUseOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionAsk, output.HookSpecificOutput.PermissionDecision)
	})

	t.Run("ignores non-PreToolUse events", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.ReadToolInput{FilePath: "/etc/passwd"})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			CWD:           gitRoot,
			ToolName:      "Read",
			ToolInput:     toolInput,
		})

		assert.Nil(t, result)
	})
}

func TestExtractBashPath(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    string
	}{
		{name: "absolute path", command: "cat /etc/passwd", want: "/etc/passwd"},
		{name: "flag then path", command: "ls -la /var/log", want: "/var/log"},
		{name: "relative path", command: "cat file.go", want: ""},
		{name: "no path", command: "go test ./...", want: ""},
		{name: "heredoc with absolute path in body", command: "git commit --file=- <<'EOF'\nfeat: add /etc/config support\nEOF", want: ""},
		{name: "heredoc with EOF marker", command: "git commit -m \"$(cat <<'EOF'\nsome message\nEOF\n)\"", want: ""},
		{name: "empty", command: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, extractBashPath(tt.command))
		})
	}
}
