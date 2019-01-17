package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/go-telegram-bot-api/telegram-bot-api"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type App struct {
	bot      *tgbotapi.BotAPI
	exitChan chan bool
	users    map[string]string
}

func NewApp() (app *App) {
	app = &App{}

	return
}

func (app *App) GetUpdatesChannel() tgbotapi.UpdatesChannel {
	app.bot.RemoveWebhook()

	if webhook := viper.GetString("webhook"); webhook != "" {
		log.Infof("starting webhook %s", webhook)

		res, err := app.bot.SetWebhook(tgbotapi.NewWebhook(webhook))

		if err != nil {
			panic("can't add webhook")
		}

		log.Info(res.Description)

		info, err := app.bot.GetWebhookInfo()
		if err != nil {
			log.Fatal(err)
		}
		if info.LastErrorDate != 0 {
			log.Infof("Telegram callback failed: %s", info.LastErrorMessage)
		}

		log.Infof("start listener on %s, path %s", viper.GetString("webhook_listen"), viper.GetString("webhook_path"))
		go http.ListenAndServe(viper.GetString("webhook_listen"), nil)

		return app.bot.ListenForWebhook(viper.GetString("webhook_path"))
	}

	log.Info("start polling")
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
			Proxy: http.ProxyURL(proxyUrl),
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
	log.Infof("registering %s", app.bot.Self.String())

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

	logger := log.WithFields(log.Fields{"from": update.Message.From.UserName, "id": update.Message.From.ID})
	logger.Infof("[%s] %s", update.Message.From.UserName, update.Message.Text)

	var ans string
	var found = false
	for _, id := range app.users {
		if id == strconv.Itoa(update.Message.From.ID) {
			found = true
			break
		}
	}

	if !found {
		ans = "с незнакомыми не разговариваю"
	} else {
		ans = CheckAnswer(update.Message.Text)
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, ans)
	//msg.ReplyToMessageID = update.Message.MessageID

	_, e := app.bot.Send(msg)

	if e != nil {
		logger.Errorf("can't send message: %s", e.Error())
	}
}

func main() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	viper.SetConfigName("botik")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()

	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	app := NewApp()
	app.users = viper.GetStringMapString("users")
	app.Run()
}
