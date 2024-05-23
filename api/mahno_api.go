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
	Name           string         `json:"name"`
	Type_          string         `json:"type"`
	HumanName      string         `json:"human_name"`
	Room           string         `json:"room"`
	Changed        time.Time      `json:"changed"`
	Checked        time.Time      `json:"checked"`
	Value          string         `json:"value"`
	RawValue       any            `json:"raw_value"`
	FormattedValue string         `json:"formatted_value"`
	Good           bool           `json:"good"`
	UI             bool           `json:"ui"`
	Tags           []string       `json:"tags,omitempty"`
	Groups         []string       `json:"groups,omitempty"`
	Meta           map[string]any `json:"meta,omitempty"`
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
