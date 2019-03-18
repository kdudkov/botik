package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

type convert func(string)

type InfluxHttpApi interface {
	Send(q string) error
	GetData(q string, fn convert) error
}

type InfluxApi struct {
	host   string
	db     string
	client *http.Client
}

func (i *InfluxApi) Send(q string) error {
	u := fmt.Sprintf("http://%s/write?db=%s", i.host, i.db)

	req, err := http.NewRequest("POST", u, bytes.NewBuffer([]byte(q)))

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

func (i *InfluxApi) GetData(q string, fn convert) error {
	params := url.Values{}
	params.Add("epoch", "s")
	params.Add("db", i.db)
	params.Add("q", q)

	url := fmt.Sprintf("http://%s/query?%s", i.host, params.Encode())

	fmt.Println(url)

	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/csv")

	resp, err := i.client.Do(req)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	r := bufio.NewReader(resp.Body)

	for true {
		l, _, err := r.ReadLine()
		if l != nil {
			fn(string(l))
		}
		if err != nil {
			break
		}
	}

	return nil
}
