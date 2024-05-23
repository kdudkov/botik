package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/kdudkov/goutils/request"
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
	AllItems() ([]*Item, error)
}

type MahnoHttpApi struct {
	host   string
	client *http.Client
	logger *slog.Logger
}

func NewMahnoApi(host string) *MahnoHttpApi {
	return &MahnoHttpApi{
		host:   host,
		client: &http.Client{Timeout: time.Second * 3},
		logger: slog.Default().With("logger", "mahno_api"),
	}
}

func (m *MahnoHttpApi) SetLogger(logger *slog.Logger) {
	m.logger = logger
}

func (m *MahnoHttpApi) doReq(method string, path string, data string) ([]byte, error) {
	r := request.New(m.client, m.logger).URL("http://" + m.host + path).Method(method)

	if data != "" {
		r.Body(strings.NewReader(data))
	}

	r.AddHeader("Content-Type", "application/json")

	return r.GetBody(context.Background())
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

func (m *MahnoHttpApi) AllItems() ([]*Item, error) {
	r := request.New(m.client, m.logger).URL("http://" + m.host + "/items")

	r.Headers(map[string]string{"Content-Type": "application/json"})

	var res []*Item

	if err := r.GetJSON(context.Background(), &res); err != nil {
		m.logger.Error("error", "error", err)
		return nil, err
	}

	return res, nil
}
