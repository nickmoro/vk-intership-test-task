package main

import (
	"net/http"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	WebhookURLPrefix = "http://"
	WebhookURL       = "87.239.111.2:8888"
)

func main() {
	// init zapSugaredLogger
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	zapLogger, _ := config.Build()
	logger := zapLogger.Sugar()
	defer logger.Sync()

	// init tgbot
	bot, err := tgbotapi.NewBotAPI(BotToken)
	if err != nil {
		logger.Panic("tgbotapi.NewBotAPI:", err)
	}

	_, err = bot.SetWebhook(tgbotapi.NewWebhook(WebhookURLPrefix + WebhookURL))
	if err != nil {
		logger.Panic("bot.SetWebhook:", err)
	}

	updates := bot.ListenForWebhook("/")
	go func() {
		http.ListenAndServe(WebhookURL, nil)
		logger.Info("Started listening", WebhookURL)
	}()

	logger.Infof(`Bot "%v" started`, bot.Self.UserName)

	for update := range updates {
		logger.Debug("got update =", update)
	}

}
