package main

import (
	"net/http"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	WebhookURL = "https://a11c-95-73-2-182.ngrok-free.app"
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

	_, err = bot.SetWebhook(tgbotapi.NewWebhook(WebhookURL))
	if err != nil {
		logger.Panic("bot.SetWebhook:", err)
	}

	updates := bot.ListenForWebhook("/")
	go func() {
		logger.Info("Listening :8080")
		logger.Fatal(http.ListenAndServe(":8080", nil))
	}()

	logger.Infof(`Bot "%v" started`, bot.Self.UserName)

	for update := range updates {
		logger.Debug("got update =", update)
		logger.Debug("Message from ", update.Message.Chat.ID)
	}
}
