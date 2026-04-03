package hook

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInputUnmarshal(t *testing.T) {
	t.Run("parses PreToolUse input", func(t *testing.T) {
		raw := `{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"git status"},"cwd":"/tmp"}`

		var input Input
		require.NoError(t, json.Unmarshal([]byte(raw), &input))
		assert.Equal(t, "PreToolUse", input.HookEventName)
		assert.Equal(t, "Bash", input.ToolName)
		assert.Equal(t, "/tmp", input.CWD)
	})

	t.Run("parses UserPromptSubmit input", func(t *testing.T) {
		raw := `{"hook_event_name":"UserPromptSubmit","prompt":"ok git commit","cwd":"/tmp"}`

		var input Input
		require.NoError(t, json.Unmarshal([]byte(raw), &input))
		assert.Equal(t, "UserPromptSubmit", input.HookEventName)
		assert.Equal(t, "ok git commit", input.Prompt)
	})
}

func TestBashToolInputUnmarshal(t *testing.T) {
	t.Run("parses command", func(t *testing.T) {
		raw := `{"command":"git commit -m 'test'"}`

		var input BashToolInput
		require.NoError(t, json.Unmarshal([]byte(raw), &input))
		assert.Equal(t, "git commit -m 'test'", input.Command)
	})
}

func TestOutputMarshal(t *testing.T) {
	t.Run("marshals PreToolUse deny output", func(t *testing.T) {
		output := PreToolUseOutputWrapper{
			HookSpecificOutput: PreToolUseOutput{
				HookEventName:            EventPreToolUse,
				PermissionDecision:       PermissionDeny,
				PermissionDecisionReason: "blocked",
			},
		}

		data, err := json.Marshal(output)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"permissionDecision":"deny"`)
		assert.Contains(t, string(data), `"permissionDecisionReason":"blocked"`)
	})

	t.Run("marshals PermissionRequest allow output", func(t *testing.T) {
		output := PermissionRequestOutputWrapper{
			HookSpecificOutput: PermissionRequestOutput{
				HookEventName: EventPermissionRequest,
				Decision: PermissionDecision{
					Behavior: PermissionAllow,
				},
			},
		}

		data, err := json.Marshal(output)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"behavior":"allow"`)
	})
}
