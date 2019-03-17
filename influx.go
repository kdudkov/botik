package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

type Influx struct {
	host   string
	db     string
	days   uint8
	client *http.Client
	logger *zap.SugaredLogger
}

func NewInflux() *Influx {
	client := &http.Client{Timeout: time.Second * 5}
	return &Influx{host: "oh.home:8086", db: "bio", days: 10, client: client}
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
		q.Cmd = ""
		return
	}

	return
}

func (i *Influx) Process(q *Q) string {
	words := q.words()

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
		if err := i.send(q.User, uint16(sys), uint16(dia)); err != nil {
			i.Errorf("send error %s", err.Error())
			return "ошибка " + err.Error()
		} else {
			return fmt.Sprintf("записано давление %d/%d", sys, dia)
		}

	}

	return "использование: \"давление\" или \"давление 120 80\""
}

func (p *Pressure) String() string {
	return fmt.Sprintf("%s %d/%d", FormatTime(p.time), p.sys, p.dia)
}

func (i *Influx) send(name string, sys uint16, dia uint16) error {
	url := fmt.Sprintf("http://%s/write?db=%s", i.host, i.db)

	q := fmt.Sprintf("pressure,name=%s sys=%d,dia=%d %d", name, sys, dia, time.Now().UnixNano())
	i.Debugf("write query %s", q)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(q)))

	if err != nil {
		return err
	}

	resp, err := i.client.Do(req)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		s, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		} else {
			return fmt.Errorf("%s", s)
		}
	}

	return nil
}

func (i *Influx) getPressure(name string, limit int) ([]Pressure, error) {
	res := make([]Pressure, 0)
	q := fmt.Sprintf("select time, sys, dia from pressure where \"name\"='%s' and time > now() - %dd limit %d", name, i.days, limit)
	params := url.Values{}
	params.Add("epoch", "s")
	params.Add("db", i.db)
	params.Add("q", q)

	url := fmt.Sprintf("http://%s/query?%s", i.host, params.Encode())

	fmt.Println(url)

	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return res, err
	}

	req.Header.Set("Accept", "application/csv")

	resp, err := i.client.Do(req)

	if err != nil {
		return res, err
	}

	defer resp.Body.Close()

	r := bufio.NewReader(resp.Body)

	for true {
		l, _, err := r.ReadLine()
		if l != nil {
			if p, ok := parseSring(string(l)); ok {
				res = append(res, p)
			}
		}
		if err != nil {
			break
		}
	}

	return res, nil
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
