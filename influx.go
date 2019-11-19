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

func getNano() int64 {
	return time.Now().Round(time.Minute).UnixNano()
}

func (x *Influx) Logf(level int8, template string, args ...interface{}) {
	Logf(x.logger, level, template, args)
}

func (x *Influx) Logw(level int8, template string, args ...interface{}) {
	Logw(x.logger, level, template, args)
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

	if IsInArray(words[0], []string{"давление", "bp"}) {
		q.Matched = true
		q.Prefix = words[0]
		q.Cmd = BP
		return
	}

	if IsInArray(words[0], []string{"вес", "weight"}) {
		q.Matched = true
		q.Prefix = words[0]
		q.Cmd = WEIGHT
		return
	}

	return
}

func (i *Influx) Process(q *Q) *Answer {
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
				return TextAnswer(res)
			} else {
				i.Logf(LOG_ERROR, "error getting pressure %s", err.Error())
				return TextAnswer(err.Error())
			}
		}

		if len(words) >= 3 {
			var note string

			sys, err := strconv.ParseInt(words[1], 10, 16)
			if err != nil {
				i.Logf(LOG_ERROR, "parse error %s", err.Error())
				return TextAnswer(err.Error())
			}
			dia, err := strconv.ParseInt(words[2], 10, 16)
			if err != nil {
				i.Logf(LOG_ERROR, "parse error %s", err.Error())
				return TextAnswer(err.Error())
			}
			if len(words) > 3 {
				note = strings.Join(words[3:], " ")
			}
			if err := i.sendBP(q.User, uint16(sys), uint16(dia), note); err != nil {
				i.Logf(LOG_ERROR, "send error %s", err.Error())
				return TextAnswer("ошибка " + err.Error())
			} else {
				return TextAnswer(fmt.Sprintf("записано давление %d/%d", sys, dia))
			}

		}

		return TextAnswer("использование: \"давление\" или \"давление 120 80\"")

	case WEIGHT:
		if len(words) == 2 {
			w, err := strconv.ParseFloat(strings.ReplaceAll(words[1], ",", "."), 10)
			if err != nil {
				i.Logf(LOG_ERROR, "parse error %s", err.Error())
				return TextAnswer(err.Error())
			}
			if err := i.sendWeight(q.User, w, 0); err != nil {
				i.Logf(LOG_ERROR, "send error %s", err.Error())
				return TextAnswer("ошибка " + err.Error())
			} else {
				return TextAnswer(fmt.Sprintf("записан вес %.1f", w))
			}

		}
		return TextAnswer("использование: \"вес \" или \"вес 95.2\"")

	default:
		return TextAnswer("invalid command " + q.Cmd)
	}
}

func (p *Pressure) String() string {
	return fmt.Sprintf("%s %d/%d", FormatTime(p.time), p.sys, p.dia)
}

func (i *Influx) sendBP(name string, sys uint16, dia uint16, note string) error {
	q := fmt.Sprintf("pressure,name=%s sys=%d,dia=%d", name, sys, dia)

	if note != "" {
		q += ",note=\"" + note + "\""
	}

	q += fmt.Sprintf(" %d", getNano())
	return i.api.Send(q)
}

func (i *Influx) sendWeight(name string, w float64, fat float64) error {
	q := fmt.Sprintf("weight,name=%s weight=%f", name, w)
	if fat != 0 {
		q += fmt.Sprintf(",fat=%f", fat)
	}

	q += fmt.Sprintf(" %d", getNano())
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
