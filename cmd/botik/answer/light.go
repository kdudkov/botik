package answer

import (
	"botik/internal/api"
	"fmt"
	"log/slog"
	"strings"
)

const (
	ON          = "ON"
	OFF         = "OFF"
	NIGHT       = "NIGHT"
	DAY         = "DAY"
	NOBODY_HOME = "NOBODY_HOME"
	STATUS      = "STATUS"
)

type Light struct {
	mahno  api.MahnoApi
	logger *slog.Logger
}

func NewLight(logger *slog.Logger, host string) *Light {
	return &Light{
		mahno:  api.NewMahnoApi(host),
		logger: logger.With("logger", "light"),
	}
}

func (l *Light) Check(user string, msg string, repl string) (q *Q) {
	m := strings.ToLower(msg)
	q = &Q{Msg: msg, User: user}

	words := q.Words()

	if HasPrefix(m, "light") {
		if len(words) == 1 {
			q.Matched = true
			q.Prefix = ""
			q.Cmd = STATUS
			return
		}
	}

	if s := LongestPrefix(m, "включи", "включить"); s != "" {
		q.Matched = true
		q.Prefix = s
		q.Cmd = ON
		if IndexOf(words, "весь", "везде", "улице", "уличный", "снаружи") > -1 {
			q.Payload = "lights_out"
		}
		return
	}

	if s := LongestPrefix(m, "выключи", "выключить"); s != "" {
		q.Matched = true
		q.Prefix = s
		q.Cmd = OFF
		if IndexOf(words, "весь", "везде") > -1 {
			q.Payload = "lights"
		}
		if IndexOf(words, "улице", "уличный", "снаружи") > -1 {
			q.Payload = "lights_out"
		}
		return
	}

	if s := LongestPrefix(m, "спать", "ночной режим", "ночь"); s != "" {
		q.Matched = true
		q.Prefix = s
		q.Cmd = NIGHT
		return
	}

	if s := LongestPrefix(m, "день"); s != "" {
		q.Matched = true
		q.Prefix = s
		q.Cmd = DAY
		return
	}

	if s := LongestPrefix(m, "жди", "все ушли", "один дома"); s != "" {
		q.Matched = true
		q.Prefix = s
		q.Cmd = NOBODY_HOME
		return
	}

	if s := LongestPrefix(m, "свет", "статус"); s != "" {
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
		if q.Payload != "" {
			l.logger.Info("lights on for " + q.Payload)
			err := l.mahno.GroupCommand(q.Payload, ON)
			if err != nil {
				return TextAnswer(fmt.Sprintf("ошибка: %s", err.Error()))
			}

			return TextAnswer("включаю свет")
		} else {
			target := getTarget(words)
			if target == "" {
				return TextAnswer(fmt.Sprintf("не понимаю %s", q.Msg))
			}

			l.logger.Info("light ON to " + target)
			err := l.mahno.ItemCommand(target, ON)

			if err != nil {
				return TextAnswer(fmt.Sprintf("ошибка: %s", err.Error()))
			}
			return TextAnswer(fmt.Sprintf("включаю %s", target))
		}

	case OFF:
		if q.Payload != "" {
			l.logger.Info("lights off for " + q.Payload)
			err := l.mahno.GroupCommand(q.Payload, OFF)
			if err != nil {
				return TextAnswer(fmt.Sprintf("ошибка: %s", err.Error()))
			}

			return TextAnswer("включаю свет")
		} else {
			target := getTarget(words)
			if target == "" {
				return TextAnswer(fmt.Sprintf("не понимаю %s", q.Msg))
			}

			l.logger.Info("light OFF to " + target)
			err := l.mahno.ItemCommand(target, OFF)

			if err != nil {
				return TextAnswer(fmt.Sprintf("ошибка: %s", err.Error()))
			}
			return TextAnswer(fmt.Sprintf("выключаю %s", target))
		}

	case DAY:
		l.logger.Info("home mode day")
		err := l.mahno.SetItemState("home_mode", "day")

		if err != nil {
			return TextAnswer(fmt.Sprintf("ошибка: %s", err.Error()))
		}
		return TextAnswer("дневной режим")

	case NIGHT:
		l.logger.Info("home mode night")
		err := l.mahno.SetItemState("home_mode", "night")

		if err != nil {
			return TextAnswer(fmt.Sprintf("ошибка: %s", err.Error()))
		}
		return TextAnswer("ночной режим")

	case NOBODY_HOME:
		l.logger.Info("home mode nobody")
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

		sb := new(strings.Builder)
		sb.WriteString("свет:\n")
		for _, i := range res {
			if IndexOf(i.Groups, "light") > -1 || i.Name == "home_mode" {
				fmt.Fprintf(sb, "\n%s %s %s %s", i.Name, i.HumanName, i.Room, i.FormattedValue)
			}
		}

		return TextAnswer(sb.String())

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
