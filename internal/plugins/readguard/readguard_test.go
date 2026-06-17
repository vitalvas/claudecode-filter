package readguard

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

				if tt.input.HookEventName == hook.EventPermissionRequest {
					var output hook.PermissionRequestOutputWrapper
					require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
					assert.Equal(t, hook.PermissionDeny, output.HookSpecificOutput.Decision.Behavior)
				} else {
					var output hook.PreToolUseOutputWrapper
					require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
					assert.Equal(t, hook.PermissionDeny, output.HookSpecificOutput.PermissionDecision)
				}
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
		result := handleRead(input, dirs, nil)

		require.NotNil(t, result)

		var output hook.PreToolUseOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionDeny, output.HookSpecificOutput.PermissionDecision)
		assert.Contains(t, output.HookSpecificOutput.PermissionDecisionReason, "/blocked/dir")
	})

	t.Run("passes file in project dir", func(t *testing.T) {
		dirs := []blockedDir{{path: "/blocked/dir"}}
		input := makeInputWithCWD("Read", hook.EventPreToolUse, hook.ReadToolInput{
			FilePath: "/project/src/file.go",
		}, "/project")
		result := handleRead(input, dirs, nil)

		assert.Nil(t, result)
	})

	t.Run("denies $GOPATH/src outside project", func(t *testing.T) {
		dirs := []blockedDir{{path: "/gopath/src", allowProject: true}}
		input := makeInputWithCWD("Read", hook.EventPreToolUse, hook.ReadToolInput{
			FilePath: "/gopath/src/github.com/other/repo/main.go",
		}, "/gopath/src/github.com/myorg/myproject")
		result := handleRead(input, dirs, nil)

		require.NotNil(t, result)

		var output hook.PreToolUseOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionDeny, output.HookSpecificOutput.PermissionDecision)
	})

	t.Run("asks $GOPATH/src outside project with marker", func(t *testing.T) {
		cwd := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(cwd, ".git"), 0o755))
		require.NoError(t, marker.Create(cwd, "allow-extread", "1"))

		dirs := []blockedDir{{path: "/gopath/src", allowProject: true}}
		input := makeInputWithCWD("Read", hook.EventPreToolUse, hook.ReadToolInput{
			FilePath: "/gopath/src/github.com/other/repo/main.go",
		}, cwd)
		result := handleRead(input, dirs, nil)

		require.NotNil(t, result)

		var output hook.PreToolUseOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionAsk, output.HookSpecificOutput.PermissionDecision)
	})

	t.Run("allows $GOPATH/src inside project", func(t *testing.T) {
		dirs := []blockedDir{{path: "/gopath/src", allowProject: true}}
		input := makeInputWithCWD("Read", hook.EventPreToolUse, hook.ReadToolInput{
			FilePath: "/gopath/src/github.com/myorg/myproject/internal/pkg/file.go",
		}, "/gopath/src/github.com/myorg/myproject")
		result := handleRead(input, dirs, nil)

		assert.Nil(t, result)
	})

	t.Run("denies read outside project", func(t *testing.T) {
		var dirs []blockedDir
		input := makeInputWithCWD("Read", hook.EventPreToolUse, hook.ReadToolInput{
			FilePath: "/other/project/file.go",
		}, "/my/project")
		result := handleRead(input, dirs, nil)

		require.NotNil(t, result)

		var output hook.PreToolUseOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionDeny, output.HookSpecificOutput.PermissionDecision)
	})

	t.Run("asks read outside project with marker", func(t *testing.T) {
		cwd := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(cwd, ".git"), 0o755))
		require.NoError(t, marker.Create(cwd, "allow-extread", "1"))

		var dirs []blockedDir
		input := makeInputWithCWD("Read", hook.EventPreToolUse, hook.ReadToolInput{
			FilePath: "/other/project/file.go",
		}, cwd)
		result := handleRead(input, dirs, nil)

		require.NotNil(t, result)

		var output hook.PreToolUseOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionAsk, output.HookSpecificOutput.PermissionDecision)
	})

	t.Run("allows read inside project", func(t *testing.T) {
		var dirs []blockedDir
		input := makeInputWithCWD("Read", hook.EventPreToolUse, hook.ReadToolInput{
			FilePath: "/my/project/src/main.go",
		}, "/my/project")
		result := handleRead(input, dirs, nil)

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

func TestHandleBashCd(t *testing.T) {
	h := hook.BuildChain(New())
	project := t.TempDir()
	outside := t.TempDir()

	t.Run("denies cd outside project", func(t *testing.T) {
		input := makeInputWithCWD("Bash", hook.EventPreToolUse,
			hook.BashToolInput{Command: fmt.Sprintf("cd %s && git diff", outside)}, project)

		result := h(input)
		require.NotNil(t, result)

		var output hook.PreToolUseOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionDeny, output.HookSpecificOutput.PermissionDecision)
		assert.Contains(t, output.HookSpecificOutput.PermissionDecisionReason, "outside project")
	})

	t.Run("allows cd inside project", func(t *testing.T) {
		input := makeInputWithCWD("Bash", hook.EventPreToolUse,
			hook.BashToolInput{Command: fmt.Sprintf("cd %s && go test ./...", filepath.Join(project, "internal"))}, project)

		assert.Nil(t, h(input))
	})

	t.Run("allows command without cd", func(t *testing.T) {
		input := makeInputWithCWD("Bash", hook.EventPreToolUse,
			hook.BashToolInput{Command: "go test ./..."}, project)

		assert.Nil(t, h(input))
	})

	t.Run("allows relative cd", func(t *testing.T) {
		input := makeInputWithCWD("Bash", hook.EventPreToolUse,
			hook.BashToolInput{Command: "cd internal && go test ./..."}, project)

		assert.Nil(t, h(input))
	})

	t.Run("asks cd outside project with marker", func(t *testing.T) {
		markerRoot := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(markerRoot, ".git"), 0o755))
		require.NoError(t, marker.Create(markerRoot, "allow-extread", "1"))

		input := makeInputWithCWD("Bash", hook.EventPreToolUse,
			hook.BashToolInput{Command: fmt.Sprintf("cd %s && git diff", outside)}, markerRoot)

		result := h(input)
		require.NotNil(t, result)

		var output hook.PreToolUseOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionAsk, output.HookSpecificOutput.PermissionDecision)
	})

	t.Run("denies on PermissionRequest event", func(t *testing.T) {
		input := makeInputWithCWD("Bash", hook.EventPermissionRequest,
			hook.BashToolInput{Command: fmt.Sprintf("cd %s && git diff", outside)}, project)

		result := h(input)
		require.NotNil(t, result)

		var output hook.PermissionRequestOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionDeny, output.HookSpecificOutput.Decision.Behavior)
	})

	t.Run("passes when CWD empty", func(t *testing.T) {
		input := makeInput("Bash", hook.EventPreToolUse,
			hook.BashToolInput{Command: fmt.Sprintf("cd %s && git diff", outside)})

		assert.Nil(t, h(input))
	})

	t.Run("denies git -C outside project", func(t *testing.T) {
		input := makeInputWithCWD("Bash", hook.EventPreToolUse,
			hook.BashToolInput{Command: fmt.Sprintf("git -C %s commit -m x", outside)}, project)

		result := h(input)
		require.NotNil(t, result)

		var output hook.PreToolUseOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionDeny, output.HookSpecificOutput.PermissionDecision)
		assert.Contains(t, output.HookSpecificOutput.PermissionDecisionReason, "outside project")
	})

	t.Run("allows git -C project root from subdir", func(t *testing.T) {
		repoRoot := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(repoRoot, ".git"), 0o755))
		subdir := filepath.Join(repoRoot, "internal")
		require.NoError(t, os.Mkdir(subdir, 0o755))

		input := makeInputWithCWD("Bash", hook.EventPreToolUse,
			hook.BashToolInput{Command: fmt.Sprintf("git -C %s status", repoRoot)}, subdir)

		assert.Nil(t, h(input))
	})

	t.Run("allows git -C project subdir", func(t *testing.T) {
		repoRoot := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(repoRoot, ".git"), 0o755))
		subdir := filepath.Join(repoRoot, "internal")
		require.NoError(t, os.Mkdir(subdir, 0o755))

		input := makeInputWithCWD("Bash", hook.EventPreToolUse,
			hook.BashToolInput{Command: fmt.Sprintf("git -C %s status", subdir)}, repoRoot)

		assert.Nil(t, h(input))
	})
}

func TestExternalDirTargets(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    []string
	}{
		{name: "no cd", command: "go test ./...", want: nil},
		{name: "single cd", command: "cd /tmp/foo && ls", want: []string{"/tmp/foo"}},
		{name: "cd with semicolon", command: "cd /tmp/foo; ls", want: []string{"/tmp/foo"}},
		{name: "quoted target", command: `cd "/tmp/foo bar" && ls`, want: []string{"/tmp/foo"}},
		{name: "multiple cd", command: "cd /a && echo x | cd /b", want: []string{"/a", "/b"}},
		{name: "cd without arg", command: "cd && ls", want: nil},
		{name: "git -C path", command: "git -C /tmp/repo status", want: []string{"/tmp/repo"}},
		{name: "git -C with commit", command: "git -C /tmp/repo commit -m x", want: []string{"/tmp/repo"}},
		{name: "git --git-dir flag", command: "git --git-dir=/tmp/repo/.git status", want: []string{"/tmp/repo/.git"}},
		{name: "git --work-tree flag", command: "git --work-tree=/tmp/repo status", want: []string{"/tmp/repo"}},
		{name: "git -C only on git", command: "make -C /tmp/repo build", want: nil},
		{name: "git -C without arg", command: "git -C", want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, externalDirTargets(tt.command))
		})
	}
}

func TestExpandHome(t *testing.T) {
	home := os.Getenv("HOME")

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "tilde path", path: "~/workspace/project/file.go", want: filepath.Join(home, "workspace/project/file.go")},
		{name: "tilde only slash", path: "~/", want: home},
		{name: "absolute path unchanged", path: "/usr/local/bin/tool", want: "/usr/local/bin/tool"},
		{name: "relative path unchanged", path: "relative/path/file.go", want: "relative/path/file.go"},
		{name: "tilde without slash", path: "~user/file.go", want: "~user/file.go"},
		{name: "empty string", path: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, expandHome(tt.path))
		})
	}
}
