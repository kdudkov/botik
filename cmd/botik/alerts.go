package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Alert struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	GroupID     string            `json:"group_id"`
	Expression  string            `json:"expression"`
	State       string            `json:"state"`
	Value       string            `json:"value"`
	Labels      map[string]string `json:"labels"`
	Annotations struct {
		Summary     string `json:"summary"`
		Description string `json:"description"`
	} `json:"annotations"`
	ActiveAt time.Time `json:"activeAt"`
}

func (app *App) processNewUrl(alertUrl string) {
	if app.numAlerts.Load() > 50 {
		app.logger.Errorf("too many active urls")
		return
	}

	if _, ok := app.alerts.LoadOrStore(alertUrl, true); ok {
		return
	}

	app.logger.Infof("new alert: %s", alertUrl)

	app.numAlerts.Inc()
	defer func() {
		app.numAlerts.Dec()
		app.alerts.Delete(alertUrl)
	}()

	resp, err := http.Get(alertUrl)
	if err != nil {
		app.logger.Errorf("error getting url %s: %s", alertUrl, err.Error())
		return
	}
	defer resp.Body.Close()

	for true {
		alert, done, err := getAlert(alertUrl)
		if err != nil {
			app.logger.Errorf("%v", err)
		}
		if done {
			return
		}

		if alert != nil {
			app.notify("kott", alert)
		}

		time.Sleep(time.Hour * 3)
	}
}

func getAlert(alertUrl string) (*Alert, bool, error) {
	cl := http.Client{Timeout: time.Second * 3}

	resp, err := cl.Get(alertUrl)
	if err != nil {
		return nil, false, fmt.Errorf("error getting url %s: %s", alertUrl, err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		return nil, true, fmt.Errorf("error getting url %s: status %d", alertUrl, resp.StatusCode)
	}

	alert := new(Alert)
	m := json.NewDecoder(resp.Body)
	if err := m.Decode(alert); err != nil {
		return nil, true, fmt.Errorf("json decode error: %s", err.Error())
	}

	return alert, false, nil
}

func (app *App) notify(name string, alert *Alert) {
	id, err := app.IdByName(name)

	if err != nil {
		app.logger.Warnf("user not found: %s", name)
		return
	}

	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("%s %s\n", alert.Name, alert.State))
	sb.WriteString(fmt.Sprintf("%s\n", alert.Annotations.Summary))
	sb.WriteString(fmt.Sprintf("%s\n", alert.Annotations.Description))

	if err := app.send(name, id, sb.String()); err != nil {
		app.logger.Errorf("can't send to %s: %s", name, err.Error())
	}
}
