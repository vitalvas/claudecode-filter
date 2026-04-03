package filter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitalvas/claudecode-filter/internal/hook"
)

func TestDebugLog(t *testing.T) {
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() { os.Chdir(origDir) })

	require.NoError(t, os.MkdirAll(".tmp", 0o755))

	t.Run("writes input and result", func(t *testing.T) {
		debugLog([]byte(`{"test":"data"}`), hook.Result{Stdout: "out", Stderr: "err", ExitCode: 1})

		data, err := os.ReadFile(filepath.Join(".tmp", "debug.log"))
		require.NoError(t, err)
		assert.Contains(t, string(data), `INPUT: {"test":"data"}`)
		assert.Contains(t, string(data), "OUTPUT: stdout=out stderr=err exit=1")
	})
}
