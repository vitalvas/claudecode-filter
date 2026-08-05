package autoallow

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitalvas/claudecode-filter/internal/hook"
)

func TestHandleBash(t *testing.T) {
	h := hook.BuildChain(New())

	t.Run("allows go test", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "go test ./..."})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PermissionRequestOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionAllow, output.HookSpecificOutput.Decision.Behavior)
	})

	t.Run("allows go build", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "go build ./..."})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PermissionRequestOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionAllow, output.HookSpecificOutput.Decision.Behavior)
	})

	t.Run("allows container command", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "container images"})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PermissionRequestOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionAllow, output.HookSpecificOutput.Decision.Behavior)
	})

	t.Run("allows yake tests", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "yake tests"})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PermissionRequestOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionAllow, output.HookSpecificOutput.Decision.Behavior)
	})

	t.Run("allows golangci-lint run", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "golangci-lint run ./..."})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PermissionRequestOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionAllow, output.HookSpecificOutput.Decision.Behavior)
	})

	t.Run("allows gh api", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "gh api repos/owner/repo/pulls"})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PermissionRequestOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionAllow, output.HookSpecificOutput.Decision.Behavior)
	})

	t.Run("allows gh repo view", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "gh repo view owner/repo"})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PermissionRequestOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionAllow, output.HookSpecificOutput.Decision.Behavior)
	})

	t.Run("allows mkdir in project", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "mkdir -p internal/newpkg"})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			CWD:           "/project",
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PermissionRequestOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionAllow, output.HookSpecificOutput.Decision.Behavior)
	})

	t.Run("blocks mkdir outside project", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "mkdir /tmp/evil"})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			CWD:           "/project",
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		assert.Nil(t, result)
	})

	t.Run("blocks mkdir with parent traversal", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "mkdir ../outside"})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			CWD:           "/project",
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		assert.Nil(t, result)
	})

	t.Run("denies ssh on PermissionRequest", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "ssh user@host"})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PreToolUseOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionDeny, output.HookSpecificOutput.PermissionDecision)
		assert.Contains(t, output.HookSpecificOutput.PermissionDecisionReason, "ssh")
	})

	t.Run("denies ssh on PreToolUse", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "ssh user@host"})
		result := h(hook.Input{
			HookEventName: hook.EventPreToolUse,
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PreToolUseOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionDeny, output.HookSpecificOutput.PermissionDecision)
	})

	t.Run("denies telnet", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "telnet host 80"})
		result := h(hook.Input{
			HookEventName: hook.EventPreToolUse,
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PreToolUseOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionDeny, output.HookSpecificOutput.PermissionDecision)
		assert.Contains(t, output.HookSpecificOutput.PermissionDecisionReason, "telnet")
	})

	t.Run("allows command with leading env vars", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{
			Command: "PATH=\"$HOME/.rustup/toolchains/stable-aarch64-apple-darwin/bin:$PATH\" cargo build -p network-manager",
		})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PermissionRequestOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionAllow, output.HookSpecificOutput.Decision.Behavior)
	})

	t.Run("does not allow unknown command", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "rm -rf /"})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		assert.Nil(t, result)
	})

	t.Run("denies cat with abs path outside project", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{
			Command: "cat /other/project/.yake.yaml",
		})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			CWD:           "/my/project",
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PermissionRequestOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionDeny, output.HookSpecificOutput.Decision.Behavior)
	})

	t.Run("denies cat with tilde path outside project", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{
			Command: "cat ~/workspace/go/src/github.com/vitalvas/gandalf/.yake.yaml",
		})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			CWD:           "/my/project",
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PermissionRequestOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionDeny, output.HookSpecificOutput.Decision.Behavior)
	})

	t.Run("allows cat with path inside project", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{
			Command: "cat /my/project/internal/main.go",
		})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			CWD:           "/my/project",
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PermissionRequestOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionAllow, output.HookSpecificOutput.Decision.Behavior)
	})

	t.Run("allows cat with relative path", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{
			Command: "cat internal/main.go",
		})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			CWD:           "/my/project",
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PermissionRequestOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionAllow, output.HookSpecificOutput.Decision.Behavior)
	})

	t.Run("allows git commit with path in message", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{
			Command: `git commit -m "feat: add /oauth2/v1/groups endpoint"`,
		})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			CWD:           "/my/project",
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		require.NotNil(t, result)

		var output hook.PermissionRequestOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionAllow, output.HookSpecificOutput.Decision.Behavior)
	})

	t.Run("does not allow partial prefix match", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "go testing"})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		assert.Nil(t, result)
	})
}
