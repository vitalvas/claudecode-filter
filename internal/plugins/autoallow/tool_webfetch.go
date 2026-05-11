package autoallow

import (
	"encoding/json"
	"net/url"

	"github.com/vitalvas/claudecode-filter/internal/hook"
	"github.com/vitalvas/gokit/xstrings"
)

var allowedWebFetchDomains = []string{
	"aws.amazon.com",
	"deepwiki.com",
	"docs.rs",
	"en.wikipedia.org",
	"github.com",
	"gitlab.com",
	"localhost",
	"raw.githubusercontent.com",
	"www.iana.org",
	"www.rfc-editor.org",
}

var allowedWebFetchPatterns = []string{
	"*.github.com",
	"*.github.io",
	"*.vitalvas.com",
	"*.vitalvas.dev",
	"*.vitalvas.net",
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

	hostname := parsed.Hostname()

	for _, domain := range allowedWebFetchDomains {
		if hostname == domain {
			return allowPermissionRequest()
		}
	}

	for _, pattern := range allowedWebFetchPatterns {
		if matched, _ := xstrings.GlobMatch(pattern, hostname); matched {
			return allowPermissionRequest()
		}
	}

	return nil
}
