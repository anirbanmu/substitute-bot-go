package substitution

import "testing"

func TestParseSubstitutionCommand(t *testing.T) {
	cases := []struct {
		input string
		cmd   *SubstitutionCommand
		err   bool
	}{
		{"", nil, true},
		{`s\4\3`, nil, true},
		{"s/m/r", &SubstitutionCommand{"m", "r"}, false},
		{"s#m#r", &SubstitutionCommand{"m", "r"}, false},
		{"\ns#m#r", nil, true}, // Command not on first line
		{" s#m#r", nil, true},
		{"s/sp ace/s pace", &SubstitutionCommand{"sp ace", "s pace"}, false},
		{"s/sp ace/s pace\nshould not be there", &SubstitutionCommand{"sp ace", "s pace"}, false},
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
				"ParseSubstitutionCommand(%s) should not have returned a command but returned SubstitutionCommand{%s, %s}",
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
				"ParseSubstitutionCommand(%s) should have returned SubstitutionCommand{%s, %s} but returned SubstitutionCommand{%s, %s}",
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
		cmd SubstitutionCommand
		out string
		err bool
	}{
		{"text", SubstitutionCommand{"ex", "ex"}, "", true},
		{"text", SubstitutionCommand{"f", "g"}, "", true},
		{"text beep", SubstitutionCommand{"ext bee", "t e"}, "tt ep", false},
		{"text", SubstitutionCommand{`\w+`, "blah"}, "blah", false}, // Accepts actual regexp
		{"23", SubstitutionCommand{`(\d)`, `<$1>`}, "<2><3>", false},
		{"23", SubstitutionCommand{`(\d`, `<$1>`}, "", true},        // Erroneous regexp
		{`\text\`, SubstitutionCommand{`\\`, `|`}, "|text|", false}, // Can work with escaped characters
	}

	for _, c := range cases {
		out, err := c.cmd.Run(c.in)
		if c.err && err == nil {
			t.Errorf("SubstitutionCommand{%s, %s}.Run(%s) should have errored but did not", c.cmd.ToReplace, c.cmd.ReplaceWith, c.in)
		}

		if !c.err && err != nil {
			t.Errorf("SubstitutionCommand{%s, %s}.Run(%s) should not have errored but did: %s", c.cmd.ToReplace, c.cmd.ReplaceWith, c.in, err)
		}

		if c.out != out {
			t.Errorf("SubstitutionCommand{%s, %s}.Run(%s) should have returned %s but returned %s", c.cmd.ToReplace, c.cmd.ReplaceWith, c.in, c.out, out)
		}
	}
}
