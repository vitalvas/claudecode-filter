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
			allowedReadDirs = append(allowedReadDirs, filepath.Join(home, ".cargo", "registry", "src"))
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
}

func handleRead(input hook.Input) *hook.Result {
	var readInput readToolInput
	if err := json.Unmarshal(input.ToolInput, &readInput); err != nil {
		return nil
	}

	for _, dir := range getAllowedReadDirs() {
		if strings.HasPrefix(readInput.FilePath, fmt.Sprintf("%s/", dir)) {
			return allowPermissionRequest()
		}
	}

	return nil
}
