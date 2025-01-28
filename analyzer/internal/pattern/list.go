package pattern

import (
	"regexp"
	"strings"

	"dev.gaijin.team/go/golib/e"
	"dev.gaijin.team/go/golib/fields"
)

var (
	ErrEmpty             = e.New("pattern can't be empty")
	ErrCompilationFailed = e.New("pattern compilation failed")
)

// List is a list of regular expressions.
type List []*regexp.Regexp

// NewList parses slice of strings to a slice of compiled regular expressions.
func NewList(strs ...string) (List, error) {
	if len(strs) == 0 {
		return nil, nil
	}

	l := make(List, 0, len(strs))

	for _, str := range strs {
		re, err := strToRe(str)
		if err != nil {
			return nil, err
		}

		l = append(l, re)
	}

	return l, nil
}

// MatchFullString matches provided string against all regexps in a slice and returns
// true if any of them matches whole string.
func (l List) MatchFullString(str string) bool {
	for i := range len(l) {
		if m := l[i].FindStringSubmatch(str); len(m) > 0 && m[0] == str {
			return true
		}
	}

	return false
}

func (l *List) Set(value string) error {
	re, err := strToRe(value)
	if err != nil {
		return err
	}

	*l = append(*l, re)

	return nil
}

func (l *List) String() string {
	res := make([]string, 0, len(*l))

	for _, re := range *l {
		res = append(res, `"`+re.String()+`"`)
	}

	return strings.Join(res, ", ")
}

// strToRe parses string to a compiled regular expression that matches full string.
func strToRe(str string) (*regexp.Regexp, error) {
	if str == "" {
		return nil, ErrEmpty
	}

	re, err := regexp.Compile(str)
	if err != nil {
		return nil, ErrCompilationFailed.Wrap(err, fields.F("pattern", str))
	}

	return re, nil
}
