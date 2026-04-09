package setup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

var config = map[string]any{
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

// Execute writes ~/.claude/settings.local.json with required hooks config.
func Execute() error {
	home := os.Getenv("HOME")
	if home == "" {
		return fmt.Errorf("HOME environment variable is not set")
	}

	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		return fmt.Errorf("failed to create %s: %w", claudeDir, err)
	}

	settingsPath := filepath.Join(claudeDir, "settings.local.json")

	data, err := json.MarshalIndent(config, "", "  ")
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
