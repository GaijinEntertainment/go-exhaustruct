package pattern

import (
	"regexp"
	"strings"

	"dev.gaijin.team/go/golib/e"
	"dev.gaijin.team/go/golib/fields"
)

// List is a collection of compiled regular expressions.
// Implements flag.Value for command-line flag binding.
type List []*regexp.Regexp //nolint:recvcheck

// NewList compiles patterns into a List.
// Returns error if any pattern is empty or invalid.
func NewList(patterns ...string) (List, error) {
	if len(patterns) == 0 {
		return nil, nil
	}

	list := make(List, 0, len(patterns))

	for _, pattern := range patterns {
		re, err := parseRx(pattern)
		if err != nil {
			return nil, err
		}

		list = append(list, re)
	}

	return list, nil
}

// MatchFullString returns true if any regex matches the entire string.
// Pattern "test" matches "test" but not "testing" or "contest".
func (l List) MatchFullString(str string) bool {
	if len(l) == 0 {
		return false
	}

	for i := range len(l) {
		if m := l[i].FindStringSubmatch(str); len(m) > 0 && m[0] == str {
			return true
		}
	}

	return false
}

// String returns comma-separated regex patterns (flag.Value interface).
func (l List) String() string {
	patterns := make([]string, len(l))
	for i, re := range l {
		patterns[i] = re.String()
	}

	return strings.Join(patterns, ",")
}

// Set compiles and appends a pattern to the List (flag.Value interface).
func (l *List) Set(value string) error {
	re, err := parseRx(value)
	if err != nil {
		return err
	}

	*l = append(*l, re)

	return nil
}

func parseRx(str string) (*regexp.Regexp, error) {
	if str == "" {
		return nil, e.New("empty regular expression is not allowed")
	}

	re, err := regexp.Compile(str)
	if err != nil {
		return nil, e.NewFrom("failed to compile regular expression", err, fields.F("pattern", str))
	}

	return re, nil
}
