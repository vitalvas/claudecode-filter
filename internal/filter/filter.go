package filter

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/vitalvas/claudecode-filter/internal/hook"
	"github.com/vitalvas/claudecode-filter/internal/plugins/autoallow"
	"github.com/vitalvas/claudecode-filter/internal/plugins/configcheck"
	"github.com/vitalvas/claudecode-filter/internal/plugins/gitguard"
	"github.com/vitalvas/claudecode-filter/internal/plugins/readguard"
	"github.com/vitalvas/claudecode-filter/internal/setup"
)

var handler = hook.BuildChain(
	configcheck.New(),
	readguard.New(),
	autoallow.New(),
	gitguard.New(),
)

// Execute handles CLI arguments and runs the appropriate command.
func Execute() {
	if cmd := subcommand(os.Args); cmd != "" {
		if err := executeCommand(cmd); err != nil {
			fmt.Fprintf(os.Stderr, "%s failed: %v\n", cmd, err)
			os.Exit(1)
		}

		return
	}

	result := run(os.Stdin)
	writeResult(os.Stdout, os.Stderr, result)
	os.Exit(result.ExitCode)
}

func run(r io.Reader) hook.Result {
	input, err := io.ReadAll(r)
	if err != nil {
		return hook.Result{
			Stderr:   fmt.Sprintf("failed to read stdin: %v", err),
			ExitCode: 1,
		}
	}

	result := process(input)

	if _, err := os.Stat(".tmp/debug"); err == nil {
		debugLog(input, result)
	}

	return result
}

func writeResult(stdout, stderr io.Writer, result hook.Result) {
	if result.Stderr != "" {
		fmt.Fprint(stderr, result.Stderr)
	}

	if result.Stdout != "" {
		fmt.Fprint(stdout, result.Stdout)
	}
}

var commands = map[string]bool{
	"setup": true,
}

func subcommand(args []string) string {
	if len(args) > 1 && commands[args[1]] {
		return args[1]
	}

	return ""
}

func executeCommand(cmd string) error {
	switch cmd {
	case "setup":
		return setup.Execute()
	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

func process(input []byte) hook.Result {
	var hookInput hook.Input
	if err := json.Unmarshal(input, &hookInput); err != nil {
		return hook.Result{
			Stderr:   fmt.Sprintf("failed to parse hook input: %v", err),
			ExitCode: 1,
		}
	}

	if result := handler(hookInput); result != nil {
		return *result
	}

	return hook.Result{}
}
