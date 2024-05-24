package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kdudkov/goutils/request"
)

type InfluxHttpApi interface {
	Send(db string, q string) error
	GetData(db string, q string) (ans InfluxAnswer, err error)
	GetSingleSeries(db string, q string) ([]map[string]interface{}, error)
}

type InfluxAnswer struct {
	Results []struct {
		StatementID int    `json:"statement_id"`
		Error       string `json:"error"`
		Series      []struct {
			Name    string            `json:"name"`
			Tags    map[string]string `json:"tags"`
			Columns []string          `json:"columns"`
			Values  [][]interface{}   `json:"values"`
		} `json:"series"`
	} `json:"results"`
	Error string `json:"error"`
}

type InfluxApi struct {
	host   string
	client *http.Client
	logger *slog.Logger
}

func NewInfluxApi(host string, client *http.Client) *InfluxApi {
	return &InfluxApi{
		host:   host,
		client: client,
		logger: slog.Default(),
	}
}

func (i *InfluxApi) Send(db, q string) error {
	r := request.New(i.client, i.logger).
		URL(fmt.Sprintf("http://%s/write", i.host)).
		Post().
		Args(map[string]string{"epoch": "ns", "db": db}).
		Body(strings.NewReader(q))

	_, err := r.GetBody(context.Background())

	return err
}

func (i *InfluxApi) GetData(db string, q string) (ans InfluxAnswer, err error) {
	params := url.Values{}
	params.Add("epoch", "ns")
	params.Add("db", db)
	params.Add("q", q)

	path := fmt.Sprintf("http://%s/query?%s", i.host, params.Encode())

	println(path)

	var req *http.Request
	if req, err = http.NewRequest("GET", path, nil); err != nil {
		return
	}
	req.Header.Set("Accept", "application/json")

	var resp *http.Response

	if resp, err = i.client.Do(req); err != nil {
		return
	}

	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)

	err = decoder.Decode(&ans)
	return
}

func (i *InfluxApi) GetSingleSeries(db string, q string) ([]map[string]interface{}, error) {
	res, err := i.GetData(db, q)

	if err != nil {
		return nil, err
	}

	if res.Error != "" {
		return nil, errors.New(res.Error)
	}

	if len(res.Results) != 1 {
		return nil, fmt.Errorf("got %d results", len(res.Results))
	}

	if len(res.Results[0].Series) > 1 {
		return nil, fmt.Errorf("got %d series", len(res.Results[0].Series))
	}

	if res.Results[0].Error != "" {
		return nil, errors.New(res.Results[0].Error)
	}

	if len(res.Results[0].Series) == 0 {
		return nil, nil
	}

	data := make([]map[string]interface{}, len(res.Results[0].Series[0].Values))

	for i, val := range res.Results[0].Series[0].Values {
		m := make(map[string]interface{})
		for fi, v := range val {
			fieldName := res.Results[0].Series[0].Columns[fi]

			if fieldName == "time" {
				if s, ok := v.(float64); ok {
					m[fieldName] = time.Unix(0, int64(s))
				} else {
					m[fieldName] = v
				}
			} else {
				m[fieldName] = v
			}
		}
		data[i] = m
	}

	return data, nil
}
