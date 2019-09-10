package main

import (
	"io/ioutil"
	"strconv"

	"github.com/aofei/air"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func runHttpServer(app *App) {
	a := air.Default
	a.Address = ":8055"

	a.POST("/send/:NAME", SendHandlerFunc(app))

	if err := a.Serve(); err != nil {
		app.logger.Errorf("server error: %v", err)
	}
}

func SendHandlerFunc(app *App) air.Handler {

	return func(req *air.Request, res *air.Response) error {
		name := req.Param("NAME")

		if name == nil {
			app.logger.Errorf("nil name")
			return air.DefaultNotFoundHandler(req, res)
		}

		if ids, ok := app.users[name.Value().String()]; ok {
			id, err := strconv.ParseInt(ids, 10, 64)

			if err != nil {
				app.logger.Errorf("can't parse int %s", ids)
				return nil
			}

			send(app, name.Value().String(), id, req, res)
			return nil
		}

		if ids, ok := app.groups[name.Value().String()]; ok {
			id, err := strconv.ParseInt(ids, 10, 64)

			if err != nil {
				app.logger.Errorf("can't parse int %s", ids)
				return nil
			}

			send(app, name.Value().String(), id, req, res)
			return nil
		}

		app.logger.Warnf("user not found: %s", name)
		return air.DefaultNotFoundHandler(req, res)
	}
}

func send(app *App, name string, id int64, req *air.Request, res *air.Response) {
	logger := app.logger.With("to", name, "id", id)
	body, _ := ioutil.ReadAll(req.Body)

	if body == nil || len(body) == 0 {
		_ = res.WriteString("empty body")
		return
	}

	if app.bot == nil {
		logger.Warnf("bot is not ready")
		_ = res.WriteString("bot is not connected")
		return
	}

	logger.Infof("message \"%s\"", string(body))

	go func(s string) {
		msg := tgbotapi.NewMessage(id, s)
		_, err := app.bot.Send(msg)

		if err != nil {
			logger.Errorf("can't send message: %s", err.Error())
		}
	}(string(body))

	_ = res.WriteString("message is sent")
}
