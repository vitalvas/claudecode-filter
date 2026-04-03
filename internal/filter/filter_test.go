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

type mockFilter struct {
	preToolUseResult  *hook.Result
	permRequestResult *hook.Result
	promptResult      *hook.Result
	sessionEndCalled  bool
}

func (m *mockFilter) OnPreToolUse(_ hook.Input) *hook.Result        { return m.preToolUseResult }
func (m *mockFilter) OnPermissionRequest(_ hook.Input) *hook.Result { return m.permRequestResult }
func (m *mockFilter) OnUserPromptSubmit(_ hook.Input) *hook.Result  { return m.promptResult }
func (m *mockFilter) OnSessionEnd(_ hook.Input)                     { m.sessionEndCalled = true }

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

	t.Run("exits 1 on invalid json", func(t *testing.T) {
		cmd := exec.Command(os.Args[0], "-test.run=TestExecute$")
		cmd.Env = append(os.Environ(), "TEST_EXECUTE=1")
		cmd.Stdin = bytes.NewReader([]byte("not json"))

		err := cmd.Run()

		var exitErr *exec.ExitError
		require.ErrorAs(t, err, &exitErr)
		assert.Equal(t, 1, exitErr.ExitCode())
	})

	t.Run("outputs deny on blocked git op", func(t *testing.T) {
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
}

func TestDispatch(t *testing.T) {
	t.Run("PreToolUse dispatches to filter", func(t *testing.T) {
		orig := filters
		t.Cleanup(func() { filters = orig })

		mock := &mockFilter{preToolUseResult: &hook.Result{Stdout: "blocked"}}
		filters = []hook.Filter{mock}

		input, _ := json.Marshal(hook.Input{HookEventName: hook.EventPreToolUse})
		result := process(input)
		assert.Equal(t, "blocked", result.Stdout)
	})

	t.Run("PreToolUse passes when filter returns nil", func(t *testing.T) {
		orig := filters
		t.Cleanup(func() { filters = orig })

		mock := &mockFilter{}
		filters = []hook.Filter{mock}

		input, _ := json.Marshal(hook.Input{HookEventName: hook.EventPreToolUse})
		result := process(input)
		assert.Empty(t, result.Stdout)
		assert.Equal(t, 0, result.ExitCode)
	})

	t.Run("PermissionRequest dispatches to filter", func(t *testing.T) {
		orig := filters
		t.Cleanup(func() { filters = orig })

		mock := &mockFilter{permRequestResult: &hook.Result{Stdout: "allowed"}}
		filters = []hook.Filter{mock}

		input, _ := json.Marshal(hook.Input{HookEventName: hook.EventPermissionRequest})
		result := process(input)
		assert.Equal(t, "allowed", result.Stdout)
	})

	t.Run("UserPromptSubmit dispatches to filter", func(t *testing.T) {
		orig := filters
		t.Cleanup(func() { filters = orig })

		mock := &mockFilter{promptResult: &hook.Result{Stdout: "handled"}}
		filters = []hook.Filter{mock}

		input, _ := json.Marshal(hook.Input{HookEventName: hook.EventUserPromptSubmit})
		result := process(input)
		assert.Equal(t, "handled", result.Stdout)
	})

	t.Run("UserPromptSubmit passes when filter returns nil", func(t *testing.T) {
		orig := filters
		t.Cleanup(func() { filters = orig })

		mock := &mockFilter{}
		filters = []hook.Filter{mock}

		input, _ := json.Marshal(hook.Input{HookEventName: hook.EventUserPromptSubmit})
		result := process(input)
		assert.Empty(t, result.Stdout)
		assert.Equal(t, 0, result.ExitCode)
	})

	t.Run("SessionEnd calls all filters", func(t *testing.T) {
		orig := filters
		t.Cleanup(func() { filters = orig })

		mock1 := &mockFilter{}
		mock2 := &mockFilter{}
		filters = []hook.Filter{mock1, mock2}

		input, _ := json.Marshal(hook.Input{HookEventName: hook.EventSessionEnd})
		result := process(input)
		assert.Equal(t, 0, result.ExitCode)
		assert.True(t, mock1.sessionEndCalled)
		assert.True(t, mock2.sessionEndCalled)
	})

	t.Run("first deny wins", func(t *testing.T) {
		orig := filters
		t.Cleanup(func() { filters = orig })

		mock1 := &mockFilter{preToolUseResult: &hook.Result{Stdout: "first"}}
		mock2 := &mockFilter{preToolUseResult: &hook.Result{Stdout: "second"}}
		filters = []hook.Filter{mock1, mock2}

		input, _ := json.Marshal(hook.Input{HookEventName: hook.EventPreToolUse})
		result := process(input)
		assert.Equal(t, "first", result.Stdout)
	})
}
