package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"botik/cmd/botik/alert"
	"botik/cmd/botik/answer"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/kdudkov/goatak/pkg/cot"
)

var (
	gitRevision string
	gitBranch   string
)

type App struct {
	conf   *AppConfig
	bot    *tg.BotAPI
	cl     *MqttClient
	logger *slog.Logger
	am     *alert.AlertManager
	ans    *answer.AnswerManager
}

func NewApp(conf *AppConfig) *App {
	app := &App{
		conf:   conf,
		logger: slog.Default(),
		ans:    answer.New(),
	}

	app.am = alert.NewManager(slog.Default().With("logger", "alerts"), app.alertNotifier)

	if s := app.conf.String("mahno.host"); s != "" {
		if err := app.ans.RegisterAnswer("light", answer.NewLight(app.logger, s)); err != nil {
			panic(err.Error())
		}
	}

	if s := app.conf.String("camera.file"); s != "" {
		if err := app.ans.RegisterAnswer("cam", answer.NewCamera(app.logger, s)); err != nil {
			panic(err.Error())
		}
	}

	if app.conf.MQTTServer() != "" {
		app.cl = NewMqttClient(app.logger, app.conf, app.onMessage)
	}

	app.ans.RegisterAnswer("alerts", answer.NewAlerts(app.logger, app.am))

	return app
}

func (app *App) GetUpdatesChannel() (tg.UpdatesChannel, error) {
	if webhook := app.conf.String("webhook.ext"); webhook != "" {
		app.logger.Info("starting webhook " + webhook)

		wh, _ := tg.NewWebhook(webhook)
		if _, err := app.bot.Request(wh); err != nil {
			return nil, err
		}

		info, err := app.bot.GetWebhookInfo()
		if err != nil {
			return nil, err
		}

		if info.LastErrorDate != 0 {
			app.logger.Error("Telegram callback failed", "error", info.LastErrorMessage)
			return nil, fmt.Errorf("callback errror with %s", info.LastErrorMessage)
		}

		app.logger.Info(fmt.Sprintf("start listener on %s, path %s", app.conf.String("webhook.listen"), app.conf.String("webhook.path")))
		go func() {
			if err := http.ListenAndServe(app.conf.String("webhook.listen"), nil); err != nil {
				panic(err)
			}
		}()

		return app.bot.ListenForWebhook(app.conf.String("webhook.path")), nil
	}

	app.logger.Info("start polling")
	app.removeWebhook()
	u := tg.NewUpdate(0)
	u.Timeout = 60

	return app.bot.GetUpdatesChan(u), nil
}

func (app *App) quit() {
	app.bot.StopReceivingUpdates()
	if webhook := app.conf.String("webhook.ext"); webhook != "" {
		app.removeWebhook()
	}
}

func (app *App) removeWebhook() {
	if _, err := app.bot.Request(tg.WebhookConfig{URL: nil}); err != nil {
		app.logger.Error("remove webhook error", "error", err)
	}
}

func (app *App) Run() {
	// for k := range app.users {
	// 	app.logger.Info("user " + k)
	// }

	var err error
	app.bot, err = tg.NewBotAPI(app.conf.String("token"))

	if err != nil {
		panic("can't start bot " + err.Error())
	}
	app.logger.Info("registering " + app.bot.Self.String())

	go runHttpServer(app)
	app.am.Start()

	if app.cl != nil {
		go app.cl.Run(context.TODO())
	}

	updates, err := app.GetUpdatesChannel()

	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case update := <-updates:
			go app.Process(update)
		case <-sigc:
			app.logger.Info("quit")
			app.quit()
			return
		}
	}
}

func (app *App) onMessage(topic string, msg []byte) {
	chunks := strings.Split(topic, "/")

	if len(chunks) == 4 && chunks[3] == "snapshot" {
		for _, user := range app.conf.Strings("notify") {
			id, err := app.IdByName(user)

			if err != nil {
				app.logger.Error("invalid user "+user, slog.Any("error", err))
				continue
			}

			msg := tg.NewPhoto(id, tg.FileBytes{Bytes: msg, Name: fmt.Sprintf("cam %s %s", chunks[1], chunks[2])})

			if _, err := app.bot.Send(msg); err != nil {
				app.logger.Error("can't send message", slog.Any("error", err))
			}
		}
	}

	if topic == "frigate/reviews" {
		app.ProcessReview(msg)
	}
}

func (app *App) alertNotifier(text string) {
	for _, user := range app.conf.Strings("notify") {
		id, err := app.IdByName(user)

		if err != nil {
			app.logger.Error("invalid user "+user, slog.Any("error", err))
			continue
		}

		go func(logger *slog.Logger, id int64, text string) {
			logger.Info("sending notification")

			if _, err := app.sendTgWithMode(id, text, "HTML"); err != nil {
				logger.Error("error send message", slog.Any("error", err))
			}
		}(app.logger.With("user", user, "id", id), id, text)
	}
}

func (app *App) Process(update tg.Update) {
	var message *tg.Message

	if update.EditedMessage != nil {
		message = update.EditedMessage
	} else {
		message = update.Message
	}

	if message == nil {
		app.logger.Warn(fmt.Sprintf("no message: %v", update))
		return
	}

	if message.From == nil {
		app.logger.Warn(fmt.Sprintf("message without from: %v", update))
		return
	}

	logger := app.logger.With(slog.String("from", message.From.UserName), slog.Int64("id", message.From.ID))

	user := app.getUser(message.From.ID)

	if user == "" {
		logger.Info(fmt.Sprintf("unknown user, msg: %s", message.Text))
		msg := tg.NewMessage(message.Chat.ID, "с незнакомыми не разговариваю")
		_, err := app.bot.Send(msg)

		if err != nil {
			logger.Error("can't send message", slog.Any("error", err))
		}
		return
	}

	// location
	if loc := getLocation(update); loc != nil {
		logger.Info(fmt.Sprintf("location: %f %f", loc.Latitude, loc.Longitude))
		if app.conf.String("cot.server") != "" {
			evt := makeEvent(fmt.Sprintf("tg-%d", message.From.ID), user, loc.Latitude, loc.Longitude)
			app.sendCotMessage(evt)
			return
		}
	}

	if message.Text == "" {
		logger.Info("empty message")
		return
	}

	logger.Info("message: " + message.Text)

	replText := ""

	if message.ReplyToMessage != nil {
		replText = message.ReplyToMessage.Text
	}

	ans := app.ans.CheckAnswer(user, message.Text, replText)
	var msg tg.Chattable

	if ans.Photo != "" {
		msg = tg.NewPhoto(message.Chat.ID, tg.FilePath(ans.Photo))
	} else {
		msg = tg.NewMessage(message.Chat.ID, ans.Msg)
	}

	//msg.ReplyToMessageID = update.Message.MessageID

	_, err := app.bot.Send(msg)

	if err != nil {
		logger.Error("can't send message", slog.Any("error", err))
	}
}

func (app *App) getUser(id int64) string {
	sid := strconv.Itoa(int(id))
	for name, uid := range app.conf.StringMap("users") {
		if uid == sid {
			return name
		}
	}
	return ""
}

func makeEvent(id, name string, lat, lon float64) *cot.Event {
	evt := cot.XMLBasicMsg("a-f-G", id, time.Hour)
	evt.Point.Lon = lon
	evt.Point.Lat = lat

	evt.AddGroup("Red", "Team member")
	evt.AddCallsign(name, "", false)
	evt.AddVersion("Telegram bot", runtime.GOARCH, runtime.GOOS, gitRevision)

	return evt
}

func getLocation(update tg.Update) *tg.Location {
	if update.EditedMessage != nil {
		return update.EditedMessage.Location
	}
	if update.Message != nil {
		return update.Message.Location
	}
	return nil
}

func (app *App) IdByName(name string) (int64, error) {
	nl := strings.ToLower(name)

	for _, gr := range []string{"users", "groups"} {
		for n, id := range app.conf.IntMap(gr) {
			if strings.ToLower(n) == nl {
				return int64(id), nil
			}
		}
	}

	return 0, fmt.Errorf("not found")
}

func (app *App) sendCotMessage(evt *cot.Event) {
	if app.conf.String("cot.server") == "" {
		return
	}

	msg, err := xml.Marshal(evt)
	if err != nil {
		app.logger.Error("marshal error", slog.Any("error", err))
		return
	}

	conn, err := net.Dial("udp", app.conf.String("cot.server"))
	if err != nil {
		app.logger.Error("connection error", slog.Any("error", err))
		return
	}

	conn.SetWriteDeadline(time.Now().Add(time.Second * 10))
	if _, err := conn.Write(msg); err != nil {
		app.logger.Error("write error", slog.Any("error", err))
	}
	conn.Close()
}

func main() {
	conf := NewAppConfig()
	conf.Load("botik.yml")

	var h slog.Handler
	if conf.Debug() {
		h = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	} else {
		h = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	}

	slog.SetDefault(slog.New(h))

	app := NewApp(conf)
	app.Run()
}
