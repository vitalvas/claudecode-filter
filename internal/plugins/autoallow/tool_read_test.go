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

	t.Run("allows read under CWD", func(t *testing.T) {
		cwd := t.TempDir()
		toolInput, _ := json.Marshal(readToolInput{
			FilePath: filepath.Join(cwd, "subdir", "file.go"),
		})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			ToolName:      "Read",
			ToolInput:     toolInput,
			CWD:           cwd,
		})

		require.NotNil(t, result)

		var output hook.PermissionRequestOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionAllow, output.HookSpecificOutput.Decision.Behavior)
	})

	t.Run("allows read with tilde path under CWD", func(t *testing.T) {
		home := os.Getenv("HOME")
		if home == "" {
			t.Skip("HOME not set")
		}

		cwd := filepath.Join(home, "workspace", "testproject")
		toolInput, _ := json.Marshal(readToolInput{
			FilePath: "~/workspace/testproject/file.go",
		})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			ToolName:      "Read",
			ToolInput:     toolInput,
			CWD:           cwd,
		})

		require.NotNil(t, result)

		var output hook.PermissionRequestOutputWrapper
		require.NoError(t, json.Unmarshal([]byte(result.Stdout), &output))
		assert.Equal(t, hook.PermissionAllow, output.HookSpecificOutput.Decision.Behavior)
	})

	t.Run("does not allow read outside CWD without allowed dir", func(t *testing.T) {
		toolInput, _ := json.Marshal(readToolInput{
			FilePath: "/some/other/path/file.go",
		})
		result := h(hook.Input{
			HookEventName: hook.EventPermissionRequest,
			ToolName:      "Read",
			ToolInput:     toolInput,
			CWD:           "/my/project",
		})

		assert.Nil(t, result)
	})
}

func TestExpandHome(t *testing.T) {
	home := os.Getenv("HOME")

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "tilde path", path: "~/workspace/file.go", want: filepath.Join(home, "workspace/file.go")},
		{name: "absolute unchanged", path: "/usr/local/bin", want: "/usr/local/bin"},
		{name: "relative unchanged", path: "relative/path", want: "relative/path"},
		{name: "tilde without slash", path: "~user/file", want: "~user/file"},
		{name: "empty", path: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, expandHome(tt.path))
		})
	}
}

func TestIsUnderDir(t *testing.T) {
	tests := []struct {
		name string
		path string
		dir  string
		want bool
	}{
		{name: "under dir", path: "/a/b/c/file.go", dir: "/a/b", want: true},
		{name: "same dir", path: "/a/b", dir: "/a/b", want: true},
		{name: "outside dir", path: "/x/y/file.go", dir: "/a/b", want: false},
		{name: "parent traversal", path: "/a/file.go", dir: "/a/b", want: false},
		{name: "dotfile under dir", path: "/a/b/.tmp/file.go", dir: "/a/b", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isUnderDir(tt.path, tt.dir))
		})
	}
}
