package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"botik/answer"
)

var (
	gitRevision string
	gitBranch   string
)

type App struct {
	bot    *tgbotapi.BotAPI
	users  map[string]string
	groups map[string]string
	logger *zap.SugaredLogger
}

func NewApp(logger *zap.SugaredLogger) (app *App) {
	app = &App{logger: logger}
	return
}

func (app *App) GetUpdatesChannel() tgbotapi.UpdatesChannel {
	if _, err := app.bot.RemoveWebhook(); err != nil {
		app.logger.Errorf("can't remove webhook", err)
	}

	if webhook := viper.GetString("webhook"); webhook != "" {
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

		app.logger.Infof("start listener on %s, path %s", viper.GetString("webhook_listen"), viper.GetString("webhook_path"))
		go http.ListenAndServe(viper.GetString("webhook_listen"), nil)

		return app.bot.ListenForWebhook(viper.GetString("webhook_path"))
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
			app.logger.Errorf("can't remove webhook", err)
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
	if update.Message == nil {
		return
	}

	logger := app.logger.With("from", update.Message.From.UserName, "id", update.Message.From.ID)
	logger.Infof("[%s] %s", update.Message.From.UserName, update.Message.Text)

	var ans *answer.Answer
	var user string
	for u, id := range app.users {
		if id == strconv.Itoa(update.Message.From.ID) {
			user = u
			break
		}
	}

	if user == "" {
		logger.Infof("invalid user %s", update.Message.From.UserName)
		ans = answer.TextAnswer("с незнакомыми не разговариваю")
	} else {
		ans = answer.CheckAnswer(user, update.Message.Text)
	}

	var msg tgbotapi.Chattable

	if ans.Photo != "" {
		msg = tgbotapi.NewPhotoUpload(update.Message.Chat.ID, ans.Photo)
	} else {
		msg = tgbotapi.NewMessage(update.Message.Chat.ID, ans.Msg)
	}

	//msg.ReplyToMessageID = update.Message.MessageID

	_, err := app.bot.Send(msg)

	if err != nil {
		logger.Errorf("can't send message: %s", err.Error())
	}
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
