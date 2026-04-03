# claudecode-filter

```json
{
  "hooks": {
    "PreToolUse": [{
      "matcher": "*",
      "hooks": [{
        "type": "command",
        "command": "claudecode-filter"
      }]
    }],
    "UserPromptSubmit": [{
      "hooks": [{
        "type": "command",
        "command": "claudecode-filter"
      }]
    }],
    "SessionEnd": [{
      "hooks": [{
        "type": "command",
        "command": "claudecode-filter"
      }]
    }]
  }
}
```
