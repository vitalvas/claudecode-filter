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

		if detectBranchCreate(seg) && !seen["branch"] {
			found = append(found, "branch")
			seen["branch"] = true
		}
	}

	return found
}

// branchCreateFlags are flags on git subcommands that create or rename a branch.
var branchCreateFlags = map[string]bool{
	"-c":       true,
	"-C":       true,
	"--copy":   true,
	"-m":       true,
	"-M":       true,
	"--move":   true,
	"-b":       true,
	"-B":       true,
	"--create": true,
	"--orphan": true,
}

// branchNonCreateFlags are flags on `git branch` that indicate a non-creating
// operation (delete, list, edit, etc.).
var branchNonCreateFlags = map[string]bool{
	"-d":                 true,
	"-D":                 true,
	"--delete":           true,
	"-l":                 true,
	"--list":             true,
	"-a":                 true,
	"--all":              true,
	"-r":                 true,
	"--remotes":          true,
	"-v":                 true,
	"-vv":                true,
	"--verbose":          true,
	"--show-current":     true,
	"--contains":         true,
	"--no-contains":      true,
	"--merged":           true,
	"--no-merged":        true,
	"--points-at":        true,
	"--edit-description": true,
	"--set-upstream-to":  true,
	"--unset-upstream":   true,
}

// detectBranchCreate returns true when the segment creates a new branch via
// `git branch <name>`, `git switch -c`, `git checkout -b`, or
// `git worktree add <path> <branch>`.
func detectBranchCreate(segment string) bool {
	words := strings.Fields(strings.TrimSpace(segment))

	for i, w := range words {
		if w != "git" {
			continue
		}

		sub, rest := gitSubcommand(words[i+1:])

		switch sub {
		case "branch":
			return branchSubcommandCreates(rest)
		case "switch", "checkout":
			return hasBranchCreateFlag(rest)
		case "worktree":
			return worktreeAddCreates(rest)
		}

		return false
	}

	return false
}

func gitSubcommand(args []string) (string, []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			if (a == "-C" || a == "-c") && i+1 < len(args) {
				i++
			}

			continue
		}

		return a, args[i+1:]
	}

	return "", nil
}

func branchSubcommandCreates(args []string) bool {
	hasNonCreate := false

	for _, a := range args {
		if branchCreateFlags[a] {
			return true
		}

		if branchNonCreateFlags[a] {
			hasNonCreate = true
		}
	}

	if hasNonCreate {
		return false
	}

	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			continue
		}

		return true
	}

	return false
}

func hasBranchCreateFlag(args []string) bool {
	for _, a := range args {
		if branchCreateFlags[a] {
			return true
		}
	}

	return false
}

func worktreeAddCreates(args []string) bool {
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			continue
		}

		return a == "add"
	}

	return false
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

var (
	conventionalCommitRe = regexp.MustCompile(`^(feat|fix|perf|deps|revert|docs|chore|style|refactor|test|build|ci)(\(.+\))?!?: `)
	breakingChangeRe     = regexp.MustCompile(`^(feat|fix|perf|deps|revert|docs|chore|style|refactor|test|build|ci)(\(.+\))?!: `)
)

var deniedCommitTypes = []string{
	"style",
	"refactor",
	"test",
	"build",
	"ci",
}

func detectDeniedCommitType(command string) (string, bool) {
	msg := extractCommitMessage(command)
	if msg == "" {
		return "", false
	}

	if !conventionalCommitRe.MatchString(msg) {
		return "", false
	}

	if breakingChangeRe.MatchString(msg) {
		return "!", true
	}

	typeRe := regexp.MustCompile(`^([a-z]+)`)
	matches := typeRe.FindStringSubmatch(msg)
	if matches == nil {
		return "", false
	}

	commitType := matches[1]
	for _, denied := range deniedCommitTypes {
		if commitType == denied {
			return commitType, true
		}
	}

	return "", false
}

var (
	commitMsgRe  = regexp.MustCompile(`-m\s+(?:"([^"]+)"|'([^']+)'|(\S+))`)
	heredocMsgRe = regexp.MustCompile(`-m\s+"\$\(cat\s+<<-?\s*'?\w+'?\s*\n`)
)

func extractCommitMessage(command string) string {
	// Try heredoc format first: -m "$(cat <<'EOF'\nmessage\nEOF\n)"
	if loc := heredocMsgRe.FindStringIndex(command); loc != nil {
		rest := command[loc[1]:]
		if idx := strings.Index(rest, "\n"); idx > 0 {
			return strings.TrimSpace(rest[:idx])
		}

		return strings.TrimSpace(rest)
	}

	// Match -m "msg" or -m 'msg' or -m msg
	matches := commitMsgRe.FindStringSubmatch(command)
	if matches == nil {
		return ""
	}

	for _, m := range matches[1:] {
		if m != "" {
			return m
		}
	}

	return ""
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
