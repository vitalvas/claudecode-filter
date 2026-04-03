package marker

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testMarker = "test-marker"

func setupGitRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0o755))

	return dir
}

func TestCreate(t *testing.T) {
	t.Run("creates marker file with value", func(t *testing.T) {
		gitRoot := setupGitRepo(t)
		require.NoError(t, Create(gitRoot, testMarker, "hello"))

		path := filepath.Join(gitRoot, ".tmp", prefix+testMarker)
		assert.FileExists(t, path)

		data, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, "hello", string(data))
	})

	t.Run("resolves git root from subdirectory", func(t *testing.T) {
		gitRoot := setupGitRepo(t)
		subDir := filepath.Join(gitRoot, "sub", "dir")
		require.NoError(t, os.MkdirAll(subDir, 0o755))

		require.NoError(t, Create(subDir, testMarker, "val"))

		path := filepath.Join(gitRoot, ".tmp", prefix+testMarker)
		assert.FileExists(t, path)
	})

	t.Run("error when no git root", func(t *testing.T) {
		assert.Error(t, Create(os.TempDir(), testMarker, "val"))
	})
}

func TestConsume(t *testing.T) {
	t.Run("returns value and true", func(t *testing.T) {
		gitRoot := setupGitRepo(t)
		require.NoError(t, Create(gitRoot, testMarker, "myvalue"))

		val, ok := Consume(gitRoot, testMarker)
		assert.True(t, ok)
		assert.Equal(t, "myvalue", val)

		path := filepath.Join(gitRoot, ".tmp", prefix+testMarker)
		assert.NoFileExists(t, path)
	})

	t.Run("returns empty and false for nonexistent", func(t *testing.T) {
		gitRoot := setupGitRepo(t)

		val, ok := Consume(gitRoot, testMarker)
		assert.False(t, ok)
		assert.Empty(t, val)
	})

	t.Run("returns empty and false when no git root", func(t *testing.T) {
		val, ok := Consume(os.TempDir(), testMarker)
		assert.False(t, ok)
		assert.Empty(t, val)
	})
}

func TestCleanup(t *testing.T) {
	t.Run("removes all prefixed files", func(t *testing.T) {
		gitRoot := setupGitRepo(t)
		require.NoError(t, Create(gitRoot, "one", "1"))
		require.NoError(t, Create(gitRoot, "two", "2"))

		// Create a non-prefixed file that should survive
		tmpPath := filepath.Join(gitRoot, ".tmp")
		require.NoError(t, os.WriteFile(filepath.Join(tmpPath, "other-file"), []byte("keep"), 0o644))

		Cleanup(gitRoot)

		assert.NoFileExists(t, filepath.Join(tmpPath, fmt.Sprintf("%s%s", prefix, "one")))
		assert.NoFileExists(t, filepath.Join(tmpPath, fmt.Sprintf("%s%s", prefix, "two")))
		assert.FileExists(t, filepath.Join(tmpPath, "other-file"))
	})

	t.Run("no error when no git root", func(_ *testing.T) {
		Cleanup(os.TempDir())
	})
}
