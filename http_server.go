package main

import (
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"io/ioutil"
	"strconv"

	"github.com/aofei/air"
)

func setHandlers(app *App) {
	air.Default.POST("/send/:NAME", SendHandlerFunc(app))
}

func runHttpServer(app *App) {
	if err := air.Default.Serve(); err != nil {
		app.logger.Errorf("server error: %v", err)
	}
}

func SendHandlerFunc(app *App) air.Handler {
	return func(req *air.Request, res *air.Response) error {
		name := req.Param("NAME")

		if name == nil {
			return fmt.Errorf("nil name")
		}

		if ids, ok := app.users[name.Value().String()]; ok {
			id, err := strconv.ParseInt(ids, 10, 64)

			if err != nil {
				return err
			}

			body, err := ioutil.ReadAll(req.Body)
			if err != nil {
				app.logger.Errorf("can't read body: %s", err.Error())
			}

			msg := tgbotapi.NewMessage(id, string(body))

			//msg.ReplyToMessageID = update.Message.MessageID

			_, err = app.bot.Send(msg)

			if err != nil {
				app.logger.Errorf("can't send message: %s", err.Error())
			}
		} else {
			return air.DefaultNotFoundHandler(req, res)
		}

		return nil
	}
}
