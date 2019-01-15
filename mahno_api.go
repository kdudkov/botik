package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/labstack/gommon/log"
)

const (
	host = "oh.tome"
)

type Item struct {
	ClassName string        `json:"class"`
	TTL       int           `json:"ttl"`
	Value     interface{}   `json:"value"`
	Value2    interface{}   `json:"_value"`
	Age       time.Duration `json:"age"`
	CheckAge  time.Duration `json:"check_age"`
	Checked   time.Time     `json:"checked"`
	Changed   time.Time     `json:"changed"`
	Tags      []string      `json:"tags"`
	Formatted string        `json:"formatted"`
	HumanName string        `json:"h_name"`
	UI        bool          `json:"ui"`
}

func ItemCommand(item string, cmd string) error {
	url := fmt.Sprintf("http://%s/items/%s", host, item)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(cmd)))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: time.Second * 5}

	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("error talking to mahno: %s", err.Error())
		return err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	log.Infof("body: %s", body)
	return nil
}

func ItemState(item string, val string) error {
	url := fmt.Sprintf("http://%s/items/%s", host, item)

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer([]byte(val)))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: time.Second * 5}

	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("error talking to mahno: %s", err.Error())
		return err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	log.Infof("body: %s", body)
	return nil
}

func AllItems() (*[]Item, error) {
	url := fmt.Sprintf("http://%s/items", host)

	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: time.Second * 5}

	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("error talking to mahno: %s", err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	res := make([]Item, 0)
	decoder := json.NewDecoder(resp.Body)
	if err = decoder.Decode(&res); err != nil {
		log.Errorf("can't decode")
		return nil, err
	}

	return &res, nil
}
