package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

const (
	ON          = "ON"
	OFF         = "OFF"
	ALL_ON      = "ALL_ON"
	ALL_OFF     = "ALL_OFF"
	NIGHT       = "NIGHT"
	DAY         = "DAY"
	NOBODY_HOME = "NOBODY_HOME"
	STATUS      = "STATUS"
)

type Light struct {
	mahno  MahnoApi
	logger *zap.SugaredLogger
}

func init() {
	if err := RegisterAnswer("light", NewLight()); err != nil {
		panic(err.Error())
	}
}

func NewLight() *Light {
	client := &http.Client{Timeout: time.Second * 5}
	return &Light{mahno: &MahnoHttpApi{host: "oh.home", client: client}}
}

func (l *Light) AddLogger(logger *zap.SugaredLogger) {
	l.logger = logger
	l.mahno.SetLogger(logger)
}

func (x *Light) Logf(level int8, template string, args ...interface{}) {
	Logf(x.logger, level, template, args)
}

func (x *Light) Logw(level int8, template string, args ...interface{}) {
	Logw(x.logger, level, template, args)
}

func (l *Light) Check(user string, msg string) (q *Q) {
	m := strings.ToLower(msg)
	q = &Q{Msg: msg, User: user}

	words := q.words()

	if s := hasPrefix(m, "включи", "включить"); s != "" {
		q.Matched = true
		q.Prefix = s
		q.Cmd = ON
		if indexOf(words, "весь", "везде") > -1 {
			q.Cmd = ALL_ON
		}
		return
	}

	if s := hasPrefix(m, "выключи", "выключить"); s != "" {
		q.Matched = true
		q.Prefix = s
		q.Cmd = OFF
		if indexOf(words, "весь", "везде") > -1 {
			q.Cmd = ALL_OFF
		}
		return
	}

	if s := hasPrefix(m, "спать", "ночной режим", "ночь"); s != "" {
		q.Matched = true
		q.Prefix = s
		q.Cmd = NIGHT
		return
	}

	if s := hasPrefix(m, "день"); s != "" {
		q.Matched = true
		q.Prefix = s
		q.Cmd = DAY
		return
	}

	if s := hasPrefix(m, "жди", "все ушли", "один дома"); s != "" {
		q.Matched = true
		q.Prefix = s
		q.Cmd = NOBODY_HOME
		return
	}

	if s := hasPrefix(m, "свет", "статус"); s != "" {
		q.Matched = true
		q.Prefix = s
		q.Cmd = STATUS
		return
	}

	q.Matched = false
	return
}

func (l *Light) Process(q *Q) string {
	words := q.words()

	switch q.Cmd {
	case ON:
		target := getTarget(words)
		if target == "" {
			return fmt.Sprintf("не понимаю %s", q.Msg)
		}

		l.Logf(LOG_INFO, "light %s ON", target)
		err := l.mahno.ItemCommand(target, ON)

		if err != nil {
			return fmt.Sprintf("ошибка: %s", err.Error())
		}
		return fmt.Sprintf("включаю %s", target)
	case OFF:
		target := getTarget(words)
		if target == "" {
			return fmt.Sprintf("не понимаю %s", q.Msg)
		}

		l.Logf(LOG_INFO, "light %s OFF", target)
		err := l.mahno.ItemCommand(target, OFF)

		if err != nil {
			return fmt.Sprintf("ошибка: %s", err.Error())
		}
		return fmt.Sprintf("выключаю %s", target)

	case ALL_ON:
		l.Logf(LOG_INFO, "all lights on")
		allLight(l.mahno, ON)
		return "включаю весь свет"

	case ALL_OFF:
		l.Logf(LOG_INFO, "all lights off")
		allLight(l.mahno, OFF)
		return "выключаю весь свет"

	case DAY:
		l.Logf(LOG_INFO, "home mode day")
		err := l.mahno.SetItemState("home_mode", "day")

		if err != nil {
			return fmt.Sprintf("ошибка: %s", err.Error())
		}
		return "дневной режим"

	case NIGHT:
		l.Logf(LOG_INFO, "home mode night")
		allLight(l.mahno, OFF)
		err := l.mahno.SetItemState("home_mode", "night")

		if err != nil {
			return fmt.Sprintf("ошибка: %s", err.Error())
		}
		return "ночной режим"

	case NOBODY_HOME:
		l.Logf(LOG_INFO, "home mode nobody")
		err := l.mahno.SetItemState("home_mode", "nobody_home")

		if err != nil {
			return fmt.Sprintf("ошибка: %s", err.Error())
		}
		return "режим отсутствия"

	case STATUS:
		res, err := l.mahno.AllItems()
		if err != nil {
			return fmt.Sprintf("ошибка: %s", err.Error())
		}

		ans := ""
		for _, i := range res {
			var pr bool = false

			for _, t := range i.Tags {
				if t == "light" {
					pr = true
					break
				}
			}

			if (pr || i.Name == "home_mode") && i.Formatted != nil {
				ans = fmt.Sprintf("%s\n%s %s", ans, i.Name, i.Formatted)
			}
		}
		return ans

	default:
		return fmt.Sprintf("не понимаю, что значит %s", q.Msg)
	}
}

func getTarget(words []string) string {
	if i := indexOf(words, "в", "на", "у"); i > -1 {
		if len(words) <= i+1 {
			return ""
		}
		return getItemName(words[i+1])
	}
	return ""
}

func getItemName(s string) string {
	var TARGET = map[string][]string{
		"max":            {"максиной", "макса"},
		"kitchen":        {"кухня", "кухне"},
		"light_room":     {"комнате", "спальне"},
		"light_corridor": {"коридоре", "корридоре", "прихожей", "прихожая"},
	}

	for k, v := range TARGET {
		for _, v1 := range v {
			if s == v1 {
				return k
			}
		}
	}

	return ""
}

func allLight(mahno MahnoApi, cmd string) {
	for _, x := range []string{"light_room", "light_corridor", "s20_1", "s20_2"} {
		mahno.ItemCommand(x, cmd)
	}
}
