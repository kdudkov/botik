package main

import (
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
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
	bot       *tgbotapi.BotAPI
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

func (app *App) GetUpdatesChannel() tgbotapi.UpdatesChannel {
	if _, err := app.bot.RemoveWebhook(); err != nil {
		app.logger.Errorf("can't remove webhook: %v", err)
	}

	if webhook := viper.GetString("webhook.ext"); webhook != "" {
		app.logger.Infof("starting webhook %s", webhook)

		res, err := app.bot.SetWebhook(tgbotapi.NewWebhook(webhook))

		if err != nil {
			panic("can't add webhook")
		}

		app.logger.Info(res.Description)

		info, err := app.bot.GetWebhookInfo()
		if err != nil {
			app.logger.Fatal(err)
		}
		if info.LastErrorDate != 0 {
			app.logger.Infof("Telegram callback failed: %s", info.LastErrorMessage)
		}

		app.logger.Infof("start listener on %s, path %s", viper.GetString("webhook.listen"), viper.GetString("webhook.path"))
		go func() {
			if err := http.ListenAndServe(viper.GetString("webhook.listen"), nil); err != nil {
				panic(err)
			}
		}()

		return app.bot.ListenForWebhook(viper.GetString("webhook.path"))
	}

	app.logger.Info("start polling")
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	ch, err := app.bot.GetUpdatesChan(u)
	if err != nil {
		panic("can't add webhook")
	}
	return ch
}

func (app *App) quit() {
	app.bot.StopReceivingUpdates()
	if viper.GetString("webhook") != "" {
		if _, err := app.bot.RemoveWebhook(); err != nil {
			app.logger.Errorf("can't remove webhook: %v", err)
		}
	}
}

func (app *App) Run() {
	for _, ans := range answer.Answers {
		ans.AddLogger(app.logger)
	}

	var err error

	if proxy := viper.GetString("proxy"); proxy != "" {
		proxyUrl, _ := url.Parse(proxy)
		myClient := &http.Client{Timeout: time.Second * 10, Transport: &http.Transport{
			Proxy:                 http.ProxyURL(proxyUrl),
			ResponseHeaderTimeout: time.Second * 5,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		}}
		app.bot, err = tgbotapi.NewBotAPIWithClient(viper.GetString("token"), myClient)
	} else {
		myClient := &http.Client{Timeout: time.Second * 10, Transport: &http.Transport{
			ResponseHeaderTimeout: time.Second * 5,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		}}
		app.bot, err = tgbotapi.NewBotAPIWithClient(viper.GetString("token"), myClient)
	}

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

	updates := app.GetUpdatesChannel()
	for {
		select {
		case update := <-updates:
			go app.Process(update)
		case <-sigc:
			app.logger.Info("quit")
			app.quit()
			break
		}
	}
}

func (app *App) Process(update tgbotapi.Update) {
	var message *tgbotapi.Message

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
		msg := tgbotapi.NewMessage(message.Chat.ID, "с незнакомыми не разговариваю")
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
	var msg tgbotapi.Chattable

	if ans.Photo != "" {
		msg = tgbotapi.NewPhotoUpload(message.Chat.ID, ans.Photo)
	} else {
		msg = tgbotapi.NewMessage(message.Chat.ID, ans.Msg)
	}

	//msg.ReplyToMessageID = update.Message.MessageID

	_, err := app.bot.Send(msg)

	if err != nil {
		logger.Errorf("can't send message: %s", err.Error())
	}
}

func (app *App) getUser(id int) string {
	sid := strconv.Itoa(id)
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

func getLocation(update tgbotapi.Update) *tgbotapi.Location {
	if update.EditedMessage != nil {
		return update.EditedMessage.Location
	}
	if update.Message != nil {
		return update.Message.Location
	}
	return nil
}

func (app *App) sendCotMessage(evt *cotxml.Event) {
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
