package autoallow

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/vitalvas/claudecode-filter/internal/hook"
)

var (
	allowedReadDirs     []string
	allowedReadDirsOnce sync.Once
)

func getAllowedReadDirs() []string {
	allowedReadDirsOnce.Do(func() {
		home := os.Getenv("HOME")
		if home != "" {
			allowedReadDirs = append(allowedReadDirs,
				filepath.Join(home, ".cargo", "registry", "src"),
				filepath.Join(home, ".rustup"),
			)
		}

		out, err := exec.Command("go", "env", "GOMODCACHE").Output()
		if err == nil {
			if dir := strings.TrimSpace(string(out)); dir != "" {
				allowedReadDirs = append(allowedReadDirs, dir)
			}
		}
	})

	return allowedReadDirs
}

type readToolInput struct {
	FilePath string `json:"file_path"`
	Path     string `json:"path"`
}

func handleRead(input hook.Input) *hook.Result {
	var readInput readToolInput
	if err := json.Unmarshal(input.ToolInput, &readInput); err != nil {
		return nil
	}

	target := readInput.FilePath
	if target == "" {
		target = readInput.Path
	}

	if target == "" {
		return nil
	}

	target = expandHome(target)

	if input.CWD != "" && isUnderDir(target, input.CWD) {
		return allowPermissionRequest()
	}

	for _, dir := range getAllowedReadDirs() {
		if strings.HasPrefix(target, fmt.Sprintf("%s/", dir)) {
			return allowPermissionRequest()
		}
	}

	return nil
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home := os.Getenv("HOME")
		if home != "" {
			return filepath.Join(home, path[2:])
		}
	}

	return path
}

func isUnderDir(filePath, dir string) bool {
	rel, err := filepath.Rel(dir, filePath)
	if err != nil {
		return false
	}

	return len(rel) > 0 && !strings.HasPrefix(rel, "..")
}
