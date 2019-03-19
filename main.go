package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"time"

	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	gitRevision string
	gitBranch   string
)

type App struct {
	bot      *tgbotapi.BotAPI
	exitChan chan bool
	users    map[string]string
	logger   *zap.SugaredLogger
}

func NewApp(logger *zap.SugaredLogger) (app *App) {
	app = &App{logger: logger}
	return
}

func (app *App) GetUpdatesChannel() tgbotapi.UpdatesChannel {
	app.bot.RemoveWebhook()

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
	if viper.GetString("webhook") != "" {
		app.bot.RemoveWebhook()
	}
	app.bot.StopReceivingUpdates()
}

func (app *App) Run() {
	var err error

	if proxy := viper.GetString("proxy"); proxy != "" {
		proxyUrl, _ := url.Parse(proxy)
		myClient := &http.Client{Transport: &http.Transport{
			Proxy:                 http.ProxyURL(proxyUrl),
			ResponseHeaderTimeout: time.Second * 30,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		}}
		app.bot, err = tgbotapi.NewBotAPIWithClient(viper.GetString("token"), myClient)
	} else {
		app.bot, err = tgbotapi.NewBotAPI(viper.GetString("token"))
	}

	if err != nil {
		panic("can't start bot " + err.Error())
	}
	app.logger.Infof("registering %s", app.bot.Self.String())

	updates := app.GetUpdatesChannel()

	for {
		select {
		case update := <-updates:
			go app.Process(update)
		case <-app.exitChan:
			app.quit()
			break
		}

		runtime.Gosched()
	}
}

func (app *App) Process(update tgbotapi.Update) {
	if update.Message == nil {
		return
	}

	logger := app.logger.With("from", update.Message.From.UserName, "id", update.Message.From.ID)
	logger.Infof("[%s] %s", update.Message.From.UserName, update.Message.Text)

	var ans string
	var user string
	for u, id := range app.users {
		if id == strconv.Itoa(update.Message.From.ID) {
			user = u
			break
		}
	}

	if user == "" {
		ans = "с незнакомыми не разговариваю"
	} else {
		ans = CheckAnswer(user, update.Message.Text)
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, ans)
	//msg.ReplyToMessageID = update.Message.MessageID

	_, e := app.bot.Send(msg)

	if e != nil {
		logger.Errorf("can't send message: %s", e.Error())
	}
}

func main() {
	viper.SetConfigName("botik")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()

	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	config := zap.NewProductionConfig()
	config.Encoding = "console"
	logger, err := config.Build()

	if err != nil {
		panic(err.Error())
	}

	sl := logger.Sugar()
	sl.Infof("starting app branch %s, rev %s", gitBranch, gitRevision)

	app := NewApp(sl)
	app.users = viper.GetStringMapString("users")

	go runHttpServer(app)

	for _, ans := range answers {
		ans.AddLogger(sl)
	}

	app.Run()
}
