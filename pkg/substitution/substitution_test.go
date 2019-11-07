package substitution

import (
	"strings"
	"testing"
)

type parseSubCommandTestCase struct {
	input string
	cmd   *Command
	err   bool
}

func TestParseSubstitutionCommand(t *testing.T) {
	baseCases := []parseSubCommandTestCase{
		{"", nil, true},
		{`s\4\3`, nil, true},
		{"s/m/r", &Command{"m", "r"}, false},
		{"s//m//r", &Command{"/m", "/r"}, false},                    // Embedded slashes
		{"s//m//r/", &Command{"/m", "/r"}, false},                   // Embedded slashes
		{"s/m/r \nshould not be there", &Command{"m", "r "}, false}, // Trailing space
		{"s/m/r/", &Command{"m", "r"}, false},                       // Trailing slash
		{"s/m/r/ \nshould not be there", &Command{"m", "r"}, false}, // Trailing slash & spaces
		{"\ns/m/r", nil, true},                                      // Command not on first line
		{" s/m/r", nil, true},                                       // Command starts with space
		{"s/sp ace /s pace ", &Command{"sp ace ", "s pace "}, false},
		{"s/sp ace /s pace \nshould not be there", &Command{"sp ace ", "s pace "}, false},
	}

	cases := baseCases[:]
	for _, c := range baseCases {
		var cmd *Command = nil
		if c.cmd != nil {
			cmd = &Command{strings.Replace(c.cmd.ToReplace, "/", "#", -1), strings.Replace(c.cmd.ReplaceWith, "/", "#", -1)}
		}
		cases = append(cases,
			parseSubCommandTestCase{
				input: strings.Replace(c.input, "/", "#", -1),
				cmd:   cmd,
				err:   c.err,
			},
		)
	}

	for _, c := range cases {
		out, err := ParseSubstitutionCommand(c.input)
		if c.err && err == nil {
			t.Errorf("ParseSubstitutionCommand(%s) should have errored but did not", c.input)
		}

		if !c.err && err != nil {
			t.Errorf("ParseSubstitutionCommand(%s) should not have errored but did: %s", c.input, err)
		}

		if c.cmd != nil && out == nil {
			t.Errorf("ParseSubstitutionCommand(%s) should have returned a command but did not", c.input)
		}

		if c.cmd == nil && out != nil {
			t.Errorf(
				"ParseSubstitutionCommand(%s) should not have returned a command but returned Command{%s, %s}",
				c.input,
				out.ToReplace,
				out.ReplaceWith,
			)
		}

		if c.cmd == nil || out == nil {
			continue
		}

		if c.cmd.ToReplace != out.ToReplace || c.cmd.ReplaceWith != out.ReplaceWith {
			t.Errorf(
				"ParseSubstitutionCommand(%s) should have returned Command{%s, %s} but returned Command{%s, %s}",
				c.input,
				c.cmd.ToReplace,
				c.cmd.ReplaceWith,
				out.ToReplace,
				out.ReplaceWith,
			)
		}
	}
}

func TestSubstitutionCommandRun(t *testing.T) {
	cases := []struct {
		in  string
		cmd Command
		out string
		err bool
	}{
		{"text", Command{"ex", "ex"}, "", true},
		{"text", Command{"f", "g"}, "", true},
		{"text beep", Command{"ext bee", "t e"}, "tt ep", false},
		{"text", Command{`\w+`, "blah"}, "blah", false}, // Accepts actual regexp
		{"23", Command{`(\d)`, `<$1>`}, "<2><3>", false},
		{"23", Command{`(\d`, `<$1>`}, "", true},        // Erroneous regexp
		{`\text\`, Command{`\\`, `|`}, "|text|", false}, // Can work with escaped characters
	}

	for _, c := range cases {
		out, err := c.cmd.Run(c.in)
		if c.err && err == nil {
			t.Errorf("Command{%s, %s}.Run(%s) should have errored but did not", c.cmd.ToReplace, c.cmd.ReplaceWith, c.in)
		}

		if !c.err && err != nil {
			t.Errorf("Command{%s, %s}.Run(%s) should not have errored but did: %s", c.cmd.ToReplace, c.cmd.ReplaceWith, c.in, err)
		}

		if c.out != out {
			t.Errorf("Command{%s, %s}.Run(%s) should have returned %s but returned %s", c.cmd.ToReplace, c.cmd.ReplaceWith, c.in, c.out, out)
		}
	}
}
