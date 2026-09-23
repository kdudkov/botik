package main

import (
	"fmt"
	"html"
	"time"

	"botik/cmd/botik/alert"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gofiber/fiber/v2"
)

func runHttpServer(app *App) {
	a := fiber.New(fiber.Config{DisableStartupMessage: true})
	//a.Use(logger.New())

	a.Post("/send/:name", SendHandlerFunc(app))
	a.Post("/grafana", GrafanaHandlerFunc(app))
	a.Post("/api/v2/alerts", AlertsHandlerFunc(app))
	a.Get("/api/alerts", AllAlertsHandlerFunc(app))
	a.Get("/api/alerts/:id/mute", GetMuteAlertHandlerFunc(app))

	app.logger.Info("start listener on " + app.conf.Listen())

	if err := a.Listen(app.conf.Listen()); err != nil {
		app.logger.Error("server error", "error", err)
	}
}

type GrafanaReq struct {
	DashboardID int `json:"dashboardId"`
	EvalMatches []struct {
		Value  int    `json:"value"`
		Metric string `json:"metric"`
		Tags   any    `json:"tags"`
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
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
}

func SendHandlerFunc(app *App) fiber.Handler {
	return func(c *fiber.Ctx) error {
		name := c.Params("name")

		if name == "" {
			app.logger.Error("nil name")

			return fiber.NewError(fiber.StatusBadRequest, "no name")
		}

		if id, err := app.IdByName(name); err == nil {
			body := c.Body()

			if len(body) == 0 {
				return fiber.NewError(fiber.StatusBadRequest, "empty body")
			}

			if _, err := app.sendTgWithMode(id, html.EscapeString(string(body)), "HTML"); err != nil {
				return err
			}

			return c.SendString("ok")
		}

		app.logger.Warn("user not found: " + name)

		return fiber.ErrNotFound
	}
}

func GrafanaHandlerFunc(app *App) fiber.Handler {
	return func(c *fiber.Ctx) error {
		r := new(GrafanaReq)
		if err := c.BodyParser(r); err != nil {
			return err
		}

		text := MakeGrafanaMsg(r)

		for _, user := range app.conf.Strings("notify") {
			id, err := app.IdByName(user)

			if err != nil {
				app.logger.Error("invalid user " + user)
				continue
			}

			if _, err := app.sendTg(id, text); err != nil {
				return err
			}
		}

		return c.SendString("ok")
	}
}

func AlertsHandlerFunc(app *App) fiber.Handler {
	return func(c *fiber.Ctx) error {
		list := make([]*alert.Alert, 0)

		if err := c.BodyParser(&list); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}

		for _, a := range list {
			app.am.Process(a)
		}

		return c.SendString("ok")
	}
}

func AllAlertsHandlerFunc(app *App) fiber.Handler {
	return func(c *fiber.Ctx) error {
		res := make([]*alert.AlertDTO, 0)

		app.am.Range(func(a *alert.Alert) bool {
			res = append(res, a.DTO())

			return true
		})

		return c.JSON(res)
	}
}

func GetMuteAlertHandlerFunc(app *App) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Params("id")

		app.am.Range(func(ar *alert.Alert) bool {
			if ar.Key() == id {
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
