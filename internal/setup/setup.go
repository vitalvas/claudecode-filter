package setup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

var requiredSettings = map[string]any{
	"alwaysThinkingEnabled":             false,
	"includeCoAuthoredBy":               false,
	"skipDangerousModePermissionPrompt": true,
	"hooks": map[string]any{
		"PreToolUse": []any{
			map[string]any{
				"matcher": "*",
				"hooks": []any{
					map[string]any{
						"type":    "command",
						"command": "claudecode-filter",
					},
				},
			},
		},
		"PermissionRequest": []any{
			map[string]any{
				"matcher": "*",
				"hooks": []any{
					map[string]any{
						"type":    "command",
						"command": "claudecode-filter",
					},
				},
			},
		},
		"UserPromptSubmit": []any{
			map[string]any{
				"hooks": []any{
					map[string]any{
						"type":    "command",
						"command": "claudecode-filter",
					},
				},
			},
		},
		"SessionEnd": []any{
			map[string]any{
				"hooks": []any{
					map[string]any{
						"type":    "command",
						"command": "claudecode-filter",
					},
				},
			},
		},
	},
}

// Execute updates ~/.claude/settings.json with required settings, preserving existing config.
func Execute() error {
	home := os.Getenv("HOME")
	if home == "" {
		return fmt.Errorf("HOME environment variable is not set")
	}

	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		return fmt.Errorf("failed to create %s: %w", claudeDir, err)
	}

	settingsPath := filepath.Join(claudeDir, "settings.json")

	existing, err := readConfig(settingsPath)
	if err != nil {
		return err
	}

	for k, v := range requiredSettings {
		existing[k] = v
	}

	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	data = append(data, '\n')

	if err := os.WriteFile(settingsPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", settingsPath, err)
	}

	fmt.Printf("Updated %s\n", settingsPath)

	return nil
}

func readConfig(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]any), nil
		}

		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("%s has invalid JSON: %w", path, err)
	}

	return config, nil
}
