package main

import (
	"log"
	"net/http"

	"tgbot/internal/handler"
	"tgbot/internal/repo"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	WebhookURL      = "https://70b7-176-59-164-45.ngrok-free.app"
	NumberOfWorkers = 4
	MongoURI        = "mongodb://mongodb:27017"
	DatabaseName    = "tgbot"
	CollectionName  = "users"
)

func main() {
	// zapSugaredLogger
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	zapLogger, err := config.Build()
	if err != nil {
		log.Fatal(err)
	}
	logger := zapLogger.Sugar()
	defer logger.Sync()

	// mongoDB
	repo, err := repo.NewMongoRepo(MongoURI, DatabaseName, CollectionName)
	if err != nil {
		logger.Panic(err)
	}

	// tgbot
	bot, err := tgbotapi.NewBotAPI(BotToken)
	if err != nil {
		logger.Panic(err)
	}

	_, err = bot.SetWebhook(tgbotapi.NewWebhook(WebhookURL))
	if err != nil {
		logger.Panic(err)
	}

	updates := bot.ListenForWebhook("/")
	logger.Infof(`Bot "%v" started`, bot.Self.UserName)

	botHandler := handler.NewBotHandler(logger, bot, repo)
	for i := 0; i < NumberOfWorkers; i++ {
		go func(workerNum int) {
			logger.Infof(`Worker %v started`, workerNum)
			botHandler.HandleUpdates(updates)
		}(i)
	}

	logger.Info("Listening :8080")
	logger.Fatal(http.ListenAndServe(":8080", nil))
}
