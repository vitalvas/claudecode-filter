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

func detectGitOp(segment string) (string, bool) {
	words := strings.Fields(strings.TrimSpace(segment))

	gitFound := false
	for _, w := range words {
		if w == "git" {
			gitFound = true
			continue
		}

		if gitFound {
			for _, op := range blockedOps {
				if w == op {
					return op, true
				}
			}
		}
	}

	return "", false
}
