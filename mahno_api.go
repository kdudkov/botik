package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"go.uber.org/zap"
)

const (
	host = "oh.home"
)

type Item struct {
	ClassName string      `json:"class"`
	Name      string      `json:"name"`
	TTL       int         `json:"ttl"`
	Value     interface{} `json:"value"`
	Value2    interface{} `json:"_value"`
	//Age       time.Duration `json:"age"`
	//CheckAge  time.Duration `json:"check_age"`
	//Checked   time.Time     `json:"checked"`
	//Changed   time.Time     `json:"changed"`
	Tags      []string    `json:"tags"`
	Formatted interface{} `json:"formatted,omitempty"`
	HumanName string      `json:"h_name,omitempty"`
	UI        bool        `json:"ui"`
}

type MahnoApi struct {
	host   string
	client *http.Client
	logger *zap.SugaredLogger
}

func NewMahnoApi() *MahnoApi {
	client := &http.Client{Timeout: time.Second * 5}
	return &MahnoApi{host: "oh.home", client: client}
}

func (x *MahnoApi) Debugf(template string, args ...interface{}) {
	if x.logger != nil {
		x.logger.Debugf(template, args)
	}
}

func (x *MahnoApi) Infof(template string, args ...interface{}) {
	if x.logger != nil {
		x.logger.Infof(template, args)
	}
}

func (x *MahnoApi) Warnf(template string, args ...interface{}) {
	if x.logger != nil {
		x.logger.Warnf(template, args)
	}
}

func (x *MahnoApi) Errorf(template string, args ...interface{}) {
	if x.logger != nil {
		x.logger.Errorf(template, args)
	}
}

func (x *MahnoApi) Debugw(template string, args ...interface{}) {
	if x.logger != nil {
		x.logger.Debugw(template, args)
	}
}

func (x *MahnoApi) Infow(template string, args ...interface{}) {
	if x.logger != nil {
		x.logger.Infow(template, args)
	}
}

func (x *MahnoApi) Warnw(template string, args ...interface{}) {
	if x.logger != nil {
		x.logger.Warnw(template, args)
	}
}

func (x *MahnoApi) Errorw(template string, args ...interface{}) {
	if x.logger != nil {
		x.logger.Errorw(template, args)
	}
}

func (m *MahnoApi) doReqReader(method string, path string, data string) (io.ReadCloser, error) {
	url := "http://" + m.host + path

	m.Debugf("url: %s", url)

	var req *http.Request
	var err error

	if data != "" {
		req, err = http.NewRequest(method, url, bytes.NewBuffer([]byte(data)))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func (m *MahnoApi) doReq(method string, path string, data string) ([]byte, error) {
	b, err := m.doReqReader(method, path, data)
	if err != nil {
		return nil, err
	}
	defer b.Close()

	res, err := ioutil.ReadAll(b)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (m *MahnoApi) ItemCommand(item string, cmd string) error {
	body, err := m.doReq("POST", "/items/"+item, cmd)

	if err != nil {
		m.Errorf("error talking to mahno: %s", err.Error())
		return err
	}

	m.Infof(fmt.Sprintf("body: %s", body))
	return nil
}

func (m *MahnoApi) SetItemState(item string, val string) error {
	body, err := m.doReq("POST", "/items/"+item, val)

	if err != nil {
		m.Errorf("error talking to mahno: %s", err.Error())
		return err
	}

	m.Infof(fmt.Sprintf("body: %s", body))
	return nil
}

func (m *MahnoApi) AllItems() ([]Item, error) {
	body, err := m.doReqReader("GET", "/items", "")

	if err != nil {
		m.Errorf("error talking to mahno: %s", err.Error())
		return nil, err
	}

	defer body.Close()
	var res []Item
	decoder := json.NewDecoder(body)

	if err = decoder.Decode(&res); err != nil {
		m.Errorf("can't decode: %v", err)
		return nil, err
	}
	return res, nil
}
