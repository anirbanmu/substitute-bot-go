package substitution

import (
	"errors"
	"regexp"
)

type SubstitutionCommand struct {
	ToReplace   string
	ReplaceWith string
}

func ParseSubstitutionCommand(txt string) (*SubstitutionCommand, error) {
	re0 := regexp.MustCompile(`(?m:\As\/(.+)\/(.*)$)`)
	re1 := regexp.MustCompile(`(?m:\As#(.+)#(.*)$)`)
	parts := re0.FindStringSubmatch(txt)
	if len(parts) != 3 {
		parts = re1.FindStringSubmatch(txt)
	}

	if len(parts) != 3 {
		return nil, errors.New("not a substitution command")
	}

	return &SubstitutionCommand{parts[1], parts[2]}, nil
}

func (s *SubstitutionCommand) Run(txt string) (string, error) {
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
