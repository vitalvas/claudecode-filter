package gitguard

import (
	"regexp"
	"strings"
)

var blockedOps = []string{
	"commit",
	"push",
	"merge",
	"rebase",
	"cherry-pick",
	"revert",
	"tag",
}

var commandSplitter = regexp.MustCompile(`\s*(?:&&|\|\||;)\s*`)

func detectBlockedOps(command string) []string {
	segments := commandSplitter.Split(command, -1)

	var found []string
	seen := make(map[string]bool)

	for _, seg := range segments {
		if op, ok := detectGitOp(seg); ok && !seen[op] {
			found = append(found, op)
			seen[op] = true
		}
	}

	return found
}

var blockedCommitHeaders = []string{
	"co-authored-by:",
	"ai-assistant:",
}

func containsBlockedCommitHeader(command string) (string, bool) {
	lower := strings.ToLower(command)

	for _, header := range blockedCommitHeaders {
		if strings.Contains(lower, header) {
			return header, true
		}
	}

	return "", false
}

func detectGitOp(segment string) (string, bool) {
	words := strings.Fields(strings.TrimSpace(segment))

	for i, w := range words {
		if w != "git" {
			continue
		}

		for j := i + 1; j < len(words); j++ {
			if strings.HasPrefix(words[j], "-") {
				// skip next word if flag takes a value (e.g. -C /path)
				if j+1 < len(words) && !strings.Contains(words[j], "=") {
					j++
				}

				continue
			}

			for _, op := range blockedOps {
				if words[j] == op {
					return op, true
				}
			}

			break
		}
	}

	return "", false
}
