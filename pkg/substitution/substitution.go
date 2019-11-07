package substitution

import (
	"errors"
	"regexp"
)

// Command represents a string substitution command
type Command struct {
	ToReplace   string
	ReplaceWith string
}

// ParseSubstitutionCommand tries to parse a VIM style substitution command from a string
func ParseSubstitutionCommand(txt string) (*Command, error) {
	re0 := regexp.MustCompile(`(?m:\As\/(.+?)\/(.*?)(?:\/\s*){0,1}$)`)
	re1 := regexp.MustCompile(`(?m:\As#(.+?)#(.*?)(?:#\s*){0,1}$)`)
	parts := re0.FindStringSubmatch(txt)
	if len(parts) != 3 {
		parts = re1.FindStringSubmatch(txt)
	}

	if len(parts) != 3 {
		return nil, errors.New("not a substitution command")
	}

	return &Command{parts[1], parts[2]}, nil
}

// Run executes a Command on a given string
func (s *Command) Run(txt string) (string, error) {
	re, err := regexp.Compile(s.ToReplace)
	if err != nil {
		return "", err
	}

	out := re.ReplaceAllString(txt, s.ReplaceWith)
	if out == txt {
		return "", errors.New("output was same as input")
	}

	return out, nil
}
