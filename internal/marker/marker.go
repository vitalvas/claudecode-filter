package marker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	Prefix = "claudecode-filter-"
	tmpDir = ".tmp"
)

// Create writes a marker file with the given value.
func Create(cwd, name, value string) error {
	root, err := findGitRoot(cwd)
	if err != nil {
		return err
	}

	dir := filepath.Join(root, tmpDir)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, Prefix+name), []byte(value), 0o644)
}

// Exists checks whether a marker file exists.
func Exists(cwd, name string) bool {
	root, err := findGitRoot(cwd)
	if err != nil {
		return false
	}

	_, err = os.Stat(filepath.Join(root, tmpDir, Prefix+name))

	return err == nil
}

// Consume removes a marker file and returns its value and whether it existed.
func Consume(cwd, name string) (string, bool) {
	root, err := findGitRoot(cwd)
	if err != nil {
		return "", false
	}

	path := filepath.Join(root, tmpDir, Prefix+name)

	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}

	os.Remove(path)

	return string(data), true
}

// Cleanup removes all marker files with the claudecode-filter- prefix.
func Cleanup(cwd string) {
	root, err := findGitRoot(cwd)
	if err != nil {
		return
	}

	dir := filepath.Join(root, tmpDir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), Prefix) {
			os.Remove(filepath.Join(dir, entry.Name()))
		}
	}
}

func findGitRoot(dir string) (string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}

	for {
		gitPath := filepath.Join(dir, ".git")

		info, err := os.Stat(gitPath)
		if err == nil && info.IsDir() {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("git root not found")
		}

		dir = parent
	}
}
