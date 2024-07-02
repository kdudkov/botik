package answer

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"unicode"
)

type AnswerManager struct {
	answerers map[string]Answerer
	mx        sync.RWMutex
}

func New() *AnswerManager {
	return &AnswerManager{
		answerers: make(map[string]Answerer),
		mx:        sync.RWMutex{},
	}
}

type Answerer interface {
	Check(user string, msg string, repl string) *Q
	Process(q *Q) *Answer
}

type Answer struct {
	Msg   string
	Photo string
}

type Q struct {
	Msg     string
	Repl    string
	Prefix  string
	Cmd     string
	Payload string
	Matched bool
	User    string
}

func TextAnswer(msg string) *Answer {
	return &Answer{Msg: msg}
}

func PhotoAnswer(file string) *Answer {
	return &Answer{Photo: file}
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

func (am *AnswerManager) RegisterAnswer(name string, ans Answerer) error {
	am.mx.Lock()
	defer am.mx.Unlock()

	_, existing := am.answerers[name]
	if existing {
		return fmt.Errorf("answer with name '%s' is already registered", name)
	}

	am.answerers[name] = ans
	return nil
}

func (am *AnswerManager) CheckAnswer(user string, msg string, repl string) *Answer {
	am.mx.RLock()
	defer am.mx.RUnlock()

	msg1 := strings.TrimLeft(msg, "/")

	for _, ans := range am.answerers {
		if q := ans.Check(user, msg1, repl); q.Matched {
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

func LongestPrefix(s string, prefixes ...string) string {
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

func HasPrefix(s string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}

	return false
}
