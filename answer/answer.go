package answer

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"go.uber.org/zap"
)

type Answerer interface {
	Check(user string, msg string) *Q
	Process(q *Q) *Answer
	AddLogger(logger *zap.SugaredLogger)
}

type Answer struct {
	Msg   string
	Photo string
}

type Q struct {
	Msg     string
	Prefix  string
	Cmd     string
	Matched bool
	User    string
}

func TextAnswer(msg string) *Answer {
	return &Answer{Msg: msg}
}

func PhotoAnswer(file string) *Answer {
	return &Answer{Photo: file}
}

func (q *Q) short() string {
	return q.Msg[len(q.Prefix):]
}

func (q *Q) Words() []string {
	res := strings.FieldsFunc(strings.ToLower(q.Msg), func(r rune) bool {
		//return unicode.IsSpace(r) || unicode.IsPunct(r)
		return unicode.IsSpace(r)
	})

	for i, s := range res {
		res[i] = strings.Trim(s, " \t\n\r!?.,;:")
	}
	return res
}

var Answers = make(map[string]Answerer, 0)

func RegisterAnswer(name string, ans Answerer) error {
	_, existing := Answers[name]
	if existing {
		return fmt.Errorf("answer with name '%s' is already registered", name)
	}

	Answers[name] = ans
	return nil
}

func CheckAnswer(user string, msg string) *Answer {
	msg1 := strings.TrimLeft(msg, "/")

	for _, ans := range Answers {
		if q := ans.Check(user, msg1); q.Matched {
			return ans.Process(q)
		}
	}

	return TextAnswer(fmt.Sprintf("я не знаю, что такое %s", msg))
}

func IndexOf(words []string, element ...string) int {
	for k, v := range words {
		for _, v1 := range element {
			if v == v1 {
				return k
			}
		}
	}
	return -1
}

func HasPrefix(s string, prefixes ...string) string {
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
