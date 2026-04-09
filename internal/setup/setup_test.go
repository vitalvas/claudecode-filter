package setup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

		path := filepath.Join(home, ".claude", "settings.local.json")
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
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)

		claudeDir := filepath.Join(home, ".claude")
		require.NoError(t, os.MkdirAll(claudeDir, 0o755))
		require.NoError(t, os.WriteFile(
			filepath.Join(claudeDir, "settings.local.json"),
			[]byte(`{"oldKey": "oldValue"}`),
			0o644,
		))

		require.NoError(t, Execute())

		data, err := os.ReadFile(filepath.Join(claudeDir, "settings.local.json"))
		require.NoError(t, err)

		var result map[string]any
		require.NoError(t, json.Unmarshal(data, &result))

		assert.NotContains(t, result, "oldKey")
		assert.Contains(t, result, "hooks")
	})
}
