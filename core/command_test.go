package core

import (
	"fmt"
	"strings"
	"testing"
)

func TestShellParsing(t *testing.T) {
	tests := []struct {
		cmd  string
		want string
	}{
		{
			"   pwd   ",
			`"pwd" []`,
		},
		{
			"|sort | cat",
			`| "sort" [] | "cat" []`,
		},
		{
			">   sort| cat  ",
			`> "sort" [] | "cat" []`,
		},
		{
			"<cat file|sort",
			`< "cat" ["file"] | "sort" []`,
		},
		{
			"ls |grep -v a",
			`"ls" [] | "grep" ["-v" "a"]`,
		},
	}

	for _, tt := range tests {
		p, err := parse(tt.cmd)
		if err != nil {
			t.Errorf("%s: unexpected syntax error: %v", tt.cmd, err)
			continue
		}
		if got := p.String(); got != tt.want {
			t.Errorf("%s:\ngot:  %v\nwant: %v", tt.cmd, got, tt.want)
		}
	}
}

func TestShellSyntaxErrors(t *testing.T) {
	tests := []struct {
		cmd string
		err string
	}{
		{"    ", errEmptyCmd.Error()},
		{" cat | | cat", "missing command"},
	}

	for _, tt := range tests {
		_, err := parse(tt.cmd)
		if err == nil {
			t.Errorf("%q: should have syntax error", tt.cmd)
			continue
		}
		if got := err.Error(); got != tt.err {
			t.Errorf("%q: got %q, want %q", tt.cmd, got, tt.err)
		}
	}
}

func (p *pipeline) String() string {
	sign := ""
	if p.pipeInput {
		sign += ">"
	}
	if p.pipeOutput {
		sign += "<"
	}
	if sign == "><" {
		sign = "|"
	}
	return strings.TrimSpace(sign + " " + p.pipe.String())
}

func (p *pipe) String() string {
	if p.prev == nil {
		return p.cmd.String()
	}
	return p.prev.String() + " | " + p.cmd.String()
}

func (c *command) String() string {
	return fmt.Sprintf("%q %q", c.cmd, c.args)
}
