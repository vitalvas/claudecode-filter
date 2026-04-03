package filter

import (
	"fmt"
	"os"

	"github.com/vitalvas/claudecode-filter/internal/hook"
)

func debugLog(input []byte, result hook.Result) {
	f, err := os.OpenFile(".tmp/debug.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()

	fmt.Fprintf(f, "INPUT: %s\nOUTPUT: stdout=%s stderr=%s exit=%d\n", input, result.Stdout, result.Stderr, result.ExitCode)
}
