package hook

import "encoding/json"

const (
	EventPreToolUse        = "PreToolUse"
	EventPermissionRequest = "PermissionRequest"
	EventUserPromptSubmit  = "UserPromptSubmit"
	EventSessionEnd        = "SessionEnd"

	PermissionAllow = "allow"
	PermissionDeny  = "deny"
)

// Handler processes a hook input and returns a result.
type Handler func(Input) *Result

// Middleware wraps a handler with additional logic.
type Middleware func(Handler) Handler

// BuildChain composes middlewares into a single handler.
// First middleware in the list runs first.
func BuildChain(middlewares ...Middleware) Handler {
	handler := Handler(func(_ Input) *Result { return nil })

	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}

	return handler
}

// Input represents the JSON payload from a Claude Code hook event.
type Input struct {
	SessionID     string `json:"session_id"`
	CWD           string `json:"cwd"`
	HookEventName string `json:"hook_event_name"`

	// PreToolUse / PermissionRequest fields
	ToolName  string          `json:"tool_name,omitempty"`
	ToolInput json.RawMessage `json:"tool_input,omitempty"`

	// UserPromptSubmit fields
	Prompt string `json:"prompt,omitempty"`
}

// BashToolInput represents the input for Bash tool calls.
type BashToolInput struct {
	Command string `json:"command"`
}

// ReadToolInput represents the input for Read tool calls.
type ReadToolInput struct {
	FilePath string `json:"file_path"`
}

// PreToolUseOutput represents the hook-specific output for PreToolUse.
type PreToolUseOutput struct {
	HookEventName            string `json:"hookEventName"`
	PermissionDecision       string `json:"permissionDecision"`
	PermissionDecisionReason string `json:"permissionDecisionReason,omitempty"`
}

// PermissionDecision represents the decision for a PermissionRequest.
type PermissionDecision struct {
	Behavior string `json:"behavior"`
}

// PermissionRequestOutput represents the hook-specific output for PermissionRequest.
type PermissionRequestOutput struct {
	HookEventName string             `json:"hookEventName"`
	Decision      PermissionDecision `json:"decision"`
}

// PreToolUseOutputWrapper represents the full hook output for PreToolUse.
type PreToolUseOutputWrapper struct {
	HookSpecificOutput PreToolUseOutput `json:"hookSpecificOutput"`
}

// PermissionRequestOutputWrapper represents the full hook output for PermissionRequest.
type PermissionRequestOutputWrapper struct {
	HookSpecificOutput PermissionRequestOutput `json:"hookSpecificOutput"`
}

// Result is the outcome of processing a hook event.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}
