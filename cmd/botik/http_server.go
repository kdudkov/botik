package main

import (
	"botik/cmd/botik/alert"
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

			return c.Status(fiber.StatusNotFound).SendString("no name")
		}

		if id, err := app.IdByName(name); err == nil {

			body := c.Body()

			if len(body) == 0 {
				return c.SendString("empty body")
			}

			if _, err := app.sendTgWithMode(id, html.EscapeString(string(body)), "HTML"); err != nil {
				return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
			}

			return c.SendString("ok")
		}

		app.logger.Warn("user not found: " + name)
		return c.SendStatus(fiber.StatusNotFound)
	}
}

func GrafanaHandlerFunc(app *App) fiber.Handler {
	return func(c *fiber.Ctx) error {
		r := new(GrafanaReq)
		if err := c.BodyParser(r); err != nil {
			return err
		}

		text := MakeGrafanaMsg(r)

		for _, user := range app.notifyUsers {
			id, err := app.IdByName(user)

			if err != nil {
				app.logger.Error("invalid user " + user)
				continue
			}

			if _, err := app.sendTg(id, text); err != nil {
				return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
			}
		}
		return c.SendString("ok")
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
		list := make([]*alert.AlertRecDTO, 0)

		app.am.Range(func(ar *alert.AlertRec) bool {
			list = append(list, ar.DTO())
			return true
		})

		return c.JSON(list)
	}
}

func GetMuteAlertHandlerFunc(app *App) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")

		app.am.Range(func(ar *alert.AlertRec) bool {
			if ar.Alert().ID == id {
				ar.Mute()
			}
			return true
		})

		return c.SendString("ok")
	}
}

func (app *App) sendTg(id int64, text string) (int, error) {
	return app.sendTgWithMode(id, text, "MarkdownV2")
}

func (app *App) sendTgWithMode(id int64, text string, mode string) (int, error) {
	logger := app.logger.With("id", id)

	if app.bot == nil {
		logger.Warn("bot is not ready")
		return 0, fmt.Errorf("bot is not connected")
	}

	msg := tg.NewMessage(id, text)
	msg.ParseMode = mode
	msg1, err := app.bot.Send(msg)

	if err != nil {
		logger.Error("can't send message", "error", err)
	}

	return msg1.MessageID, nil
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
