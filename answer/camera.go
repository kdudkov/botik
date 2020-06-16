package answer

import (
	"strings"

	"go.uber.org/zap"

	"botik/util"
)

type Camera struct {
	img    string
	logger *zap.SugaredLogger
}

func NewCamera() *Camera {
	return &Camera{img: "/home/motion/lastsnap.jpg"}
}

func init() {
	if err := RegisterAnswer("camera", NewCamera()); err != nil {
		panic(err.Error())
	}
}

func (cam *Camera) AddLogger(logger *zap.SugaredLogger) {
	cam.logger = logger.Named("camera")
}

func (cam *Camera) Check(user string, msg string) (q *Q) {
	q = &Q{Msg: msg, User: strings.ToLower(user)}

	words := q.Words()

	if util.IsInArray(words[0], []string{"камера", "cam"}) {
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
