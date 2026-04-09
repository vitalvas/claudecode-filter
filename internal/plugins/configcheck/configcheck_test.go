package configcheck

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitalvas/claudecode-filter/internal/hook"
)

func TestValidateJSONFile(t *testing.T) {
	t.Run("valid JSON", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "valid.json")
		require.NoError(t, os.WriteFile(path, []byte(`{"key": "value"}`), 0o644))

		assert.NoError(t, validateJSONFile(path))
	})

	t.Run("invalid JSON", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "invalid.json")
		require.NoError(t, os.WriteFile(path, []byte(`{"key": {`), 0o644))

		assert.Error(t, validateJSONFile(path))
	})

	t.Run("file does not exist", func(t *testing.T) {
		assert.NoError(t, validateJSONFile("/nonexistent/path.json"))
	})

	t.Run("empty file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "empty.json")
		require.NoError(t, os.WriteFile(path, []byte{}, 0o644))

		assert.Error(t, validateJSONFile(path))
	})
}

func TestValidateConfigs(t *testing.T) {
	t.Run("all valid", func(t *testing.T) {
		dir := t.TempDir()
		claudeDir := filepath.Join(dir, ".claude")
		require.NoError(t, os.MkdirAll(claudeDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(`{}`), 0o644))

		result := validateConfigs(nil, dir)
		assert.Nil(t, result)
	})

	t.Run("invalid project config", func(t *testing.T) {
		dir := t.TempDir()
		claudeDir := filepath.Join(dir, ".claude")
		require.NoError(t, os.MkdirAll(claudeDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(`{invalid`), 0o644))

		result := validateConfigs(nil, dir)
		require.NotNil(t, result)
		assert.Contains(t, result.Stderr, "invalid JSON")
		assert.Contains(t, result.Stderr, "settings.json")
		assert.Equal(t, 0, result.ExitCode)
	})

	t.Run("invalid user config", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "settings.json")
		require.NoError(t, os.WriteFile(path, []byte(`{bad`), 0o644))

		result := validateConfigs([]string{path}, "")
		require.NotNil(t, result)
		assert.Contains(t, result.Stderr, "invalid JSON")
	})

	t.Run("missing files are fine", func(t *testing.T) {
		result := validateConfigs([]string{"/nonexistent/settings.json"}, "/nonexistent/project")
		assert.Nil(t, result)
	})
}

func TestConfigcheckMiddleware(t *testing.T) {
	h := hook.BuildChain(New())

	t.Run("passes on valid config", func(t *testing.T) {
		result := h(hook.Input{
			HookEventName: hook.EventUserPromptSubmit,
			CWD:           t.TempDir(),
			Prompt:        "hello",
		})

		assert.Nil(t, result)
	})

	t.Run("warns on invalid project config", func(t *testing.T) {
		dir := t.TempDir()
		claudeDir := filepath.Join(dir, ".claude")
		require.NoError(t, os.MkdirAll(claudeDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(`{broken`), 0o644))

		result := h(hook.Input{
			HookEventName: hook.EventUserPromptSubmit,
			CWD:           dir,
			Prompt:        "hello",
		})

		require.NotNil(t, result)
		assert.Contains(t, result.Stderr, "invalid JSON")
	})

	t.Run("ignores non-UserPromptSubmit events", func(t *testing.T) {
		result := h(hook.Input{
			HookEventName: hook.EventPreToolUse,
			ToolName:      "Bash",
		})

		assert.Nil(t, result)
	})
}
