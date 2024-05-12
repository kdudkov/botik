package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log/slog"
	"net/http"
	"time"
)

type Item struct {
	ClassName string      `json:"class"`
	Name      string      `json:"name"`
	HumanName string      `json:"human_name,omitempty"`
	Value     interface{} `json:"value"`
	Formatted string      `json:"formatted_value,omitempty"`
	Checked   time.Time   `json:"checked"`
	Changed   time.Time   `json:"changed"`
	Tags      []string    `json:"tags"`
}

type MahnoApi interface {
	ItemCommand(item string, cmd string) error
	SetItemState(item string, val string) error
	AllItems() ([]Item, error)
	SetLogger(logger *slog.Logger)
}

type MahnoHttpApi struct {
	host   string
	client *http.Client
	logger *slog.Logger
}

func NewMahnoApi() *MahnoHttpApi {
	client := &http.Client{Timeout: time.Second * 3}
	return &MahnoHttpApi{host: "oh.home", client: client}
}

func (m *MahnoHttpApi) SetLogger(logger *slog.Logger) {
	m.logger = logger
}

func (m *MahnoHttpApi) doReqReader(method string, path string, data string) (io.ReadCloser, error) {
	url := "http://" + m.host + path

	m.logger.Debug("url: " + url)

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

func (m *MahnoHttpApi) doReq(method string, path string, data string) ([]byte, error) {
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

func (m *MahnoHttpApi) ItemCommand(item string, cmd string) error {
	body, err := m.doReq("POST", "/items/"+item, cmd)

	if err != nil {
		m.logger.Error("error talking to mahno", "error", err)
		return err
	}

	m.logger.Info(fmt.Sprintf("body: " + string(body)))
	return nil
}

func (m *MahnoHttpApi) SetItemState(item string, val string) error {
	body, err := m.doReq("POST", "/items/"+item, val)

	if err != nil {
		m.logger.Error("error talking to mahno", "error", err)
		return err
	}

	m.logger.Info(fmt.Sprintf("body: " + string(body)))
	return nil
}

func (m *MahnoHttpApi) AllItems() ([]Item, error) {
	body, err := m.doReqReader("GET", "/items", "")

	if err != nil {
		m.logger.Error("error talking to mahno", "error", err)
		return nil, err
	}

	defer body.Close()
	var res []Item
	decoder := json.NewDecoder(body)

	if err = decoder.Decode(&res); err != nil {
		m.logger.Error("can't decode", "error", err)
		return nil, err
	}

	return res, nil
}
