package answer

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"botik/api"
	"botik/util"
)

const (
	BP     = "bp"
	WEIGHT = "weight"
	DB     = "bio"
)

type Influx struct {
	api    api.InfluxHttpApi
	logger *slog.Logger
	days   uint16
}

func NewInflux() *Influx {
	client := &http.Client{Timeout: time.Second * 5}
	return &Influx{
		api:    api.NewInfluxApi("192.168.0.1:8086", client),
		days:   10,
		logger: slog.Default().With("logger", "influx"),
	}
}

func init() {
	if err := RegisterAnswer("influx", NewInflux()); err != nil {
		panic(err.Error())
	}
}

func getNano() int64 {
	return time.Now().Round(time.Minute).UnixNano()
}

type Pressure struct {
	Time time.Time
	Sys  uint16
	Dia  uint16
}

func (i *Influx) Check(user string, msg string) (q *Q) {
	q = &Q{Msg: msg, User: strings.ToLower(user)}

	words := q.Words()

	if util.IsInArray(words[0], []string{"давление", "bp"}) {
		q.Matched = true
		q.Prefix = words[0]
		q.Cmd = BP
		return
	}

	if util.IsInArray(words[0], []string{"вес", "weight"}) {
		q.Matched = true
		q.Prefix = words[0]
		q.Cmd = WEIGHT
		return
	}

	return
}

func (i *Influx) Process(q *Q) *Answer {
	words := q.Words()

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
				i.logger.Error("error getting pressure", "error", err)
				return TextAnswer(err.Error())
			}
		}

		if len(words) >= 3 {
			var note string

			sys, err := strconv.ParseInt(words[1], 10, 16)
			if err != nil {
				i.logger.Error("parse error", "error", err)
				return TextAnswer(err.Error())
			}
			dia, err := strconv.ParseInt(words[2], 10, 16)
			if err != nil {
				i.logger.Error("parse error", "error", err)
				return TextAnswer(err.Error())
			}
			if len(words) > 3 {
				note = strings.Join(words[3:], " ")
			}
			if err := i.sendBP(q.User, uint16(sys), uint16(dia), note); err != nil {
				i.logger.Error("send error", "error", err)
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
				i.logger.Error("parse error", "error", err)
				return TextAnswer(err.Error())
			}
			if err := i.sendWeight(q.User, w, 0); err != nil {
				i.logger.Error("send error", "error", err)
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
	return fmt.Sprintf("%s %d/%d", util.FormatTime(p.Time), p.Sys, p.Dia)
}

func (i *Influx) sendBP(name string, sys uint16, dia uint16, note string) error {
	q := fmt.Sprintf("pressure,name=%s sys=%d,dia=%d", name, sys, dia)

	if note != "" {
		q += ",note=\"" + note + "\""
	}

	q += fmt.Sprintf(" %d", getNano())
	return i.api.Send(DB, q)
}

func (i *Influx) sendWeight(name string, w float64, fat float64) error {
	q := fmt.Sprintf("weight,name=%s weight=%f", name, w)
	if fat != 0 {
		q += fmt.Sprintf(",fat=%f", fat)
	}

	q += fmt.Sprintf(" %d", getNano())
	return i.api.Send(DB, q)
}

func (i *Influx) getPressure(name string, limit int) ([]Pressure, error) {
	q := fmt.Sprintf("select time, sys, dia from pressure where \"name\"='%s' and time > now() - %dd limit %d", name, i.days, limit)

	r, err := i.api.GetSingleSeries(DB, q)
	if err != nil {
		return nil, err
	}

	res := make([]Pressure, 0)
	for _, record := range r {
		if p, err := MapToPressure(record); err == nil {
			res = append(res, *p)
		}
	}
	return res, nil
}

func MapToPressure(record map[string]interface{}) (*Pressure, error) {
	if util.HasAllKeys(record, "time", "sys", "dia") {
		p := Pressure{}
		if v, ok := record["time"].(time.Time); ok {
			p.Time = v
		} else {
			return nil, errors.New("bad time field")
		}
		if v, ok := record["sys"].(float64); ok {
			p.Sys = uint16(v)
		} else {
			return nil, errors.New("bad sys field")
		}
		if v, ok := record["dia"].(float64); ok {
			p.Dia = uint16(v)
		} else {
			return nil, errors.New("bad time field")
		}
		return &p, nil
	}
	return nil, errors.New("not all fields")
}
