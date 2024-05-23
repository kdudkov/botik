package main

import (
	"botik/cmd/botik/alert"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"
)

const (
	notifyUser = "kott"
)

//go:embed template/alert
var alertTpl string

func (app *App) processNewUrl(alertUrl string) {
	if _, ok := app.alerts.Load(alertUrl); ok {
		return
	}

	app.logger.Info("new alert: " + alertUrl)

	alertInfo, err := fetchAlertInfo(alertUrl)

	if err != nil {
		app.logger.Error("error getting alert", "error", err)
	}

	if alertInfo != nil && alertInfo.State != "inactive" {
		ar := alert.NewAlertRec(alertInfo, alertUrl)
		app.alerts.Store(alertUrl, ar)
		go app.notifyUser(notifyUser, ar, false)
	}
}

func (app *App) alertProcessor() {
	for {
		app.alerts.Range(func(key, value interface{}) bool {
			if alertRec, ok := value.(*alert.AlertRec); ok {
				alertInfo, err := fetchAlertInfo(key.(string))

				if err != nil {
					app.logger.Error(fmt.Sprintf("error getting alert %v", key), "error", err.Error())
					return true
				}

				if alertInfo == nil {
					app.logger.Info(fmt.Sprintf("remove %s alert (404)", key))
					app.alerts.Delete(key)
					go app.notifyUser(notifyUser, alertRec, true)
					return true
				}

				alertRec.SetAlert(alertInfo)

				if alertInfo.State == "inactive" {
					app.logger.Info(fmt.Sprintf("alert %s inactive", key))
					app.alerts.Delete(key)
					go app.notifyUser(notifyUser, alertRec, true)
					return true
				}

				if alertRec.NeedToNotify() {
					go app.notifyUser(notifyUser, alertRec, false)
				}
			} else {
				app.logger.Error(fmt.Sprintf("invalid value: %v", value))
			}

			return true
		})

		time.Sleep(time.Second)
	}
}
func fetchAlertInfo(alertUrl string) (*alert.Alert, error) {
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
	alert := new(alert.Alert)
	m := json.NewDecoder(resp.Body)
	if err := m.Decode(alert); err != nil {
		return nil, fmt.Errorf("json decode error %v", err)
	}

	return alert, nil
}

func (app *App) notifyUser(userName string, ar *alert.AlertRec, good bool) {
	id, err := app.IdByName(userName)

	if err != nil {
		app.logger.Warn("user not found: " + userName)
		return
	}

	if mid, err := app.sendTgWithMode(id, getMsg(ar.Alert(), good), "HTML"); err != nil {
		app.logger.Error("can't send to "+userName, "error", err)
	} else {
		ar.Notified(mid)
	}
}

func getMsg(alert *alert.Alert, good bool) string {
	tmpl, err := template.New("name").Parse(alertTpl)
	if err != nil {
		return err.Error()
	}

	var severity = "unknown"
	if sev, ok := alert.Labels["severity"]; ok {
		severity = sev
	}

	sb := new(strings.Builder)
	if err := tmpl.Execute(sb, map[string]any{"good": good, "alert": alert, "severity": severity}); err != nil {
		return err.Error()
	}

	return sb.String()
}
