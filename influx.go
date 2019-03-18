package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

const (
	BP     = "bp"
	WEIGHT = "weight"
)

type Influx struct {
	api    InfluxHttpApi
	logger *zap.SugaredLogger
	days   uint16
}

func NewInflux() *Influx {
	client := &http.Client{Timeout: time.Second * 5}
	return &Influx{api: &InfluxApi{host: "oh.home:8086", db: "bio", client: client}, days: 10}
}

func init() {
	if err := RegisterAnswer("influx", NewInflux()); err != nil {
		panic(err.Error())
	}
}

func (i *Influx) Debugf(template string, args ...interface{}) {
	if i.logger != nil {
		i.logger.Debugf(template, args)
	}
}

func (i *Influx) Infof(template string, args ...interface{}) {
	if i.logger != nil {
		i.logger.Infof(template, args)
	}
}

func (i *Influx) Warnf(template string, args ...interface{}) {
	if i.logger != nil {
		i.logger.Warnf(template, args)
	}
}

func (i *Influx) Errorf(template string, args ...interface{}) {
	if i.logger != nil {
		i.logger.Errorf(template, args)
	}
}

func (i *Influx) Debugw(template string, args ...interface{}) {
	if i.logger != nil {
		i.logger.Debugw(template, args)
	}
}

func (i *Influx) Infow(template string, args ...interface{}) {
	if i.logger != nil {
		i.logger.Infow(template, args)
	}
}

func (i *Influx) Warnw(template string, args ...interface{}) {
	if i.logger != nil {
		i.logger.Warnw(template, args)
	}
}

func (i *Influx) Errorw(template string, args ...interface{}) {
	if i.logger != nil {
		i.logger.Errorw(template, args)
	}
}

type Pressure struct {
	time time.Time
	sys  uint16
	dia  uint16
}

func (i *Influx) AddLogger(logger *zap.SugaredLogger) {
	i.logger = logger
}

func (i *Influx) Check(user string, msg string) (q *Q) {
	q = &Q{Msg: msg, User: strings.ToLower(user)}

	words := q.words()

	if words[0] == "давление" {
		q.Matched = true
		q.Prefix = "давление"
		q.Cmd = BP
		return
	}

	if words[0] == "вес" {
		q.Matched = true
		q.Prefix = "вес"
		q.Cmd = WEIGHT
		return
	}

	return
}

func (i *Influx) Process(q *Q) string {
	words := q.words()

	switch q.Cmd {
	case BP:
		if len(words) == 1 {
			p, err := i.getPressure(q.User, 50)
			if err == nil {
				res := fmt.Sprintf("Давление за последние %d дней для %s\n\n", i.days, q.User)
				for _, pp := range p {
					res += pp.String() + "\n"
				}
				return res
			} else {
				i.Errorf("error getting pressure %s", err.Error())
				return err.Error()
			}
		}

		if len(words) == 3 {
			sys, err := strconv.ParseInt(words[1], 10, 16)
			if err != nil {
				i.Errorf("parse error %s", err.Error())
				return err.Error()
			}
			dia, err := strconv.ParseInt(words[2], 10, 16)
			if err != nil {
				i.Errorf("parse error %s", err.Error())
				return err.Error()
			}
			if err := i.sendBP(q.User, uint16(sys), uint16(dia)); err != nil {
				i.Errorf("send error %s", err.Error())
				return "ошибка " + err.Error()
			} else {
				return fmt.Sprintf("записано давление %d/%d", sys, dia)
			}

		}

		return "использование: \"давление\" или \"давление 120 80\""

	case WEIGHT:
		if len(words) == 2 {
			w, err := strconv.ParseFloat(strings.ReplaceAll(words[1], ",", "."), 10)
			if err != nil {
				i.Errorf("parse error %s", err.Error())
				return err.Error()
			}
			if err := i.sendWeight(q.User, w, 0); err != nil {
				i.Errorf("send error %s", err.Error())
				return "ошибка " + err.Error()
			} else {
				return fmt.Sprintf("записан вес %.1f", w)
			}

		}
		return "использование: \"вес \" или \"вес 95.2\""

	default:
		return "invalid command " + q.Cmd
	}
}

func (p *Pressure) String() string {
	return fmt.Sprintf("%s %d/%d", FormatTime(p.time), p.sys, p.dia)
}

func (i *Influx) sendBP(name string, sys uint16, dia uint16) error {
	q := fmt.Sprintf("pressure,name=%s sys=%d,dia=%d %d", name, sys, dia, time.Now().UnixNano())
	return i.api.Send(q)
}

func (i *Influx) sendWeight(name string, w float64, fat float64) error {
	q := fmt.Sprintf("weight,name=%s weight=%f", name, w)
	if fat != 0 {
		q += fmt.Sprintf(",fat=%f", fat)
	}

	q += fmt.Sprintf(" %d", time.Now().UnixNano())
	return i.api.Send(q)
}

func (i *Influx) getPressure(name string, limit int) ([]Pressure, error) {
	res := make([]Pressure, 0)
	q := fmt.Sprintf("select time, sys, dia from pressure where \"name\"='%s' and time > now() - %dd limit %d", name, i.days, limit)

	err := i.api.GetData(q, func(s string) {
		if p, ok := parseSring(s); ok {
			res = append(res, p)
		}
	})

	return res, err
}

func parseSring(s string) (Pressure, bool) {
	p := Pressure{}
	ss := strings.Split(s, ",")

	if ss[0] == "pressure" {
		d, err := strconv.ParseInt(ss[2], 10, 64)
		if err != nil {
			return p, false
		}
		p.time = time.Unix(d, 0)

		d, err = strconv.ParseInt(ss[3], 10, 16)
		if err != nil {
			return p, false
		}
		p.sys = uint16(d)

		d, err = strconv.ParseInt(ss[4], 10, 16)
		if err != nil {
			return p, false
		}
		p.dia = uint16(d)
		return p, true
	}

	return p, false
}
