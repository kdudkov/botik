package main

import (
	"bytes"
	"fmt"
	"github.com/labstack/gommon/log"
	"io/ioutil"
	"net/http"
	"time"
)

type item struct {
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
	url := fmt.Sprintf("http://oh.home/items/%s", item)

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
	url := fmt.Sprintf("http://oh.home/items/%s", item)

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