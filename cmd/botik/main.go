package main

import (
	"encoding/xml"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/kdudkov/goatak/cotxml"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"botik/answer"
)

var (
	gitRevision string
	gitBranch   string
)

type App struct {
	bot       *tg.BotAPI
	users     map[string]string
	groups    map[string]string
	logger    *zap.SugaredLogger
	alerts    sync.Map
	alertUrls chan string
}

func NewApp(logger *zap.SugaredLogger) (app *App) {
	app = &App{
		logger:    logger,
		alertUrls: make(chan string, 20),
		alerts:    sync.Map{},
	}
	return
}

func (app *App) GetUpdatesChannel() (tg.UpdatesChannel, error) {
	if webhook := viper.GetString("webhook.ext"); webhook != "" {
		app.logger.Infof("starting webhook %s", webhook)

		wh, _ := tg.NewWebhook(webhook)
		if _, err := app.bot.Request(wh); err != nil {
			return nil, err
		}

		info, err := app.bot.GetWebhookInfo()
		if err != nil {
			return nil, err
		}

		if info.LastErrorDate != 0 {
			app.logger.Errorf("Telegram callback failed: %s", info.LastErrorMessage)
			return nil, fmt.Errorf(info.LastErrorMessage)
		}

		app.logger.Infof("start listener on %s, path %s", viper.GetString("webhook.listen"), viper.GetString("webhook.path"))
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
		app.logger.Errorf("remove webhook error: %v", err)
	}
}

func (app *App) Run() {
	for _, ans := range answer.Answers {
		ans.AddLogger(app.logger)
	}

	var err error

	app.bot, err = tg.NewBotAPI(viper.GetString("token"))

	if err != nil {
		panic("can't start bot " + err.Error())
	}
	app.logger.Infof("registering %s", app.bot.Self.String())

	go runHttpServer(app)

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	go func() {
		for alertUrl := range app.alertUrls {
			go app.processNewUrl(alertUrl)
		}
	}()

	go app.alertProcessor()

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

func (app *App) Process(update tg.Update) {
	var message *tg.Message

	if update.EditedMessage != nil {
		message = update.EditedMessage
	} else {
		message = update.Message
	}

	if message == nil {
		app.logger.Warnf("no message: %v", update)
		return
	}

	if message.From == nil {
		app.logger.Warnf("message without from: %v", update)
		return
	}

	logger := app.logger.With("from", message.From.UserName, "id", message.From.ID)

	user := app.getUser(message.From.ID)

	if user == "" {
		logger.Infof("unknown user, msg: %s", message.Text)
		msg := tg.NewMessage(message.Chat.ID, "с незнакомыми не разговариваю")
		_, err := app.bot.Send(msg)

		if err != nil {
			logger.Errorf("can't send message: %s", err.Error())
		}
		return
	}

	// location
	if loc := getLocation(update); loc != nil {
		logger.Infof("location: %f %f", loc.Latitude, loc.Longitude)
		if viper.GetString("cot.server") != "" {
			evt := makeEvent(fmt.Sprintf("tg-%d", message.From.ID), user, loc.Latitude, loc.Longitude)
			app.sendCotMessage(evt)
			return
		}
	}

	if message.Text == "" {
		logger.Infof("empty message")
		return
	}

	if update.Message == nil {
		// edited message
		return
	}

	logger.Infof("message: %s", message.Text)

	ans := answer.CheckAnswer(user, message.Text)
	var msg tg.Chattable

	if ans.Photo != "" {
		msg = tg.NewPhoto(message.Chat.ID, tg.FilePath(ans.Photo))
	} else {
		msg = tg.NewMessage(message.Chat.ID, ans.Msg)
	}

	//msg.ReplyToMessageID = update.Message.MessageID

	_, err := app.bot.Send(msg)

	if err != nil {
		logger.Errorf("can't send message: %s", err.Error())
	}
}

func (app *App) getUser(id int64) string {
	sid := strconv.Itoa(int(id))
	for u, id := range app.users {
		if id == sid {
			return u
		}
	}
	return ""
}

func makeEvent(id, name string, lat, lon float64) *cotxml.Event {
	evt := cotxml.BasicMsg("a-f-G", id, time.Hour)
	evt.Detail = cotxml.Detail{
		Group:   &cotxml.Group{Name: "Red", Role: "Team Member"},
		Contact: &cotxml.Contact{Callsign: name},
	}
	evt.Point.Lon = lon
	evt.Point.Lat = lat
	evt.Detail.TakVersion = &cotxml.TakVersion{}
	evt.Detail.TakVersion.Platform = "Telegram bot"
	evt.Detail.TakVersion.Version = "0.1"
	evt.Detail.TakVersion.Os = "linux-amd64"

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
	if ids, ok := app.users[name]; ok {
		if id, err := strconv.ParseInt(ids, 10, 64); err == nil {
			return id, nil
		} else {
			app.logger.Errorf("can't parse int %s", ids)
			return 0, err
		}
	}

	if ids, ok := app.groups[name]; ok {
		if id, err := strconv.ParseInt(ids, 10, 64); err == nil {
			return id, nil
		} else {
			app.logger.Errorf("can't parse int %s", ids)
			return 0, err
		}
	}

	return 0, fmt.Errorf("not found")
}

func (app *App) sendCotMessage(evt *cotxml.Event) {
	if viper.GetString("cot.server") == "" {
		return
	}

	msg, err := xml.Marshal(evt)
	if err != nil {
		app.logger.Errorf("marshal error: %v", err)
		return
	}

	conn, err := net.Dial("udp", viper.GetString("cot.server"))
	if err != nil {
		app.logger.Errorf("connection error: %v", err)
		return
	}

	conn.SetWriteDeadline(time.Now().Add(time.Second * 10))
	if _, err := conn.Write(msg); err != nil {
		app.logger.Errorf("write error: %v", err)
	}
	conn.Close()
}

func main() {
	viper.SetConfigName("botik")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	config := zap.NewProductionConfig()
	config.Encoding = "console"

	logger, err := config.Build()
	defer logger.Sync()

	if err != nil {
		panic(err.Error())
	}

	sl := logger.Sugar()
	sl.Infof("starting app branch %s, rev %s", gitBranch, gitRevision)

	app := NewApp(sl)
	app.users = viper.GetStringMapString("users")
	app.groups = viper.GetStringMapString("groups")

	app.Run()
}
