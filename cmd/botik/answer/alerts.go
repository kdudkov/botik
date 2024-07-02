package answer

import (
	"botik/cmd/botik/alert"
	"botik/internal/util"
	"fmt"
	"log/slog"
	"strings"
)

type Alerts struct {
	logger *slog.Logger
	am     *alert.AlertManager
}

func NewAlerts(logger *slog.Logger, am *alert.AlertManager) *Alerts {
	return &Alerts{
		logger: logger.With("logger", "alerts"),
		am:     am,
	}
}

func (cam *Alerts) Check(user string, msg string, repl string) (q *Q) {
	q = &Q{Msg: msg, User: strings.ToLower(user)}

	words := q.Words()

	if util.IsInArray(words[0], "mute", "выкл") {
		q.Matched = true
		q.Prefix = words[0]
		q.Cmd = "mute"
		q.Repl = repl
		return
	}

	if util.IsInArray(words[0], "alerts", "алерты") {
		q.Matched = true
		q.Prefix = words[0]
		q.Cmd = "alerts"
		q.Repl = repl
		return
	}

	return
}

func (cam *Alerts) Process(q *Q) *Answer {
	switch q.Cmd {
	case "mute":
		var id string
		for _, s := range strings.Split(q.Repl, "\n") {
			if strings.HasPrefix(s, "id:") {
				id = s[3:]
				break
			}
		}

		if id == "" {
			return nil
		}

		cam.logger.Info("mute id " + id)

		var ans string
		cam.am.Range(func(ar *alert.AlertRec) bool {
			if ar.Alert().ID == id {
				ar.Mute()
				ans = fmt.Sprintf("alert %s is muted", ar.Alert().Name)
				return false
			}

			return true
		})

		if ans != "" {
			return TextAnswer(ans)
		} else {
			return TextAnswer("alert with is not found")
		}

	case "alerts":
		var ans string
		cam.am.Range(func(ar *alert.AlertRec) bool {
			ans += ar.String() + "\n"

			return true
		})

		if ans != "" {
			return TextAnswer(ans)
		} else {
			return TextAnswer("нет активных алертов")
		}

	default:
		return TextAnswer("invalid command " + q.Cmd)
	}
}
