package alert

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	notifyUser = "kott"
)

//go:embed template/alert
var alertTpl string

type AlertManager struct {
	logger   *slog.Logger
	alerts   sync.Map
	client   *http.Client
	tpl      *template.Template
	notifier func(msg string)
}

func NewManager(logger *slog.Logger, notifier func(msg string)) *AlertManager {
	tmpl, err := template.New("alert").Parse(alertTpl)

	if err != nil {
		panic(err)
	}

	return &AlertManager{
		logger:   logger,
		alerts:   sync.Map{},
		client:   &http.Client{Timeout: time.Second * 3},
		tpl:      tmpl,
		notifier: notifier,
	}
}

func (a *AlertManager) Start() {
	go a.alertProcessor()
}

func (a *AlertManager) AddURL(alertUrl string) {
	if _, ok := a.alerts.Load(alertUrl); ok {
		return
	}

	a.logger.Info("new alert: " + alertUrl)

	alertInfo, err := a.fetchAlertInfo(alertUrl)

	if err != nil {
		a.logger.Error("error getting alert", "error", err)
	}

	if alertInfo != nil && alertInfo.State != "inactive" {
		ar := NewAlertRec(alertInfo, alertUrl)
		a.alerts.Store(alertUrl, ar)
		//a.notifier(a.getMsg(ar.Alert(), false))
	}
}

func (a *AlertManager) Range(f func(a *AlertRec) bool) {
	a.alerts.Range(func(_, value any) bool {
		if alertRec, ok := value.(*AlertRec); ok {
			return f(alertRec)
		}

		return true
	})
}

func (a *AlertManager) alertProcessor() {
	for {
		a.alerts.Range(func(key, value interface{}) bool {
			if alertRec, ok := value.(*AlertRec); ok {
				alertInfo, err := a.fetchAlertInfo(key.(string))

				if err != nil {
					a.logger.Error(fmt.Sprintf("error getting alert %v", key), "error", err.Error())
					return true
				}

				if alertInfo == nil {
					a.logger.Info(fmt.Sprintf("remove %s alert (404)", key))
					a.alerts.Delete(key)
					a.notifier(a.getMsg(alertRec.Alert(), true))
					return true
				}

				alertRec.SetAlert(alertInfo)

				if alertInfo.State == "inactive" {
					a.logger.Info(fmt.Sprintf("alert %s inactive", key))
					a.alerts.Delete(key)
					a.notifier(a.getMsg(alertRec.Alert(), true))
					return true
				}

				if alertRec.NeedToNotify() {
					a.notifier(a.getMsg(alertRec.Alert(), false))
					alertRec.Notified()
				}
			} else {
				a.logger.Error(fmt.Sprintf("invalid value: %v", value))
			}

			return true
		})

		time.Sleep(time.Second)
	}
}
func (a *AlertManager) fetchAlertInfo(alertUrl string) (*Alert, error) {
	resp, err := a.client.Get(alertUrl)
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
	al := new(Alert)
	m := json.NewDecoder(resp.Body)
	if err := m.Decode(al); err != nil {
		return nil, fmt.Errorf("json decode error %v", err)
	}

	return al, nil
}

func (a *AlertManager) getMsg(alert *Alert, good bool) string {
	sb := new(strings.Builder)
	if err := a.tpl.Execute(sb, map[string]any{"good": good, "alert": alert, "severity": alert.Severity()}); err != nil {
		a.logger.Error("error in template", "error", err)
		return ""
	}

	return sb.String()
}
