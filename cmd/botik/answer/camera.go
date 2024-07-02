package answer

import (
	"botik/internal/util"
	"log/slog"
	"strings"
)

type Camera struct {
	img    string
	logger *slog.Logger
}

func NewCamera(logger *slog.Logger, img string) *Camera {
	return &Camera{
		img:    img,
		logger: logger.With("logger", "camera"),
	}
}

func (cam *Camera) Check(user string, msg string, repl string) (q *Q) {
	q = &Q{Msg: msg, User: strings.ToLower(user)}

	words := q.Words()

	if util.IsInArray(words[0], "камера", "cam") {
		q.Matched = true
		q.Prefix = words[0]
		q.Cmd = "camera"
		return
	}

	return
}

func (cam *Camera) Process(q *Q) *Answer {
	switch q.Cmd {
	case "camera":
		return PhotoAnswer(cam.img)

	default:
		return TextAnswer("invalid command " + q.Cmd)
	}
}
