package writeguard

import (
	"regexp"
	"strings"
)

var writeCommands = []string{
	"chmod",
	"chown",
	"cp",
	"dd",
	"ln",
	"mkdir",
	"mkfifo",
	"mknod",
	"mv",
	"rm",
	"rmdir",
	"touch",
	"truncate",
	"unlink",
}

var redirectPattern = regexp.MustCompile(`[>|]`)

var commandSplitter = regexp.MustCompile(`\s*(?:&&|\|\||;)\s*`)

func detectWriteCommand(command string) bool {
	if redirectPattern.MatchString(command) {
		return true
	}

	segments := commandSplitter.Split(command, -1)

	for _, seg := range segments {
		words := strings.Fields(strings.TrimSpace(seg))
		if len(words) == 0 {
			continue
		}

		for _, cmd := range writeCommands {
			if words[0] == cmd {
				return true
			}
		}
	}

	return false
}
