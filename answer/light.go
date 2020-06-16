package answer

import (
	"fmt"
	"strings"

	"go.uber.org/zap"

	"botik/api"
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
	mahno  api.MahnoApi
	logger *zap.SugaredLogger
}

func init() {
	if err := RegisterAnswer("light", NewLight()); err != nil {
		panic(err.Error())
	}
}

func NewLight() *Light {
	return &Light{mahno: api.NewMahnoApi()}
}

func (l *Light) AddLogger(logger *zap.SugaredLogger) {
	l.logger = logger.Named("light")
	l.mahno.SetLogger(logger.Named("mahno_api"))
}

func (l *Light) Check(user string, msg string) (q *Q) {
	m := strings.ToLower(msg)
	q = &Q{Msg: msg, User: user}

	words := q.Words()

	if strings.HasPrefix(m, "light") {
		if len(words) == 1 {
			q.Matched = true
			q.Prefix = ""
			q.Cmd = STATUS
			return
		}
	}

	if s := HasPrefix(m, "включи", "включить"); s != "" {
		q.Matched = true
		q.Prefix = s
		q.Cmd = ON
		if IndexOf(words, "весь", "везде") > -1 {
			q.Cmd = ALL_ON
		}
		return
	}

	if s := HasPrefix(m, "выключи", "выключить"); s != "" {
		q.Matched = true
		q.Prefix = s
		q.Cmd = OFF
		if IndexOf(words, "весь", "везде") > -1 {
			q.Cmd = ALL_OFF
		}
		return
	}

	if s := HasPrefix(m, "спать", "ночной режим", "ночь"); s != "" {
		q.Matched = true
		q.Prefix = s
		q.Cmd = NIGHT
		return
	}

	if s := HasPrefix(m, "день"); s != "" {
		q.Matched = true
		q.Prefix = s
		q.Cmd = DAY
		return
	}

	if s := HasPrefix(m, "жди", "все ушли", "один дома"); s != "" {
		q.Matched = true
		q.Prefix = s
		q.Cmd = NOBODY_HOME
		return
	}

	if s := HasPrefix(m, "свет", "статус"); s != "" {
		q.Matched = true
		q.Prefix = s
		q.Cmd = STATUS
		return
	}

	q.Matched = false
	return
}

func (l *Light) Process(q *Q) *Answer {
	words := q.Words()

	switch q.Cmd {
	case ON:
		target := getTarget(words)
		if target == "" {
			return TextAnswer(fmt.Sprintf("не понимаю %s", q.Msg))
		}

		l.logger.Infof("light %s ON", target)
		err := l.mahno.ItemCommand(target, ON)

		if err != nil {
			return TextAnswer(fmt.Sprintf("ошибка: %s", err.Error()))
		}
		return TextAnswer(fmt.Sprintf("включаю %s", target))
	case OFF:
		target := getTarget(words)
		if target == "" {
			return TextAnswer(fmt.Sprintf("не понимаю %s", q.Msg))
		}

		l.logger.Infof("light %s OFF", target)
		err := l.mahno.ItemCommand(target, OFF)

		if err != nil {
			return TextAnswer(fmt.Sprintf("ошибка: %s", err.Error()))
		}
		return TextAnswer(fmt.Sprintf("выключаю %s", target))

	case ALL_ON:
		l.logger.Infof("all lights on")
		allLight(l.mahno, ON)
		return TextAnswer("включаю весь свет")

	case ALL_OFF:
		l.logger.Infof("all lights off")
		allLight(l.mahno, OFF)
		return TextAnswer("выключаю весь свет")

	case DAY:
		l.logger.Infof("home mode day")
		err := l.mahno.SetItemState("home_mode", "day")

		if err != nil {
			return TextAnswer(fmt.Sprintf("ошибка: %s", err.Error()))
		}
		return TextAnswer("дневной режим")

	case NIGHT:
		l.logger.Infof("home mode night")
		allLight(l.mahno, OFF)
		err := l.mahno.SetItemState("home_mode", "night")

		if err != nil {
			return TextAnswer(fmt.Sprintf("ошибка: %s", err.Error()))
		}
		return TextAnswer("ночной режим")

	case NOBODY_HOME:
		l.logger.Infof("home mode nobody")
		err := l.mahno.SetItemState("home_mode", "nobody_home")

		if err != nil {
			return TextAnswer(fmt.Sprintf("ошибка: %s", err.Error()))
		}
		return TextAnswer("режим отсутствия")

	case STATUS:
		res, err := l.mahno.AllItems()
		if err != nil {
			return TextAnswer(fmt.Sprintf("ошибка: %s", err.Error()))
		}

		ans := "свет:\n"
		for _, i := range res {
			if IndexOf(i.Tags, "light") > -1 || i.Name == "home_mode" {
				ans = fmt.Sprintf("%s\n%s %s", ans, i.Name, i.Formatted)
			}
		}
		return TextAnswer(ans)

	default:
		return TextAnswer(fmt.Sprintf("не понимаю, что значит %s", q.Msg))
	}
}

func getTarget(words []string) string {
	if i := IndexOf(words, "в", "на", "у"); i > -1 {
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

func allLight(mahno api.MahnoApi, cmd string) {
	for _, x := range []string{"light_room", "light_corridor", "s20_1", "s20_2"} {
		mahno.ItemCommand(x, cmd)
	}
}
