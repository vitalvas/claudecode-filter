package autoallow

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/vitalvas/claudecode-filter/internal/hook"
)

var (
	goModCache     string
	goModCacheOnce sync.Once
)

func getGoModCache() string {
	goModCacheOnce.Do(func() {
		out, err := exec.Command("go", "env", "GOMODCACHE").Output()
		if err != nil {
			return
		}

		goModCache = strings.TrimSpace(string(out))
	})

	return goModCache
}

type readToolInput struct {
	FilePath string `json:"file_path"`
}

func (p *Plugin) handleRead(input hook.Input) *hook.Result {
	var readInput readToolInput
	if err := json.Unmarshal(input.ToolInput, &readInput); err != nil {
		return nil
	}

	modCache := getGoModCache()
	if modCache == "" {
		return nil
	}

	if strings.HasPrefix(readInput.FilePath, fmt.Sprintf("%s/", modCache)) {
		return allowPermissionRequest()
	}

	return nil
}
