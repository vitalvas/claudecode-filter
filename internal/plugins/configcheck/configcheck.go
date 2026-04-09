package configcheck

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vitalvas/claudecode-filter/internal/hook"
)

// New creates the configcheck middleware.
func New() hook.Middleware {
	settingsFiles := settingsPaths()

	return func(next hook.Handler) hook.Handler {
		return func(input hook.Input) *hook.Result {
			if input.HookEventName == hook.EventUserPromptSubmit {
				if result := validateConfigs(settingsFiles, input.CWD); result != nil {
					return result
				}
			}

			return next(input)
		}
	}
}

func settingsPaths() []string {
	home := os.Getenv("HOME")
	if home == "" {
		return nil
	}

	return []string{
		filepath.Join(home, ".claude", "settings.json"),
		filepath.Join(home, ".claude", "settings.local.json"),
	}
}

func validateConfigs(settingsFiles []string, cwd string) *hook.Result {
	paths := make([]string, 0, len(settingsFiles)+2)
	paths = append(paths, settingsFiles...)

	if cwd != "" {
		paths = append(paths,
			filepath.Join(cwd, ".claude", "settings.json"),
			filepath.Join(cwd, ".claude", "settings.local.json"),
		)
	}

	for _, path := range paths {
		if err := validateJSONFile(path); err != nil {
			return &hook.Result{
				Stderr:   fmt.Sprintf("WARNING: %s has invalid JSON: %v\n", path, err),
				ExitCode: 0,
			}
		}
	}

	return nil
}

func validateJSONFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	if !json.Valid(data) {
		return fmt.Errorf("malformed JSON")
	}

	return nil
}
