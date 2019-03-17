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
	logger *zap.SugaredLogger
}

func (m *MahnoApi) Info(msg string, fields ...zap.Field) {
	if m.logger != nil {
		m.logger.Info(msg, fields)
	}
}

func (m *MahnoApi) Warn(msg string, fields ...zap.Field) {
	if m.logger != nil {
		m.logger.Warn(msg, fields)
	}
}

func (m *MahnoApi) Error(msg string, fields ...zap.Field) {
	if m.logger != nil {
		m.logger.Error(msg, fields)
	}
}

func (m *MahnoApi) doReqReader(method string, path string, data string) (io.ReadCloser, error) {
	url := "http://" + m.host + path

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
	client := &http.Client{Timeout: time.Second * 5}

	resp, err := client.Do(req)
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
		m.Error("error talking to mahno: " + err.Error())
		return err
	}

	m.Info(fmt.Sprintf("body: %s", body))
	return nil
}

func (m *MahnoApi) SetItemState(item string, val string) error {
	body, err := m.doReq("POST", "/items/"+item, val)

	if err != nil {
		m.Error("error talking to mahno: " + err.Error())
		return err
	}

	m.Info(fmt.Sprintf("body: %s", body))
	return nil
}

func (m *MahnoApi) AllItems() ([]Item, error) {
	body, err := m.doReqReader("GET", "/items", "")

	if err != nil {
		m.Error("error talking to mahno: " + err.Error())
		return nil, err
	}

	defer body.Close()
	var res []Item
	decoder := json.NewDecoder(body)

	if err = decoder.Decode(&res); err != nil {
		fmt.Println("can't decode: ", err)
		return nil, err
	}
	return res, nil
}
