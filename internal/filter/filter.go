package filter

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/vitalvas/claudecode-filter/internal/hook"
	"github.com/vitalvas/claudecode-filter/internal/plugins/gitguard"
)

var filters = []hook.Filter{
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
	case "PreToolUse":
		return handlePreToolUse(hookInput)
	case "UserPromptSubmit":
		return handleUserPromptSubmit(hookInput)
	case "SessionEnd":
		handleSessionEnd(hookInput)
		return hook.Result{}
	default:
		return hook.Result{}
	}
}

func handlePreToolUse(input hook.Input) hook.Result {
	for _, f := range filters {
		if result := f.OnPreToolUse(input); result != nil {
			return *result
		}
	}

	return hook.Result{}
}

func handleUserPromptSubmit(input hook.Input) hook.Result {
	for _, f := range filters {
		if result := f.OnUserPromptSubmit(input); result != nil {
			return *result
		}
	}

	return hook.Result{}
}

func handleSessionEnd(input hook.Input) {
	for _, f := range filters {
		f.OnSessionEnd(input)
	}
}
