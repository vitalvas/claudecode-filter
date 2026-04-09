package autoallow

import (
	"encoding/json"
	"net/url"

	"github.com/vitalvas/claudecode-filter/internal/hook"
)

var allowedWebFetchDomains = []string{
	"blog.vitalvas.com",
	"en.wikipedia.org",
	"github.com",
	"localhost",
	"raw.githubusercontent.com",
	"www.iana.org",
	"www.rfc-editor.org",
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
