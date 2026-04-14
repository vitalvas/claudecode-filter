package writeguard

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectWriteCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    bool
	}{
		{name: "rm file", command: "rm file.go", want: true},
		{name: "rm -rf", command: "rm -rf dir", want: true},
		{name: "mv file", command: "mv old.go new.go", want: true},
		{name: "cp file", command: "cp src dst", want: true},
		{name: "touch file", command: "touch file.go", want: true},
		{name: "mkdir", command: "mkdir newdir", want: true},
		{name: "rmdir", command: "rmdir olddir", want: true},
		{name: "chmod", command: "chmod 644 file", want: true},
		{name: "chown", command: "chown user file", want: true},
		{name: "ln symlink", command: "ln -s src dst", want: true},
		{name: "truncate", command: "truncate -s 0 file", want: true},
		{name: "unlink", command: "unlink file", want: true},
		{name: "redirect stdout", command: "echo test > file", want: true},
		{name: "redirect append", command: "echo test >> file", want: true},
		{name: "pipe", command: "cat file | tee output", want: true},
		{name: "chained write", command: "ls && rm file", want: true},
		{name: "read only ls", command: "ls -la", want: false},
		{name: "read only cat", command: "cat file.go", want: false},
		{name: "read only grep", command: "grep pattern file", want: false},
		{name: "read only git status", command: "git status", want: false},
		{name: "read only go test", command: "go test ./...", want: false},
		{name: "empty", command: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, detectWriteCommand(tt.command))
		})
	}
}
