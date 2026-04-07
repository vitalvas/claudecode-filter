package readguard

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitalvas/claudecode-filter/internal/hook"
)

func TestReadguard(t *testing.T) {
	h := hook.BuildChain(New())

	tests := []struct {
		name    string
		input   hook.Input
		blocked bool
	}{
		{
			name:    "blocks *.key file",
			input:   makeInput("Read", hook.EventPreToolUse, hook.ReadToolInput{FilePath: "/home/user/secret.key"}),
			blocked: true,
		},
		{
			name:    "blocks *.key file in subdir",
			input:   makeInput("Read", hook.EventPreToolUse, hook.ReadToolInput{FilePath: "/some/path/id_rsa.key"}),
			blocked: true,
		},
		{
			name:    "passes non-key file",
			input:   makeInput("Read", hook.EventPreToolUse, hook.ReadToolInput{FilePath: "/home/user/config.yaml"}),
			blocked: false,
		},
		{
			name:    "passes file with key in name but wrong extension",
			input:   makeInput("Read", hook.EventPreToolUse, hook.ReadToolInput{FilePath: "/home/user/keyfile.txt"}),
			blocked: false,
		},
		{
			name:    "blocks *.key file on PermissionRequest",
			input:   makeInput("Read", hook.EventPermissionRequest, hook.ReadToolInput{FilePath: "/home/user/secret.key"}),
			blocked: true,
		},
		{
			name:    "passes non-Read event",
			input:   makeInput("Read", hook.EventUserPromptSubmit, hook.ReadToolInput{FilePath: "/home/user/secret.key"}),
			blocked: false,
		},
		{
			name:    "passes non-Read tool",
			input:   makeInput("Bash", hook.EventPreToolUse, hook.ReadToolInput{FilePath: "/home/user/secret.key"}),
			blocked: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h(tt.input)

			if tt.blocked {
				require.NotNil(t, result)

				var output hook.PreToolUseOutputWrapper
				require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
				assert.Equal(t, hook.PermissionDeny, output.HookSpecificOutput.PermissionDecision)
			} else {
				assert.Nil(t, result)
			}
		})
	}
}

func makeInput(toolName, event string, toolInput any) hook.Input {
	data, _ := json.Marshal(toolInput)

	return hook.Input{
		HookEventName: event,
		ToolName:      toolName,
		ToolInput:     data,
	}
}
