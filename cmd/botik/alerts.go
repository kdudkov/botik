package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	notifyDelay = time.Hour * 6
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

type AlertRec struct {
	Alert      *Alert    `json:"alert"`
	Url        string    `json:"url"`
	LastNotify time.Time `json:"last_notify"`
}

func (app *App) processNewUrl(alertUrl string) {
	if _, ok := app.alerts.Load(alertUrl); ok {
		return
	}

	app.logger.Infof("new alert: %s", alertUrl)

	alert, _, err := getAlert(alertUrl)

	if err != nil {
		app.logger.Errorf("error getting alert: %s", err.Error())
	}

	if alert != nil && alert.State != "inactive" {
		app.alerts.Store(alertUrl, &AlertRec{Alert: alert, Url: alertUrl, LastNotify: time.Now()})
		app.notify("kott", alert, false)
	}
}

func (app *App) alertProcessor() {
	for {
		app.alerts.Range(func(key, value interface{}) bool {
			alert, stop, err := getAlert(key.(string))

			if stop {
				app.logger.Infof("remove %s alert (404)", key)
				if ar, ok := value.(*AlertRec); ok {
					app.notify("kott", ar.Alert, true)
				}
				app.alerts.Delete(key)
				return true
			}

			if err != nil {
				app.logger.Errorf("error getting alert %s: %s", key, err.Error())
				return true
			}

			if alert.State == "inactive" {
				app.logger.Infof("alert %s inactive", key)
				app.alerts.Delete(key)
				app.notify("kott", alert, true)
				return true
			}

			if ar, ok := value.(*AlertRec); ok {
				ar.Alert = alert
				if time.Now().After(ar.LastNotify.Add(notifyDelay)) {
					app.notify("kott", alert, false)
					ar.LastNotify = time.Now()
				}
				app.alerts.Store(key, ar)
			} else {
				app.logger.Errorf("invalid value: %v", value)
				app.alerts.Store(key, &AlertRec{Alert: alert, LastNotify: time.Now()})
			}
			return true
		})

		time.Sleep(time.Second)
	}
}
func getAlert(alertUrl string) (*Alert, bool, error) {
	cl := http.Client{Timeout: time.Second * 3}

	resp, err := cl.Get(alertUrl)
	if err != nil {
		return nil, false, fmt.Errorf("error getting url %s: %s", alertUrl, err.Error())
	}

	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, true, fmt.Errorf("error getting url %s: status %d", alertUrl, resp.StatusCode)
	}

	if resp.StatusCode > 299 {
		return nil, false, fmt.Errorf("error getting url %s: status %d", alertUrl, resp.StatusCode)
	}

	alert := new(Alert)
	m := json.NewDecoder(resp.Body)
	if err := m.Decode(alert); err != nil {
		return nil, true, fmt.Errorf("json decode error: %s", err.Error())
	}

	return alert, false, nil
}

func (app *App) notify(name string, alert *Alert, good bool) {
	id, err := app.IdByName(name)

	if err != nil {
		app.logger.Warnf("user not found: %s", name)
		return
	}

	sb := strings.Builder{}
	if good {
		sb.WriteString(fmt.Sprintf("\u2705 %s is good\n", alert.Name))
	} else {
		sb.WriteString(fmt.Sprintf("\u26a0 %s %s\n", alert.Name, alert.State))
		if sev, ok := alert.Labels["severity"]; ok {
			sb.WriteString(fmt.Sprintf("Severity: %s\n", sev))
		}
		sb.WriteString(fmt.Sprintf("%s\n", alert.Annotations.Summary))
		sb.WriteString(fmt.Sprintf("%s\n", alert.Annotations.Description))
		for k, v := range alert.Labels {
			sb.WriteString(k)
			sb.WriteString(": ")
			sb.WriteString(v)
			sb.WriteString(", ")
		}
	}
	if err := app.sendMode(name, id, sb.String(), "HTML"); err != nil {
		app.logger.Errorf("can't send to %s: %s", name, err.Error())
	}
}
