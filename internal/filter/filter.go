package filter

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/vitalvas/claudecode-filter/internal/hook"
	"github.com/vitalvas/claudecode-filter/internal/plugins/autoallow"
	"github.com/vitalvas/claudecode-filter/internal/plugins/gitguard"
)

var filters = []hook.Filter{
	autoallow.New(),
	gitguard.New(),
}

// Execute reads hook input from stdin, processes it, and exits.
func Execute() {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read stdin: %v", err)
		os.Exit(1)
	}

	result := process(input)

	if _, err := os.Stat(".tmp/debug"); err == nil {
		debugLog(input, result)
	}

	if result.Stderr != "" {
		fmt.Fprint(os.Stderr, result.Stderr)
	}

	if result.Stdout != "" {
		fmt.Print(result.Stdout)
	}

	os.Exit(result.ExitCode)
}

func process(input []byte) hook.Result {
	var hookInput hook.Input
	if err := json.Unmarshal(input, &hookInput); err != nil {
		return hook.Result{
			Stderr:   fmt.Sprintf("failed to parse hook input: %v", err),
			ExitCode: 1,
		}
	}

	switch hookInput.HookEventName {
	case hook.EventPreToolUse:
		for _, f := range filters {
			if result := f.OnPreToolUse(hookInput); result != nil {
				return *result
			}
		}

	case hook.EventPermissionRequest:
		for _, f := range filters {
			if result := f.OnPermissionRequest(hookInput); result != nil {
				return *result
			}
		}

	case hook.EventUserPromptSubmit:
		for _, f := range filters {
			if result := f.OnUserPromptSubmit(hookInput); result != nil {
				return *result
			}
		}

	case hook.EventSessionEnd:
		for _, f := range filters {
			f.OnSessionEnd(hookInput)
		}
	}

	return hook.Result{}
}
