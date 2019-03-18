package main

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"go.uber.org/zap"
)

type Answer interface {
	Check(user string, msg string) *Q
	Process(q *Q) string
	AddLogger(logger *zap.SugaredLogger)
}

type Q struct {
	Msg     string
	Prefix  string
	Cmd     string
	Matched bool
	User    string
}

func (q *Q) short() string {
	return q.Msg[len(q.Prefix):]
}

func (q *Q) words() []string {
	return strings.FieldsFunc(strings.ToLower(q.Msg), func(r rune) bool {
		//return unicode.IsSpace(r) || unicode.IsPunct(r)
		return unicode.IsSpace(r)
	})
}

var answers = make(map[string]Answer, 0)

func RegisterAnswer(name string, ans Answer) error {
	_, existing := answers[name]
	if existing {
		return fmt.Errorf("answer with name '%s' is already registered", name)
	}

	answers[name] = ans
	return nil
}

func CheckAnswer(user string, msg string) string {
	for _, ans := range answers {

		if q := ans.Check(user, msg); q.Matched {
			return ans.Process(q)
		}
	}

	return fmt.Sprintf("я не знаю, что такое %s", msg)
}

func indexOf(words []string, element ...string) int {
	for k, v := range words {
		for _, v1 := range element {
			if v == v1 {
				return k
			}
		}
	}
	return -1
}

func hasPrefix(s string, prefixes ...string) string {
	sort.Slice(prefixes, func(i, j int) bool {
		return len(prefixes[i]) > len(prefixes[j])
	})

	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			return prefix
		}
	}

	return ""
}
