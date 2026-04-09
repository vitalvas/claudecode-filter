package readguard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitalvas/claudecode-filter/internal/hook"
)

func TestReadguard(t *testing.T) {
	h := hook.BuildChain(New())
	homeDir := os.Getenv("HOME")

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
		// SSH directory
		{
			name:    "blocks file under $HOME/.ssh",
			input:   makeInput("Read", hook.EventPreToolUse, hook.ReadToolInput{FilePath: filepath.Join(homeDir, ".ssh", "config")}),
			blocked: true,
		},
		{
			name:    "blocks nested file under $HOME/.ssh",
			input:   makeInput("Read", hook.EventPreToolUse, hook.ReadToolInput{FilePath: filepath.Join(homeDir, ".ssh", "keys", "deploy")}),
			blocked: true,
		},
		{
			name:    "passes .ssh outside HOME",
			input:   makeInput("Read", hook.EventPreToolUse, hook.ReadToolInput{FilePath: "/tmp/.ssh/config"}),
			blocked: false,
		},
		// .env files
		{
			name:    "blocks .env",
			input:   makeInput("Read", hook.EventPreToolUse, hook.ReadToolInput{FilePath: "/project/.env"}),
			blocked: true,
		},
		{
			name:    "blocks .env.local",
			input:   makeInput("Read", hook.EventPreToolUse, hook.ReadToolInput{FilePath: "/project/.env.local"}),
			blocked: true,
		},
		{
			name:    "blocks .env.production",
			input:   makeInput("Read", hook.EventPreToolUse, hook.ReadToolInput{FilePath: "/project/.env.production"}),
			blocked: true,
		},
		{
			name:    "passes env.example",
			input:   makeInput("Read", hook.EventPreToolUse, hook.ReadToolInput{FilePath: "/project/env.example"}),
			blocked: false,
		},
		// Private key files
		{
			name:    "blocks id_rsa",
			input:   makeInput("Read", hook.EventPreToolUse, hook.ReadToolInput{FilePath: "/some/path/id_rsa"}),
			blocked: true,
		},
		{
			name:    "blocks id_rsa_custom",
			input:   makeInput("Read", hook.EventPreToolUse, hook.ReadToolInput{FilePath: "/some/path/id_rsa_custom"}),
			blocked: true,
		},
		{
			name:    "blocks id_ecdsa",
			input:   makeInput("Read", hook.EventPreToolUse, hook.ReadToolInput{FilePath: "/some/path/id_ecdsa"}),
			blocked: true,
		},
		{
			name:    "blocks id_ecdsa_sk",
			input:   makeInput("Read", hook.EventPreToolUse, hook.ReadToolInput{FilePath: "/some/path/id_ecdsa_sk"}),
			blocked: true,
		},
		{
			name:    "blocks id_ed25519",
			input:   makeInput("Read", hook.EventPreToolUse, hook.ReadToolInput{FilePath: "/some/path/id_ed25519"}),
			blocked: true,
		},
		{
			name:    "blocks id_ed25519_custom",
			input:   makeInput("Read", hook.EventPreToolUse, hook.ReadToolInput{FilePath: "/some/path/id_ed25519_custom"}),
			blocked: true,
		},
		// Public key exceptions
		{
			name:    "allows id_rsa.pub",
			input:   makeInput("Read", hook.EventPreToolUse, hook.ReadToolInput{FilePath: "/some/path/id_rsa.pub"}),
			blocked: false,
		},
		{
			name:    "allows id_ecdsa.pub",
			input:   makeInput("Read", hook.EventPreToolUse, hook.ReadToolInput{FilePath: "/some/path/id_ecdsa.pub"}),
			blocked: false,
		},
		{
			name:    "allows id_ed25519.pub",
			input:   makeInput("Read", hook.EventPreToolUse, hook.ReadToolInput{FilePath: "/some/path/id_ed25519.pub"}),
			blocked: false,
		},
		{
			name:    "allows id_rsa.pub under $HOME/.ssh",
			input:   makeInput("Read", hook.EventPreToolUse, hook.ReadToolInput{FilePath: filepath.Join(homeDir, ".ssh", "id_rsa.pub")}),
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

func blockedDirPaths(dirs []blockedDir) []string {
	paths := make([]string, 0, len(dirs))
	for _, d := range dirs {
		paths = append(paths, d.path)
	}

	return paths
}

func TestBlockedDirectories(t *testing.T) {
	home := os.Getenv("HOME")

	t.Run("includes $HOME/.ssh always", func(t *testing.T) {
		paths := blockedDirPaths(blockedDirectories())
		assert.Contains(t, paths, filepath.Join(home, ".ssh"))
	})

	t.Run("includes $HOME/go when GOPATH differs", func(t *testing.T) {
		goPath := os.Getenv("GOPATH")
		defaultGoPath := filepath.Join(home, "go")

		if goPath != "" && goPath != defaultGoPath {
			paths := blockedDirPaths(blockedDirectories())
			assert.Contains(t, paths, defaultGoPath)
		}
	})

	t.Run("includes $GOPATH/src when GOPATH set", func(t *testing.T) {
		goPath := os.Getenv("GOPATH")
		if goPath == "" {
			t.Skip("GOPATH not set")
		}

		dirs := blockedDirectories()

		var found bool
		for _, d := range dirs {
			if d.path == filepath.Join(goPath, "src") {
				found = true
				assert.True(t, d.allowProject)
			}
		}

		assert.True(t, found)
	})
}

func TestHandleReadBlockedDirs(t *testing.T) {
	t.Run("blocks file under blocked dir", func(t *testing.T) {
		dirs := []blockedDir{{path: "/blocked/dir"}}
		input := makeInput("Read", hook.EventPreToolUse, hook.ReadToolInput{FilePath: "/blocked/dir/some/file.go"})
		result := handleRead(input, dirs)

		require.NotNil(t, result)

		var output hook.PreToolUseOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionDeny, output.HookSpecificOutput.PermissionDecision)
		assert.Contains(t, output.HookSpecificOutput.PermissionDecisionReason, "/blocked/dir")
	})

	t.Run("passes file outside blocked dir", func(t *testing.T) {
		dirs := []blockedDir{{path: "/blocked/dir"}}
		input := makeInput("Read", hook.EventPreToolUse, hook.ReadToolInput{FilePath: "/other/dir/file.go"})
		result := handleRead(input, dirs)

		assert.Nil(t, result)
	})

	t.Run("blocks $GOPATH/src outside project", func(t *testing.T) {
		dirs := []blockedDir{{path: "/gopath/src", allowProject: true}}
		input := makeInputWithCWD("Read", hook.EventPreToolUse, hook.ReadToolInput{
			FilePath: "/gopath/src/github.com/other/repo/main.go",
		}, "/gopath/src/github.com/myorg/myproject")
		result := handleRead(input, dirs)

		require.NotNil(t, result)

		var output hook.PreToolUseOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionDeny, output.HookSpecificOutput.PermissionDecision)
	})

	t.Run("allows $GOPATH/src inside project", func(t *testing.T) {
		dirs := []blockedDir{{path: "/gopath/src", allowProject: true}}
		input := makeInputWithCWD("Read", hook.EventPreToolUse, hook.ReadToolInput{
			FilePath: "/gopath/src/github.com/myorg/myproject/internal/pkg/file.go",
		}, "/gopath/src/github.com/myorg/myproject")
		result := handleRead(input, dirs)

		assert.Nil(t, result)
	})
}

func makeInput(toolName, event string, toolInput any) hook.Input {
	data, _ := json.Marshal(toolInput)

	return hook.Input{
		HookEventName: event,
		ToolName:      toolName,
		ToolInput:     data,
	}
}

func makeInputWithCWD(toolName, event string, toolInput any, cwd string) hook.Input {
	input := makeInput(toolName, event, toolInput)
	input.CWD = cwd

	return input
}
