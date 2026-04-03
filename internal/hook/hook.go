package hook

import "encoding/json"

// Filter is the interface for pluggable hook filters.
type Filter interface {
	OnPreToolUse(input Input) *Result
	OnUserPromptSubmit(input Input) *Result
	OnSessionEnd(input Input)
}

// Input represents the JSON payload from a Claude Code hook event.
type Input struct {
	SessionID     string `json:"session_id"`
	CWD           string `json:"cwd"`
	HookEventName string `json:"hook_event_name"`

	// PreToolUse fields
	ToolName  string          `json:"tool_name,omitempty"`
	ToolInput json.RawMessage `json:"tool_input,omitempty"`

	// UserPromptSubmit fields
	Prompt string `json:"prompt,omitempty"`
}

// BashToolInput represents the input for Bash tool calls.
type BashToolInput struct {
	Command string `json:"command"`
}

// PreToolUseOutput represents the hook-specific output for PreToolUse.
type PreToolUseOutput struct {
	HookEventName            string `json:"hookEventName"`
	PermissionDecision       string `json:"permissionDecision"`
	PermissionDecisionReason string `json:"permissionDecisionReason,omitempty"`
}

// Output represents the full hook output.
type Output struct {
	HookSpecificOutput PreToolUseOutput `json:"hookSpecificOutput"`
}

// Result is the outcome of processing a hook event.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}
