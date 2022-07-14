package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/aofei/air"
	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/spf13/viper"
)

func runHttpServer(app *App) {
	a := air.Default
	a.Address = viper.GetString("http.address")

	a.POST("/send/:NAME", SendHandlerFunc(app))
	a.POST("/grafana", GrafanaHandlerFunc(app))
	a.POST("/api/v2/alerts", AlertsHandlerFunc(app))
	a.GET("/api/alerts", GetAlertsHandlerFunc(app))
	a.GET("/api/alerts/:ID/mute", GetMuteAlertHandlerFunc(app))

	app.logger.Infof("start listener on %s", a.Address)

	if err := a.Serve(); err != nil {
		app.logger.Errorf("server error: %v", err)
	}
}

type GrafanaReq struct {
	DashboardID int `json:"dashboardId"`
	EvalMatches []struct {
		Value  int         `json:"value"`
		Metric string      `json:"metric"`
		Tags   interface{} `json:"tags"`
	} `json:"evalMatches"`
	Message  string `json:"message"`
	OrgID    int    `json:"orgId"`
	PanelID  int    `json:"panelId"`
	RuleID   int    `json:"ruleId"`
	RuleName string `json:"ruleName"`
	RuleURL  string `json:"ruleUrl"`
	State    string `json:"state"`
	Tags     struct {
	} `json:"tags"`
	Title string `json:"title"`
}

type AlertReq struct {
	StartsAt     time.Time         `json:"startsAt"`
	GeneratorURL string            `json:"generatorURL"`
	EndsAt       time.Time         `json:"endsAt"`
	Labels       map[string]string `json:"labels"`
	Annotations  struct {
		Summary string `json:"summary"`
	} `json:"annotations"`
}

func SendHandlerFunc(app *App) air.Handler {
	return func(req *air.Request, res *air.Response) error {
		name := req.Param("NAME")

		if name == nil {
			app.logger.Errorf("nil name")
			return air.DefaultNotFoundHandler(req, res)
		}

		if id, err := app.IdByName(name.Value().String()); err == nil {
			body, _ := ioutil.ReadAll(req.Body)

			if body == nil || len(body) == 0 {
				_ = res.WriteString("empty body")
				return nil
			}

			if err := app.send(name.Value().String(), id, string(body)); err != nil {
				_ = res.WriteString(err.Error())
				return nil
			}
			_ = res.WriteString("ok")
			return nil
		}

		app.logger.Warnf("user not found: %s", name)
		return air.DefaultNotFoundHandler(req, res)
	}
}

func GrafanaHandlerFunc(app *App) air.Handler {
	return func(req *air.Request, res *air.Response) error {
		name := notifyUser

		if id, err := app.IdByName(name); err == nil {
			r := new(GrafanaReq)

			m := json.NewDecoder(req.Body)
			if err := m.Decode(r); err != nil {
				return err
			}

			text := MakeGrafanaMsg(r)

			if err := app.send(name, id, text); err != nil {
				_ = res.WriteString(err.Error())
				return nil
			}
			_ = res.WriteString("ok")
			return nil
		}

		app.logger.Warnf("user not found: %s", name)
		return air.DefaultNotFoundHandler(req, res)
	}
}

func AlertsHandlerFunc(app *App) air.Handler {
	return func(req *air.Request, res *air.Response) error {
		list := new([]AlertReq)

		m := json.NewDecoder(req.Body)
		if err := m.Decode(list); err != nil {
			app.logger.Errorf("error decoding json, %s", err.Error())
			return err
		}

		if list == nil {
			return nil
		}

		for _, a := range *list {
			app.alertUrls <- a.GeneratorURL
		}

		return res.WriteString("ok")
	}
}

func GetAlertsHandlerFunc(app *App) air.Handler {
	return func(req *air.Request, res *air.Response) error {
		list := make([]*AlertRec, 0)

		app.alerts.Range(func(_, value interface{}) bool {
			if ar, ok := value.(*AlertRec); ok {
				list = append(list, ar)
			}
			return true
		})

		return res.WriteJSON(list)
	}
}

func GetMuteAlertHandlerFunc(app *App) air.Handler {
	return func(req *air.Request, res *air.Response) error {
		id := req.Param("ID").Value().String()

		app.alerts.Range(func(_, value interface{}) bool {
			if ar, ok := value.(*AlertRec); ok {
				if ar.Alert.ID == id {
					ar.Muted = true
				}
			}
			return true
		})

		return res.WriteString("ok")
	}
}

func (app *App) send(name string, id int64, text string) error {
	return app.sendWithMode(name, id, text, "MarkdownV2")
}

func (app *App) sendWithMode(name string, id int64, text string, mode string) error {
	logger := app.logger.With("to", name, "id", id)

	if app.bot == nil {
		logger.Warnf("bot is not ready")
		return fmt.Errorf("bot is not connected")
	}

	go func(s string) {
		msg := tg.NewMessage(id, s)
		msg.ParseMode = mode
		_, err := app.bot.Send(msg)

		if err != nil {
			logger.Errorf("can't send message: %s", err.Error())
		}
	}(text)

	return nil
}

func MakeGrafanaMsg(r *GrafanaReq) string {
	if r == nil {
		return "empty message"
	}

	switch r.State {
	case "ok":
		return fmt.Sprintf("✅ %s\n\n%s\n%s", r.Title, r.Message, r.RuleURL)
	case "no_data":
		return fmt.Sprintf("❕ %s\n\n%s\n%s", r.Title, r.Message, r.RuleURL)
	default:
		return fmt.Sprintf("❗ %s\n\n%s\n%s", r.Title, r.Message, r.RuleURL)

	}
}
