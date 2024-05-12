package main

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"
)

const (
	notifyDelay = time.Hour * 3
	notifyUser  = "kott"
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
	Muted      bool      `json:"muted"`
}

func (app *App) processNewUrl(alertUrl string) {
	if _, ok := app.alerts.Load(alertUrl); ok {
		return
	}

	app.logger.Info("new alert: " + alertUrl)

	alert, err := fetchAlertInfo(alertUrl)

	if err != nil {
		app.logger.Error("error getting alert", "error", err)
	}

	if alert != nil && alert.State != "inactive" {
		app.alerts.Store(alertUrl, &AlertRec{Alert: alert, Url: alertUrl, LastNotify: time.Now()})
		app.notify(notifyUser, alert, false)
	}
}

func (app *App) alertProcessor() {
	for {
		app.alerts.Range(func(key, value interface{}) bool {
			if alertRec, ok := value.(*AlertRec); ok {
				alert, err := fetchAlertInfo(key.(string))

				if err != nil {
					app.logger.Error(fmt.Sprintf("error getting alert %v", key), "error", err.Error())
					return true
				}

				if alert == nil {
					app.logger.Info(fmt.Sprintf("remove %s alert (404)", key))
					app.notify(notifyUser, alertRec.Alert, true)
					app.alerts.Delete(key)
					return true
				}

				if alert.State == "inactive" {
					app.logger.Info(fmt.Sprintf("alert %s inactive", key))
					app.alerts.Delete(key)
					app.notify(notifyUser, alert, true)
					return true
				}

				var severity = "unknown"
				if sev, ok := alert.Labels["severity"]; ok {
					severity = sev
				}

				alertRec.Alert = alert

				if !alertRec.Muted && severity == "critical" && time.Now().After(alertRec.LastNotify.Add(notifyDelay)) {
					app.notify(notifyUser, alert, false)
					alertRec.LastNotify = time.Now()
				}
				app.alerts.Store(key, alertRec)

			} else {
				app.logger.Error(fmt.Sprintf("invalid value: %v", value))
			}

			return true
		})

		time.Sleep(time.Second)
	}
}
func fetchAlertInfo(alertUrl string) (*Alert, error) {
	cl := http.Client{Timeout: time.Second * 3}

	resp, err := cl.Get(alertUrl)
	if err != nil {
		return nil, fmt.Errorf("error getting url %s: %s", alertUrl, err.Error())
	}

	if resp.StatusCode == 404 {
		return nil, nil
	}

	if resp.StatusCode > 299 {
		return nil, fmt.Errorf("error getting url %s: status %d", alertUrl, resp.StatusCode)
	}

	defer resp.Body.Close()
	alert := new(Alert)
	m := json.NewDecoder(resp.Body)
	if err := m.Decode(alert); err != nil {
		return nil, fmt.Errorf("json decode error", "error", err)
	}

	return alert, nil
}

func (app *App) notify(name string, alert *Alert, good bool) {
	id, err := app.IdByName(name)

	if err != nil {
		app.logger.Warn("user not found: " + name)
		return
	}

	var severity = "unknown"
	if sev, ok := alert.Labels["severity"]; ok {
		severity = sev
	}

	if severity != "critical" {
		return
	}

	sb := strings.Builder{}
	if good {
		sb.WriteString(fmt.Sprintf("%s %s is good\n", em_green_square, alert.Name))
	} else {
		icon := em_yellow_square
		if severity == "critical" {
			icon = em_red_square
		}
		sb.WriteString(fmt.Sprintf("%s %s [%s]\n\n", icon, alert.Name, severity))
		sb.WriteString(fmt.Sprintf("%s\n", alert.Annotations.Summary))
		sb.WriteString(fmt.Sprintf("%s\n", alert.Annotations.Description))
		for k, v := range alert.Labels {
			sb.WriteString(k)
			sb.WriteString(": ")
			sb.WriteString(v)
			sb.WriteString(", ")
		}
	}
	if err := app.sendWithMode(name, id, html.EscapeString(sb.String()), "HTML"); err != nil {
		app.logger.Error("can't send to "+name, "error", err)
	}
}
