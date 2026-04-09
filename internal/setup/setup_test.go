package setup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadConfig(t *testing.T) {
	t.Run("reads existing file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "settings.json")
		require.NoError(t, os.WriteFile(path, []byte(`{"key": "value"}`), 0o644))

		config, err := readConfig(path)
		require.NoError(t, err)
		assert.Equal(t, "value", config["key"])
	})

	t.Run("returns empty map for missing file", func(t *testing.T) {
		config, err := readConfig("/nonexistent/settings.json")
		require.NoError(t, err)
		assert.Empty(t, config)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "settings.json")
		require.NoError(t, os.WriteFile(path, []byte(`{broken`), 0o644))

		_, err := readConfig(path)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid JSON")
	})
}

func TestExecuteSetup(t *testing.T) {
	t.Run("fails when HOME is empty", func(t *testing.T) {
		t.Setenv("HOME", "")

		err := Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "HOME")
	})

	t.Run("creates config file", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)

		require.NoError(t, Execute())

		path := filepath.Join(home, ".claude", "settings.json")
		data, err := os.ReadFile(path)
		require.NoError(t, err)

		var result map[string]any
		require.NoError(t, json.Unmarshal(data, &result))

		hooks, ok := result["hooks"].(map[string]any)
		require.True(t, ok)
		assert.Contains(t, hooks, "PreToolUse")
		assert.Contains(t, hooks, "PermissionRequest")
		assert.Contains(t, hooks, "UserPromptSubmit")
		assert.Contains(t, hooks, "SessionEnd")
		assert.Equal(t, false, result["includeCoAuthoredBy"])
	})

	t.Run("preserves existing settings", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)

		claudeDir := filepath.Join(home, ".claude")
		require.NoError(t, os.MkdirAll(claudeDir, 0o755))
		require.NoError(t, os.WriteFile(
			filepath.Join(claudeDir, "settings.json"),
			[]byte(`{"enabledPlugins": {"gopls": true}, "customKey": "customValue"}`),
			0o644,
		))

		require.NoError(t, Execute())

		data, err := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
		require.NoError(t, err)

		var result map[string]any
		require.NoError(t, json.Unmarshal(data, &result))

		assert.Equal(t, "customValue", result["customKey"])
		assert.Contains(t, result, "enabledPlugins")
		assert.Contains(t, result, "hooks")
		assert.Equal(t, false, result["includeCoAuthoredBy"])
	})

	t.Run("overwrites hooks", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)

		claudeDir := filepath.Join(home, ".claude")
		require.NoError(t, os.MkdirAll(claudeDir, 0o755))
		require.NoError(t, os.WriteFile(
			filepath.Join(claudeDir, "settings.json"),
			[]byte(`{"hooks": {"old": "data"}}`),
			0o644,
		))

		require.NoError(t, Execute())

		data, err := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
		require.NoError(t, err)

		var result map[string]any
		require.NoError(t, json.Unmarshal(data, &result))

		hooks, ok := result["hooks"].(map[string]any)
		require.True(t, ok)
		assert.Contains(t, hooks, "PreToolUse")
		assert.NotContains(t, hooks, "old")
	})
}
