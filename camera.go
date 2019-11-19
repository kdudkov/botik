package main

import (
	"go.uber.org/zap"
	"strings"
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
	cam.logger = logger
}

func (cam *Camera) Check(user string, msg string) (q *Q) {
	q = &Q{Msg: msg, User: strings.ToLower(user)}

	words := q.words()

	if IsInArray(words[0], []string{"камера", "cam"}) {
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
