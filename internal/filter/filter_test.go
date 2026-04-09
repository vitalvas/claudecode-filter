package filter

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitalvas/claudecode-filter/internal/hook"
)

func TestExecute(t *testing.T) {
	if os.Getenv("TEST_EXECUTE") == "1" {
		Execute()
		return
	}

	t.Run("exits 0 on valid input", func(t *testing.T) {
		input, _ := json.Marshal(hook.Input{HookEventName: "UnknownEvent"})

		cmd := exec.Command(os.Args[0], "-test.run=TestExecute$")
		cmd.Env = append(os.Environ(), "TEST_EXECUTE=1")
		cmd.Stdin = bytes.NewReader(input)

		err := cmd.Run()
		assert.NoError(t, err)
	})

	t.Run("outputs result on stdout", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "git commit -m 'test'"})
		input, _ := json.Marshal(hook.Input{
			HookEventName: hook.EventPreToolUse,
			CWD:           t.TempDir(),
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		cmd := exec.Command(os.Args[0], "-test.run=TestExecute$")
		cmd.Env = append(os.Environ(), "TEST_EXECUTE=1")
		cmd.Stdin = bytes.NewReader(input)

		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		err := cmd.Run()
		assert.NoError(t, err)
		assert.Contains(t, stdout.String(), "deny")
	})

	t.Run("exits 1 on invalid json", func(t *testing.T) {
		cmd := exec.Command(os.Args[0], "-test.run=TestExecute$")
		cmd.Env = append(os.Environ(), "TEST_EXECUTE=1")
		cmd.Stdin = bytes.NewReader([]byte("not json"))

		err := cmd.Run()

		var exitErr *exec.ExitError
		require.ErrorAs(t, err, &exitErr)
		assert.Equal(t, 1, exitErr.ExitCode())
	})
}

func TestExecuteCommand(t *testing.T) {
	t.Run("setup succeeds", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())

		err := executeCommand("setup")
		assert.NoError(t, err)
	})

	t.Run("unknown command fails", func(t *testing.T) {
		err := executeCommand("unknown")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown command")
	})
}

func TestWriteResult(t *testing.T) {
	t.Run("writes stdout", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		writeResult(&stdout, &stderr, hook.Result{Stdout: "output"})

		assert.Equal(t, "output", stdout.String())
		assert.Empty(t, stderr.String())
	})

	t.Run("writes stderr", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		writeResult(&stdout, &stderr, hook.Result{Stderr: "error"})

		assert.Empty(t, stdout.String())
		assert.Equal(t, "error", stderr.String())
	})

	t.Run("writes both", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		writeResult(&stdout, &stderr, hook.Result{Stdout: "out", Stderr: "err"})

		assert.Equal(t, "out", stdout.String())
		assert.Equal(t, "err", stderr.String())
	})

	t.Run("empty result writes nothing", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		writeResult(&stdout, &stderr, hook.Result{})

		assert.Empty(t, stdout.String())
		assert.Empty(t, stderr.String())
	})
}

func TestSubcommand(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "setup", args: []string{"bin", "setup"}, want: "setup"},
		{name: "no args", args: []string{"bin"}, want: ""},
		{name: "empty", args: []string{}, want: ""},
		{name: "unknown ignored", args: []string{"bin", "unknown"}, want: ""},
		{name: "test flag ignored", args: []string{"bin", "-test.run=Foo"}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, subcommand(tt.args))
		})
	}
}

func TestRun(t *testing.T) {
	t.Run("valid input", func(t *testing.T) {
		input, _ := json.Marshal(hook.Input{HookEventName: "UnknownEvent"})
		result := run(bytes.NewReader(input))

		assert.Equal(t, 0, result.ExitCode)
	})

	t.Run("invalid json", func(t *testing.T) {
		result := run(bytes.NewReader([]byte("not json")))

		assert.Equal(t, 1, result.ExitCode)
		assert.Contains(t, result.Stderr, "failed to parse")
	})

	t.Run("handler result", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "git commit -m 'test'"})
		input, _ := json.Marshal(hook.Input{
			HookEventName: hook.EventPreToolUse,
			CWD:           t.TempDir(),
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		result := run(bytes.NewReader(input))
		assert.NotEmpty(t, result.Stdout)
	})
}

func TestProcess(t *testing.T) {
	t.Run("invalid json", func(t *testing.T) {
		result := process([]byte("not json"))
		assert.Equal(t, 1, result.ExitCode)
		assert.Contains(t, result.Stderr, "failed to parse")
	})

	t.Run("unknown hook event", func(t *testing.T) {
		input, err := json.Marshal(hook.Input{HookEventName: "UnknownEvent"})
		require.NoError(t, err)

		result := process(input)
		assert.Equal(t, 0, result.ExitCode)
		assert.Empty(t, result.Stdout)
	})

	t.Run("handler result returned", func(t *testing.T) {
		toolInput, _ := json.Marshal(hook.BashToolInput{Command: "git commit -m 'test'"})
		input, _ := json.Marshal(hook.Input{
			HookEventName: hook.EventPreToolUse,
			CWD:           t.TempDir(),
			ToolName:      "Bash",
			ToolInput:     toolInput,
		})

		result := process(input)
		assert.NotEmpty(t, result.Stdout)
	})
}

func TestMiddlewareChain(t *testing.T) {
	t.Run("first middleware can short-circuit", func(t *testing.T) {
		first := func(_ hook.Handler) hook.Handler {
			return func(_ hook.Input) *hook.Result {
				return &hook.Result{Stdout: "first"}
			}
		}
		second := func(_ hook.Handler) hook.Handler {
			return func(_ hook.Input) *hook.Result {
				return &hook.Result{Stdout: "second"}
			}
		}

		h := hook.BuildChain(first, second)
		result := h(hook.Input{})
		assert.Equal(t, "first", result.Stdout)
	})

	t.Run("passes to next when nil", func(t *testing.T) {
		first := func(next hook.Handler) hook.Handler {
			return func(input hook.Input) *hook.Result {
				return next(input)
			}
		}
		second := func(_ hook.Handler) hook.Handler {
			return func(_ hook.Input) *hook.Result {
				return &hook.Result{Stdout: "second"}
			}
		}

		h := hook.BuildChain(first, second)
		result := h(hook.Input{})
		assert.Equal(t, "second", result.Stdout)
	})

	t.Run("returns nil when no middleware handles", func(t *testing.T) {
		passthrough := func(next hook.Handler) hook.Handler {
			return func(input hook.Input) *hook.Result {
				return next(input)
			}
		}

		h := hook.BuildChain(passthrough)
		result := h(hook.Input{})
		assert.Nil(t, result)
	})
}
