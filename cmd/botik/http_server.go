package main

import (
	"fmt"
	"html"
	"strings"
	"time"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gofiber/fiber/v2"
	"github.com/spf13/viper"
)

func runHttpServer(app *App) {
	a := fiber.New()
	//a.Use(logger.New())

	a.Post("/send/:name", SendHandlerFunc(app))
	a.Post("/grafana", GrafanaHandlerFunc(app))
	a.Post("/api/v2/alerts", AlertsHandlerFunc(app))
	a.Get("/api/alerts", GetAlertsHandlerFunc(app))
	a.Get("/api/alerts/:id/mute", GetMuteAlertHandlerFunc(app))

	app.logger.Info("start listener on " + viper.GetString("http.address"))

	if err := a.Listen(viper.GetString("http.address")); err != nil {
		app.logger.Error("server error", "error", err)
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
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Labels       map[string]string `json:"labels"`
	Annotations  struct {
		Summary string `json:"summary"`
	} `json:"annotations"`
}

func SendHandlerFunc(app *App) fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")

		if name == "" {
			app.logger.Error("nil name")
			c.WriteString("no name")
			return c.SendStatus(fiber.StatusNotFound)
		}

		if id, err := app.IdByName(name); err == nil {

			body := c.Body()

			if len(body) == 0 {
				c.WriteString("empty body")
				return nil
			}

			if err := app.sendWithMode(name, id, html.EscapeString(string(body)), "HTML"); err != nil {
				_, _ = c.WriteString(err.Error())
				return nil
			}
			_, _ = c.WriteString("ok")
			return nil
		}

		app.logger.Warn("user not found: " + name)
		return c.SendStatus(fiber.StatusNotFound)
	}
}

func GrafanaHandlerFunc(app *App) fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := notifyUser

		if id, err := app.IdByName(name); err == nil {
			r := new(GrafanaReq)
			if err := c.BodyParser(r); err != nil {
				return err
			}

			text := MakeGrafanaMsg(r)

			if err := app.send(name, id, text); err != nil {
				_, _ = c.WriteString(err.Error())
				return nil
			}
			_, _ = c.WriteString("ok")
			return nil
		}

		app.logger.Warn("user not found: " + name)
		return c.SendStatus(fiber.StatusNotFound)
	}
}

func AlertsHandlerFunc(app *App) fiber.Handler {
	return func(c *fiber.Ctx) error {
		list := new([]AlertReq)

		if err := c.BodyParser(list); err != nil {
			return err
		}

		if list == nil {
			return nil
		}

		for _, a := range *list {
			url := strings.ReplaceAll(a.GeneratorURL, "/vmalert/alert?", "/api/v1/alert?")
			app.alertUrls <- url
		}

		_, _ = c.WriteString("ok")
		return nil
	}
}

func GetAlertsHandlerFunc(app *App) fiber.Handler {
	return func(c *fiber.Ctx) error {
		list := make([]*AlertRec, 0)

		app.alerts.Range(func(_, value interface{}) bool {
			if ar, ok := value.(*AlertRec); ok {
				list = append(list, ar)
			}
			return true
		})

		return c.JSON(list)
	}
}

func GetMuteAlertHandlerFunc(app *App) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")

		app.alerts.Range(func(_, value interface{}) bool {
			if ar, ok := value.(*AlertRec); ok {
				if ar.Alert.ID == id {
					ar.Muted = true
				}
			}
			return true
		})

		_, _ = c.WriteString("ok")
		return nil
	}
}

func (app *App) send(name string, id int64, text string) error {
	return app.sendWithMode(name, id, text, "MarkdownV2")
}

func (app *App) sendWithMode(name string, id int64, text string, mode string) error {
	logger := app.logger.With("to", name, "id", id)

	if app.bot == nil {
		logger.Warn("bot is not ready")
		return fmt.Errorf("bot is not connected")
	}

	go func(s string) {
		msg := tg.NewMessage(id, s)
		msg.ParseMode = mode
		_, err := app.bot.Send(msg)

		if err != nil {
			logger.Error("can't send message", "error", err)
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
