package autoallow

import (
	"encoding/json"
	"net/url"

	"github.com/vitalvas/claudecode-filter/internal/hook"
)

var allowedWebFetchDomains = []string{
	"github.com",
	"raw.githubusercontent.com",
}

func handleWebFetch(input hook.Input) *hook.Result {
	var fetchInput hook.WebFetchToolInput
	if err := json.Unmarshal(input.ToolInput, &fetchInput); err != nil {
		return nil
	}

	parsed, err := url.Parse(fetchInput.URL)
	if err != nil {
		return nil
	}

	for _, domain := range allowedWebFetchDomains {
		if parsed.Hostname() == domain {
			return allowPermissionRequest()
		}
	}

	return nil
}
