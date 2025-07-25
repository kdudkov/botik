package main

import (
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
	"github.com/spf13/viper"
)

var (
	gitRevision string
	gitBranch   string
)

type App struct {
	bot         *tg.BotAPI
	users       map[string]string
	groups      map[string]string
	notifyUsers []string
	logger      *slog.Logger
	am          *alert.AlertManager
	ans         *answer.AnswerManager
}

func NewApp() *App {
	app := &App{
		logger: slog.Default(),
		ans:    answer.New(),
	}

	app.am = alert.NewManager(slog.Default().With("logger", "alerts"), app.alertNotifier)

	if h := viper.GetString("mahno.host"); h != "" {
		if err := app.ans.RegisterAnswer("light", answer.NewLight(app.logger, h)); err != nil {
			panic(err.Error())
		}
	}

	if h := viper.GetString("camera.file"); h != "" {
		if err := app.ans.RegisterAnswer("cam", answer.NewCamera(app.logger, h)); err != nil {
			panic(err.Error())
		}
	}

	app.ans.RegisterAnswer("alerts", answer.NewAlerts(app.logger, app.am))

	return app
}

func (app *App) GetUpdatesChannel() (tg.UpdatesChannel, error) {
	if webhook := viper.GetString("webhook.ext"); webhook != "" {
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
			return nil, fmt.Errorf(info.LastErrorMessage)
		}

		app.logger.Info(fmt.Sprintf("start listener on %s, path %s", viper.GetString("webhook.listen"), viper.GetString("webhook.path")))
		go func() {
			if err := http.ListenAndServe(viper.GetString("webhook.listen"), nil); err != nil {
				panic(err)
			}
		}()

		return app.bot.ListenForWebhook(viper.GetString("webhook.path")), nil
	}

	app.logger.Info("start polling")
	app.removeWebhook()
	u := tg.NewUpdate(0)
	u.Timeout = 60

	return app.bot.GetUpdatesChan(u), nil
}

func (app *App) quit() {
	app.bot.StopReceivingUpdates()
	if webhook := viper.GetString("webhook.ext"); webhook != "" {
		app.removeWebhook()
	}
}

func (app *App) removeWebhook() {
	if _, err := app.bot.Request(tg.WebhookConfig{URL: nil}); err != nil {
		app.logger.Error("remove webhook error", "error", err)
	}
}

func (app *App) Run() {
	for k := range app.users {
		app.logger.Info("user " + k)
	}

	var err error
	app.bot, err = tg.NewBotAPI(viper.GetString("token"))

	if err != nil {
		panic("can't start bot " + err.Error())
	}
	app.logger.Info("registering " + app.bot.Self.String())

	go runHttpServer(app)

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)

	app.am.Start()

	updates, err := app.GetUpdatesChannel()

	if err != nil {
		app.logger.Error(err.Error())
		return
	}

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

func (app *App) alertNotifier(text string) {
	for _, user := range app.notifyUsers {
		id, err := app.IdByName(user)

		if err != nil {
			app.logger.Error("invalid user "+user, "error", err)
			continue
		}

		go func(logger *slog.Logger, id int64, text string) {
			logger.Info("sending notification")

			if _, err := app.sendTgWithMode(id, text, "HTML"); err != nil {
				logger.Error("error send message", "error", err)
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

	logger := app.logger.With("from", message.From.UserName, "id", message.From.ID)

	user := app.getUser(message.From.ID)

	if user == "" {
		logger.Info(fmt.Sprintf("unknown user, msg: %s", message.Text))
		msg := tg.NewMessage(message.Chat.ID, "с незнакомыми не разговариваю")
		_, err := app.bot.Send(msg)

		if err != nil {
			logger.Error("can't send message", "error", err.Error())
		}
		return
	}

	// location
	if loc := getLocation(update); loc != nil {
		logger.Info(fmt.Sprintf("location: %f %f", loc.Latitude, loc.Longitude))
		if viper.GetString("cot.server") != "" {
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
		logger.Error("can't send message", "error", err.Error())
	}
}

func (app *App) getUser(id int64) string {
	sid := strconv.Itoa(int(id))
	for name, uid := range app.users {
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
	if ids, ok := app.users[strings.ToLower(name)]; ok {
		if id, err := strconv.ParseInt(ids, 10, 64); err == nil {
			return id, nil
		} else {
			app.logger.Error("can't parse int " + ids)
			return 0, err
		}
	}

	if ids, ok := app.groups[strings.ToLower(name)]; ok {
		if id, err := strconv.ParseInt(ids, 10, 64); err == nil {
			return id, nil
		} else {
			app.logger.Error("can't parse int " + ids)
			return 0, err
		}
	}

	return 0, fmt.Errorf("not found")
}

func (app *App) sendCotMessage(evt *cot.Event) {
	if viper.GetString("cot.server") == "" {
		return
	}

	msg, err := xml.Marshal(evt)
	if err != nil {
		app.logger.Error("marshal error", "error", err)
		return
	}

	conn, err := net.Dial("udp", viper.GetString("cot.server"))
	if err != nil {
		app.logger.Error("connection error", "error", err)
		return
	}

	conn.SetWriteDeadline(time.Now().Add(time.Second * 10))
	if _, err := conn.Write(msg); err != nil {
		app.logger.Error("write error", "error", err)
	}
	conn.Close()
}

func main() {
	viper.SetConfigName("botik")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})

	slog.SetDefault(slog.New(h))

	app := NewApp()
	app.users = viper.GetStringMapString("users")
	app.groups = viper.GetStringMapString("groups")
	app.notifyUsers = viper.GetStringSlice("notify")

	app.Run()
}
